package logic

import (
	tracing "codewave-timer/codewaveTimer/pkg/trace"
	"context"

	"codewave-timer/codewaveTimer/internal/svc"
	"codewave-timer/codewaveTimer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableTimerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 停止定时任务
func NewDisableTimerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableTimerLogic {
	return &DisableTimerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableTimerLogic) DisableTimer(req *types.DisableTimerRequest) (resp *types.Response, err error) {
	// 从上下文中获取 traceID，如果存在的话
	traceID, ok := tracing.TraceIDFromContext(l.ctx)
	if !ok {
		traceID = "new-trace-id" // 或生成一个新的 traceID
	}

	// 创建新的上下文和 Span
	ctx, span := tracing.NewSpan(tracing.WithTraceID(l.ctx, traceID), "DisableTimerLogic", "DisableTimer")
	defer span.End()

	resp, err = l.svcCtx.Service.DisableTimer(ctx, req)
	if err != nil {
		return nil, err
	}

	// 在响应中添加 traceID
	resp.TraceID = traceID

	return resp, nil
}
