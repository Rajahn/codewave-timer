// trigger.go
package biz

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/constant"
	"codewave-timer/codewaveTimer/internal/utils"
	"codewave-timer/codewaveTimer/pkg/log"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TriggerUseCase struct {
	confData  *config.Data
	timerRepo JobRepo
	taskRepo  TimerTaskRepo
	taskCache TaskCache
	pool      utils.WorkerPool
	slowPool  utils.WorkerPool
	executor  *ExecutorUseCase
}

func NewTriggerUseCase(confData *config.Data, timerRepo JobRepo, taskRepo TimerTaskRepo, taskCache TaskCache, executorUseCase *ExecutorUseCase) *TriggerUseCase {
	return &TriggerUseCase{
		confData:  confData,
		timerRepo: timerRepo,
		taskRepo:  taskRepo,
		taskCache: taskCache,
		pool:      utils.NewGoWorkerPool(confData.Trigger.WorkersNum),
		slowPool:  utils.NewGoWorkerPool(confData.Trigger.SlowWorkersNum),
		executor:  executorUseCase,
	}
}

func (t *TriggerUseCase) Work(ctx context.Context, minuteBucketKey string, ack func()) error {

	// 进行为时一分钟的 zrange 处理
	startTime, err := getStartMinute(minuteBucketKey)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(t.confData.Trigger.ZrangeGapSeconds) * time.Second)
	defer ticker.Stop()

	notifier := utils.NewSafeChan(int(time.Minute / (time.Duration(t.confData.Trigger.ZrangeGapSeconds) * time.Second)))
	defer notifier.Close()

	endTime := startTime.Add(time.Minute)

	//TODO 此时建立下一分钟的缓存, JobID-Job详情

	var wg sync.WaitGroup
	for range ticker.C {
		select {
		case e := <-notifier.GetChan():
			err, _ = e.(error)
			return err
		default:
		}

		wg.Add(1)
		go func(startTime time.Time) {
			defer wg.Done()
			if err := t.handleBatch(ctx, minuteBucketKey, startTime, startTime.Add(time.Duration(t.confData.Trigger.ZrangeGapSeconds)*time.Second)); err != nil {
				notifier.Put(err)
			}
		}(startTime)
		//每个协程启动后, 更新一次startTime, 启动下一个协程
		if startTime = startTime.Add(time.Duration(t.confData.Trigger.ZrangeGapSeconds) * time.Second); startTime.Equal(endTime) || startTime.After(endTime) {
			break
		}
	}

	wg.Wait()
	select {
	case e := <-notifier.GetChan():
		err, _ = e.(error)
		return err
	default:
	}
	ack()
	log.InfoContextf(ctx, "ack success, key: %s", minuteBucketKey)
	return nil
}

func (t *TriggerUseCase) handleBatch(ctx context.Context, key string, start, end time.Time) error {
	bucket, err := getBucket(key)
	if err != nil {
		return err
	}

	tasks, err := t.getTasksByTime(ctx, key, bucket, start, end)
	if err != nil || len(tasks) == 0 {
		return err
	} else {
		//for _, task := range tasks {
		//	fmt.Print("task: ", task.TimerID, " ", task.RunTimer, " ")
		//}
		//fmt.Print("\n")
	}

	for _, task := range tasks {
		task := task
		if err := t.pool.Submit(func() {
			if err := t.executor.Work(ctx, utils.UnionTimerIDUnix(uint(task.TimerID), task.RunTimer)); err != nil {
				log.ErrorContextf(ctx, "executor work failed, err: %v", err)
			}
		}); err != nil {
			return err
		}
	}
	return nil
}

func (t *TriggerUseCase) getTasksByTime(ctx context.Context, key string, bucket int, start, end time.Time) ([]*TaskTimer, error) {
	// 先走缓存
	tasks, err := t.taskCache.GetTasksByTime(ctx, key, start.UnixMilli(), end.UnixMilli())
	if err == nil {
		return tasks, nil
	}

	// 倘若缓存查询报错，数据库兜底
	tasks, err = t.taskRepo.GetTasksByTimeRange(ctx, start.UnixMilli(), end.UnixMilli(), constant.NotRunned.ToInt())
	if err != nil {
		return nil, err
	}

	maxBucket := t.confData.Scheduler.BucketsNum
	var validTask []*TaskTimer
	for _, task := range tasks {
		if uint(task.TimerID)%uint(maxBucket) != uint(bucket) {
			continue
		}
		validTask = append(validTask, task)
	}

	return validTask, nil
}

func getStartMinute(slice string) (time.Time, error) {
	timeBucket := strings.Split(slice, "_")
	if len(timeBucket) != 2 {
		return time.Time{}, fmt.Errorf("invalid format of msg key: %s", slice)
	}

	return utils.GetStartMinute(timeBucket[0])
}

func getBucket(slice string) (int, error) {
	timeBucket := strings.Split(slice, "_")
	if len(timeBucket) != 2 {
		return -1, fmt.Errorf("invalid format of msg key: %s", slice)
	}
	return strconv.Atoi(timeBucket[1])
}
