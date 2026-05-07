package memory

import (
	"300Bot/conf"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type QdrantRepository struct {
	httpClient      *http.Client
	baseURL         string
	apiKey          string
	collectionUser  string
	collectionGroup string
	vectorSize      int
	distance        string
	embedder        Embedder
	dedupWindow     int
}

var (
	qdrantRepo    *QdrantRepository
	qdrantInitErr error
	qdrantRepoMu  sync.RWMutex
)

func init() {
	if !conf.Memory.MemoryEnabled {
		return
	}
	if err := InitQdrantRepository(); err != nil {
		log.Printf("memory qdrant init skipped: %v", err)
	}
}

func InitQdrantRepository() error {
	qdrantRepoMu.RLock()
	if qdrantRepo != nil {
		qdrantRepoMu.RUnlock()
		return nil
	}
	qdrantRepoMu.RUnlock()

	repo, err := NewQdrantRepository(conf.Memory, conf.Qdrant)
	if err != nil {
		qdrantRepoMu.Lock()
		qdrantInitErr = err
		qdrantRepoMu.Unlock()
		return err
	}
	if err = repo.EnsureCollections(); err != nil {
		qdrantRepoMu.Lock()
		qdrantInitErr = err
		qdrantRepoMu.Unlock()
		return err
	}

	qdrantRepoMu.Lock()
	qdrantRepo = repo
	qdrantInitErr = nil
	qdrantRepoMu.Unlock()
	return nil
}

func GetQdrantRepository() (*QdrantRepository, error) {
	if err := InitQdrantRepository(); err != nil {
		return nil, err
	}
	qdrantRepoMu.RLock()
	defer qdrantRepoMu.RUnlock()
	if qdrantRepo == nil {
		return nil, fmt.Errorf("qdrant repository 未初始化")
	}
	return qdrantRepo, nil
}

func NewQdrantRepository(memoryCfg conf.MemoryConfig, qdrantCfg conf.QdrantConfig) (*QdrantRepository, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(qdrantCfg.QdrantUrl), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("qdrantUrl 未配置")
	}
	if strings.TrimSpace(qdrantCfg.QdrantCollectionUser) == "" || strings.TrimSpace(qdrantCfg.QdrantCollectionGroup) == "" {
		return nil, fmt.Errorf("qdrant collection 配置不能为空")
	}
	timeout := time.Duration(qdrantCfg.QdrantTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &QdrantRepository{
		httpClient:      &http.Client{Timeout: timeout},
		baseURL:         baseURL,
		apiKey:          strings.TrimSpace(qdrantCfg.QdrantApiKey),
		collectionUser:  strings.TrimSpace(qdrantCfg.QdrantCollectionUser),
		collectionGroup: strings.TrimSpace(qdrantCfg.QdrantCollectionGroup),
		vectorSize:      qdrantCfg.QdrantVectorSize,
		distance:        normalizeQdrantDistance(qdrantCfg.QdrantDistance),
		embedder:        NewEmbedder(memoryCfg, timeout),
		dedupWindow:     memoryCfg.MemoryDedupWindowSec,
	}, nil
}

func (r *QdrantRepository) EnsureCollections() error {
	if err := r.ensureOneCollection(r.collectionUser); err != nil {
		return err
	}
	return r.ensureOneCollection(r.collectionGroup)
}

func (r *QdrantRepository) ensureOneCollection(collection string) error {
	path := "/collections/" + url.PathEscape(collection)
	respBody, statusCode, err := r.doJSON(http.MethodGet, path, nil)
	if err != nil && statusCode != http.StatusNotFound {
		return err
	}
	if statusCode == http.StatusOK {
		return nil
	}
	if statusCode != http.StatusNotFound {
		return fmt.Errorf("检查集合失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}

	req := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     r.vectorSize,
			"distance": r.distance,
		},
	}
	respBody, statusCode, err = r.doJSON(http.MethodPut, path, req)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("创建集合失败 collection=%s status=%d body=%s", collection, statusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (r *QdrantRepository) UpsertMemorySummary(summary MemorySummary) (string, error) {
	collection, ownerId, err := r.collectionByScope(summary.Scope, summary.UserId, summary.GroupId)
	if err != nil {
		return "", err
	}

	text := strings.TrimSpace(summary.Text)
	if text == "" {
		text = strings.TrimSpace(summary.Summary)
	}
	if text == "" {
		return "", fmt.Errorf("summary 文本为空")
	}

	vector, err := r.embedder.Embed(text)
	if err != nil {
		return "", err
	}
	if len(vector) != r.vectorSize {
		return "", fmt.Errorf("embedding 维度不匹配 got=%d want=%d", len(vector), r.vectorSize)
	}

	dedupKey := BuildDedupKey(summary.Scope, ownerId, text)
	pointID := BuildPointID(dedupKey)
	payload := r.buildPayload(summary, text, dedupKey)
	point := map[string]interface{}{
		"id":      pointID,
		"vector":  vector,
		"payload": payload,
	}

	req := map[string]interface{}{
		"points": []interface{}{point},
	}
	upsertPath := "/collections/" + url.PathEscape(collection) + "/points?wait=true"
	respBody, statusCode, err := r.doJSON(http.MethodPut, upsertPath, req)
	if err != nil {
		return "", err
	}
	if statusCode < 200 || statusCode >= 300 {
		return "", fmt.Errorf("qdrant upsert 失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}
	return dedupKey, nil
}

func UpsertMemorySummary(summary MemorySummary) (string, error) {
	repo, err := GetQdrantRepository()
	if err != nil {
		return "", err
	}
	return repo.UpsertMemorySummary(summary)
}

func (r *QdrantRepository) collectionByScope(scope string, userId string, groupId string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "user":
		userId = strings.TrimSpace(userId)
		if userId == "" {
			return "", "", fmt.Errorf("scope=user 时 user_id 不能为空")
		}
		return r.collectionUser, userId, nil
	case "group":
		groupId = strings.TrimSpace(groupId)
		if groupId == "" {
			return "", "", fmt.Errorf("scope=group 时 group_id 不能为空")
		}
		return r.collectionGroup, groupId, nil
	default:
		return "", "", fmt.Errorf("不支持的 scope: %s", scope)
	}
}

func (r *QdrantRepository) buildPayload(summary MemorySummary, text string, dedupKey string) map[string]interface{} {
	createdAt := summary.CreatedAt
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	return map[string]interface{}{
		"scope":            strings.ToLower(strings.TrimSpace(summary.Scope)),
		"user_id":          strings.TrimSpace(summary.UserId),
		"group_id":         strings.TrimSpace(summary.GroupId),
		"session_id":       strings.TrimSpace(summary.SessionId),
		"message_id":       strings.TrimSpace(summary.MessageId),
		"source":           strings.TrimSpace(summary.Source),
		"text":             text,
		"summary":          strings.TrimSpace(summary.Summary),
		"tags":             summary.Tags,
		"importance":       summary.Importance,
		"confidence":       summary.Confidence,
		"created_at":       createdAt,
		"dedup_key":        dedupKey,
		"dedup_window_sec": r.dedupWindow,
	}
}

func (r *QdrantRepository) doJSON(method string, path string, payload interface{}) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, r.baseURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("api-key", r.apiKey)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

func normalizeQdrantDistance(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dot":
		return "Dot"
	case "euclid":
		return "Euclid"
	default:
		return "Cosine"
	}
}
