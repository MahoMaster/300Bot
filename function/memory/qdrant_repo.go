package memory

import (
	"300Bot/conf"
	"300Bot/function/memory/recall"
	"bytes"
	"context"
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

	dedupKey := BuildDedupKey(summary.Scope, ownerId, text)
	pointID := BuildPointID(dedupKey)

	// dedup_window 真实生效（P14）：窗口内重复记忆直接跳过，不刷时间戳、不重复 embed
	if r.dedupWindow > 0 {
		fresh, err := r.pointFresh(collection, pointID)
		if err != nil {
			log.Printf("memory dedup check failed scope=%s err=%v (continue upsert)", summary.Scope, err)
		} else if fresh {
			return dedupKey, nil
		}
	}

	vector, err := r.embedder.Embed(text)
	if err != nil {
		return "", err
	}
	if len(vector) != r.vectorSize {
		return "", fmt.Errorf("embedding 维度不匹配 got=%d want=%d", len(vector), r.vectorSize)
	}

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

// pointFresh 检查同 dedupKey 的点是否已存在且仍在去重窗口内（now - created_at < dedupWindow）；
// 404/无 created_at 返回 false
func (r *QdrantRepository) pointFresh(collection string, pointID string) (bool, error) {
	path := "/collections/" + url.PathEscape(collection) + "/points/" + url.PathEscape(pointID)
	respBody, statusCode, err := r.doJSON(http.MethodGet, path, nil)
	if err != nil {
		return false, err
	}
	if statusCode == http.StatusNotFound {
		return false, nil
	}
	if statusCode < 200 || statusCode >= 300 {
		return false, fmt.Errorf("qdrant point get 失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}
	var got struct {
		Result struct {
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &got); err != nil {
		return false, err
	}
	createdAt, _ := got.Result.Payload["created_at"].(float64)
	if createdAt <= 0 {
		return false, nil
	}
	return time.Now().Unix()-int64(createdAt) < int64(r.dedupWindow), nil
}

// EmbedQuery 将查询文本转向量，供召回链路在两个 collection 间共用（只 embed 一次）
func (r *QdrantRepository) EmbedQuery(query string) ([]float32, error) {
	return r.embedder.Embed(query)
}

// Search 在 scope 对应集合中检索与 vector 最相似的 topK 个记忆点，
// payload filter 按 ownerId 过滤（user→user_id、group→group_id），带 ctx 预算控制
func (r *QdrantRepository) Search(ctx context.Context, scope string, ownerId string, vector []float32, topK int) ([]recall.MemoryHit, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	var collection, filterKey string
	switch scope {
	case "user":
		collection, filterKey = r.collectionUser, "user_id"
	case "group":
		collection, filterKey = r.collectionGroup, "group_id"
	default:
		return nil, fmt.Errorf("不支持的 scope: %s", scope)
	}
	ownerId = strings.TrimSpace(ownerId)
	if ownerId == "" {
		return nil, fmt.Errorf("scope=%s 时 ownerId 不能为空", scope)
	}
	if topK <= 0 {
		topK = 4
	}
	if len(vector) != r.vectorSize {
		return nil, fmt.Errorf("embedding 维度不匹配 got=%d want=%d", len(vector), r.vectorSize)
	}

	req := map[string]interface{}{
		"vector":       vector,
		"limit":        topK,
		"with_payload": true,
		"filter": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"key":   filterKey,
					"match": map[string]interface{}{"value": ownerId},
				},
			},
		},
	}
	path := "/collections/" + url.PathEscape(collection) + "/points/search"
	respBody, statusCode, err := r.doJSONCtx(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("qdrant search 失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}
	return recall.ParseSearchResponse(respBody, scope)
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
	return r.doJSONCtx(context.Background(), method, path, payload)
}

// doJSONCtx 与 doJSON 相同，但请求携带 ctx，供召回等带预算的链路使用
func (r *QdrantRepository) doJSONCtx(ctx context.Context, method string, path string, payload interface{}) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, body)
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
