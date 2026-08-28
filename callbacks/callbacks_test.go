package callbacks

import (
	"context"
	"fmt"
	"testing"
)

type rec struct {
	order []string
}

type recordHandler struct {
	rec *rec
}

func (h *recordHandler) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
	h.rec.order = append(h.rec.order, fmt.Sprintf("start:%s", info.Type))
	return ctx
}

func (h *recordHandler) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
	h.rec.order = append(h.rec.order, fmt.Sprintf("end:%s", info.Type))
	return ctx
}

func (h *recordHandler) OnError(ctx context.Context, info *RunInfo, err error) context.Context {
	h.rec.order = append(h.rec.order, fmt.Sprintf("error:%s", info.Type))
	return ctx
}

func TestWithCallbacks_FiresAllTimings(t *testing.T) {
	r := &rec{}
	ctx := WithCallbacks(t.Context(), &recordHandler{rec: r})
	ctx = EnsureRunInfo(ctx, "dashscope", ComponentImage)
	ctx = OnStart(ctx, nil)
	OnEnd(ctx, nil)
	OnError(ctx, fmt.Errorf("boom"))

	want := []string{"start:dashscope", "end:dashscope", "error:dashscope"}
	if len(r.order) != len(want) {
		t.Fatalf("got %v, want %v", r.order, want)
	}
	for i := range want {
		if r.order[i] != want[i] {
			t.Fatalf("got %v, want %v", r.order, want)
		}
	}
}

func TestInitCallbacks_Replaces(t *testing.T) {
	r1 := &rec{}
	r2 := &rec{}
	ctx := WithCallbacks(t.Context(), &recordHandler{rec: r1})
	ctx = InitCallbacks(t.Context(), &RunInfo{Type: "agnes", Component: ComponentImage}, &recordHandler{rec: r2})
	ctx = OnStart(ctx, nil)

	if len(r1.order) != 0 {
		t.Fatalf("old handler should not fire, got %v", r1.order)
	}
	if len(r2.order) != 1 || r2.order[0] != "start:agnes" {
		t.Fatalf("new handler should fire with new RunInfo, got %v", r2.order)
	}
}

type namedHandler struct {
	name string
	rec  *rec
}

func (h *namedHandler) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
	h.rec.order = append(h.rec.order, "start:"+h.name)
	return ctx
}

func (h *namedHandler) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
	h.rec.order = append(h.rec.order, "end:"+h.name)
	return ctx
}

func (h *namedHandler) OnError(ctx context.Context, info *RunInfo, err error) context.Context {
	h.rec.order = append(h.rec.order, "error:"+h.name)
	return ctx
}

func TestOnStart_ReverseOrder_OnEndForward(t *testing.T) {
	r := &rec{}
	ctx := WithCallbacks(t.Context(),
		&namedHandler{name: "h1", rec: r}, // 先注册
		&namedHandler{name: "h2", rec: r}, // 后注册
	)
	ctx = EnsureRunInfo(ctx, "x", ComponentAudio)
	OnStart(ctx, nil)
	OnEnd(ctx, nil)

	// OnStart 倒序（h2 先于 h1），OnEnd 正序（h1 先于 h2）
	want := []string{"start:h2", "start:h1", "end:h1", "end:h2"}
	if len(r.order) != len(want) {
		t.Fatalf("got %v, want %v", r.order, want)
	}
	for i := range want {
		if r.order[i] != want[i] {
			t.Fatalf("got %v, want %v", r.order, want)
		}
	}
}

func TestHandlerBuilder_StatePassing(t *testing.T) {
	type key struct{}
	r := &rec{}
	h := NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
			r.order = append(r.order, "start")
			return context.WithValue(ctx, key{}, "hello")
		}).
		OnEndFn(func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
			if v, _ := ctx.Value(key{}).(string); v != "hello" {
				t.Fatalf("state lost between OnStart and OnEnd: %v", v)
			}
			r.order = append(r.order, "end")
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *RunInfo, err error) context.Context {
			r.order = append(r.order, "error")
			return ctx
		}).
		Build()

	ctx := WithCallbacks(t.Context(), h)
	ctx = EnsureRunInfo(ctx, "x", ComponentImage)
	ctx = OnStart(ctx, nil)
	OnEnd(ctx, nil)

	if len(r.order) != 2 || r.order[0] != "start" || r.order[1] != "end" {
		t.Fatalf("got %v", r.order)
	}
}

