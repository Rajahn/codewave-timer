// cron_job_biz.go
package biz

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/constant"
	_ "codewave-timer/codewaveTimer/internal/constant"
	"codewave-timer/codewaveTimer/internal/utils"
	"codewave-timer/codewaveTimer/pkg/lock"
	"codewave-timer/codewaveTimer/pkg/log"
	"fmt"

	"context"
	_ "fmt"
	"time"
)

// 解耦biz与data层, job是一个cron表达式, task是具体时间点上要执行的任务
type JobTimer struct {
	TimerId         int64 `gorm:"column:id"`
	App             string
	Name            string
	Status          int
	Cron            string
	NotifyHTTPParam string     `gorm:"column:notify_http_param;NOT NULL" json:"notify_http_param,omitempty"` // Http 回调参数
	CreateTime      *time.Time `gorm:"column:create_time;default:null"`
	ModifyTime      *time.Time `gorm:"column:modify_time;default:null"`
}

// TableName 表名
func (p *JobTimer) TableName() string {
	return "cron_job"
}

type JobRepo interface {
	Save(context.Context, *JobTimer) (*JobTimer, error)
	Update(context.Context, *JobTimer) (*JobTimer, error)
	FindByID(context.Context, int64) (*JobTimer, error)
	FindByIDs(context.Context, []int64) ([]*JobTimer, error)
	FindByStatus(context.Context, int) ([]*JobTimer, error)
	Delete(context.Context, int64) error
}

// xtimerUseCase is a User usecase.
type CronJobUseCase struct {
	confData  *config.Data
	timerRepo JobRepo
	taskRepo  TimerTaskRepo
	taskCache TaskCache
	tm        Transaction
	muc       *MigratorUseCase
}

// NewUserUseCase new a User usecase.
func NewCronJobUseCase(confData *config.Data, timerRepo JobRepo, taskRepo TimerTaskRepo, taskCache TaskCache, tm Transaction, muc *MigratorUseCase) *CronJobUseCase {
	return &CronJobUseCase{confData: confData, timerRepo: timerRepo, taskRepo: taskRepo, taskCache: taskCache, tm: tm, muc: muc}
}

func (uc *CronJobUseCase) CreateTimer(ctx context.Context, g *JobTimer) (*JobTimer, error) {
	return uc.timerRepo.Save(ctx, g)
}

// 包装一批待执行的task
func (t *JobTimer) BatchTasksFromTimer(executeTimes []time.Time) []*TaskTimer {
	tasks := make([]*TaskTimer, 0, len(executeTimes))
	for _, executeTime := range executeTimes {
		tasks = append(tasks, &TaskTimer{
			App:      t.App,
			TimerID:  t.TimerId,
			Status:   constant.NotRunned.ToInt(),
			RunTimer: executeTime.UnixMilli(),
		})
	}
	return tasks
}

const defaultEnableGapSeconds = int64(3)

func (uc *CronJobUseCase) EnableTimer(ctx context.Context, app string, timerId int64) error {
	// 限制激活和去激活频次
	locker := lock.NewRedisLock(utils.GetEnableLockKey(app),
		lock.WithExpireSeconds(defaultEnableGapSeconds),
		lock.WithWatchDogMode())

	defer func(locker *lock.RedisLock, ctx context.Context) {
		err := locker.Unlock(ctx)
		if err != nil {
			log.ErrorContextf(ctx, "EnableTimer 自动解锁失败", err.Error())
		}
	}(locker, ctx)

	err := locker.Lock(ctx)
	if err != nil {
		log.InfoContextf(ctx, "激活/去激活操作过于频繁，请稍后再试！", err.Error())
		// 抢锁失败, 直接跳过执行, 下一轮
		return nil
	}

	// 开启事务
	err = uc.tm.InTx(ctx, func(ctx context.Context) error {
		// 1. 数据库获取Timer
		timer, err := uc.timerRepo.FindByID(ctx, timerId)
		if err != nil {
			log.ErrorContextf(ctx, "激活失败，timer不存在：timerId, err: %v", err)
			return err
		}

		// 2. 校验状态
		if timer.Status != constant.Unabled.ToInt() {
			return fmt.Errorf("Timer非Unable状态，激活失败，timerId:: %d", timerId)
		}

		// 修改timer为激活状态
		timer.Status = constant.Enabled.ToInt()
		_, err = uc.timerRepo.Update(ctx, timer)
		if err != nil {
			log.ErrorContextf(ctx, "激活失败，timer不存在：timerId, err: %v", err)
			return err
		}

		// 迁移数据
		if err := uc.muc.MigratorTimer(ctx, timer); err != nil {
			log.ErrorContextf(ctx, "迁移timer失败: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (uc *CronJobUseCase) DisableTimer(ctx context.Context, app string, timerId int64) interface{} {

	timer, err := uc.timerRepo.FindByID(ctx, timerId)

	timer.Status = constant.Unabled.ToInt()
	_, err = uc.timerRepo.Update(ctx, timer)
	if err != nil {
		log.ErrorContextf(ctx, "关闭失败，timer不存在：timerId, err: %v", err)
		return err
	}
	return nil
}
