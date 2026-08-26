package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// BaseConfig 存放历史基础配置，保持向后兼容 conf.Config 的使用方式。
type BaseConfig struct {
	Name string `json:"name"`

	Port string `json:"port"`
	// 调用发送等api的端口
	ApiPort string `json:"apiPort"`
	ApiUrl  string `json:"apiUrl"`
	Host    string `json:"host"`
	// 机器人qq号
	BotQQ   string `json:"botQQ"`
	BotName string `json:"botName"`
	// 最高权限QQ
	Manager string `json:"manager"`

	DatabaseHost     string `json:"databaseHost"`
	DatabaseUser     string `json:"databaseUser"`
	DatabasePassword string `json:"databasePassword"`
	BotDatabaseName  string `json:"botDatabaseName"`
	HeroDatabaseName string `json:"heroDatabaseName"`
	ImmortalbaseName string `json:"immortalbaseName"`

	ChatGPTKey     string `json:"chatGPTkey"`
	ChatGPTBaseUrl string `json:"chatGPTbaseUrl"`

	DashScopeKey  string `json:"dashScopeKey"`
	WetherApiCode string `json:"wetherApiCode"`
	VPN           string `json:"VPN"`

	// Chat 并发调度可选项，未配置时代码内补默认值
	ChatConcurrency int `json:"chatConcurrency"` // 交互池全局并发上限，默认 3
	ChatQueueDepth  int `json:"chatQueueDepth"`  // 每会话队列深度，默认 8
	BgConcurrency   int `json:"bgConcurrency"`   // 后台池全局并发上限，默认 2
	LLMTimeoutSec   int `json:"llmTimeoutSec"`   // 单次 LLM 生成超时秒数，默认 120

	// 群聊上下文滑动窗口可选项，未配置时代码内补默认值
	CtxWindowSize     int `json:"ctxWindowSize"`     // 每群窗口保留条数，默认 50
	CtxWindowMaxChars int `json:"ctxWindowMaxChars"` // 注入群聊记录的字符预算，默认 4000
	CtxIdleMinutes    int `json:"ctxIdleMinutes"`    // 会话空闲超时分钟数，默认 30
	CtxBackfillCount  int `json:"ctxBackfillCount"`  // NapCat 历史补拉条数，默认 20

	// LLM 交互 JSON 化（阶段四）开关；bool 缺失即 false，需在 json 中显式写 true 启用
	ChatJsonMode            bool `json:"chatJsonMode"`            // 请求 response_format: json_object
	ChatMemoryInlineEnabled bool `json:"chatMemoryInlineEnabled"` // 回复内联记忆候选入队

	// 模型名（阶段五）可选项，未配置时代码内补默认值
	ChatModel  string `json:"chatModel"`  // 聊天模型，默认 qwen3.5-plus-2026-04-20
	StoryModel string `json:"storyModel"` // 修仙故事/JustChatGpt 模型，默认 deepseek-r1

	MoneyList []string `json:"moneyList"` //赞助列表
}

type MemoryConfig struct {
	MemoryEnabled         bool   `json:"memoryEnabled"`
	MemoryRawStoreEnabled bool   `json:"memoryRawStoreEnabled"`
	MemoryBatchEnabled    bool   `json:"memoryBatchEnabled"`
	MemoryBatchMaxTurns   int    `json:"memoryBatchMaxTurns"`
	MemoryBatchMaxChars   int    `json:"memoryBatchMaxChars"`
	MemoryBatchMaxWaitSec int    `json:"memoryBatchMaxWaitSec"`
	MemoryAsyncQueueSize  int    `json:"memoryAsyncQueueSize"`
	MemoryWorkerCount     int    `json:"memoryWorkerCount"`
	MemoryRetryTimes      int    `json:"memoryRetryTimes"`
	MemoryMinImportance   int    `json:"memoryMinImportance"`
	MemoryDedupWindowSec  int    `json:"memoryDedupWindowSec"`
	MemoryFallbackToMysql bool   `json:"memoryFallbackToMysql"`

	// 收尾项（阶段五）可选项，未配置时代码内补默认值
	MemorySummaryModel     string `json:"memorySummaryModel"`     // 总结器模型，默认 qwen3.5-plus-2026-04-20
	MemoryRawRetentionDays int    `json:"memoryRawRetentionDays"` // summarized 记录保留天数，默认 30
	MemoryRawQueueSize     int    `json:"memoryRawQueueSize"`     // CollectInput 异步队列容量，默认 2000
	MemoryRawBatchSize     int    `json:"memoryRawBatchSize"`     // 批量 insert 条数，默认 20

	EmbeddingProvider     string `json:"embeddingProvider"`
	EmbeddingApiKey       string `json:"embeddingApiKey"`
	EmbeddingModel        string `json:"embeddingModel"`
	EmbeddingApiUrl       string `json:"embeddingApiUrl"`
	EmbeddingDimension    int    `json:"embeddingDimension"`

	// 记忆召回（阶段三）可选项；Enabled 未显式配置时为 false 不启用
	MemoryRecallEnabled  bool    `json:"memoryRecallEnabled"`
	MemoryRecallBudgetMs int     `json:"memoryRecallBudgetMs"` // 召回总预算毫秒，默认 2000
	MemoryRecallTopK     int     `json:"memoryRecallTopK"`     // 每集合召回条数，默认 4
	MemoryRecallMinScore float64 `json:"memoryRecallMinScore"` // 相似度阈值，默认 0.35
	MemoryRecallMaxChars int     `json:"memoryRecallMaxChars"` // 注入 prompt 的字符预算，默认 2000
}

