package service

import (
	"codewave-timer/codewaveTimer/internal/biz"
	"codewave-timer/codewaveTimer/internal/constant"
	"codewave-timer/codewaveTimer/internal/types"
	"context"
	"encoding/json"
	"errors"
	"strconv"
)

type CodewaveTimerService struct {
	timerUC     *biz.CronJobUseCase
	schedulerUC *biz.SchedulerUseCase
	migratorUC  *biz.MigratorUseCase
}

func NewCodewaveTimerService(timerUC *biz.CronJobUseCase, schedulerUC *biz.SchedulerUseCase, migratorUC *biz.MigratorUseCase) *CodewaveTimerService {
	return &CodewaveTimerService{timerUC: timerUC, schedulerUC: schedulerUC, migratorUC: migratorUC}
}

func (s *CodewaveTimerService) CreateTimer(ctx context.Context, req *types.CreateTimerRequest) (*types.Response, error) {
	param, err := json.Marshal(req.NotifyHTTPParam)
	if err != nil {
		return nil, err
	}
	timer, err := s.timerUC.CreateTimer(ctx, &biz.JobTimer{
		App:             req.App,
		Name:            req.Name,
		Status:          constant.Unabled.ToInt(),
		Cron:            req.Cron,
		NotifyHTTPParam: string(param),
	})
	if err != nil {
		return nil, errors.Join(errors.New("create timer failed"), err)
	}
	resp := types.Response{
		Code:    0,
		Message: "ok",
		Data:    strconv.FormatInt(timer.TimerId, 10),
	}

	return &resp, nil
}

func (s *CodewaveTimerService) EnableTimer(ctx context.Context, req *types.EnableTimerRequest) (*types.Response, error) {

	err := s.timerUC.EnableTimer(ctx, req.App, req.TimerId)
	if err != nil {
		return nil, errors.Join(errors.New("enable timer failed"), err)
	}

	resp := types.Response{
		Code:    0,
		Message: "ok",
		Data:    strconv.FormatInt(req.TimerId, 10),
	}
	return &resp, nil
}

func (s *CodewaveTimerService) DisableTimer(ctx context.Context, req *types.DisableTimerRequest) (*types.Response, error) {
	err := s.timerUC.DisableTimer(ctx, req.App, req.TimerId)
	if err != nil {
		return nil, errors.New("disable timer failed")
	}
	resp := types.Response{
		Code:    0,
		Message: "ok",
		Data:    strconv.FormatInt(req.TimerId, 10),
	}
	return &resp, nil
}
