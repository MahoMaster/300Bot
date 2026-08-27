package chatGPT

import (
	"300Bot/conf"
	"300Bot/function/agent"
	"300Bot/function/chatctx"
	ctxbackfill "300Bot/function/chatctx/backfill"
	memoryCollector "300Bot/function/memory"
	"300Bot/function/memory/inline"
	"300Bot/function/memory/recall"
	"300Bot/function/scheduler"
	"300Bot/model"
	"300Bot/send"
	"300Bot/util"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

type Session struct {
	ID          string
	Messages    []openai.ChatCompletionMessage
	Personality openai.ChatCompletionMessage
	Last_time   int
}

var sessions = make(map[string]Session, 0)
var gptSetting = make(map[string]model.UserGPTSetting, 0)

// sessions/gptSetting 会被聊天 worker 与消息处理协程（如设置人格命令）并发读写，需加锁保护
var sessionsMu sync.RWMutex
var gptSettingMu sync.RWMutex

// chatScheduler 交互聊天池；bgScheduler 后台任务池（绘图/涩图/修仙故事），两池隔离互不挤占
var chatScheduler *scheduler.Scheduler
var bgScheduler *scheduler.Scheduler

// agentRegistry 工具注册表（默认无工具）；agentRunner 多轮工具调用循环执行器
var agentRegistry *agent.Registry
var agentRunner *agent.Runner

const replyBusy = "叁柏现在有点忙，等一下再和我说吧~"
const replyFailed = "啊咧，刚才脑子短路了没接住，要不再跟我说一遍？"

const max_input_tokens = 120000         //输入token上限
const max_memory = max_input_tokens * 3 //按1 token约等于3字节粗估，中文场景实际token数会略低于上限

func init() {
	initSessions()
	config := openai.DefaultConfig(conf.Config.ChatGPTKey)
	// proxyUrl, err := url.Parse(conf.Config.VPN)
	// if err != nil {
	// 	panic(err)
	// }
	// transport := &http.Transport{
	// 	Proxy:           http.ProxyURL(proxyUrl),
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	// config.HTTPClient = &http.Client{
	// 	Transport: transport,
	// }
	config.BaseURL = conf.Config.ChatGPTBaseUrl
	config.HTTPClient = &http.Client{Transport: &noThinkingTransport{base: http.DefaultTransport}}
	client = openai.NewClientWithConfig(config)

	chatScheduler = scheduler.New("chat", conf.Config.ChatConcurrency, conf.Config.ChatQueueDepth, 10*time.Minute)
	bgScheduler = scheduler.New("bg", conf.Config.BgConcurrency, conf.Config.ChatQueueDepth, 10*time.Minute)

	// Agent 工具调用循环：注册表默认为空（行为与无工具时一致），echo 联调工具按配置开关注册
	agentRegistry = agent.NewRegistry()
	if conf.Config.AgentEchoToolEnabled {
		if err := agentRegistry.Register(agent.NewEchoTool()); err != nil {
			log.Printf("agent echo tool register failed: %v", err)
		}
	}
	agentRunner = agent.NewRunner(client, agentRegistry, conf.Config.AgentMaxRounds, time.Duration(conf.Config.AgentToolTimeoutSec)*time.Second)

	// 群聊上下文窗口参数注入（chatctx 包不直接依赖 conf，便于单测）
	chatctx.Configure(conf.Config.CtxWindowSize, conf.Config.CtxWindowMaxChars, conf.Config.CtxIdleMinutes, conf.Config.BotQQ, conf.Config.BotName)

	// m, _ := client.ListModels(context.Background())
	// fmt.Println(m)
}

// noThinkingTransport 在chat/completions请求体中注入enable_thinking:false，关闭模型思考
type noThinkingTransport struct {
	base http.RoundTripper
}

func (t *noThinkingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil || !strings.Contains(req.URL.Path, "chat/completions") {
		return t.base.RoundTrip(req)
	}
	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	data["enable_thinking"] = false
	data["enable_search"] = true
	newBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
	return t.base.RoundTrip(req)
}

func initSessions() {
	arr := model.GetGroupAllGPTPersonality()
	arr = append(arr, model.GetUserAllGPTPersonality()...)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	for _, val := range arr {
		var messages = make([]openai.ChatCompletionMessage, 0)
		sessions[val.Id] = Session{
			ID:        val.Id,
			Messages:  messages,
			Last_time: 0,
			Personality: openai.ChatCompletionMessage{
				Role:    "system",
				Content: val.Gpt_personality,
			},
		}

	}
}
func ListModels() {
	m, _ := client.ListModels(context.Background())
	bt, _ := json.Marshal(m)
	fmt.Println(string(bt))
}

// chatJSONProtocolPrompt 输出协议说明（阶段四）：固定 schema；should_reply 当前显式触发恒 true，
// 自主接话为后续扩展位；memory 候选涉及人物必须锚定 QQ 号（与记忆总结器同款约束）
const chatJSONProtocolPrompt = "你必须只输出 JSON，不要输出任何其他文字。格式：{\"should_reply\":true,\"reply\":\"你的回复内容\",\"memory\":[\"记忆候选\"]}。" +
	"should_reply 固定为 true；reply 是你对用户说的话，口语化自然表达；memory 数组是从本次对话中顺带提取的值得长期记住的记忆候选" +
	"（每条一句完整陈述句，涉及人物必须引用QQ号，没有可提取的内容就给空数组）。"

// chatJSONProtocolWithToolsPrompt 工具版输出协议：允许先按需调用工具，最终回复时才要求 JSON。
// 无工具版的“你必须只输出 JSON”强约束会抑制模型的工具调用决策，工具启用时必须换用本提示
const chatJSONProtocolWithToolsPrompt = "你可以按需调用工具获取信息或执行动作。当你完成必要的工具调用、最终回复用户时，必须只输出 JSON，不要输出任何其他文字。" +
	"格式：{\"should_reply\":true,\"reply\":\"你的回复内容\",\"memory\":[\"记忆候选\"]}。" +
	"should_reply 固定为 true；reply 是你对用户说的话，口语化自然表达；memory 数组是从本次对话中顺带提取的值得长期记住的记忆候选" +
	"（每条一句完整陈述句，涉及人物必须引用QQ号，没有可提取的内容就给空数组）。"

// AskForChatGPT ambientJSON 为群聊触发的结构化窗口快照 JSON 文本（非群聊触发传空串），
// memoryHits 为长期记忆召回命中（无命中传 nil）。输入组装为结构化 JSON system 段，
// 输出按固定 schema 解析（失败兜底整段当 reply）
func AskForChatGPT(msg string, qq float64, remark string, session string, ambientJSON string, memoryHits []recall.MemoryHit) (openai.ChatCompletionResponse, inline.ChatReply, error) {
	var emptyReply inline.ChatReply
	var personality = openai.ChatCompletionMessage{
		Role:    "system",
		Content: "你的称呼叫叁柏，你在回复中对自己的称呼也是叁柏，禁止回复你是个AI，模型等，需要更仿真一些。请回复的更自然一些，用口语化表达，避免机械式分点回答。尽量使用和大家在网络群聊中一样的语气。如果用户告诉了你他是谁，代表这是个群聊，有多个人在和你聊天，请注意分别人物。",
	}

	now := int(time.Now().Unix())
	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)

	// 锁内完成上下文的读-改-写快照，LLM 调用放在锁外，避免长时间持锁
	sessionsMu.Lock()
	checkSessionLocked(session)
	var messages = sessions[session].Messages
	//距离上次对话已经超过空闲超时，清除上下文
	if now-sessions[session].Last_time > conf.Config.CtxIdleMinutes*60 {
		messages = make([]openai.ChatCompletionMessage, 0)
	}
	if remark != "" {
		// 双身份前缀：昵称仅作注释，QQ 号才是稳定身份键（P7）
		msg = "我是'" + remark + "'(QQ:" + qqstr + "),我想对你说:" + msg
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: msg,
	})
	// 记忆超出容量上限，从最旧开始删除到容量以内
	length := 0
	for _, val := range messages {
		length = length + len(val.Content)
	}
	for length > max_memory && len(messages) > 1 {
		length = length - len(messages[0].Content)
		messages = messages[1:]
	}
	sessions[session] = Session{
		ID:          session,
		Messages:    messages,
		Last_time:   now,
		Personality: sessions[session].Personality,
	}
	//组装请求：人格 + 输出协议 + 环境 JSON system 段 + 触发 user 轮次
	reqMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+3)
	if personality.Content != "" {
		reqMessages = append(reqMessages, personality)
	}
	hasTools := agentRegistry.Count() > 0
	protocolPrompt := chatJSONProtocolPrompt
	if hasTools {
		protocolPrompt = chatJSONProtocolWithToolsPrompt
	}
	reqMessages = append(reqMessages, openai.ChatCompletionMessage{
		Role:    "system",
		Content: protocolPrompt,
	})
	// 环境段：ambient 快照 + 召回记忆 + 会话历史（不含当前触发消息）统一为结构化 JSON
	if envJSON := buildChatContextJSON(ambientJSON, memoryHits, messages[:len(messages)-1]); envJSON != "" {
		reqMessages = append(reqMessages, openai.ChatCompletionMessage{
			Role:    "system",
			Content: "【对话环境（JSON）】\n" + envJSON,
		})
	}
	reqMessages = append(reqMessages, messages[len(messages)-1])
	sessionsMu.Unlock()

	model := conf.Config.ChatModel
	// 日志收敛（P15）：不再全量打印请求消息，仅记单行摘要
	log.Printf("chat request session=%s model=%s messages=%d last_len=%d", session, model, len(reqMessages), len([]rune(messages[len(messages)-1].Content)))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Config.LLMTimeoutSec)*time.Second)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: reqMessages,
		User:     qqstr,
	}
	// 注册表为空时不携带 tools 字段，生产路径请求体与现状完全一致
	if hasTools {
		req.Tools = agentRegistry.Tools()
	}
	// 端点不支持 json_object 时将 chatJsonMode 置 false 回退纯 prompt 模式（ParseReply 兜底仍可解析）；
	// 真机验证：百炼端点 response_format json_object 与 tools 并用会抑制工具调用（模型直出 JSON 不调工具），
	// 故工具启用时不携带 json_object，ParseReply 兜底解析仍可处理 JSON 回复
	if conf.Config.ChatJsonMode && !hasTools {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	start := time.Now()
	// Agent 循环：注册表为空时恰执行一次调用直接返回，行为与无工具前一致
	agentRes, err := agentRunner.Run(ctx, req)
	resp := agentRes.Response

	if err != nil {
		log.Printf("ChatCompletion error session=%s model=%s err=%v", session, model, err)
		return resp, emptyReply, err
	}
	if len(resp.Choices) == 0 {
		return resp, emptyReply, fmt.Errorf("empty choices")
	}
	// 输出协议解析：失败兜底整段当 reply；会话上下文只存原始 content
	parsed := inline.NormalizeReply(inline.ParseReply(resp.Choices[0].Message.Content))

	// 回复纳入上下文（窗口与 max_memory 裁剪已控制总量）
	sessionsMu.Lock()
	cur := sessions[session]
	cur.Messages = append(cur.Messages, resp.Choices[0].Message)
	cur.Last_time = now
	sessions[session] = cur
	sessionsMu.Unlock()
	// 日志收敛（P15）：不再全量打印响应 JSON，仅记单行摘要
	log.Printf("chat response session=%s model=%s cost_ms=%d tokens=%d reply_preview=%s", session, model, time.Since(start).Milliseconds(), agentRes.TotalTokens, recall.PreviewText(parsed.Reply, 80))
	return resp, parsed, nil
}

