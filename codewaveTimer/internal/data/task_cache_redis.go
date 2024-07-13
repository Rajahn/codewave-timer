// task_cache_redis.go
package data

import (
	"codewave-timer/codewaveTimer/internal/biz"
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/constant"
	"codewave-timer/codewaveTimer/internal/utils"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

type RedisCache struct {
	confData *config.Data
	data     *Data
}

func NewRedisCache(confData *config.Data, data *Data) *RedisCache {
	return &RedisCache{confData: confData, data: data}
}

func (t *RedisCache) BatchCreateTasks(ctx context.Context, tasks []*biz.TaskTimer) error {
	if len(tasks) == 0 {
		return nil
	}

	err := t.data.Cache.Pipeline(ctx, func(pipe redis.Pipeliner) error {

		for _, task := range tasks {
			unix := task.RunTimer
			tableName := t.GetTableName(task)
			var members []redis.Z
			// 以执行时机为score, timerID和unix为member, 直接用timerID, 一个timerId下对应多个task,需要区分开,否则只能插入一次
			member := redis.Z{
				Score:  float64(unix),
				Member: utils.UnionTimerIDUnix(uint(task.TimerID), unix),
			}
			members = append(members, member)
			pipe.ZAdd(ctx, tableName, members...)

			// 设置过期时间为一天, 起到清理作用
			aliveDuration := time.Until(time.UnixMilli(task.RunTimer).Add(24 * time.Hour))
			pipe.Expire(ctx, tableName, aliveDuration)
		}
		return nil
	})
	return err
}

func (t *RedisCache) GetTasksByTime(ctx context.Context, table string, start, end int64) ([]*biz.TaskTimer, error) {
	timerIDUnixs, err := t.data.Cache.ZRangeByScore(ctx, table, strconv.FormatInt(start, 10), strconv.FormatInt(end-1, 10))
	if err != nil {
		return nil, err
	}

	tasks := make([]*biz.TaskTimer, 0, len(timerIDUnixs))
	for _, timerIDUnix := range timerIDUnixs {
		timerID, unix, _ := utils.SplitTimerIDUnix(timerIDUnix)
		tasks = append(tasks, &biz.TaskTimer{
			TimerID:  int64(timerID),
			RunTimer: unix,
		})
	}

	return tasks, nil
}

func (t *RedisCache) GetTableName(task *biz.TaskTimer) string {
	maxBucket := t.confData.Scheduler.BucketsNum
	return fmt.Sprintf("%s_%d", time.UnixMilli(task.RunTimer).Format(constant.MinuteFormat), task.TimerID%int64(maxBucket))
}
