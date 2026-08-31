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

// 条目记忆 status 取值：软删语义，永不硬删，可回滚；存量点无该字段不受 must_not 过滤影响
const (
	memoryStatusActive  = "active"
	memoryStatusDeleted = "deleted"
	memoryStatusExpired = "expired"
	memoryStatusMerged  = "merged"
)

// MemoryPointRecord scroll 返回的单条点：点 ID + 全量 payload，供 Manager 裁决与生命周期任务使用
type MemoryPointRecord struct {
	Id      string
	Payload map[string]interface{}
}

func (r MemoryPointRecord) PayloadString(key string) string {
	if v, ok := r.Payload[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func (r MemoryPointRecord) PayloadInt64(key string) int64 {
	if v, ok := r.Payload[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func (r MemoryPointRecord) PayloadFloat(key string) float64 {
	if v, ok := r.Payload[key].(float64); ok {
		return v
	}
	return 0
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
	if err := r.ensureOneCollection(r.collectionGroup); err != nil {
		return err
	}
	// payload 索引失败不阻断启动：小集合下无索引过滤检索仍可用，仅性能退化
	r.ensurePayloadIndexes(r.collectionUser)
	r.ensurePayloadIndexes(r.collectionGroup)
	return nil
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

// ensurePayloadIndexes 幂等创建 keyword 索引：subject_id/mem_type/mem_key/status，
// 支撑 Manager 的精确过滤检索（按人按属性查旧记忆），避免随点数增长线性退化；
// 重复创建返回 200，失败仅告警不阻断启动
func (r *QdrantRepository) ensurePayloadIndexes(collection string) {
	for _, field := range []string{"subject_id", "mem_type", "mem_key", "status"} {
		req := map[string]interface{}{
			"field_name":  field,
			"field_schema": "keyword",
		}
		respBody, statusCode, err := r.doJSON(http.MethodPut, "/collections/"+url.PathEscape(collection)+"/index", req)
		if err != nil {
			log.Printf("memory qdrant payload index failed collection=%s field=%s err=%v (continue)", collection, field, err)
			continue
		}
		if statusCode < 200 || statusCode >= 300 {
			log.Printf("memory qdrant payload index failed collection=%s field=%s status=%d body=%s (continue)", collection, field, statusCode, strings.TrimSpace(string(respBody)))
		}
	}
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
	if summary.IsEntry() && conf.Memory.MemoryStructuredPayloadEnabled {
		// 条目记忆用三元组键：同 key 记忆文本更新后仍落同一点，为 Manager 的 UPDATE 铺路；
		// legacy 记忆仍用文本哈希键，存量点与旧行为不变
		dedupKey = BuildEntryDedupKey(summary.Scope, ownerId, summary.SubjectId, summary.Type, summary.Key)
	}
	pointID := BuildPointID(dedupKey)

	// dedup_window 真实生效（P14）：窗口内重复记忆直接跳过，不刷时间戳、不重复 embed；
	// 同点但文本已变更（条目更新/合并）不算新鲜，放行覆盖
	if r.dedupWindow > 0 {
		fresh, err := r.pointFresh(collection, pointID, text)
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
// 404/无 created_at 返回 false；点存在但存储文本与传入文本不一致时同样返回 false（放行覆盖）
func (r *QdrantRepository) pointFresh(collection string, pointID string, text string) (bool, error) {
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
	if storedText, ok := got.Result.Payload["text"].(string); ok && strings.TrimSpace(storedText) != "" && strings.TrimSpace(storedText) != text {
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
		// must_not 排除死亡状态点；缺失 status 字段的存量点不受影响（Qdrant 对缺失字段不命中 must_not）
		"filter": deadStatusFilter([]interface{}{
			map[string]interface{}{
				"key":   filterKey,
				"match": map[string]interface{}{"value": ownerId},
			},
		}),
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
	payload := map[string]interface{}{
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
	if summary.IsEntry() && conf.Memory.MemoryStructuredPayloadEnabled {
		// 条目型记忆追加结构化字段；召回只读 text/summary，新字段对其透明；
		// legacy 记忆输出逐字节不变，存量回灌与旧点零影响
		evidenceCount := summary.EvidenceCount
		if evidenceCount <= 0 {
			evidenceCount = 1
		}
		payload["subject_id"] = strings.TrimSpace(summary.SubjectId)
		payload["mem_type"] = strings.TrimSpace(summary.Type)
		payload["mem_key"] = strings.TrimSpace(summary.Key)
		payload["status"] = memoryStatusActive
		payload["schema"] = "entry_v1"
		payload["evidence"] = strings.TrimSpace(summary.Evidence)
		payload["evidence_count"] = evidenceCount
		payload["updated_at"] = createdAt
	}
	return payload
}

// deadStatusFilter 包装 must 条件并追加 must_not 排除死亡状态（删除/过期/已并入）的点；
// 存量点无 status 字段，Qdrant 对缺失字段不命中 must_not，不会被误排除——安全方向硬约束
func deadStatusFilter(must []interface{}) map[string]interface{} {
	return map[string]interface{}{
		"must": must,
		"must_not": []interface{}{
			map[string]interface{}{
				"key": "status",
				"match": map[string]interface{}{
					"any": []string{memoryStatusDeleted, memoryStatusExpired, memoryStatusMerged},
				},
			},
		},
	}
}

// ScrollEntries 精确过滤检索（不带向量）：owner + subject_id + mem_type[+mem_key] 取旧记忆，
// 供 Manager 裁决前查同人同属性已有条目；比纯向量搜索确定，程序精确匹配优先原则的落点。
// memType/memKey 为空时放宽对应条件；limit 默认 8。
func (r *QdrantRepository) ScrollEntries(ctx context.Context, scope string, ownerId string, subjectId string, memType string, memKey string, limit int) ([]MemoryPointRecord, error) {
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
	subjectId = strings.TrimSpace(subjectId)
	if ownerId == "" || subjectId == "" {
		return nil, fmt.Errorf("scroll 需要 ownerId 与 subjectId 均非空")
	}
	if limit <= 0 {
		limit = 8
	}

	must := []interface{}{
		map[string]interface{}{"key": filterKey, "match": map[string]interface{}{"value": ownerId}},
		map[string]interface{}{"key": "subject_id", "match": map[string]interface{}{"value": subjectId}},
	}
	if memType = strings.TrimSpace(memType); memType != "" {
		must = append(must, map[string]interface{}{"key": "mem_type", "match": map[string]interface{}{"value": memType}})
	}
	if memKey = strings.TrimSpace(memKey); memKey != "" {
		must = append(must, map[string]interface{}{"key": "mem_key", "match": map[string]interface{}{"value": memKey}})
	}

	req := map[string]interface{}{
		"limit":        limit,
		"with_payload": true,
		"filter":       deadStatusFilter(must),
	}
	path := "/collections/" + url.PathEscape(collection) + "/points/scroll"
	respBody, statusCode, err := r.doJSONCtx(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("qdrant scroll 失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}
	var resp struct {
		Result struct {
			Points []struct {
				Id      string                 `json:"id"`
				Payload map[string]interface{} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}
	records := make([]MemoryPointRecord, 0, len(resp.Result.Points))
	for _, point := range resp.Result.Points {
		if strings.TrimSpace(point.Id) == "" {
			continue
		}
		records = append(records, MemoryPointRecord{Id: point.Id, Payload: point.Payload})
	}
	return records, nil
}

// ScrollDecayBatch 分页扫描条目型记忆（仅命中 schema=entry_v1，存量点不受影响），供生命周期任务；
// offset 传上一页返回的 next_page_offset（首页空串），返回本页点与下一页偏移（空串表示无更多）
func (r *QdrantRepository) ScrollDecayBatch(ctx context.Context, scope string, offset string, limit int) ([]MemoryPointRecord, string, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	var collection string
	switch scope {
	case "user":
		collection = r.collectionUser
	case "group":
		collection = r.collectionGroup
	default:
		return nil, "", fmt.Errorf("不支持的 scope: %s", scope)
	}
	if limit <= 0 {
		limit = 200
	}
	req := map[string]interface{}{
		"limit":        limit,
		"with_payload": true,
		"filter": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{"key": "schema", "match": map[string]interface{}{"value": "entry_v1"}},
			},
		},
	}
	if offset = strings.TrimSpace(offset); offset != "" {
		req["offset"] = offset
	}
	path := "/collections/" + url.PathEscape(collection) + "/points/scroll"
	respBody, statusCode, err := r.doJSONCtx(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, "", err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, "", fmt.Errorf("qdrant decay scroll 失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}
	var resp struct {
		Result struct {
			Points []struct {
				Id      string                 `json:"id"`
				Payload map[string]interface{} `json:"payload"`
			} `json:"points"`
			NextPageOffset string `json:"next_page_offset"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, "", err
	}
	records := make([]MemoryPointRecord, 0, len(resp.Result.Points))
	for _, point := range resp.Result.Points {
		if strings.TrimSpace(point.Id) == "" {
			continue
		}
		records = append(records, MemoryPointRecord{Id: point.Id, Payload: point.Payload})
	}
	return records, strings.TrimSpace(resp.Result.NextPageOffset), nil
}

// SetPayloads 仅更新 payload 不动向量（POST /points/payload）：
// evidence_count/status/updated_at 等元数据变更零重新 embed，是裁决省钱的关键路径；
// 点 ID 不存在时 Qdrant 返回 400，由调用方按失败处置

func (r *QdrantRepository) SetPayloads(scope string, userId string, groupId string, pointIDs []string, kv map[string]interface{}) error {
	collection, _, err := r.collectionByScope(scope, userId, groupId)
	if err != nil {
		return err
	}
	if len(pointIDs) == 0 || len(kv) == 0 {
		return nil
	}
	req := map[string]interface{}{
		"points":  pointIDs,
		"payload": kv,
	}
	path := "/collections/" + url.PathEscape(collection) + "/points/payload?wait=true"
	respBody, statusCode, err := r.doJSON(http.MethodPost, path, req)
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("qdrant set payload 失败 status=%d body=%s", statusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
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
