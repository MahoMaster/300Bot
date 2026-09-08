// Package vision 图片视觉识别：调用 DashScope 原生多模态端点，把图片描述为简短中文文本，
// 供聊天上下文注入（让文本 LLM “看懂” 表情包/图片）。
// 本包为叶子包，仅依赖 conf 与标准库，不依赖 chatctx/chatGPT，避免导入环。
package vision

import (
	"300Bot/conf"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxImagesPerMsg  = 4       // 单条消息最多识别的图片数
	maxFileSizeBytes = 8 << 20 // 单图最大字节数（8MB），用分段 file_size 提前跳过
	cacheTTL         = 10 * time.Minute
	failCacheTTL     = 30 * time.Second // 识别失败只短暂占位，避免长期缓存空结果
	maxCacheEntries  = 512
	multimodalPath   = "/api/v1/services/aigc/multimodal-generation/generation"
)

// cqImageRe 匹配 raw_message 中的图片 CQ 码，与 chatctx.sanitizeText 使用同款模式
var cqImageRe = regexp.MustCompile(`\[CQ:image[^\]]*\]`)

// ImageRef 一张待识别图片的引用
type ImageRef struct {
	URL      string // 图片下载地址（如 multimedia.nt.qq.com.cn/...）
	FileSize int64  // 分段/CQ 码里的 file_size，用于提前跳过大图
}

// cacheEntry 识别结果缓存项；done 关闭表示识别已结束（成功或失败），desc 为最终描述
type cacheEntry struct {
	desc     string
	done     chan struct{}
	expireAt time.Time
}

var (
	cacheMu sync.Mutex
	cache   = make(map[string]*cacheEntry)

	semOnce sync.Once
	sem     chan struct{}
)

// semaphore 惰性建立并发信号量，容量取 ImageRecogConcurrency（默认 3），限制同时进行的识别数
func semaphore() chan struct{} {
	semOnce.Do(func() {
		n := conf.Config.ImageRecogConcurrency
		if n <= 0 {
			n = 3
		}
		sem = make(chan struct{}, n)
	})
	return sem
}

// ExtractImages 按出现顺序提取图片引用：优先遍历 msg["message"] 分段（type=="image"），
// 分段缺失时回退解析 msg["raw_message"] 里的 [CQ:image,...]。
// 超过 maxImagesPerMsg 截断；file_size 超过 maxFileSizeBytes 的项跳过。无图返回 nil。
func ExtractImages(msg map[string]interface{}) []ImageRef {
	if msg == nil {
		return nil
	}
	var refs []ImageRef
	if segs, ok := msg["message"].([]interface{}); ok && len(segs) > 0 {
		for _, segAny := range segs {
			seg, ok := segAny.(map[string]interface{})
			if !ok {
				continue
			}
			if t, _ := seg["type"].(string); t != "image" {
				continue
			}
			data, _ := seg["data"].(map[string]interface{})
			if data == nil {
				continue
			}
			u := normalizeURL(firstString(data["url"], data["file"]))
			if u == "" {
				continue
			}
			var size int64
			if fs, ok := data["file_size"].(float64); ok {
				size = int64(fs)
			}
			if size > maxFileSizeBytes {
				continue
			}
			refs = append(refs, ImageRef{URL: u, FileSize: size})
			if len(refs) >= maxImagesPerMsg {
				break
			}
		}
		if len(refs) > 0 {
			return refs
		}
	}
	// 回退：从 raw_message 逐个解析图片 CQ 码
	raw, _ := msg["raw_message"].(string)
	if raw == "" {
		return nil
	}
	for _, cq := range cqImageRe.FindAllString(raw, -1) {
		ref, ok := parseCQImage(cq)
		if !ok || ref.FileSize > maxFileSizeBytes {
			continue
		}
		refs = append(refs, ref)
		if len(refs) >= maxImagesPerMsg {
			break
		}
	}
	return refs
}

