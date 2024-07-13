// cron_job.go
package data

import (
	"codewave-timer/codewaveTimer/internal/biz"
	"context"
)

type jobRepo struct {
	data *Data
}

func NewJobRepo(data *Data) biz.JobRepo {
	return &jobRepo{
		data: data,
	}
}

func (r *jobRepo) Save(ctx context.Context, g *biz.JobTimer) (*biz.JobTimer, error) {
	err := r.data.DB(ctx).Create(g).Error
	return g, err
}

func (r *jobRepo) Update(ctx context.Context, g *biz.JobTimer) (*biz.JobTimer, error) {
	err := r.data.Db.WithContext(ctx).Where("id = ?", g.TimerId).Updates(g).Error
	return g, err
}

func (r *jobRepo) Delete(ctx context.Context, id int64) error {
	return r.data.DB(ctx).Where("id = ?", id).Delete(&biz.JobTimer{}).Error
}

func (r *jobRepo) FindByID(ctx context.Context, timerId int64) (*biz.JobTimer, error) {
	var timer biz.JobTimer
	err := r.data.Db.WithContext(ctx).Where("id = ?", timerId).First(&timer).Error
	if err != nil {
		return nil, err
	}
	return &timer, nil
}

func (r *jobRepo) FindByStatus(ctx context.Context, status int) ([]*biz.JobTimer, error) {
	var timers []*biz.JobTimer
	err := r.data.Db.WithContext(ctx).Where("status = ?", status).Find(&timers).Error
	if err != nil {
		return nil, err
	}
	return timers, nil
}