type QdrantConfig struct {
	QdrantUrl             string `json:"qdrantUrl"`
	QdrantApiKey          string `json:"qdrantApiKey"`
	QdrantCollectionUser  string `json:"qdrantCollectionUser"`
	QdrantCollectionGroup string `json:"qdrantCollectionGroup"`
	QdrantVectorSize      int    `json:"qdrantVectorSize"`
	QdrantDistance        string `json:"qdrantDistance"`
	QdrantTimeoutMs       int    `json:"qdrantTimeoutMs"`
}

type AppConfig struct {
	Base   BaseConfig
	Memory MemoryConfig
	Qdrant QdrantConfig
}

const (
	// JwtKey 私钥
	SecretKey = ""
	// passWd 加盐
	// Salt = ""
)

var (
	Config BaseConfig
	Memory MemoryConfig
	Qdrant QdrantConfig
	App    AppConfig
)

func fileGetContents(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func loadJSONConfig(path string, out interface{}) error {
	content, err := fileGetContents(path)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(content, out); err != nil {
		return err
	}
	return nil
}

// applyBaseConfigDefaults 对未配置的可选并发项补代码默认值，避免强制要求本地配置
func applyBaseConfigDefaults(cfg *BaseConfig) {
	if cfg.ChatConcurrency <= 0 {
		cfg.ChatConcurrency = 3
	}
	if cfg.ChatQueueDepth <= 0 {
		cfg.ChatQueueDepth = 8
	}
	if cfg.BgConcurrency <= 0 {
		cfg.BgConcurrency = 2
	}
	if cfg.LLMTimeoutSec <= 0 {
		cfg.LLMTimeoutSec = 120
	}
	if cfg.CtxWindowSize <= 0 {
		cfg.CtxWindowSize = 50
	}
	if cfg.CtxWindowMaxChars <= 0 {
		cfg.CtxWindowMaxChars = 4000
	}
	if cfg.CtxIdleMinutes <= 0 {
		cfg.CtxIdleMinutes = 30
	}
	if cfg.CtxBackfillCount <= 0 {
		cfg.CtxBackfillCount = 20
	}
	if strings.TrimSpace(cfg.ChatModel) == "" {
		cfg.ChatModel = "qwen3.5-plus-2026-04-20"
	}
	if strings.TrimSpace(cfg.StoryModel) == "" {
		cfg.StoryModel = "deepseek-r1"
	}
}

func validateBaseConfig(cfg BaseConfig) error {
	if strings.TrimSpace(cfg.DatabaseHost) == "" ||
		strings.TrimSpace(cfg.DatabaseUser) == "" ||
		strings.TrimSpace(cfg.BotDatabaseName) == "" {
		return errors.New("conf.local.json 缺少数据库必要字段")
	}
	if strings.TrimSpace(cfg.Port) == "" {
		return errors.New("conf.local.json 缺少 port")
	}
	return nil
}

// applyMemoryConfigDefaults 对未配置的召回可选项补代码默认值
func applyMemoryConfigDefaults(cfg *MemoryConfig) {
	if cfg.MemoryRecallBudgetMs <= 0 {
		cfg.MemoryRecallBudgetMs = 2000
	}
	if cfg.MemoryRecallTopK <= 0 {
		cfg.MemoryRecallTopK = 4
	}
	if cfg.MemoryRecallMinScore <= 0 {
		cfg.MemoryRecallMinScore = 0.35
	}
	if cfg.MemoryRecallMaxChars <= 0 {
		cfg.MemoryRecallMaxChars = 2000
	}
	if strings.TrimSpace(cfg.MemorySummaryModel) == "" {
		cfg.MemorySummaryModel = "qwen3.5-plus-2026-04-20"
	}
	if cfg.MemoryRawRetentionDays <= 0 {
		cfg.MemoryRawRetentionDays = 30
	}
	if cfg.MemoryRawQueueSize <= 0 {
		cfg.MemoryRawQueueSize = 2000
	}
	if cfg.MemoryRawBatchSize <= 0 {
		cfg.MemoryRawBatchSize = 20
	}
}

func validateMemoryConfig(cfg MemoryConfig) error {
	if cfg.MemoryBatchMaxTurns <= 0 || cfg.MemoryBatchMaxChars <= 0 || cfg.MemoryBatchMaxWaitSec <= 0 {
		return errors.New("memory.local.json 阈值字段必须 > 0")
	}
	if cfg.MemoryAsyncQueueSize <= 0 || cfg.MemoryWorkerCount <= 0 {
		return errors.New("memory.local.json 队列与 worker 配置必须 > 0")
	}
	if cfg.MemoryRetryTimes < 0 {
		return errors.New("memory.local.json memoryRetryTimes 不能 < 0")
	}
	if cfg.MemoryMinImportance < 1 || cfg.MemoryMinImportance > 5 {
		return errors.New("memory.local.json memoryMinImportance 必须在 1-5")
	}
	if strings.TrimSpace(cfg.EmbeddingApiUrl) == "" {
		return errors.New("memory.local.json 缺少 embeddingApiUrl")
	}
	if strings.TrimSpace(cfg.EmbeddingModel) == "" {
		return errors.New("memory.local.json 缺少 embeddingModel")
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.EmbeddingProvider))
	if provider == "ali" && cfg.EmbeddingDimension <= 0 {
		return errors.New("memory.local.json provider=ali 时 embeddingDimension 必须 > 0")
	}
	if cfg.MemoryRecallEnabled {
		if cfg.MemoryRecallTopK < 1 || cfg.MemoryRecallTopK > 10 {
			return errors.New("memory.local.json memoryRecallTopK 必须在 1-10")
		}
		if cfg.MemoryRecallMinScore < 0 || cfg.MemoryRecallMinScore > 1 {
			return errors.New("memory.local.json memoryRecallMinScore 必须在 0-1")
		}
		if cfg.MemoryRecallBudgetMs > 10000 {
			return errors.New("memory.local.json memoryRecallBudgetMs 不能超过 10000")
		}
	}
	return nil
}