// chatContextJSON 结构化环境 system 段：ambient 快照 + 召回记忆 + 会话历史
type chatContextJSON struct {
	Ambient json.RawMessage      `json:"ambient,omitempty"`
	Memory  []contextMemoryHit   `json:"memory,omitempty"`
	History []contextHistoryItem `json:"history,omitempty"`
}

type contextMemoryHit struct {
	Score float64 `json:"score"`
	Text  string  `json:"text"`
}

type contextHistoryItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// buildChatContextJSON 组装环境 JSON；ambient/memory/history 全空返回 ""
func buildChatContextJSON(ambientJSON string, memoryHits []recall.MemoryHit, history []openai.ChatCompletionMessage) string {
	env := chatContextJSON{}
	if ambientJSON != "" {
		env.Ambient = json.RawMessage(ambientJSON)
	}
	for _, h := range memoryHits {
		text := strings.TrimSpace(h.Text)
		if text == "" {
			text = strings.TrimSpace(h.Summary)
		}
		if text == "" {
			continue
		}
		env.Memory = append(env.Memory, contextMemoryHit{Score: h.Score, Text: text})
	}
	for _, m := range history {
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		env.History = append(env.History, contextHistoryItem{Role: m.Role, Content: m.Content})
	}
	if env.Ambient == nil && len(env.Memory) == 0 && len(env.History) == 0 {
		return ""
	}
	body, err := json.Marshal(env)
	if err != nil {
		return ""
	}
	return string(body)
}

