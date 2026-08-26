package chatGPT

import (
	"300Bot/conf"
	"300Bot/function/chatctx"
	ctxbackfill "300Bot/function/chatctx/backfill"
	memoryCollector "300Bot/function/memory"
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

// AskForChatGPT ambient 为触发时刻快照的群聊记录文本（非群聊触发传空串），
// 以 system 段注入，让机器人感知未触发自己的普通群聊（P1）
func AskForChatGPT(msg string, qq float64, remark string, session string, ambient string) (openai.ChatCompletionResponse, error) {
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
	//组装请求：人格 + 环境群聊记录 + 会话消息
	reqMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+2)
	if personality.Content != "" {
		reqMessages = append(reqMessages, personality)
	}
	if ambient != "" {
		reqMessages = append(reqMessages, openai.ChatCompletionMessage{
			Role:    "system",
			Content: "以下是触发我这条消息之前的群聊记录（发言人身份以QQ号为准，昵称仅作注释）：\n" + ambient,
		})
	}
	reqMessages = append(reqMessages, messages...)
	sessionsMu.Unlock()

	fmt.Println("------------")
	fmt.Println(reqMessages)
	fmt.Println("------------")

	model := "qwen3.5-plus-2026-04-20"
	// if qqstr == "675559614" {
	// 	model = "deepseek-r1"
	// }

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Config.LLMTimeoutSec)*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: reqMessages,
			User:     qqstr,
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return resp, err
	}

	// 回复纳入上下文（窗口与 max_memory 裁剪已控制总量）
	sessionsMu.Lock()
	cur := sessions[session]
	cur.Messages = append(cur.Messages, resp.Choices[0].Message)
	cur.Last_time = now
	sessions[session] = cur
	sessionsMu.Unlock()
	json, err := json.Marshal(resp)
	fmt.Println(string(json))
	return resp, err
}

func JustChatGpt(msg string, qq string) (openai.ChatCompletionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Config.LLMTimeoutSec)*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: "deepseek-r1",
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
		fmt.Printf("ChatCompletion error: %v\n", err)
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
	submitted := chatScheduler.Submit(session, func() {
		checkSession(session)
		remark := msg["sender"].(map[string]interface{})["nickname"].(string)
		if msg["sender"].(map[string]interface{})["card"].(string) != "" {
			remark = msg["sender"].(map[string]interface{})["card"].(string)
		}
		res, err := AskForChatGPT(msgStr, msg["user_id"].(float64), remark, session, ambient)

		if err == nil && res.Choices[0].Message.Content != "" {
			replyText := strings.TrimSpace(res.Choices[0].Message.Content)
			send.SendGroupPost(msg["group_id"].(float64), replyText)
			// NapCat 不回推机器人自己的群消息，回复需手动入窗
			chatctx.AppendBotReply(groupIdStr, replyText)
			memoryCollector.CollectOutput("group", "group", session, msg, replyText)
			// send.SendTTS(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			model.LogUserUseTokens(msg["user_id"].(float64), res.Usage.TotalTokens, res.ID)
			return
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
	submitted := chatScheduler.Submit(session, func() {
		checkSession(session)
		// 私聊消息本就全量进会话上下文，无需环境记录注入
		res, err := AskForChatGPT(msgStr, msg["user_id"].(float64), "", session, "")

		if err == nil && res.Choices[0].Message.Content != "" {
			replyText := strings.TrimSpace(res.Choices[0].Message.Content)
			send.SendPrivatePost(msg["user_id"].(float64), replyText)
			memoryCollector.CollectOutput("user", "private", session, msg, replyText)
			// send.SendTTS(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			model.LogUserUseTokens(msg["user_id"].(float64), res.Usage.TotalTokens, res.ID)
			return
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
