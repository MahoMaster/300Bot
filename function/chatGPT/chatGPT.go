package chatGPT

import (
	"300Bot/conf"
	"300Bot/model"
	"300Bot/send"
	"300Bot/util"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

const memory = 3000

func init() {
	initSessions()
	config := openai.DefaultConfig(conf.Config.ChatGPTKey)
	proxyUrl, err := url.Parse(conf.Config.VPN)
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxyUrl),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	config.HTTPClient = &http.Client{
		Transport: transport,
	}

	client = openai.NewClientWithConfig(config)

	// m, _ := client.ListModels(context.Background())
	// fmt.Println(m)
}

func initSessions() {
	arr := model.GetGroupAllGPTPersonality()
	arr = append(arr, model.GetUserAllGPTPersonality()...)
	for _, val := range arr {
		var messages = make([]openai.ChatCompletionMessage, 0)
		sessions[val.Id] = Session{
			ID:        val.Id,
			Messages:  messages,
			Last_time: 0,
			Personality: openai.ChatCompletionMessage{
				Role:    "user",
				Content: val.Gpt_personality,
			},
		}

	}
}

func AskForChatGPT(msg string, qq float64, session string) (openai.ChatCompletionResponse, error) {

	var messages = sessions[session].Messages
	var personality = sessions[session].Personality

	//距离上次对话已经超过30分钟，清除上下文
	now := int(time.Now().Unix())
	if now-sessions[session].Last_time > 1800 {
		messages = make([]openai.ChatCompletionMessage, 0)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: msg,
	})
	// 记忆超出，准备删除部分
	// if len(messages) > 4 {
	// 	messages = append(messages[0:1], messages[2:]...)
	// }
	length := 0
	for _, val := range messages {
		length = length + len(val.Content)
	}
	for {
		if length <= memory { //除去人设，记忆超过容量，删除到容量以内
			break
		}
		length = length - len(messages[0].Content)
		messages = messages[1:]
	}

	//log本次对话
	sessions[session] = Session{
		ID:          session,
		Messages:    messages,
		Last_time:   now,
		Personality: sessions[session].Personality,
	}

	//如果有人格设定，拼接人格
	if personality.Content != "" {
		messages = append([]openai.ChatCompletionMessage{personality}, messages...)
	}

	fmt.Println("------------")
	fmt.Println(messages)
	fmt.Println("------------")

	qqstr := strconv.FormatFloat(qq, 'f', -1, 64)

	model := openai.GPT3Dot5Turbo0613
	// if qqstr == "675559614" {
	// 	model = openai.GPT3Dot5Turbo16K0613
	// }

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			User:     qqstr,
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return resp, err
	}

	if resp.Usage.CompletionTokens < 200 { //如果回复消耗的token比较少，也可以纳入上下文
		sessions[session] = Session{
			ID:          session,
			Messages:    append(sessions[session].Messages, resp.Choices[0].Message),
			Last_time:   now,
			Personality: sessions[session].Personality,
		}
	}
	// json, err := json.Marshal(resp)
	// fmt.Println(string(json))
	return resp, err
}

func JustChatGpt(msg string, qq string) (openai.ChatCompletionResponse, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo0301,
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

	resp := util.ChatGPTHttpPost("https://api.openai.com/v1/images/generations", data)
	var res map[string]interface{}
	err := json.Unmarshal(resp, &res)
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println(res)
	error, has := res["error"]
	if has {
		return false, error.(map[string]interface{})["message"].(string)
	}
	// type redata struct {
	// 	Url string `json:"url"`
	// }
	resdata, _ := res["data"]
	// fmt.Println(resdata)
	resdata1 := resdata.([]interface{})
	// fmt.Println(resdata1)
	//if has {
	return true, resdata1[0].(map[string]interface{})["url"].(string)
	//	} else {
	//return false, ""
	//}

}

func EditImg(filePath string, msgStr string, qq float64) (bool, string) {
	return true, ""
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

	if flag {
		// var messages = sessions[qq].Messages
		// messages[0].Content = msgStr
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
		// var messages = make([]openai.ChatCompletionMessage, 0)
		// messages = append(messages, openai.ChatCompletionMessage{
		// 	Role:    "user",
		// 	Content: msgStr,
		// })
		// sessions[qq] = Session{
		// 	ID:       qq,
		// 	Messages: messages,
		// }
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
	send.SendGroupPost(msg["group_id"].(float64), `已修改`)

}

func checkSession(id string) bool {
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
	gptInfo, ok := gptSetting[qq]
	if !ok {
		gptInfo = model.GetChatGptInfo(msg["user_id"].(float64))
		gptSetting[qq] = gptInfo
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

var g = goroutineNew(1)

func AddPlan(msgStr string, msg map[string]interface{}) {
	g.goroutineRun(func() {
		// test()
		session := getUserGptSetting(msg, 0)
		if session == "" { //被ban了
			return
		}
		checkSession(session)
		res, err := AskForChatGPT(msgStr, msg["user_id"].(float64), session)

		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendGroupPost(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			// send.SendTTS(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			model.LogUserUseTokens(msg["user_id"].(float64), res.Usage.TotalTokens, res.ID)
		}
	})
}
func AddPlanPrivate(msgStr string, msg map[string]interface{}) {
	g.goroutineRun(func() {
		// test()
		session := getUserGptSetting(msg, 1)
		if session == "" { //被ban了
			return
		}
		checkSession(session)
		res, err := AskForChatGPT(msgStr, msg["user_id"].(float64), session)

		if err == nil && res.Choices[0].Message.Content != "" {

			send.SendPrivatePost(msg["user_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			// send.SendTTS(msg["group_id"].(float64), strings.TrimSpace(res.Choices[0].Message.Content))
			model.LogUserUseTokens(msg["user_id"].(float64), res.Usage.TotalTokens, res.ID)
		}
	})
}
func AddImgPlan(msgStr string, msg map[string]interface{}) {
	g.goroutineRun(func() {
		// test()
		session := getUserGptSetting(msg, 0)
		if session == "" { //被ban了
			return
		}
		_, url := CreateImg(msgStr, msg["user_id"].(float64))
		//if flag {
		//	if len(url) != 0 {
		//img := `[CQ:image,file=` + url + `]`
		//		send.SendGroupPost(msg["group_id"].(float64), url)
		//}
		//} else {
		//	if len(url) != 0 {
		send.SendGroupPost(msg["group_id"].(float64), url)
		//	}
		//}

	})
}