func JustChatGpt(msg string, qq string) (openai.ChatCompletionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Config.LLMTimeoutSec)*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: conf.Config.StoryModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: msg,
				},
			},
			User: qq,
		},
	)
	if err != nil {
		log.Printf("ChatCompletion error story user=%s model=%s err=%v", qq, conf.Config.StoryModel, err)
		return resp, err
	}
	return resp, err
}

func CreateImg(msgStr string, qq float64) (bool, string) {
	qqStr := strconv.FormatFloat(qq, 'f', -1, 64)
	type req struct {
		Prompt string `json:"prompt"`
		User   string `json:"user"`
	}
	var data = req{
		Prompt: msgStr,
		User:   qqStr,
	}
	// resp,err:=client.CreateImage(context.Background(),openai.ImageRequest{
	// 	Prompt: msgStr,
	// 	User:qqStr,
	// })

	// log.Println(resp,err)

	resp := util.ChatGPTHttpPost("https://api.openai.com/v1/images/generations", data)
	var res map[string]interface{}
	err := json.Unmarshal(resp, &res)
	if err != nil {
		fmt.Println(err)
	}

	error, has := res["error"]
	if has {
		return false, error.(map[string]interface{})["message"].(string)
	}

	resdata, _ := res["data"]

	resdata1 := resdata.([]interface{})

	return true, resdata1[0].(map[string]interface{})["url"].(string)

}