// Describe 识别单图，返回简短中文描述；内置 TTL 缓存 + singleflight 去重 + 并发信号量。
// 未启用/失败/超时一律返回 ""（调用方据此保留 [图片] 占位，静默降级）。
// 到达识别与触发注入对同一 URL 共享同一次调用，避免重复计费。
func Describe(imageURL string) string {
	imageURL = normalizeURL(imageURL)
	if imageURL == "" || !conf.Config.ImageRecogEnabled {
		return ""
	}
	timeout := time.Duration(conf.Config.ImageRecogTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	cacheMu.Lock()
	evictExpiredLocked()
	if e, ok := cache[imageURL]; ok && time.Now().Before(e.expireAt) {
		cacheMu.Unlock()
		// 命中：已完成则直接返回，进行中则等待其 done（带超时）
		select {
		case <-e.done:
			log.Printf("vision describe cache hit url=%s desc_len=%d", imageURL, len([]rune(e.desc)))
			return e.desc
		case <-time.After(timeout):
			log.Printf("vision describe wait timeout url=%s", imageURL)
			return ""
		}
	}
	// 未命中或已过期：占位并发起识别
	e := &cacheEntry{done: make(chan struct{}), expireAt: time.Now().Add(cacheTTL)}
	cache[imageURL] = e
	enforceCacheSizeLocked()
	cacheMu.Unlock()

	desc := recognize(imageURL, timeout)

	cacheMu.Lock()
	e.desc = desc
	if desc == "" {
		e.expireAt = time.Now().Add(failCacheTTL)
	}
	cacheMu.Unlock()
	close(e.done)
	return desc
}

// InjectDescriptions 把 msgStr 中的每个 [CQ:image...] 按序替换为 "[图片: <desc>]"。
// URL 直接从 msgStr 的 CQ 码解析，保证与占位 1:1 对齐（不受 ExtractImages 过滤影响），
// 且与到达识别共享 Describe 缓存。未启用/无图时原样返回；desc 为空则该处保留 "[图片]"。
// msg 参数保留用于签名对称与未来扩展，当前实现只依赖 msgStr。
func InjectDescriptions(msgStr string, msg map[string]interface{}) string {
	if !conf.Config.ImageRecogEnabled || msgStr == "" {
		return msgStr
	}
	if !cqImageRe.MatchString(msgStr) {
		return msgStr
	}
	total, described := 0, 0
	out := cqImageRe.ReplaceAllStringFunc(msgStr, func(cq string) string {
		total++
		ref, ok := parseCQImage(cq)
		if !ok || ref.FileSize > maxFileSizeBytes {
			return "[图片]"
		}
		desc := Describe(ref.URL)
		if strings.TrimSpace(desc) == "" {
			return "[图片]"
		}
		described++
		return "[图片: " + desc + "]"
	})
	log.Printf("vision inject done images=%d described=%d", total, described)
	return out
}

// recognize 在信号量与总超时约束下完成一次识别：下载转 base64（主）→ 直传 URL（回退）→ 调用多模态端点
func recognize(imageURL string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	s := semaphore()
	select {
	case s <- struct{}{}:
		defer func() { <-s }()
	case <-ctx.Done():
		log.Printf("vision semaphore acquire timeout url=%s", imageURL)
		return ""
	}

	ep := endpoint()
	if ep == "" {
		log.Printf("vision endpoint empty, skip url=%s", imageURL)
		return ""
	}
	apiKey := strings.TrimSpace(conf.Config.DashScopeKey)
	if apiKey == "" {
		log.Printf("vision dashScopeKey empty, skip url=%s", imageURL)
		return ""
	}

	log.Printf("vision recognize start url=%s endpoint=%s timeout=%s", imageURL, ep, timeout)
	start := time.Now()
	// 主路径：机器人下载图片转 base64（规避 DashScope 拉取 QQ CDN 的不确定性）；失败回退直传 URL
	imageField := imageURL
	if b, err := download(ctx, imageURL); err == nil && len(b) > 0 {
		mime := http.DetectContentType(b)
		if !strings.HasPrefix(mime, "image/") {
			mime = "image/jpeg"
		}
		imageField = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
		log.Printf("vision recognize downloaded url=%s bytes=%d mime=%s cost_ms=%d", imageURL, len(b), mime, time.Since(start).Milliseconds())
	} else if err != nil {
		log.Printf("vision download failed, fallback to url, url=%s err=%v", imageURL, err)
	}

	desc, err := callVision(ctx, ep, apiKey, imageField)
	if err != nil {
		log.Printf("vision call failed url=%s cost_ms=%d err=%v", imageURL, time.Since(start).Milliseconds(), err)
		return ""
	}
	desc = truncate(desc, conf.Config.ImageRecogMaxDescChars)
	if strings.TrimSpace(desc) == "" {
		log.Printf("vision recognize empty url=%s cost_ms=%d", imageURL, time.Since(start).Milliseconds())
	} else {
		log.Printf("vision recognize done url=%s cost_ms=%d desc_len=%d preview=%s", imageURL, time.Since(start).Milliseconds(), len([]rune(desc)), truncate(desc, 60))
	}
	return desc
}

// endpoint 派生多模态端点：优先用 VisionApiUrl 覆盖，否则从 ChatGPTBaseUrl 的 scheme+host 拼接路径，
// 对专属 MaaS 主机与标准 dashscope.aliyuncs.com 均成立；无法派生返回 ""（静默降级）
func endpoint() string {
	if u := strings.TrimSpace(conf.Config.VisionApiUrl); u != "" {
		return u
	}
	base := strings.TrimSpace(conf.Config.ChatGPTBaseUrl)
	if base == "" {
		return ""
	}
	pu, err := url.Parse(base)
	if err != nil || pu.Host == "" {
		return ""
	}
	scheme := pu.Scheme
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + pu.Host + multimodalPath
}

// download 带 ctx 超时下载图片字节，限制读取上限避免超大响应
func download(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxFileSizeBytes+1))
}

// ---- DashScope 多模态请求/响应结构 ----

type visionContentItem struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

type visionMessage struct {
	Role    string              `json:"role"`
	Content []visionContentItem `json:"content"`
}

type visionInput struct {
	Messages []visionMessage `json:"messages"`
}

