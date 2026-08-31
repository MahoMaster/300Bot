package memory

import (
	"300Bot/conf"
	"300Bot/model"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const memorySummaryConfidenceThreshold = 0.65

// 单次证据封顶：仅一次行为不得升级为高置信长期偏好（证据原则的程序兑现）
const memorySingleEvidenceConfidenceCap = 0.75

// 提取/裁决 LLM 调用超时：修复原先 context.Background() 无超时，慢响应长期霸占 owner 锁的隐患
const memoryLLMCallTimeoutSec = 60

type memorySummaryResult struct {
	Summary     string   `json:"summary"`
	Facts       []string `json:"facts"`
	Preferences []string `json:"preferences"`
	Rules       []string `json:"rules"`
	Goals       []string `json:"goals"`
	Importance  int      `json:"importance"`
	Confidence  float64  `json:"confidence"`
}

var (
	memorySummaryClientOnce sync.Once
	memorySummaryClient     *openai.Client
	memorySummaryClientErr  error
	memoryOwnerLocks        sync.Map
)

func TryBatchSummarizeOwner(scope string, userId string, groupId string) {
	tryBatchSummarizeOwner(scope, userId, groupId, false)
}

// TryBatchSummarizeOwnerForced 事件触发通道：跳过批量三阈值立即提取存量 pending turns；
// 其余逻辑（owner 锁/转写/门槛/入队/状态标记）与批量路径完全复用。
// 触发器只负责"提前"，定性仍走提取+裁决两道 LLM 关卡，玩笑话不会被直接当指令入库。
func TryBatchSummarizeOwnerForced(scope string, userId string, groupId string) {
	tryBatchSummarizeOwner(scope, userId, groupId, true)
}

func tryBatchSummarizeOwner(scope string, userId string, groupId string, force bool) {
	if !conf.Memory.MemoryEnabled || !conf.Memory.MemoryBatchEnabled {
		return
	}
	scope = strings.TrimSpace(scope)
	ownerId := selectOwnerID(scope, userId, groupId)
	if ownerId == "" {
		return
	}

	lockKey := scope + ":" + ownerId
	lock := getOwnerLock(lockKey)
	lock.Lock()
	defer lock.Unlock()

	turns := model.GetPendingMemoryRawTurnsByOwner(scope, ownerId, conf.Memory.MemoryBatchMaxTurns*3)
	if len(turns) == 0 {
		return
	}

	totalChars := 0
	for _, turn := range turns {
		totalChars += len([]rune(turn.InputText))
		totalChars += len([]rune(turn.ReplyText))
	}
	waitSec := 0
	if turns[0].CreatedAt > 0 {
		waitSec = int(nowUnix() - turns[0].CreatedAt)
	}

	if !force && len(turns) < conf.Memory.MemoryBatchMaxTurns &&
		totalChars < conf.Memory.MemoryBatchMaxChars &&
		waitSec < conf.Memory.MemoryBatchMaxWaitSec {
		return
	}

	selectedTurns, transcript := buildSummaryTranscript(turns)
	if len(selectedTurns) == 0 || transcript == "" {
		return
	}
	ids := make([]int64, 0, len(selectedTurns))
	for _, turn := range selectedTurns {
		ids = append(ids, turn.Id)
	}

	// 条目化提取（开关）：输出结构化候选逐条入队；关闭时走原分类式总结路径，行为不变
	if conf.Memory.MemoryEntryExtractEnabled {
		summarizeEntries(scope, ownerId, transcript, selectedTurns, ids)
		return
	}

	summary, err := callStructuredSummary(scope, ownerId, transcript)
	if err != nil {
		log.Printf("memory summarize failed scope=%s owner=%s err=%v", scope, ownerId, err)
		_ = model.MarkMemoryRawTurnsStatus(ids, "failed")
		return
	}

	passImportance := summary.Importance >= conf.Memory.MemoryMinImportance
	passConfidence := summary.Confidence >= memorySummaryConfidenceThreshold
	if passImportance && passConfidence {
		log.Printf("memory summarized scope=%s owner=%s turns=%d importance=%d confidence=%.2f summary=%s", scope, ownerId, len(selectedTurns), summary.Importance, summary.Confidence, summary.Summary)
		memorySummary := buildMemorySummary(scope, selectedTurns, summary)
		if err = EnqueueMemoryTask(memorySummary); err != nil {
			log.Printf("memory enqueue failed scope=%s owner=%s err=%v", scope, ownerId, err)
		}
	} else {
		log.Printf("memory summary filtered scope=%s owner=%s turns=%d importance=%d confidence=%.2f", scope, ownerId, len(selectedTurns), summary.Importance, summary.Confidence)
	}

	if err := model.MarkMemoryRawTurnsStatus(ids, "summarized"); err != nil {
		log.Printf("memory mark summarized failed scope=%s owner=%s err=%v", scope, ownerId, err)
	}
}

// summarizeEntries 条目化提取分流：收紧版 prompt 提取结构化候选 → 逐条门槛过滤 → 逐条入队。
// 候选为空（闲聊批次常态）同样把 turns 标为 summarized，避免反复重提；
// 提取失败与旧路径一致不标记，留给 5 分钟补扫重试。
func summarizeEntries(scope string, ownerId string, transcript string, selectedTurns []model.MemoryRawTurn, ids []int64) {
	entries, err := callEntryExtraction(scope, ownerId, transcript)
	if err != nil {
		log.Printf("memory entry extract failed scope=%s owner=%s err=%v", scope, ownerId, err)
		return
	}
	entries = sanitizeEntries(entries)
	accepted := 0
	for _, entry := range entries {
		if entry.Importance < conf.Memory.MemoryMinImportance {
			continue
		}
		if entry.Confidence < memorySummaryConfidenceThreshold {
			continue
		}
		if entry.Confidence > memorySingleEvidenceConfidenceCap {
			entry.Confidence = memorySingleEvidenceConfidenceCap
		}
		summary := buildEntrySummary(scope, selectedTurns, entry)
		if err = EnqueueMemoryTask(summary); err != nil {
			log.Printf("memory entry enqueue failed scope=%s owner=%s key=%s err=%v", scope, ownerId, entry.Key, err)
			continue
		}
		accepted++
		log.Printf("memory entry extracted scope=%s owner=%s subject=%s type=%s key=%s importance=%d confidence=%.2f value=%s",
			scope, ownerId, entry.SubjectId, entry.Type, entry.Key, entry.Importance, entry.Confidence, entry.Value)
	}
	log.Printf("memory entry batch done scope=%s owner=%s turns=%d candidates=%d accepted=%d", scope, ownerId, len(selectedTurns), len(entries), accepted)
	if err := model.MarkMemoryRawTurnsStatus(ids, "summarized"); err != nil {
		log.Printf("memory mark summarized failed scope=%s owner=%s err=%v", scope, ownerId, err)
	}
}

// callEntryExtraction 收紧版提取：与旧总结器共用 client，但带超时且输出结构化候选。
// 提取器优化目标是尽量不漏，价值判断交给逐条门槛与后续 Manager 裁决。
func callEntryExtraction(scope string, ownerId string, transcript string) ([]MemoryEntry, error) {
	client, err := getMemorySummaryClient()
	if err != nil {
		return nil, err
	}
	userPrompt := fmt.Sprintf("scope=%s owner_id=%s\n以下是群聊回合转写，请提取长期记忆候选。\n%s", scope, ownerId, transcript)
	ctx, cancel := context.WithTimeout(context.Background(), memoryLLMCallTimeoutSec*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: conf.Memory.MemorySummaryModel,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: memoryEntryExtractPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
			User: ownerId,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices")
	}
	return parseEntryExtraction(resp.Choices[0].Message.Content)
}

// buildEntrySummary 候选 → MemorySummary（队列/入库唯一载体）：
// Text 为 embedding 文本、Tags=[type]；结构化三元组随车传递供后续结构化落库与裁决。
// 归属字段沿用 buildMemorySummary 的隔离约定：群维度不绑定单一 user_id。
func buildEntrySummary(scope string, turns []model.MemoryRawTurn, entry MemoryEntry) MemorySummary {
	summary := MemorySummary{
		Scope:         strings.ToLower(strings.TrimSpace(scope)),
		Text:          renderEntryText(entry),
		Summary:       entry.Value,
		Tags:          []string{entry.Type},
		Importance:    entry.Importance,
		Confidence:    entry.Confidence,
		SubjectId:     entry.SubjectId,
		Type:          entry.Type,
		Key:           entry.Key,
		Evidence:      entry.Evidence,
		EvidenceCount: 1,
		CreatedAt:     nowUnix(),
	}
	if len(turns) > 0 {
		lastTurn := turns[len(turns)-1]
		if summary.Scope == "user" {
			summary.UserId = strings.TrimSpace(lastTurn.UserId)
		}
		summary.GroupId = strings.TrimSpace(lastTurn.GroupId)
		summary.SessionId = strings.TrimSpace(lastTurn.SessionId)
		summary.MessageId = strings.TrimSpace(lastTurn.MessageId)
		summary.Source = strings.TrimSpace(lastTurn.Source)
		if lastTurn.CreatedAt > 0 {
			summary.CreatedAt = lastTurn.CreatedAt
		}
	}
	return summary
}

func callStructuredSummary(scope string, ownerId string, transcript string) (memorySummaryResult, error) {
	result := memorySummaryResult{}
	client, err := getMemorySummaryClient()
	if err != nil {
		return result, err
	}
	systemPrompt := "你是记忆总结器。请仅输出 JSON，不要输出其他文字。字段要求：summary(string)、facts(string[])、preferences(string[])、rules(string[])、goals(string[])、importance(1-5整数)、confidence(0-1小数)。若信息不足请给空数组，并降低importance/confidence。QQ号是唯一稳定身份键，昵称仅作注释；输出记忆中涉及人物必须引用QQ号，禁止仅凭昵称区分人物。"
	userPrompt := fmt.Sprintf("scope=%s owner_id=%s\n请根据以下回合内容提取长期记忆候选。\n%s", scope, ownerId, transcript)
	ctx, cancel := context.WithTimeout(context.Background(), memoryLLMCallTimeoutSec*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: conf.Memory.MemorySummaryModel,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
			User: ownerId,
		},
	)
	if err != nil {
		return result, err
	}
	if len(resp.Choices) == 0 {
		return result, fmt.Errorf("empty choices")
	}
	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	raw = extractJSONBody(raw)
	if raw == "" {
		return result, fmt.Errorf("empty json payload")
	}
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		return result, err
	}
	result.Importance = clampInt(result.Importance, 1, 5)
	result.Confidence = clampFloat(result.Confidence, 0, 1)
	result.Summary = strings.TrimSpace(result.Summary)
	return result, nil
}

