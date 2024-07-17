// scheduler.go
package biz

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/utils"
	"codewave-timer/codewaveTimer/pkg/lock"
	"codewave-timer/codewaveTimer/pkg/log"
	"context"
	"errors"
	"time"
)

// 调度器, 将任务task传递给trigger 协程执行
type SchedulerUseCase struct {
	confData  *config.Data
	timerRepo JobRepo
	taskRepo  TimerTaskRepo
	taskCache TaskCache
	pool      utils.WorkerPool
	trigger   *TriggerUseCase
}

func NewSchedulerUseCase(confData *config.Data, timerRepo JobRepo, taskRepo TimerTaskRepo, taskCache TaskCache, trigger *TriggerUseCase) *SchedulerUseCase {
	return &SchedulerUseCase{
		confData:  confData,
		timerRepo: timerRepo,
		taskRepo:  taskRepo,
		taskCache: taskCache,
		pool:      utils.NewGoWorkerPool(confData.Scheduler.WorkersNum),
		trigger:   trigger,
	}
}

func (w *SchedulerUseCase) Work(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is nil")
	}

	ticker := time.NewTicker(time.Duration(w.confData.Scheduler.TryLockGapMilliSeconds) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ctx.Done():
			log.WarnContextf(ctx, "stopped")
			return nil
		default:
		}

		w.handleSlices(ctx)
	}
	return nil
}

func (w *SchedulerUseCase) handleSlices(ctx context.Context) {
	for i := 0; i < w.getValidBucket(ctx); i++ {
		//fmt.Print("handleSlices: ", i, "\n")
		w.handleSlice(ctx, i)
	}
}

// 动态分桶
func (w *SchedulerUseCase) getValidBucket(ctx context.Context) int {
	return w.confData.Scheduler.BucketsNum
}

func (w *SchedulerUseCase) handleSlice(ctx context.Context, bucketID int) {

	defer func() {
		// 捕获异常，继续执行剩下的任务
		if r := recover(); r != nil {
			log.ErrorContextf(ctx, "handleSlice %v run err. Recovered from panic:%v", bucketID, r)
		}
	}()

	log.InfoContextf(ctx, "scheduler_%v start: %v", bucketID, time.Now())
	now := time.Now()
	//先把上一分钟的任务重复拉一遍, 避免因scheduler尚未触发完全部桶时宕机, 重启后所有未被触发的桶中任务全部丢失
	//配合分布式锁延时, 避免重复触发
	if err := w.pool.Submit(func() {
		w.asyncHandleSlice(ctx, now.Add(-time.Minute), bucketID)
	}); err != nil {
		log.ErrorContextf(ctx, "[handle slice] submit task failed, err: %v", err)
	}

	if err := w.pool.Submit(func() {
		w.asyncHandleSlice(ctx, now, bucketID)
	}); err != nil {
		log.ErrorContextf(ctx, "[handle slice] submit task failed, err: %v", err)
	}

	log.InfoContextf(ctx, "scheduler_%v end: %v", bucketID, time.Now())

}

func (w *SchedulerUseCase) asyncHandleSlice(ctx context.Context, t time.Time, bucketID int) {
	locker := lock.NewRedisLock(utils.GetTimeBucketLockKey(t, bucketID),
		lock.WithExpireSeconds(int64(w.confData.Scheduler.TryLockSeconds)))
	err := locker.Lock(ctx)
	if err != nil {
		log.InfoContextf(ctx, "asyncHandleSlice 获取分布式锁失败: %v", err.Error())
		// 抢锁失败, 直接跳过执行, 下一轮
		return
	}

	log.InfoContextf(ctx, "get scheduler lock success, key: %s", utils.GetTimeBucketLockKey(t, bucketID))
	//如果work没有成功执行, 应允许其他trigger协程抢锁, 默认释放时间70s, work执行成功后, 延长锁的释放时间为130s 确保即便发生重启,重复拉取上一分钟任务时不会反复执行已经触发过的任务
	ack := func() {
		if err := locker.DelayExpire(ctx, int64(w.confData.Scheduler.SuccessExpireSeconds)); err != nil {
			log.ErrorContextf(ctx, "expire lock failed, lock key: %s, err: %v", utils.GetTimeBucketLockKey(t, bucketID), err)
		}
	}

	if err := w.trigger.Work(ctx, utils.GetSliceMsgKey(t, bucketID), ack); err != nil {
		log.ErrorContextf(ctx, "trigger work failed, SliceMsgKey[%v] err: %v", utils.GetSliceMsgKey(t, bucketID), err)
	}
}
