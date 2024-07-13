package handler

import (
	"net/http"

	"codewave-timer/codewaveTimer/internal/logic"
	"codewave-timer/codewaveTimer/internal/svc"
	"codewave-timer/codewaveTimer/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 停止定时任务
func disableTimerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DisableTimerRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewDisableTimerLogic(r.Context(), svcCtx)
		resp, err := l.DisableTimer(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