func TestHandlerBuilder_NeededSkipsUnregistered(t *testing.T) {
	r := &rec{}
	h := NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
			r.order = append(r.order, "start")
			return ctx
		}).
		Build()

	ctx := WithCallbacks(t.Context(), h)
	ctx = EnsureRunInfo(ctx, "x", ComponentImage)
	OnStart(ctx, nil)
	OnEnd(ctx, nil) // 未注册 OnEnd，必须被 Needed 跳过（而非 panic）

	if len(r.order) != 1 || r.order[0] != "start" {
		t.Fatalf("got %v", r.order)
	}
}

func TestNoManager_NoOp(t *testing.T) {
	r := &rec{}
	ctx := OnStart(t.Context(), nil)
	OnEnd(ctx, nil)
	OnError(ctx, fmt.Errorf("boom"))
	if len(r.order) != 0 {
		t.Fatalf("no callbacks should fire without manager, got %v", r.order)
	}
}

func TestEnsureRunInfo_KeepsExisting(t *testing.T) {
	r := &rec{}
	ctx := InitCallbacks(t.Context(), &RunInfo{Name: "outer", Type: "mimo", Component: ComponentAudio}, &recordHandler{rec: r})
	ctx = EnsureRunInfo(ctx, "dashscope", ComponentImage) // 已存在 RunInfo，不应覆盖
	OnStart(ctx, nil)

	if len(r.order) != 1 || r.order[0] != "start:mimo" {
		t.Fatalf("got %v", r.order)
	}
}

func TestWithCallbacks_AppendsToExisting(t *testing.T) {
	r := &rec{}
	ctx := WithCallbacks(t.Context(), &namedHandler{name: "h1", rec: r})
	ctx = WithCallbacks(ctx, &namedHandler{name: "h2", rec: r}) // 追加到已有 manager
	ctx = EnsureRunInfo(ctx, "x", ComponentImage)
	OnStart(ctx, nil)

	// 两个 handler 都触发，OnStart 倒序：h2 先于 h1
	want := []string{"start:h2", "start:h1"}
	if len(r.order) != len(want) {
		t.Fatalf("got %v, want %v", r.order, want)
	}
	for i := range want {
		if r.order[i] != want[i] {
			t.Fatalf("got %v, want %v", r.order, want)
		}
	}
}

func TestReuseHandlers_NestedIsolation(t *testing.T) {
	var outerEnds, innerEnds []string
	h := NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
			if info.Type == "outer" {
				// 模拟组件嵌套：内层组件用 ReuseHandlers 复用 handlers、独立 RunInfo
				innerCtx := ReuseHandlers(ctx, &RunInfo{Type: "inner", Component: ComponentAudio})
				innerCtx = OnStart(innerCtx, nil)
				OnEnd(innerCtx, nil)
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
			if info.Type == "outer" {
				outerEnds = append(outerEnds, "end:"+info.Type)
			}
			if info.Type == "inner" {
				innerEnds = append(innerEnds, "end:"+info.Type)
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *RunInfo, err error) context.Context {
			return ctx
		}).
		Build()

	ctx := WithCallbacks(t.Context(), h)
	ctx = EnsureRunInfo(ctx, "outer", ComponentImage)
	ctx = OnStart(ctx, nil)
	OnEnd(ctx, nil)

	// 内层 OnEnd 用内层 RunInfo
	if len(innerEnds) != 1 || innerEnds[0] != "end:inner" {
		t.Fatalf("inner OnEnd should use inner RunInfo, got %v", innerEnds)
	}
	// 外层 OnEnd 不受内层影响，仍用外层 RunInfo
	if len(outerEnds) != 1 || outerEnds[0] != "end:outer" {
		t.Fatalf("outer OnEnd got polluted RunInfo, got %v", outerEnds)
	}
}