func buildSummaryTranscript(turns []model.MemoryRawTurn) ([]model.MemoryRawTurn, string) {
	maxTurns := conf.Memory.MemoryBatchMaxTurns
	maxChars := conf.Memory.MemoryBatchMaxChars
	if maxTurns <= 0 || maxChars <= 0 {
		return nil, ""
	}
	selected := make([]model.MemoryRawTurn, 0, maxTurns)
	lines := make([]string, 0, maxTurns*2)
	charCount := 0
	for _, turn := range turns {
		if len(selected) >= maxTurns {
			break
		}
		turnChars := len([]rune(turn.InputText)) + len([]rune(turn.ReplyText))
		if len(selected) > 0 && charCount+turnChars > maxChars {
			break
		}
		selected = append(selected, turn)
		charCount += turnChars
		if text := strings.TrimSpace(turn.InputText); text != "" {
			prefix := "USER"
			if strings.EqualFold(strings.TrimSpace(turn.Scope), "group") {
				uid := strings.TrimSpace(turn.UserId)
				nickname := strings.TrimSpace(turn.Nickname)
				switch {
				case uid != "" && nickname != "":
					prefix = "用户[" + nickname + "](QQ:" + uid + ")"
				case uid != "":
					prefix = "用户(QQ:" + uid + ")"
				}
			}
			lines = append(lines, prefix+": "+text)
		}
		if text := strings.TrimSpace(turn.ReplyText); text != "" {
			lines = append(lines, "BOT: "+text)
		}
	}
	return selected, strings.Join(lines, "\n")
}