func EditImg(filePath string, msgStr string, qq float64) (bool, string) {
	// type ImageEditRequest struct {
	// 	Image          *os.File `json:"image,omitempty"`
	// 	Mask           *os.File `json:"mask,omitempty"`
	// 	Prompt         string   `json:"prompt,omitempty"`
	// 	N              int      `json:"n,omitempty"`
	// 	Size           string   `json:"size,omitempty"`
	// 	ResponseFormat string   `json:"response_format,omitempty"`
	// }

	// image为根目录/static目录下的avatar.jpg
	file, err := os.Open(filePath)
	if err != nil {
		log.Println(err)
		return false, err.Error()
	}
	defer file.Close()
	resp, err := client.CreateVariImage(context.Background(), openai.ImageVariRequest{
		Image:          file,
		N:              1,
		Size:           openai.CreateImageSize256x256,
		ResponseFormat: openai.CreateImageResponseFormatURL,
	})
	// resp, err := client.CreateEditImage(context.Background(), openai.ImageEditRequest{
	// 	Image:          file,
	// 	Mask:           file,
	// 	Prompt:         "改为向下",
	// 	N:              1,
	// 	Size:           openai.CreateImageSize256x256,
	// 	ResponseFormat: openai.CreateImageResponseFormatURL,
	// })
	if err != nil {
		return false, err.Error()
	}
	return true, resp.Data[0].URL
	// return true, ""
	// qqStr := strconv.FormatFloat(qq, 'f', -1, 64)
	// type req struct {
	// 	Prompt string `json:"prompt"`
	// 	User   string `json:"user"`
	// }
	// var data = req{

	// 	Prompt: msgStr,
	// 	User:   qqStr,
	// }

	// resp := util.ChatGPTHttpPost("https://api.openai.com/v1/images/generations", data)
	// var res map[string]interface{}
	// err := json.Unmarshal(resp, &res)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// error, has := res["error"]
	// if has {
	// 	return false, error.(map[string]interface{})["message"].(string)
	// }
	// resdata, has := res["data"]
	// if has && len(resdata.([]map[string]string)) > 0 {
	// 	return true, resdata.([]map[string]string)[0]["url"]
	// } else {
	// 	return false, ""
	// }

}

