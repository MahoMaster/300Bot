package memory

import (
	"encoding/json"
	"fmt"
	"strings"
)

// prompts.go 集中管理记忆提取与裁决的 prompt、输出 schema 结构体与纯解析函数，
// 便于独立迭代；解析/规范化均为纯函数，单元测试不依赖 conf。

// 条目化提取每批上限（与 inline.MaxCandidates 对齐，控制逐条入队的队列压力）
const memoryEntryMaxPerBatch = 5

// 单条 value 截断长度（与 inline.MaxCandidateRunes 对齐）
const memoryEntryMaxValueRunes = 500

// memoryEntryExtractPrompt 收紧版提取 prompt：只提取"以后再次遇到这个人时仍有价值"的信息，
// 明确禁止一次性事件；输出结构化 memories[] 而非分类式 facts/preferences
const memoryEntryExtractPrompt = `你是群聊长期记忆提取器。

你的目标不是总结聊天内容，而是识别"以后再次遇到这个人时，仍然有价值的信息"。核心原则：记住人，而不是记住聊天。

【高价值记忆】
1. 身份信息：QQ号对应的姓名、昵称、别名、曾用名；群内称呼、外号之间的对应关系（type=identity，key=alias）
2. 稳定个人信息：职业、长期居住地、兴趣、家庭情况、宠物等（type=profile）
3. 明确偏好：喜欢/讨厌的游戏、作品、食物、事物（type=preference / type=dislike）
4. 性格和长期行为特征：经常性、稳定可见的行为模式与性格特点（type=trait / type=habit）
5. 群成员关系：朋友、情侣、同事、固定搭档、经常一起活动或互相吐槽的稳定关系（type=relationship）
6. 群内文化：固定梗、固定称呼、特定词语对应的人（type=group_meme）
7. 当前仍然有效的重要目标：正在找工作、准备考试、正在做某项目等（type=goal）
8. 用户明确要求机器人记住的内容（type=rule，importance 给高）

【通常不要记录】
1. 一次性的提问、天气/新闻/搜索/知识咨询
2. 普通闲聊内容
3. 单次游戏行为、单次情绪表达
4. 没有长期价值的事件
5. AI根据上下文推测出来、但用户没有明确表达的信息

【特别注意】"发生过"不等于"值得长期记忆"。
"某人询问了成都今天的天气"——不是长期记忆。
"某人长期生活在成都"——是长期记忆。
"某人今天抱怨朋友不和他打游戏"——通常不是长期记忆。
"某人经常和某朋友一起打游戏，并且经常吐槽他"——可能是长期记忆。
"某人今天凌晨3点还在聊天"——不是长期记忆。
"某人经常凌晨活跃"——可能是长期记忆。

【人物识别】
QQ号是唯一稳定身份键，昵称只是别名或当前显示名称。
转写中形如 用户[昵称](QQ:123) 的前缀标明了发言者身份。
如果有充分证据表明多个昵称属于同一个QQ号，提取为 type=identity、key=alias 的条目；
不得仅凭昵称相似推断为同一个人。涉及关系/称呼类记忆时，value 中引用人物必须用QQ号。

【证据原则】
只有聊天内容明确支持的信息才能提取，每条给出简短原文依据（evidence）。
不要因为一次行为推断长期性格或偏好；对于"喜欢、讨厌、经常、性格"等长期属性需要较强证据，
仅一次行为时通常不要记录；勉强记录的必须压低 importance（≤2）。

【输出】
仅输出 JSON，不要输出其他文字。每条记忆独立评估 importance(1-5整数) 与 confidence(0-1小数)。
type 只能取：identity|profile|preference|dislike|trait|relationship|habit|goal|rule|group_meme
key 为具体属性的小写英文短语（如 current_location、alias、game_dbd）。
{"memories":[{"subject_id":"QQ号","type":"类型","key":"属性键","value":"记忆内容（自然句）","importance":1,"confidence":0.0,"evidence":"原文依据"}]}
如果没有值得长期保存的信息，输出：{"memories":[]}`

// memoryReconcilePrompt 裁决 prompt：Memory Manager 用，判断新候选与相关旧记忆的关系。
// 与提取器优化目标分离：提取器尽量不漏，裁决器尽量不错。
const memoryReconcilePrompt = `你是记忆裁决器。给定一条新的记忆候选与相关的已有记忆列表，判断新候选与已有记忆的关系，输出且仅输出以下决定之一：
- add：没有相关旧记忆，或新候选与旧记忆不冲突且互不重复 → 新增候选
- update：新候选是某条旧记忆的最新状态（如搬家、换工作、改称呼），旧值已过时 → 用新值覆盖该旧记忆
- merge：新候选与某条旧记忆是同一事实的不同表述或互相补充 → 合并为一条
- delete：新信息表明某条旧记忆已不再成立或本来就是错的 → 作废该旧记忆，候选本身也不保留
- ignore：候选本身无长期价值，或与旧记忆完全重复无新信息 → 丢弃候选

已有记忆列表为空时只能输出 add 或 ignore。
仅输出 JSON：{"decision":"add|update|merge|delete|ignore","target_point_id":"被 update/merge/delete 的已有记忆ID，add/ignore 时为空","merged_value":"update/merge 后最终保留的记忆文本，其他情况为空","confidence":0.0}`

