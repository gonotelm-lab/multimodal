package callbacks

import "context"

type Component string

const (
	ComponentImage Component = "image"
	ComponentAudio Component = "audio"
)

type RunInfo struct {
	Name      string
	Type      string
	Component Component
}

type CallbackInput any

type CallbackOutput any

type CallbackTiming uint8

const (
	TimingOnStart CallbackTiming = iota
	TimingOnEnd
	TimingOnError
)

type TimingChecker interface {
	Needed(ctx context.Context, info *RunInfo, timing CallbackTiming) bool
}

type Handler interface {
	OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	OnError(ctx context.Context, info *RunInfo, err error) context.Context
}

// --- manager ---

type ctxKey struct{}

type runInfoCtxKey struct{}

type manager struct {
	handlers []Handler
	runInfo  *RunInfo
}

func ctxWithManager(ctx context.Context, m *manager) context.Context {
	return context.WithValue(ctx, ctxKey{}, m)
}

func managerFromCtx(ctx context.Context) (*manager, bool) {
	m, ok := ctx.Value(ctxKey{}).(*manager)
	if !ok || m == nil {
		return nil, false
	}
	return m, true
}

// WithCallbacks 将 handlers 追加到 ctx。若 ctx 已带 manager 则追加到其后，否则新建。
func WithCallbacks(ctx context.Context, handlers ...Handler) context.Context {
	if len(handlers) == 0 {
		return ctx
	}
	m, ok := managerFromCtx(ctx)
	if !ok {
		return ctxWithManager(ctx, &manager{handlers: handlers})
	}
	nh := make([]Handler, 0, len(m.handlers)+len(handlers))
	nh = append(nh, m.handlers...)
	nh = append(nh, handlers...)
	return ctxWithManager(ctx, &manager{handlers: nh, runInfo: m.runInfo})
}

// InitCallbacks 完全替换 ctx 中的 RunInfo 与 handlers。
func InitCallbacks(ctx context.Context, info *RunInfo, handlers ...Handler) context.Context {
	if len(handlers) == 0 {
		return ctx
	}
	return ctxWithManager(ctx, &manager{handlers: handlers, runInfo: info})
}

// EnsureRunInfo 补齐 RunInfo（Type、Component）。ctx 中已有 RunInfo 时不覆盖。
func EnsureRunInfo(ctx context.Context, typ string, comp Component) context.Context {
	m, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}
	if m.runInfo != nil {
		return ctx
	}
	return ctxWithManager(ctx, &manager{handlers: m.handlers, runInfo: &RunInfo{Type: typ, Component: comp}})
}

// ReuseHandlers 保留 ctx 中已有的 handlers，替换 RunInfo。
// 用于组件嵌套：内层组件复用外层的 handlers，但使用自己的 RunInfo，
// 内层 OnStart/OnEnd/OnError 产生的 ctx 不影响外层链。
func ReuseHandlers(ctx context.Context, info *RunInfo) context.Context {
	m, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}
	return ctxWithManager(ctx, &manager{handlers: m.handlers, runInfo: info})
}

func runInfoFromCtx(ctx context.Context) *RunInfo {
	info, _ := ctx.Value(runInfoCtxKey{}).(*RunInfo)
	return info
}

func filterHandlers(ctx context.Context, hs []Handler, info *RunInfo, timing CallbackTiming) []Handler {
	ret := make([]Handler, 0, len(hs))
	for _, h := range hs {
		tc, ok := h.(TimingChecker)
		if !ok || tc.Needed(ctx, info, timing) {
			ret = append(ret, h)
		}
	}
	return ret
}

// OnStart 触发所有 handler 的 OnStart，按注册倒序执行。
func OnStart(ctx context.Context, input CallbackInput) context.Context {
	m, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}
	var info *RunInfo
	if m.runInfo != nil {
		info = m.runInfo
		m = &manager{handlers: m.handlers}
		ctx = context.WithValue(ctx, runInfoCtxKey{}, info)
	} else {
		info = runInfoFromCtx(ctx)
	}
	hs := filterHandlers(ctx, m.handlers, info, TimingOnStart)
	for i := len(hs) - 1; i >= 0; i-- {
		ctx = hs[i].OnStart(ctx, info, input)
	}
	return ctxWithManager(ctx, m)
}

// OnEnd 触发所有 handler 的 OnEnd，按注册正序执行。
func OnEnd(ctx context.Context, output CallbackOutput) context.Context {
	m, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}
	info := m.runInfo
	if info == nil {
		info = runInfoFromCtx(ctx)
	}
	for _, h := range filterHandlers(ctx, m.handlers, info, TimingOnEnd) {
		ctx = h.OnEnd(ctx, info, output)
	}
	return ctxWithManager(ctx, m)
}

// OnError 触发所有 handler 的 OnError，按注册正序执行。
func OnError(ctx context.Context, err error) context.Context {
	m, ok := managerFromCtx(ctx)
	if !ok {
		return ctx
	}
	info := m.runInfo
	if info == nil {
		info = runInfoFromCtx(ctx)
	}
	for _, h := range filterHandlers(ctx, m.handlers, info, TimingOnError) {
		ctx = h.OnError(ctx, info, err)
	}
	return ctxWithManager(ctx, m)
}
