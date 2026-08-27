package chatGPT

import (
	"300Bot/conf"
	"300Bot/function/ambient"
	"300Bot/function/chatctx"
	"300Bot/function/memory/inline"
	"300Bot/function/memory/recall"
	"300Bot/function/scheduler"
	"300Bot/send"
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// ambientScheduler 自主插话独立调度池：与交互聊天池隔离，插话任务绝不挤占 @ 回复的并发
var ambientScheduler *scheduler.Scheduler

// initAmbient 在 chatGPT 包 init 末尾调用：建独立调度池，并把闸门参数与决策回调注入 ambient 包
func initAmbient() {
	ambientScheduler = scheduler.New("ambient", 1, 2, 10*time.Minute)
	// 白名单开关配置层已保证非 nil（未配置补默认 true）
	ambient.Configure(conf.Ambient.AmbientEnabled, *conf.Ambient.AmbientWhitelistEnabled, conf.Ambient.AmbientGroups, conf.Ambient.AmbientChance,
		conf.Ambient.AmbientCooldownSec, conf.Ambient.AmbientThinkMinSec, conf.Ambient.AmbientThinkMaxSec, conf.Config.BotQQ)
	ambient.SetDecideCallback(AddAmbientPlan)
}

// ambientProtocolPrompt 环境插话输出协议：与显式触发的关键差异是 should_reply 由模型真实决策；
// 基调是绝大多数时候保持沉默，避免为插话而插话
const ambientProtocolPrompt = "你正在旁观一个群聊，没有人@你或叫你的名字。你必须只输出 JSON，不要输出任何其他文字。格式：{\"should_reply\":false,\"reply\":\"你的插话内容\",\"memory\":[]}。" +
	"should_reply 由你自己决定：大多数时候你应该保持沉默，输出 false；只有当话题与你有关、或者话题有趣、或者你能自然地接一句话、或有人可能需要你的时候，才输出 true。" +
	`可以偶尔幽默、吐槽、接梗，但不要持续玩梗。

不要频繁使用：
“哈哈”“嘿嘿”“噗”“哎哟喂”等语气词。

不要频繁使用网络流行语。

不要连续使用表情符号。

正常情况下可以完全不使用表情符号。

不要为了显得亲切而频繁称呼对方名字。

不要刻意夸张情绪，不要把普通对话演绎成段子。

用户比较平静时，叁柏也应该比较平静。
用户明显在开玩笑时，可以适当接梗。
用户认真讨论时，应该认真回应。

	` +
	"不要接你自己刚说过的话，不要为了插话而插话。若 should_reply 为 true，reply 要口语、简短、克制，通常一句话就够；若为 false，reply 留空字符串。" +
	"memory 数组是从这次群聊中顺带提取的值得长期记住的记忆候选（每条一句完整陈述句，涉及人物必须引用QQ号，没有可提取的内容就给空数组）。"

// AddAmbientPlan ambient 闸门放行后的决策入口：提交到独立调度池，所有失败分支静默（不发任何兜底话术）。
// 队列满丢弃也静默——插话是可有可无的，绝不能像交互路径那样发 replyBusy 提示
func AddAmbientPlan(groupIdStr string) {
	submitted := ambientScheduler.Submit(groupIdStr, func() {
		// 执行时刻取新鲜快照：闸门思考延迟已过数秒，窗口内容比放行时更新
		ambientJSON := chatctx.SnapshotJSON(groupIdStr)
		if ambientJSON == "" {
			log.Printf("ambient skip group=%s reason=empty_snapshot", groupIdStr)
			return
		}
		parsed, err := AskForChatGPTAmbient(groupIdStr, ambientJSON)
		if err != nil {
			return
		}
		if !parsed.ShouldReply || parsed.Reply == "" {
			log.Printf("ambient silent group=%s reason=model_decided_no_reply", groupIdStr)
			return
		}
		groupId, err := strconv.ParseFloat(groupIdStr, 64)
		if err != nil {
			return
		}
		send.SendGroupPost(groupId, parsed.Reply)
		// NapCat 不回推机器人自己的群消息，插话也需手动入窗
		chatctx.AppendBotReply(groupIdStr, parsed.Reply)
		ambient.NotifyReplied(groupIdStr)
	})
	if !submitted {
		log.Printf("ambient drop group=%s reason=queue_full", groupIdStr)
	}
}

// AskForChatGPTAmbient 环境插话决策：完全不读写 sessions（避免决策 prompt 污染交互上下文），
// 不走 Agent 工具循环（也规避 tools 与 JSON-only 输出提示互相抑制的已知坑）；
// 人格 + 插话协议 + 窗口快照组成一次性决策请求，输出按严格协议解析（失败即沉默）
func AskForChatGPTAmbient(groupIdStr, ambientJSON string) (inline.ChatReply, error) {
	var empty inline.ChatReply
	reqMessages := []openai.ChatCompletionMessage{
		{Role: "system", Content: basePersonalityPrompt},
		{Role: "system", Content: ambientProtocolPrompt},
		{Role: "system", Content: "【对话环境（JSON）】\n" + ambientJSON},
	}
	model := conf.Config.ChatModel
	log.Printf("ambient request group=%s model=%s", groupIdStr, model)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Config.LLMTimeoutSec)*time.Second)
	defer cancel()
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: reqMessages,
	}
	// 此路径不带 tools，json_object 无抑制问题
	if conf.Config.ChatJsonMode {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	start := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Printf("ambient ChatCompletion error group=%s model=%s err=%v", groupIdStr, model, err)
		return empty, err
	}
	if len(resp.Choices) == 0 {
		return empty, fmt.Errorf("empty choices")
	}
	parsed := inline.NormalizeReply(inline.ParseAmbientReply(resp.Choices[0].Message.Content))
	log.Printf("ambient response group=%s model=%s cost_ms=%d should_reply=%v reply_preview=%s",
		groupIdStr, model, time.Since(start).Milliseconds(), parsed.ShouldReply, recall.PreviewText(parsed.Reply, 80))
	return parsed, nil
}
