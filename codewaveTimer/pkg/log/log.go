package log

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	ERROR LogLevel = iota
	WARN
	INFO
)

type Logger struct {
	mu      sync.Mutex
	level   LogLevel
	loggers map[LogLevel]*log.Logger
}

var (
	defaultLogger *Logger
	once          sync.Once
)

func init() {
	once.Do(func() {
		defaultLogger = &Logger{
			level: WARN,
			loggers: map[LogLevel]*log.Logger{
				ERROR: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
				WARN:  log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile),
				INFO:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
			},
		}
	})
}

func SetLevel(level LogLevel) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

func logf(ctx context.Context, level LogLevel, format string, args ...interface{}) {
	if level <= defaultLogger.level {
		defaultLogger.mu.Lock()
		defer defaultLogger.mu.Unlock()
		if logger, ok := defaultLogger.loggers[level]; ok {
			logger.Output(3, fmt.Sprintf(format, args...))
		}
	}
}

func ErrorContextf(ctx context.Context, format string, args ...interface{}) {
	logf(ctx, ERROR, format, args...)
}

func WarnContextf(ctx context.Context, format string, args ...interface{}) {
	logf(ctx, WARN, format, args...)
}

func InfoContextf(ctx context.Context, format string, args ...interface{}) {
	logf(ctx, INFO, format, args...)
}
