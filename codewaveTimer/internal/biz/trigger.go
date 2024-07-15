// trigger.go
package biz

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/constant"
	"codewave-timer/codewaveTimer/internal/utils"
	"codewave-timer/codewaveTimer/pkg/cache"
	"codewave-timer/codewaveTimer/pkg/log"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TriggerUseCase struct {
	confData     *config.Data
	timerRepo    JobRepo
	taskRepo     TimerTaskRepo
	taskCache    TaskCache
	taskMemCache *cache.TaskMemCache
	pool         utils.WorkerPool
	slowPool     utils.WorkerPool
	executor     *ExecutorUseCase
}

func NewTriggerUseCase(confData *config.Data, timerRepo JobRepo, taskRepo TimerTaskRepo, taskCache TaskCache, executorUseCase *ExecutorUseCase) *TriggerUseCase {
	return &TriggerUseCase{
		confData:     confData,
		timerRepo:    timerRepo,
		taskRepo:     taskRepo,
		taskCache:    taskCache,
		taskMemCache: cache.NewTaskMemCache(),
		pool:         utils.NewGoWorkerPool(confData.Trigger.WorkersNum),
		slowPool:     utils.NewGoWorkerPool(confData.Trigger.SlowWorkersNum),
		executor:     executorUseCase,
	}
}

func (t *TriggerUseCase) Work(ctx context.Context, minuteBucketKey string, ack func()) error {

	// trigger的每次触发,负责对指定桶号下, 一分钟的zrange task进行处理
	startTime, err := getStartMinute(minuteBucketKey)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(t.confData.Trigger.ZrangeGapSeconds) * time.Second)
	defer ticker.Stop()

	notifier := utils.NewSafeChan(int(time.Minute / (time.Duration(t.confData.Trigger.ZrangeGapSeconds) * time.Second)))
	defer notifier.Close()

	endTime := startTime.Add(time.Minute)

	log.WarnContextf(ctx, "start trigger, key: %s, start: %s, end: %s", minuteBucketKey, startTime, endTime)

	// 预加载任务到内存
	err = t.preloadTasks(ctx, minuteBucketKey, startTime, endTime)
	if err != nil {
		return err
	}

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

func (t *TriggerUseCase) preloadTasks(ctx context.Context, key string, start, end time.Time) error {

	bucket, err := getBucket(key)
	if err != nil {
		return err
	}

	tasks, err := t.getTasksByTime(ctx, key, bucket, start, end)

	timerIDMap := make(map[int64]struct{})
	timerIDs := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		if _, exists := timerIDMap[task.TimerID]; !exists {
			timerIDMap[task.TimerID] = struct{}{}
			timerIDs = append(timerIDs, task.TimerID)
		}
	}

	// 根据任务 ID 从数据库获取任务详情 timerid-详情
	jobs, err := t.timerRepo.FindByIDs(ctx, timerIDs)
	if err != nil {
		return fmt.Errorf("get timer failed", err)
	}

	// 将任务详情加载到内存缓存, 以timerid为key
	for _, job := range jobs {
		t.taskMemCache.Set(cache.Timer{
			TimerID:         job.TimerId,
			Status:          job.Status,
			App:             job.App,
			NotifyHTTPParam: job.NotifyHTTPParam,
			CreateTime:      job.CreateTime,
		})
	}
	log.InfoContextf(ctx, "preload tasks success, key: %s, tasks: %d", key, len(tasks))
	return nil
}

func (t *TriggerUseCase) handleBatch(ctx context.Context, key string, start, end time.Time) error {
	bucket, err := getBucket(key)
	if err != nil {
		return err
	}

	log.WarnContextf(ctx, "handleBatch: %s, %d, %s, %s", key, bucket, start, end)

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
			curTask, ok := t.taskMemCache.Get(task.TimerID)
			useCache := true
			//TODO
			if !ok {
				log.ErrorContextf(ctx, "task not found in memory cache, timerID: %d", task.TimerID)
				useCache = false
			} else {
				log.WarnContextf(ctx, "task found in memory cache, timerID: %d", task.TimerID)
			}

			if err := t.executor.Work(ctx, utils.UnionTimerIDUnix(uint(task.TimerID), task.RunTimer), curTask, useCache); err != nil {
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