// MemoryEntry 提取器输出的单条结构化记忆候选
type MemoryEntry struct {
	SubjectId  string  `json:"subject_id"`
	Type       string  `json:"type"`
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	Importance int     `json:"importance"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// entryExtractionResult 提取器输出 schema：{"memories":[...]}
type entryExtractionResult struct {
	Memories []MemoryEntry `json:"memories"`
}

// reconcileDecision 裁决器输出 schema
type reconcileDecision struct {
	Decision      string  `json:"decision"`
	TargetPointId string  `json:"target_point_id"`
	MergedValue   string  `json:"merged_value"`
	Confidence    float64 `json:"confidence"`
}

// 合法记忆类型集合；未知类型归一到 profile（对 LLM 输出宽容）
var memoryEntryValidTypes = map[string]struct{}{
	"identity": {}, "profile": {}, "preference": {}, "dislike": {}, "trait": {},
	"relationship": {}, "habit": {}, "goal": {}, "rule": {}, "group_meme": {},
}

// parseEntryExtraction 解析提取器输出为候选列表；空数组是合法结果（闲聊批次常态）
func parseEntryExtraction(raw string) ([]MemoryEntry, error) {
	body := extractJSONBody(raw)
	if body == "" {
		return nil, fmt.Errorf("empty json payload")
	}
	var result entryExtractionResult
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return nil, err
	}
	return result.Memories, nil
}

// parseReconcileDecision 解析裁决器输出；解析失败由调用方降级为 ADD（只增不删，安全方向）
func parseReconcileDecision(raw string) (reconcileDecision, error) {
	body := extractJSONBody(raw)
	if body == "" {
		return reconcileDecision{}, fmt.Errorf("empty json payload")
	}
	var decision reconcileDecision
	if err := json.Unmarshal([]byte(body), &decision); err != nil {
		return reconcileDecision{}, err
	}
	decision.Decision = strings.ToLower(strings.TrimSpace(decision.Decision))
	decision.TargetPointId = strings.TrimSpace(decision.TargetPointId)
	decision.MergedValue = strings.TrimSpace(decision.MergedValue)
	decision.Confidence = clampFloat(decision.Confidence, 0, 1)
	return decision, nil
}

// normalizeMemoryEntryType 类型归一：小写去空白，未知类型回落到 profile
func normalizeMemoryEntryType(raw string) string {
	t := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := memoryEntryValidTypes[t]; ok {
		return t
	}
	return "profile"
}

// sanitizeEntries 规范化候选列表：清洗/截断/夹取/按三元组去重/限批上限。
// 非法条目（subject 或 value 为空）直接丢弃；候选为空是合法结果。
func sanitizeEntries(entries []MemoryEntry) []MemoryEntry {
	cleaned := make([]MemoryEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		entry.SubjectId = strings.TrimSpace(entry.SubjectId)
		entry.Key = strings.ToLower(strings.TrimSpace(entry.Key))
		entry.Value = strings.TrimSpace(entry.Value)
		entry.Evidence = strings.TrimSpace(entry.Evidence)
		entry.Type = normalizeMemoryEntryType(entry.Type)
		entry.Importance = clampInt(entry.Importance, 1, 5)
		entry.Confidence = clampFloat(entry.Confidence, 0, 1)
		if entry.SubjectId == "" || entry.Value == "" || entry.Key == "" {
			continue
		}
		if runes := []rune(entry.Value); len(runes) > memoryEntryMaxValueRunes {
			entry.Value = string(runes[:memoryEntryMaxValueRunes])
		}
		dedupTriple := entry.SubjectId + "|" + entry.Type + "|" + entry.Key
		if _, dup := seen[dedupTriple]; dup {
			continue
		}
		seen[dedupTriple] = struct{}{}
		cleaned = append(cleaned, entry)
		if len(cleaned) >= memoryEntryMaxPerBatch {
			break
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

// renderEntryText 将候选渲染为供 embedding 的自然句文本；
// value 已含 QQ 号时不重复拼接，避免向量被噪声稀释
func renderEntryText(entry MemoryEntry) string {
	value := strings.TrimSpace(entry.Value)
	if strings.Contains(value, entry.SubjectId) {
		return value
	}
	return "QQ:" + entry.SubjectId + " " + value
}
