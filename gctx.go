package micro

import (
	"context"
	"time"
)

type GrpcCtx struct {
	context.Context
	f context.CancelFunc
}

func (ctx *GrpcCtx) Close() {
	ctx.f()
}

func WithGCtxTimeout(timeout time.Duration) *GrpcCtx {
	gc := &GrpcCtx{}

	gc.Context, gc.f = context.WithTimeout(context.Background(), timeout)
	return gc
}

func WithGCtxCancel() *GrpcCtx {
	gc := &GrpcCtx{}

	gc.Context, gc.f = context.WithCancel(context.Background())
	return gc
}
