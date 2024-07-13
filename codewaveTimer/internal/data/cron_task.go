// cron_task.go
package data

import (
	"codewave-timer/codewaveTimer/internal/biz"
	"context"

	"gorm.io/gorm/clause"
)

type taskRepo struct {
	data *Data
}

func NewTaskRepo(data *Data) biz.TimerTaskRepo {
	return &taskRepo{
		data: data,
	}
}

func (r *taskRepo) BatchSave(ctx context.Context, g []*biz.TaskTimer) error {
	err := r.data.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "timer_id"}, {Name: "run_timer"}},
		DoUpdates: clause.AssignmentColumns([]string{}),
	}).Create(g).Error
	return err
}

func (r *taskRepo) Update(ctx context.Context, g *biz.TaskTimer) (*biz.TaskTimer, error) {
	err := r.data.Db.WithContext(ctx).Where("task_id = ?", g.TaskID).Updates(g).Error
	return g, err
}

func (r *taskRepo) GetTasksByTimeRange(ctx context.Context, startTime int64, endTime int64, status int) ([]*biz.TaskTimer, error) {
	var tasks []*biz.TaskTimer
	err := r.data.Db.WithContext(ctx).
		Where("run_timer >= ?", startTime).
		Where("run_timer <= ?", endTime).
		Where("status = ?", status).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepo) GetTasksByTimerIdAndRunTimer(ctx context.Context, timerId int64, runTimer int64) (*biz.TaskTimer, error) {
	var task *biz.TaskTimer
	err := r.data.Db.WithContext(ctx).
		Where("timer_id = ?", timerId).
		Where("run_timer = ?", runTimer).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
}
