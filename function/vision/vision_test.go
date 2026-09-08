package vision

import (
	"300Bot/conf"
	"strings"
	"testing"
)

// withConf 临时覆盖 conf.Config 字段，用例结束后自动恢复，避免污染其它用例
func withConf(t *testing.T, mutate func(*conf.BaseConfig)) {
	t.Helper()
	orig := conf.Config
	mutate(&conf.Config)
	t.Cleanup(func() { conf.Config = orig })
}

func imgSeg(url string, size float64) map[string]interface{} {
	data := map[string]interface{}{"url": url}
	if size >= 0 {
		data["file_size"] = size
	}
	return map[string]interface{}{"type": "image", "data": data}
}

func TestExtractImagesFromSegments(t *testing.T) {
	msg := map[string]interface{}{
		"message": []interface{}{
			imgSeg("https://multimedia.nt.qq.com.cn/download?appid=1407&rkey=abc", 173757),
			map[string]interface{}{"type": "text", "data": map[string]interface{}{"text": "dyx"}},
		},
	}
	refs := ExtractImages(msg)
	if len(refs) != 1 {
		t.Fatalf("期望 1 张图, got %d", len(refs))
	}
	if refs[0].URL != "https://multimedia.nt.qq.com.cn/download?appid=1407&rkey=abc" {
		t.Fatalf("URL 提取错误: %s", refs[0].URL)
	}
	if refs[0].FileSize != 173757 {
		t.Fatalf("file_size 提取错误: %d", refs[0].FileSize)
	}
}

func TestExtractImagesSkipOversize(t *testing.T) {
	msg := map[string]interface{}{
		"message": []interface{}{
			imgSeg("https://x/big.jpg", float64(9<<20)),
			imgSeg("https://x/ok.jpg", 1000),
		},
	}
	refs := ExtractImages(msg)
	if len(refs) != 1 || refs[0].URL != "https://x/ok.jpg" {
		t.Fatalf("超大图未跳过: %+v", refs)
	}
}

func TestExtractImagesCapAndOrder(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e", "f"}
	segs := make([]interface{}, 0, len(names))
	for _, n := range names {
		segs = append(segs, imgSeg("https://x/"+n+".jpg", 100))
	}
	refs := ExtractImages(map[string]interface{}{"message": segs})
	if len(refs) != maxImagesPerMsg {
		t.Fatalf("应截断到 %d 张, got %d", maxImagesPerMsg, len(refs))
	}
	if refs[0].URL != "https://x/a.jpg" || refs[len(refs)-1].URL != "https://x/d.jpg" {
		t.Fatalf("顺序/截断错误: %+v", refs)
	}
}

func TestExtractImagesFallbackRawMessage(t *testing.T) {
	// 无 message 分段时回退解析 raw_message，&amp; 应还原为 &
	msg := map[string]interface{}{
		"raw_message": "[CQ:image,file=a.jpg,sub_type=0,url=https://x/d?a=1&amp;b=2,file_size=100][CQ:at,qq=1] dyx",
	}
	refs := ExtractImages(msg)
	if len(refs) != 1 {
		t.Fatalf("期望 1 张图, got %d", len(refs))
	}
	if refs[0].URL != "https://x/d?a=1&b=2" {
		t.Fatalf("URL 反转义错误: %s", refs[0].URL)
	}
	if refs[0].FileSize != 100 {
		t.Fatalf("file_size 解析错误: %d", refs[0].FileSize)
	}
}

func TestExtractImagesNone(t *testing.T) {
	if refs := ExtractImages(map[string]interface{}{"raw_message": "纯文本"}); refs != nil {
		t.Fatalf("无图应返回 nil, got %+v", refs)
	}
	if refs := ExtractImages(nil); refs != nil {
		t.Fatalf("nil msg 应返回 nil, got %+v", refs)
	}
}

func TestEndpointDerivation(t *testing.T) {
	withConf(t, func(c *conf.BaseConfig) {
		c.VisionApiUrl = ""
		c.ChatGPTBaseUrl = "https://ws-abc.cn-beijing.maas.aliyuncs.com/compatible-mode/v1"
	})
	want := "https://ws-abc.cn-beijing.maas.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	if got := endpoint(); got != want {
		t.Fatalf("端点派生错误:\n got=%s\nwant=%s", got, want)
	}

	withConf(t, func(c *conf.BaseConfig) { c.VisionApiUrl = "https://custom/ep" })
	if got := endpoint(); got != "https://custom/ep" {
		t.Fatalf("显式覆盖未生效: %s", got)
	}

	withConf(t, func(c *conf.BaseConfig) { c.VisionApiUrl = ""; c.ChatGPTBaseUrl = "" })
	if got := endpoint(); got != "" {
		t.Fatalf("无法派生应返回空, got=%s", got)
	}
}

func TestInjectDescriptionsDisabled(t *testing.T) {
	withConf(t, func(c *conf.BaseConfig) { c.ImageRecogEnabled = false })
	msgStr := "看这个[CQ:image,file=a.jpg,url=https://x/dis.jpg]好看"
	if got := InjectDescriptions(msgStr, nil); got != msgStr {
		t.Fatalf("开关关闭应原样返回, got=%s", got)
	}
}

func TestInjectDescriptionsReplaceWhenNoEndpoint(t *testing.T) {
	// 开关开但端点/密钥为空 → Describe 返回 ""（endpoint 为空时在下载前短路，无网络），
	// 验证 [CQ:image...] 被替换为 [图片] 占位、周边文本保留
	withConf(t, func(c *conf.BaseConfig) {
		c.ImageRecogEnabled = true
		c.VisionApiUrl = ""
		c.ChatGPTBaseUrl = ""
		c.DashScopeKey = ""
	})
	msgStr := "看这个[CQ:image,file=a.jpg,url=https://x/noendpoint.jpg]好看"
	got := InjectDescriptions(msgStr, nil)
	if strings.Contains(got, "CQ:image") {
		t.Fatalf("CQ 码应被替换: %s", got)
	}
	if !strings.Contains(got, "[图片]") {
		t.Fatalf("应替换为 [图片] 占位: %s", got)
	}
	if !strings.Contains(got, "看这个") || !strings.Contains(got, "好看") {
		t.Fatalf("周边文本应保留: %s", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("  你好世界  ", 0); got != "你好世界" {
		t.Fatalf("maxChars<=0 应只去空白不截断, got=%q", got)
	}
	if got := truncate("你好世界", 2); got != "你好" {
		t.Fatalf("应按 rune 截断, got=%q", got)
	}
	if got := truncate("短", 10); got != "短" {
		t.Fatalf("未超长应原样, got=%q", got)
	}
}

func TestNormalizeURL(t *testing.T) {
	if got := normalizeURL("  https://x/d?a=1&amp;b=2  "); got != "https://x/d?a=1&b=2" {
		t.Fatalf("反转义/去空白错误: %q", got)
	}
	if got := normalizeURL("notaurl"); got != "" {
		t.Fatalf("非 http(s) 应返回空: %q", got)
	}
	if got := normalizeURL(""); got != "" {
		t.Fatalf("空串应返回空: %q", got)
	}
}