func validateQdrantConfig(cfg QdrantConfig) error {
	if strings.TrimSpace(cfg.QdrantUrl) == "" {
		return errors.New("qdrant.local.json 缺少 qdrantUrl")
	}
	if cfg.QdrantVectorSize <= 0 || cfg.QdrantTimeoutMs <= 0 {
		return errors.New("qdrant.local.json 向量维度和超时时间必须 > 0")
	}
	distance := strings.ToLower(strings.TrimSpace(cfg.QdrantDistance))
	if distance != "cosine" && distance != "dot" && distance != "euclid" {
		return errors.New("qdrant.local.json qdrantDistance 仅支持 Cosine/Dot/Euclid")
	}
	return nil
}

func loadAllLocalConfig() error {
	if err := loadJSONConfig("./conf/conf.local.json", &Config); err != nil {
		return fmt.Errorf("读取 conf.local.json 失败: %w", err)
	}
	applyBaseConfigDefaults(&Config)
	if err := validateBaseConfig(Config); err != nil {
		return err
	}

	if err := loadJSONConfig("./conf/memory.local.json", &Memory); err != nil {
		return fmt.Errorf("读取 memory.local.json 失败: %w", err)
	}
	applyMemoryConfigDefaults(&Memory)
	if err := validateMemoryConfig(Memory); err != nil {
		return err
	}

	if err := loadJSONConfig("./conf/qdrant.local.json", &Qdrant); err != nil {
		return fmt.Errorf("读取 qdrant.local.json 失败: %w", err)
	}
	if err := validateQdrantConfig(Qdrant); err != nil {
		return err
	}
	if Memory.EmbeddingDimension > 0 && Memory.EmbeddingDimension != Qdrant.QdrantVectorSize {
		return fmt.Errorf("embeddingDimension(%d) 与 qdrantVectorSize(%d) 不一致", Memory.EmbeddingDimension, Qdrant.QdrantVectorSize)
	}

	App = AppConfig{
		Base:   Config,
		Memory: Memory,
		Qdrant: Qdrant,
	}
	return nil
}

func init() {
	if err := loadAllLocalConfig(); err != nil {
		fmt.Println("配置文件读取失败")
		panic(err)
	}
	fmt.Println("run on ", Config.Port)
}
