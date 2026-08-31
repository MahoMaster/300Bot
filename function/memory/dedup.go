package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

// BuildDedupKey 生成幂等去重键：hash(scope + owner + normalized_text)。
func BuildDedupKey(scope string, ownerId string, text string) string {
	normalized := normalizeMemoryText(text)
	raw := strings.ToLower(strings.TrimSpace(scope)) + "|" + strings.TrimSpace(ownerId) + "|" + normalized
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// BuildEntryDedupKey 条目型记忆的幂等去重键：hash(scope + owner + subject + type + key)。
// 与文本无关——同一三元组的记忆无论文本如何更新都落同一个点，UPDATE 天然成立；
// 旧 BuildDedupKey 原样保留给 legacy 记忆，存量点不受影响。
func BuildEntryDedupKey(scope string, ownerId string, subjectId string, memType string, key string) string {
	raw := strings.ToLower(strings.TrimSpace(scope)) + "|" + strings.TrimSpace(ownerId) + "|" +
		strings.TrimSpace(subjectId) + "|" + strings.ToLower(strings.TrimSpace(memType)) + "|" + strings.ToLower(strings.TrimSpace(key))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// BuildPointID 生成满足 Qdrant 要求的稳定 UUID（基于 dedupKey 派生）。
func BuildPointID(dedupKey string) string {
	key := strings.ToLower(strings.TrimSpace(dedupKey))
	if len(key) < 32 {
		sum := sha256.Sum256([]byte(key))
		key = hex.EncodeToString(sum[:])
	}
	key = key[:32]
	return key[0:8] + "-" + key[8:12] + "-" + key[12:16] + "-" + key[16:20] + "-" + key[20:32]
}

func normalizeMemoryText(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	lastSpace := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}