func buildMemorySummary(scope string, turns []model.MemoryRawTurn, result memorySummaryResult) MemorySummary {
	summary := MemorySummary{
		Scope:      strings.ToLower(strings.TrimSpace(scope)),
		Summary:    strings.TrimSpace(result.Summary),
		Text:       buildMemorySummaryText(result),
		Tags:       buildMemorySummaryTags(result),
		Importance: clampInt(result.Importance, 1, 5),
		Confidence: clampFloat(result.Confidence, 0, 1),
		CreatedAt:  nowUnix(),
	}
	if len(turns) > 0 {
		lastTurn := turns[len(turns)-1]
		normalizedScope := strings.ToLower(strings.TrimSpace(scope))
		if normalizedScope == "user" {
			summary.UserId = strings.TrimSpace(lastTurn.UserId)
		} else {
			// 群维度总结不绑定到单一 user_id，避免误导“归属到某个用户”。
			summary.UserId = ""
		}
		summary.GroupId = strings.TrimSpace(lastTurn.GroupId)
		summary.SessionId = strings.TrimSpace(lastTurn.SessionId)
		summary.MessageId = strings.TrimSpace(lastTurn.MessageId)
		summary.Source = strings.TrimSpace(lastTurn.Source)
		if lastTurn.CreatedAt > 0 {
			summary.CreatedAt = lastTurn.CreatedAt
		}
	}
	return summary
}

