package agenttool

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"300Bot/function/memory/recall"
)

// fakeSearch 记录调用入参并按脚本返回，用于断言执行器的路解析与降级行为
type fakeSearch struct {
	called    bool
	userId    string
	groupId   string
	query     string
	userHits  []recall.MemoryHit
	groupHits []recall.MemoryHit
	err       error
}

func (f *fakeSearch) fn() SearchFn {
	return func(ctx context.Context, userId, groupId, query string) ([]recall.MemoryHit, []recall.MemoryHit, error) {
		f.called = true
		f.userId, f.groupId, f.query = userId, groupId, query
		return f.userHits, f.groupHits, f.err
	}
}

func newOpts(f *fakeSearch) RecallToolOptions {
	return RecallToolOptions{Search: f.fn(), TopK: 3, MinScore: 0.1, MaxChars: 500, Budget: time.Second}
}

func identityCtx(userQQ, groupID string) context.Context {
	return WithRecallIdentity(context.Background(), RecallIdentity{UserQQ: userQQ, GroupID: groupID})
}

// ① 缺省 scope：两路都查，合并渲染输出
func TestRecallBothScopes(t *testing.T) {
	f := &fakeSearch{
		userHits:  []recall.MemoryHit{{Score: 0.9, Text: "123喜欢猫"}},
		groupHits: []recall.MemoryHit{{Score: 0.8, Text: "群里讨论过天气"}},
	}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(identityCtx("123", "456"), `{"query":"猫"}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !f.called || f.userId != "123" || f.groupId != "456" || f.query != "猫" {
		t.Fatalf("search args mismatch: called=%v userId=%s groupId=%s query=%s", f.called, f.userId, f.groupId, f.query)
	}
	if !strings.Contains(out, "【关于对方的既有记忆】") || !strings.Contains(out, "123喜欢猫") || !strings.Contains(out, "群里讨论过天气") {
		t.Fatalf("rendered output missing hits: %s", out)
	}
}

// ② scope=user：只传 user 路
func TestRecallScopeUser(t *testing.T) {
	f := &fakeSearch{userHits: []recall.MemoryHit{{Score: 0.9, Text: "123喜欢猫"}}}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(identityCtx("123", "456"), `{"query":"猫","scope":"user"}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if f.userId != "123" || f.groupId != "" {
		t.Fatalf("expected user-only search, got userId=%s groupId=%s", f.userId, f.groupId)
	}
	if !strings.Contains(out, "123喜欢猫") || strings.Contains(out, "群里讨论过天气") {
		t.Fatalf("unexpected output: %s", out)
	}
}

// ③ scope=group 且无私聊群号：软提示且不发起检索
func TestRecallScopeGroupInPrivate(t *testing.T) {
	f := &fakeSearch{}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(identityCtx("123", ""), `{"query":"群","scope":"group"}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if f.called {
		t.Fatal("search should not be called in private chat group scope")
	}
	if !strings.Contains(out, "私聊") {
		t.Fatalf("expected private-chat hint, got: %s", out)
	}
}

// ④ query 为空：软提示
func TestRecallEmptyQuery(t *testing.T) {
	f := &fakeSearch{}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(identityCtx("123", "456"), `{"query":"  "}`)
	if err != nil || f.called {
		t.Fatalf("expected soft hint without search, err=%v called=%v", err, f.called)
	}
	if !strings.Contains(out, "不能为空") {
		t.Fatalf("expected empty-query hint, got: %s", out)
	}
}

// ⑤ 身份缺失：软提示
func TestRecallNoIdentity(t *testing.T) {
	f := &fakeSearch{}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(context.Background(), `{"query":"猫"}`)
	if err != nil || f.called {
		t.Fatalf("expected soft hint without search, err=%v called=%v", err, f.called)
	}
	if !strings.Contains(out, "缺少发言人身份") {
		t.Fatalf("expected identity hint, got: %s", out)
	}
}

// ⑥ 零命中：固定提示文案
func TestRecallNoHits(t *testing.T) {
	f := &fakeSearch{}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(identityCtx("123", "456"), `{"query":"不存在的话题"}`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out != "未检索到相关记忆。" {
		t.Fatalf("expected no-hit message, got: %s", out)
	}
}

// ⑦ Search 报错：软降级提示，不返回 error
func TestRecallSearchError(t *testing.T) {
	f := &fakeSearch{err: errors.New("qdrant down")}
	tool := NewRecallMemoryTool(newOpts(f))
	out, err := tool.Run(identityCtx("123", "456"), `{"query":"猫"}`)
	if err != nil {
		t.Fatalf("expected soft failure, got err: %v", err)
	}
	if !strings.Contains(out, "暂时不可用") {
		t.Fatalf("expected degraded hint, got: %s", out)
	}
}

// ⑧ WithRecallIdentity 存取往返 + 非法参数返回 error
func TestRecallIdentityRoundTrip(t *testing.T) {
	id, ok := recallIdentityFrom(identityCtx("123", "456"))
	if !ok || id.UserQQ != "123" || id.GroupID != "456" {
		t.Fatalf("identity round trip failed: ok=%v id=%+v", ok, id)
	}
	if _, ok := recallIdentityFrom(context.Background()); ok {
		t.Fatal("expected no identity on bare context")
	}

	tool := NewRecallMemoryTool(newOpts(&fakeSearch{}))
	if _, err := tool.Run(identityCtx("123", "456"), `{invalid json`); err == nil {
		t.Fatal("expected error on invalid args JSON")
	}
}
