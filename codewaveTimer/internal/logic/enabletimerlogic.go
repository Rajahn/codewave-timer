package logic

import (
	tracing "codewave-timer/codewaveTimer/pkg/trace"
	"context"

	"codewave-timer/codewaveTimer/internal/svc"
	"codewave-timer/codewaveTimer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableTimerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 启动定时任务
func NewEnableTimerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableTimerLogic {
	return &EnableTimerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnableTimerLogic) EnableTimer(req *types.EnableTimerRequest) (resp *types.Response, err error) {
	// 从上下文中获取 traceID，如果存在的话
	traceID, ok := tracing.TraceIDFromContext(l.ctx)
	if !ok {
		traceID = "new-trace-id" // 或生成一个新的 traceID
	}

	// 创建新的上下文和 Span
	ctx, span := tracing.NewSpan(tracing.WithTraceID(l.ctx, traceID), "EnableTimerLogic", "EnableTimer")
	defer span.End()

	resp, err = l.svcCtx.Service.EnableTimer(ctx, req)
	if err != nil {
		return nil, err
	}

	// 在响应中添加 traceID
	resp.TraceID = traceID

	return resp, nil
}