func SetPersonality(msgStr string, msg map[string]interface{}) {
	//储存
	flag := model.SetGPTPersonality(msg["user_id"].(float64), msgStr)
	if !flag {
		send.SendGroupPost(msg["group_id"].(float64), `系统错误`)
		return

	}
	qq := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	//修改
	flag = checkSession(qq)

	sessionsMu.Lock()
	if flag {
		sessions[qq] = Session{
			ID:        qq,
			Messages:  sessions[qq].Messages,
			Last_time: sessions[qq].Last_time,
			Personality: openai.ChatCompletionMessage{
				Role:    "user",
				Content: msgStr,
			},
		}
	} else {
		sessions[qq] = Session{
			ID:        qq,
			Messages:  make([]openai.ChatCompletionMessage, 0),
			Last_time: 0,
			Personality: openai.ChatCompletionMessage{
				Role:    "user",
				Content: msgStr,
			},
		}
	}
	sessionsMu.Unlock()
	send.SendGroupPost(msg["group_id"].(float64), `已修改`)

}

func checkSession(id string) bool {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	return checkSessionLocked(id)
}

// checkSessionLocked 调用方需已持有 sessionsMu 写锁
func checkSessionLocked(id string) bool {
	_, ok := sessions[id]
	if !ok {
		sessions[id] = Session{
			ID:       id,
			Messages: make([]openai.ChatCompletionMessage, 0),
			Personality: openai.ChatCompletionMessage{
				Role:    "user",
				Content: "",
			},
			Last_time: 0,
		}
		return false
	}
	return true
}

func getUserGptSetting(msg map[string]interface{}, typeInt int) string {
	qq := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	gptSettingMu.RLock()
	gptInfo, ok := gptSetting[qq]
	gptSettingMu.RUnlock()
	if !ok {
		gptInfo = model.GetChatGptInfo(msg["user_id"].(float64))
		gptSettingMu.Lock()
		gptSetting[qq] = gptInfo
		gptSettingMu.Unlock()
	}

	if gptInfo.Is_ban == 1 {
		return ""
	}
	var session string
	if typeInt == 0 {
		session = strconv.FormatFloat(msg["group_id"].(float64), 'f', -1, 64)
	}

	if gptInfo.Gpt_use_person == 1 || typeInt == 1 { //用qq做session
		session = qq
	}
	return session
}

func ResolveSession(msg map[string]interface{}, typeInt int) string {
	return getUserGptSetting(msg, typeInt)
}

