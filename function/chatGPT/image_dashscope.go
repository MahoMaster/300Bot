package chatGPT

import (
	"300Bot/conf"
	"300Bot/send"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dashScopeImageURL = "https://ws-6esmsaoyn605x2mf.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"

var pseudoSexyKeywords = []string{"来张涩图", "来张色图", "整点二次元", "涩图", "色图"}

type dashScopeImageRequest struct {
	Model      string                   `json:"model"`
	Input      dashScopeImageInput      `json:"input"`
	Parameters dashScopeImageParameters `json:"parameters"`
}

type dashScopeImageInput struct {
	Messages []dashScopeImageMessage `json:"messages"`
}

type dashScopeImageMessage struct {
	Role    string                      `json:"role"`
	Content []dashScopeImageMessageItem `json:"content"`
}

type dashScopeImageMessageItem struct {
	Text string `json:"text"`
}

type dashScopeImageParameters struct {
	NegativePrompt     string `json:"negative_prompt"`
	PromptExtend       bool   `json:"prompt_extend"`
	Watermark          bool   `json:"watermark"`
	Size               string `json:"size"`
	NumImagesPerPrompt int    `json:"num_images_per_prompt"`
}

type dashScopeImageResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Image string `json:"image"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func AddPseudoSexyImagePlan(msgStr string, msg map[string]interface{}) {
	session := getUserGptSetting(msg, 0)
	if session == "" {
		return
	}
	bgScheduler.Submit(session, func() {
		imageURL, requestID, errMsg := createPseudoSexyImage(msgStr, msg["user_id"].(float64))
		if errMsg != "" {
			reply := "生成失败，你踏马违规了知道吗"
			if requestID != "" {
				reply = reply + "（request_id: " + requestID + "）"
			}
			send.SendGroupPost(msg["group_id"].(float64), reply)
			return
		}
		// log.Printf("imageURL: %s", imageURL)
		// log.Printf("requestID: %s", requestID)
		// log.Printf("errMsg: %s", errMsg)
		send.SendGroupPost(msg["group_id"].(float64), `[CQ:image,file=`+imageURL+`]`)
	})
}

//	func init() {
//		var a float64 = 123456
//		AddPseudoSexyImagePlan("来张涩图 初音", map[string]interface{}{
//			"user_id":  a,
//			"group_id": a,
//		})
//	}
func createPseudoSexyImage(msgStr string, qq float64) (string, string, string) {
	apiKey := strings.TrimSpace(conf.Config.DashScopeKey)
	if apiKey == "" {
		return "", "", "dashScopeKey is empty"
	}

	prompt := buildPseudoSexyPrompt(msgStr)
	reqBody := dashScopeImageRequest{
		Model: "qwen-image-2.0-pro",
		Input: dashScopeImageInput{
			Messages: []dashScopeImageMessage{
				{
					Role: "user",
					Content: []dashScopeImageMessageItem{
						{Text: prompt},
					},
				},
			},
		},
		Parameters: dashScopeImageParameters{
			NegativePrompt:     "低分辨率，低画质，肢体畸形，手指畸形，性行为，未成年人，过度光滑，蜡像感，文字模糊，扭曲，构图混乱",
			PromptExtend:       true,
			Watermark:          false,
			Size:               "1024*1024",
			NumImagesPerPrompt: 1,
		},
	}

	jsonStr, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err.Error()
	}

	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest("POST", dashScopeImageURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", "", err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-DashScope-User-Id", strconv.FormatFloat(qq, 'f', -1, 64))

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err.Error()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err.Error()
	}

	var parsed dashScopeImageResponse
	if err = json.Unmarshal(body, &parsed); err != nil {
		return "", "", err.Error()
	}

	if parsed.Code != "" {
		return "", parsed.RequestID, fmt.Sprintf("%s: %s", parsed.Code, parsed.Message)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", parsed.RequestID, parsed.Message
	}

	if len(parsed.Output.Choices) == 0 ||
		len(parsed.Output.Choices[0].Message.Content) == 0 ||
		strings.TrimSpace(parsed.Output.Choices[0].Message.Content[0].Image) == "" {
		return "", parsed.RequestID, "empty image url"
	}

	return parsed.Output.Choices[0].Message.Content[0].Image, parsed.RequestID, ""
}

func buildPseudoSexyPrompt(msgStr string) string {
	basePrompt := `
杰作，最佳画质，超精细，

纯二次元，
平涂风，
日式萌系，
非写实，
手绘风，
日系动漫风，anime style，
二次元插画，anime illustration，
赛璐璐上色，cel shading，
干净线稿，clean lineart

动漫女性，
身材纤细，
柔软感，
性感脸，

看向镜头，
诱人的表情，
微张嘴，

过膝袜，
柔软布料，
清爽的服饰，
strategically covered,
barely covered,
covered cleavage,

低角度视角，
近距离构图，
动态透视，
轻微虚化，

景深，
高饱和配色，
`
	extraPrompt := extractPromptExtra(msgStr)
	if extraPrompt == "" {
		return basePrompt
	}
	return basePrompt + " 场景补充：" + extraPrompt
}

func extractPromptExtra(msgStr string) string {
	trimmed := strings.TrimSpace(msgStr)
	for _, keyword := range pseudoSexyKeywords {
		if strings.HasPrefix(trimmed, keyword) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, keyword))
		}
	}
	return ""
}