type visionRequest struct {
	Model string      `json:"model"`
	Input visionInput `json:"input"`
}

type visionResp struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// callVision 组装并发送多模态识别请求，拼接返回文本；code 非空或状态码 >=400 视为错误
func callVision(ctx context.Context, ep, apiKey, imageField string) (string, error) {
	model := strings.TrimSpace(conf.Config.VisionModel)
	if model == "" {
		model = conf.Config.ChatModel
	}
	prompt := strings.TrimSpace(conf.Config.ImageRecogPrompt)
	if prompt == "" {
		prompt = "这是一张聊天里的图片，请简要描述这张图片的内容。"
	}
	reqBody := visionRequest{
		Model: model,
		Input: visionInput{
			Messages: []visionMessage{
				{
					Role: "user",
					Content: []visionContentItem{
						{Image: imageField},
						{Text: prompt},
					},
				},
			},
		},
	}
	jsonStr, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	imageKind := "url"
	if strings.HasPrefix(imageField, "data:") {
		imageKind = "base64"
	}
	log.Printf("vision call request model=%s endpoint=%s image_kind=%s image_len=%d", model, ep, imageKind, len(imageField))

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var parsed visionResp
	if err = json.Unmarshal(body, &parsed); err != nil {
		log.Printf("vision call unmarshal failed model=%s status=%d cost_ms=%d body_preview=%s err=%v", model, resp.StatusCode, time.Since(start).Milliseconds(), truncate(string(body), 200), err)
		return "", err
	}
	if parsed.Code != "" {
		log.Printf("vision call api error model=%s request_id=%s code=%s message=%s", model, parsed.RequestID, parsed.Code, parsed.Message)
		return "", fmt.Errorf("%s: %s (request_id=%s)", parsed.Code, parsed.Message, parsed.RequestID)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		log.Printf("vision call http error model=%s request_id=%s status=%d message=%s", model, parsed.RequestID, resp.StatusCode, parsed.Message)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, parsed.Message)
	}
	if len(parsed.Output.Choices) == 0 {
		log.Printf("vision call empty choices model=%s request_id=%s status=%d", model, parsed.RequestID, resp.StatusCode)
		return "", fmt.Errorf("empty choices")
	}
	var sb strings.Builder
	for _, c := range parsed.Output.Choices[0].Message.Content {
		sb.WriteString(c.Text)
	}
	text := strings.TrimSpace(sb.String())
	log.Printf("vision call response model=%s request_id=%s status=%d cost_ms=%d text_len=%d", model, parsed.RequestID, resp.StatusCode, time.Since(start).Milliseconds(), len([]rune(text)))
	return text, nil
}

// ---- 辅助函数 ----

// parseCQImage 解析单个图片 CQ 码为 ImageRef：提取 url（回退 http(s) 的 file）与 file_size，
// url 值做 &amp; 反转义；无有效 http(s) url 返回 ok=false
func parseCQImage(cq string) (ImageRef, bool) {
	var ref ImageRef
	inner := strings.TrimSuffix(strings.TrimPrefix(cq, "[CQ:image"), "]")
	for _, part := range strings.Split(inner, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch k {
		case "url":
			v = strings.ReplaceAll(v, "&amp;", "&")
			if isHTTPURL(v) {
				ref.URL = v
			}
		case "file":
			if ref.URL == "" && isHTTPURL(v) {
				ref.URL = strings.ReplaceAll(v, "&amp;", "&")
			}
		case "file_size":
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				ref.FileSize = n
			}
		}
	}
	return ref, ref.URL != ""
}

// normalizeURL 去空白并把 HTML 转义的 &amp; 还原为 &，使分段 URL 与 CQ URL 得到一致的缓存键
func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	u = strings.ReplaceAll(u, "&amp;", "&")
	if !isHTTPURL(u) {
		return ""
	}
	return u
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// firstString 返回入参中第一个非空字符串（用于 url 优先、file 回退）
func firstString(vals ...interface{}) string {
	for _, v := range vals {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// truncate 按 rune 截断到 maxChars，避免撑大上下文；maxChars<=0 时不截断
func truncate(s string, maxChars int) string {
	s = strings.TrimSpace(s)
	if maxChars <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= maxChars {
		return s
	}
	return string(r[:maxChars])
}

// evictExpiredLocked 清理已过期且已完成的缓存项，调用方需持有 cacheMu；进行中的项不删
func evictExpiredLocked() {
	now := time.Now()
	for k, e := range cache {
		if !now.After(e.expireAt) {
			continue
		}
		select {
		case <-e.done:
			delete(cache, k)
		default:
		}
	}
}

// enforceCacheSizeLocked 缓存超上限时清理过期项以限界，调用方需持有 cacheMu
func enforceCacheSizeLocked() {
	if len(cache) <= maxCacheEntries {
		return
	}
	now := time.Now()
	for k, e := range cache {
		if len(cache) <= maxCacheEntries {
			break
		}
		if !now.After(e.expireAt) {
			continue
		}
		select {
		case <-e.done:
			delete(cache, k)
		default:
		}
	}
}
