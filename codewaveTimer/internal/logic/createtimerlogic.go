package logic

import (
	"codewave-timer/codewaveTimer/internal/svc"
	"codewave-timer/codewaveTimer/internal/types"
	tracing "codewave-timer/codewaveTimer/pkg/trace"
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTimerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建一个新的定时任务
func NewCreateTimerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTimerLogic {
	return &CreateTimerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTimerLogic) CreateTimer(req *types.CreateTimerRequest) (resp *types.Response, err error) {
	// 创建新的上下文和 Span
	ctx, span := tracing.NewSpan(l.ctx, "CreateTimerLogic", "CreateTimer")
	defer span.End()

	resp, err = l.svcCtx.Service.CreateTimer(ctx, req)
	if err != nil {
		return nil, err
	}

	// 在响应中添加 traceID
	traceID := span.SpanContext().TraceID().String()
	resp.TraceID = traceID

	return resp, nil
}
