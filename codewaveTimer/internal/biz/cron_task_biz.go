// cron_task_biz.go
package biz

import "context"

// 表定义放在biz层, 解耦biz与data层
type TaskTimer struct {
	TaskID   int64  `gorm:"column:task_id"`
	App      string `gorm:"column:app;NOT NULL"`           // 定义ID
	TimerID  int64  `gorm:"column:timer_id;NOT NULL"`      // 定义ID
	Output   string `gorm:"column:output;default:null"`    // 执行结果
	RunTimer int64  `gorm:"column:run_timer;default:null"` // 执行时间
	CostTime int    `gorm:"column:cost_time"`              // 执行耗时
	DiffTime int    `gorm:"column:diff_time"`              // 实际的执行时间和预期执行时间的差值
	Status   int    `gorm:"column:status;NOT NULL"`        // 当前状态
}

func (t *TaskTimer) TableName() string {
	return "cron_task"
}

// JobRepo is a Greater timerRepo.
type TimerTaskRepo interface {
	BatchSave(context.Context, []*TaskTimer) error
	Update(context.Context, *TaskTimer) (*TaskTimer, error)
	GetTasksByTimeRange(context.Context, int64, int64, int) ([]*TaskTimer, error)
	GetTasksByTimerIdAndRunTimer(context.Context, int64, int64) (*TaskTimer, error)
}

type TaskCache interface {
	BatchCreateTasks(ctx context.Context, tasks []*TaskTimer) error
	GetTasksByTime(ctx context.Context, table string, start, end int64) ([]*TaskTimer, error)
}
