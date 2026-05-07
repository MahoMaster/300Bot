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

	ChatGPTKey    string `json:"chatGPTkey"`
	DashScopeKey  string `json:"dashScopeKey"`
	WetherApiCode string `json:"wetherApiCode"`
	VPN           string `json:"VPN"`

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
	EmbeddingProvider     string `json:"embeddingProvider"`
	EmbeddingApiKey       string `json:"embeddingApiKey"`
	EmbeddingModel        string `json:"embeddingModel"`
	EmbeddingApiUrl       string `json:"embeddingApiUrl"`
	EmbeddingDimension    int    `json:"embeddingDimension"`
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
	if err := validateBaseConfig(Config); err != nil {
		return err
	}

	if err := loadJSONConfig("./conf/memory.local.json", &Memory); err != nil {
		return fmt.Errorf("读取 memory.local.json 失败: %w", err)
	}
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