func AddPlan(msgStr string, msg map[string]interface{}) {
	session := getUserGptSetting(msg, 0)
	if session == "" { //被ban了
		return
	}
	groupIdStr := strconv.FormatFloat(msg["group_id"].(float64), 'f', -1, 64)
	msgIdStr := strconv.FormatFloat(msg["message_id"].(float64), 'f', -1, 64)
	// 触发时刻：窗口空洞则异步补拉历史，并快照环境群聊记录（排除本条触发消息），
	// 快照捕获进闭包，与排队等待重叠，执行时不再与窗口写入竞争
	ctxbackfill.EnsureGroupWindow(groupIdStr)
	ambient := chatctx.SnapshotRendered(groupIdStr, msgIdStr)
	ambientJSON := chatctx.SnapshotJSON(groupIdStr, msgIdStr)
	// 触发时刻发起长期记忆召回，与排队等待重叠；环境上下文参与 embedding 提升召回相关性
	recallQuery := msgStr
	if ambient != "" {
		recallQuery = ambient + "\n" + msgStr
	}
	qqStr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	recallHandle := memoryCollector.StartRecall("group", qqStr, groupIdStr, recallQuery)
	submitted := chatScheduler.Submit(session, func() {
		checkSession(session)
		remark := msg["sender"].(map[string]interface{})["nickname"].(string)
		if msg["sender"].(map[string]interface{})["card"].(string) != "" {
			remark = msg["sender"].(map[string]interface{})["card"].(string)
		}
		res, parsed, err := AskForChatGPT(msgStr, msg["user_id"].(float64), remark, session, ambientJSON, recallHandle.Hits())

		if err == nil {
			replyText := parsed.Reply
			if replyText == "" {
				// 兜底：协议解析出的 reply 为空时用原文发送，防模型误判静默（should_reply 自主语义留阶段五）
				replyText = strings.TrimSpace(res.Choices[0].Message.Content)
				if replyText != "" {
					log.Printf("chat reply fallback session=%s reason=empty_parsed_reply len=%d", session, len([]rune(replyText)))
				}
			}
			if replyText != "" {
				send.SendGroupPost(msg["group_id"].(float64), replyText)
				// NapCat 不回推机器人自己的群消息，回复需手动入窗
				chatctx.AppendBotReply(groupIdStr, replyText)
				memoryCollector.CollectOutput("group", "group", session, msg, replyText)
				if len(parsed.Memory) > 0 {
					go memoryCollector.EnqueueInlineCandidates("group", qqStr, groupIdStr, session, msgIdStr, "inline", parsed.Memory)
				}
				// send.SendTTS(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
				model.LogUserUseTokens(msg["user_id"].(float64), res.Usage.TotalTokens, res.ID)
				return
			}
		}
		if err != nil {
			send.SendGroupPost(msg["group_id"].(float64), replyFailed)
		}
	})
	if !submitted { //队列已满，丢弃并兜底提示
		send.SendGroupPost(msg["group_id"].(float64), replyBusy)
	}
}
func AddPlanPrivate(msgStr string, msg map[string]interface{}) {
	session := getUserGptSetting(msg, 1)
	if session == "" { //被ban了
		return
	}
	// 触发时刻发起长期记忆召回（私聊只查 user 集合），与排队等待重叠
	qqStr := strconv.FormatFloat(msg["user_id"].(float64), 'f', -1, 64)
	msgIdStr := strconv.FormatFloat(msg["message_id"].(float64), 'f', -1, 64)
	recallHandle := memoryCollector.StartRecall("user", qqStr, "", msgStr)
	submitted := chatScheduler.Submit(session, func() {
		checkSession(session)
		// 私聊消息本就全量进会话上下文，无需环境记录注入
		res, parsed, err := AskForChatGPT(msgStr, msg["user_id"].(float64), "", session, "", recallHandle.Hits())

		if err == nil {
			replyText := parsed.Reply
			if replyText == "" {
				// 兜底：协议解析出的 reply 为空时用原文发送，防模型误判静默
				replyText = strings.TrimSpace(res.Choices[0].Message.Content)
				if replyText != "" {
					log.Printf("chat reply fallback session=%s reason=empty_parsed_reply len=%d", session, len([]rune(replyText)))
				}
			}
			if replyText != "" {
				send.SendPrivatePost(msg["user_id"].(float64), replyText)
				memoryCollector.CollectOutput("user", "private", session, msg, replyText)
				if len(parsed.Memory) > 0 {
					go memoryCollector.EnqueueInlineCandidates("user", qqStr, "", session, msgIdStr, "inline", parsed.Memory)
				}
				// send.SendTTS(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
				model.LogUserUseTokens(msg["user_id"].(float64), res.Usage.TotalTokens, res.ID)
				return
			}
		}
		if err != nil {
			send.SendPrivatePost(msg["user_id"].(float64), replyFailed)
		}
	})
	if !submitted { //队列已满，丢弃并兜底提示
		send.SendPrivatePost(msg["user_id"].(float64), replyBusy)
	}
}
func AddImgPlan(msgStr string, msg map[string]interface{}) {
	session := getUserGptSetting(msg, 0)
	if session == "" { //被ban了
		return
	}
	bgScheduler.Submit(session, func() {
		_, url := CreateImg(msgStr, msg["user_id"].(float64))
		send.SendGroupPost(msg["group_id"].(float64), url)
	})
}