func buildMemorySummaryText(result memorySummaryResult) string {
	lines := make([]string, 0, 1+len(result.Facts)+len(result.Preferences)+len(result.Rules)+len(result.Goals))
	if text := strings.TrimSpace(result.Summary); text != "" {
		lines = append(lines, text)
	}
	appendMemoryList(&lines, "事实", result.Facts)
	appendMemoryList(&lines, "偏好", result.Preferences)
	appendMemoryList(&lines, "规则", result.Rules)
	appendMemoryList(&lines, "目标", result.Goals)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func appendMemoryList(lines *[]string, label string, values []string) {
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		*lines = append(*lines, label+"："+item)
	}
}

func buildMemorySummaryTags(result memorySummaryResult) []string {
	tags := make([]string, 0, 4)
	if len(result.Facts) > 0 {
		tags = append(tags, "fact")
	}
	if len(result.Preferences) > 0 {
		tags = append(tags, "preference")
	}
	if len(result.Rules) > 0 {
		tags = append(tags, "rule")
	}
	if len(result.Goals) > 0 {
		tags = append(tags, "goal")
	}
	return tags
}

func getMemorySummaryClient() (*openai.Client, error) {
	memorySummaryClientOnce.Do(func() {
		if strings.TrimSpace(conf.Config.ChatGPTKey) == "" {
			memorySummaryClientErr = fmt.Errorf("chatgpt key is empty")
			return
		}
		cfg := openai.DefaultConfig(conf.Config.ChatGPTKey)
		cfg.BaseURL = conf.Config.ChatGPTBaseUrl
		memorySummaryClient = openai.NewClientWithConfig(cfg)
	})
	if memorySummaryClientErr != nil {
		return nil, memorySummaryClientErr
	}
	return memorySummaryClient, nil
}

func extractJSONBody(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	left := strings.Index(raw, "{")
	right := strings.LastIndex(raw, "}")
	if left < 0 || right < 0 || right <= left {
		return ""
	}
	return raw[left : right+1]
}

func selectOwnerID(scope string, userId string, groupId string) string {
	if scope == "user" {
		return strings.TrimSpace(userId)
	}
	if scope == "group" {
		return strings.TrimSpace(groupId)
	}
	return ""
}

func getOwnerLock(key string) *sync.Mutex {
	lockAny, _ := memoryOwnerLocks.LoadOrStore(key, &sync.Mutex{})
	return lockAny.(*sync.Mutex)
}

func clampInt(v int, min int, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampFloat(v float64, min float64, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func nowUnix() int64 {
	return time.Now().Unix()
}
