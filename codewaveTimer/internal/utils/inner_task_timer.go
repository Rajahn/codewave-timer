package utils

import (
	"github.com/robfig/cron/v3"
	"log"
)

type TimerTask struct {
	cron *cron.Cron
}

func NewTimerTask() *TimerTask {
	return &TimerTask{
		cron: cron.New(cron.WithSeconds()), // 支持秒级调度
	}
}

// Start 启动codewave-timer本身的定时任务调度器
func (t *TimerTask) Start() {
	t.cron.Start()
}

// Stop 停止定时任务调度器
func (t *TimerTask) Stop() {
	t.cron.Stop()
}

// AddStartupJob 添加在服务启动时运行一次的任务
func (t *TimerTask) AddStartupJob(job func()) {
	go job() // 服务启动时立即异步运行一次
}

// AddRecurringJob 添加定时任务
func (t *TimerTask) AddRecurringJob(schedule string, job func()) error {
	_, err := t.cron.AddFunc(schedule, func() {
		go job() // 定时任务异步运行
	})
	if err != nil {
		log.Fatalf("Error adding cron job: %v", err)
		return err
	}
	return nil
}
