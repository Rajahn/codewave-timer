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
	confData   *config.Data
	timerRepo  JobRepo
	taskRepo   TimerTaskRepo
	redisCache RedisCache
	localCache *cache.CodewaveCache
	pool       utils.WorkerPool
	slowPool   utils.WorkerPool
	executor   *ExecutorUseCase
}

func NewTriggerUseCase(confData *config.Data, timerRepo JobRepo, taskRepo TimerTaskRepo, redisCache RedisCache, executorUseCase *ExecutorUseCase) *TriggerUseCase {
	return &TriggerUseCase{
		confData:   confData,
		timerRepo:  timerRepo,
		taskRepo:   taskRepo,
		redisCache: redisCache,
		localCache: cache.NewCache(confData.Scheduler.BucketsNum),
		pool:       utils.NewGoWorkerPool(confData.Trigger.WorkersNum),
		slowPool:   utils.NewGoWorkerPool(confData.Trigger.SlowWorkersNum),
		executor:   executorUseCase,
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
	go t.buildCache(ctx, minuteBucketKey)

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
	log.InfoLog(ctx, "ack success, key: %s", minuteBucketKey)
	return nil
}

func (t *TriggerUseCase) buildCache(ctx context.Context, minuteBucketKey string) error {
	startTime, _ := getStartMinute(minuteBucketKey)
	nextStartTime := startTime.Add(time.Minute)
	nextEndTime := nextStartTime.Add(time.Minute)

	bucket, _ := getBucket(minuteBucketKey)
	key := utils.GetSliceMsgKey(nextStartTime, bucket)
	tasks, _ := t.getTasksByTime(ctx, key, bucket, nextStartTime, nextEndTime)

	timerIDMap := make(map[int64]struct{})
	timerIDs := make([]int64, 0, len(tasks))

	//下一分钟的task中, 提出timerId并去重, 减少查询数据库次数
	for _, task := range tasks {
		if _, exists := timerIDMap[task.TimerID]; !exists {
			timerIDMap[task.TimerID] = struct{}{}
			timerIDs = append(timerIDs, task.TimerID)
		}
	}

	timers, err := t.timerRepo.FindByIDs(ctx, timerIDs)
	if err != nil {
		return fmt.Errorf("get timer failed, id: %d, err: %w", timerIDs, err)
	}

	for _, timer := range timers {
		t.localCache.Set(strconv.Itoa(int(timer.TimerId)), timer, 1*time.Minute)
	}

	log.WarnLog(ctx, "buildCache success, key: %s", key)
	return nil
}

func (t *TriggerUseCase) DisableCache(timerId int64) {
	t.localCache.Delete(strconv.Itoa(int(timerId)))
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
			if err := t.executor.Work(ctx, utils.UnionTimerIDUnix(uint(task.TimerID), task.RunTimer), t.localCache); err != nil {
				log.ErrorLog(ctx, "executor work failed, err: %v", err)
			}
		}); err != nil {
			return err
		}
	}
	return nil
}

func (t *TriggerUseCase) getTasksByTime(ctx context.Context, key string, bucket int, start, end time.Time) ([]*TaskTimer, error) {
	// 先走缓存
	tasks, err := t.redisCache.GetTasksByTime(ctx, key, start.UnixMilli(), end.UnixMilli())
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
