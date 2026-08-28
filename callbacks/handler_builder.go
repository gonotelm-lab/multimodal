package callbacks

import "context"

// HandlerBuilder 只注册关心的时机，构建出的 handler 自动实现 TimingChecker。
type HandlerBuilder struct {
	onStartFn func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	onEndFn   func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	onErrorFn func(ctx context.Context, info *RunInfo, err error) context.Context
}

func NewHandlerBuilder() *HandlerBuilder {
	return &HandlerBuilder{}
}

func (b *HandlerBuilder) OnStartFn(fn func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context) *HandlerBuilder {
	b.onStartFn = fn
	return b
}

func (b *HandlerBuilder) OnEndFn(fn func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context) *HandlerBuilder {
	b.onEndFn = fn
	return b
}

func (b *HandlerBuilder) OnErrorFn(fn func(ctx context.Context, info *RunInfo, err error) context.Context) *HandlerBuilder {
	b.onErrorFn = fn
	return b
}

func (b *HandlerBuilder) Build() Handler {
	return &handlerImpl{
		onStartFn: b.onStartFn,
		onEndFn:   b.onEndFn,
		onErrorFn: b.onErrorFn,
	}
}

type handlerImpl struct {
	onStartFn func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	onEndFn   func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	onErrorFn func(ctx context.Context, info *RunInfo, err error) context.Context
}

func (h *handlerImpl) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
	if h.onStartFn == nil {
		return ctx
	}
	return h.onStartFn(ctx, info, input)
}

func (h *handlerImpl) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
	if h.onEndFn == nil {
		return ctx
	}
	return h.onEndFn(ctx, info, output)
}

func (h *handlerImpl) OnError(ctx context.Context, info *RunInfo, err error) context.Context {
	if h.onErrorFn == nil {
		return ctx
	}
	return h.onErrorFn(ctx, info, err)
}

func (h *handlerImpl) Needed(_ context.Context, _ *RunInfo, timing CallbackTiming) bool {
	switch timing {
	case TimingOnStart:
		return h.onStartFn != nil
	case TimingOnEnd:
		return h.onEndFn != nil
	case TimingOnError:
		return h.onErrorFn != nil
	default:
		return false
	}
}
