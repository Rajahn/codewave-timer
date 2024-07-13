// executor.go
package biz

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/constant"
	"codewave-timer/codewaveTimer/internal/types"
	"codewave-timer/codewaveTimer/internal/utils"
	"codewave-timer/codewaveTimer/pkg/log"
	"context"
	"encoding/json"
	"fmt"

	"time"
)

// xtimerUseCase is a User usecase.
type ExecutorUseCase struct {
	confData   *config.Data
	httpClient *utils.JSONClient
	timerRepo  JobRepo
	taskRepo   TimerTaskRepo
}

// NewUserUseCase new a User usecase.
func NewExecutorUseCase(confData *config.Data, timerRepo JobRepo, taskRepo TimerTaskRepo, taskCache TaskCache, httpClient *utils.JSONClient) *ExecutorUseCase {
	return &ExecutorUseCase{
		confData:   confData,
		timerRepo:  timerRepo,
		taskRepo:   taskRepo,
		httpClient: httpClient,
	}
}

func (w *ExecutorUseCase) Work(ctx context.Context, timerIDUnixKey string) error {
	// 拿到消息，查询一次完整的 timer 定义
	timerID, unix, err := utils.SplitTimerIDUnix(timerIDUnixKey)
	if err != nil {
		return err
	}
	return w.executeAndPostProcess(ctx, timerID, unix)
}

func (w *ExecutorUseCase) executeAndPostProcess(ctx context.Context, timerID int64, unix int64) error {
	// 查询 timer 完整的定义，执行回调
	timer, err := w.timerRepo.FindByID(ctx, timerID)
	if err != nil {
		return fmt.Errorf("get timer failed, id: %d, err: %w", timerID, err)
	}

	// 定时器已经处于去激活态，则无需处理任务
	if timer.Status != constant.Enabled.ToInt() {
		log.WarnContextf(ctx, "timer has alread been unabled, timerID: %d", timerID)
		return nil
	}

	execTime := time.Now()
	resp, err := w.execute(ctx, timer)
	return w.postProcess(ctx, resp, err, timer.App, uint(timerID), unix, execTime)
}

func (w *ExecutorUseCase) execute(ctx context.Context, timer *JobTimer) (map[string]interface{}, error) {
	var (
		resp map[string]interface{}
		err  error
	)

	notifyHTTPParam := types.NotifyHTTPParam{}
	err = json.Unmarshal([]byte(timer.NotifyHTTPParam), &notifyHTTPParam)
	if err != nil {
		//log.Errorf("json unmarshal for NotifyHTTPParam err %s", err.Error())
		return nil, err
	}

	// 简单POST
	//fmt.Println("Go! notifyHTTPParam.Url: ", notifyHTTPParam.Url, " notifyHTTPParam.Headers: ", notifyHTTPParam.Headers, " notifyHTTPParam.Body: ", notifyHTTPParam.Body)
	err = w.httpClient.Post(ctx, notifyHTTPParam.Url, notifyHTTPParam.Headers, notifyHTTPParam.Body, &resp)
	return resp, err
}

func (w *ExecutorUseCase) postProcess(ctx context.Context, resp map[string]interface{}, execErr error, app string, timerID uint, unix int64, execTime time.Time) error {

	task, err := w.taskRepo.GetTasksByTimerIdAndRunTimer(ctx, int64(timerID), unix)
	if err != nil {
		return fmt.Errorf("get task failed, timerID: %d, runTimer: %d, err: %w", timerID, time.UnixMilli(unix), err)
	}

	// output
	if execErr != nil {
		task.Output = execErr.Error()
	} else {
		respBody, _ := json.Marshal(resp)
		task.Output = string(respBody)
	}

	// Status
	if execErr != nil {
		task.Status = constant.Failed.ToInt()
	} else {
		task.Status = constant.Successed.ToInt()
	}

	// gapTime
	//task.CostTime = int(execTime.UnixMilli() - task.RunTimer)
	task.DiffTime = int(execTime.UnixMilli() - task.RunTimer)
	//定时任务执行的时间cost time
	task.CostTime = int(time.Now().UnixMilli() - execTime.UnixMilli())

	_, err = w.taskRepo.Update(ctx, task)
	if err != nil {
		return fmt.Errorf("task postProcess failed, timerID: %d, runTimer: %d, err: %w", timerID, time.UnixMilli(unix), err)
	}

	return nil
}
