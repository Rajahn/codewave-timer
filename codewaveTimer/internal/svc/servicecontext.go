package svc

import (
	"codewave-timer/codewaveTimer/internal/biz"
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/data"
	"codewave-timer/codewaveTimer/internal/service"
	"codewave-timer/codewaveTimer/internal/utils"
	"codewave-timer/codewaveTimer/pkg/cache"
	tracing "codewave-timer/codewaveTimer/pkg/trace"
	"context"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/trace"

	"gorm.io/gorm"
	"log"
)

type ServiceContext struct {
	Config    config.GlobalConfig
	DB        *gorm.DB
	Cache     *cache.Client
	JobRepo   biz.JobRepo
	TaskRepo  biz.TimerTaskRepo
	TxManager biz.Transaction

	RedisCache  *data.RedisCache
	CronJobUC   *biz.CronJobUseCase
	SchedulerUC *biz.SchedulerUseCase
	MigratorUC  *biz.MigratorUseCase
	TimerTask   *utils.TimerTask

	Service *service.CodewaveTimerService

	TracerProvider *trace.TracerProvider
}

func NewServiceContext(c config.GlobalConfig) *ServiceContext {
	// 初始化数据库连接
	db := data.GetDataBase(&c.Data.Database)
	// 初始化缓存
	cacheClient := data.GetCache(&c.Data.Redis)

	// 初始化数据层
	dataLayer := &data.Data{
		Db:    db,
		Cache: cacheClient,
	}

	// 初始化事务管理
	txManager := data.NewTransaction(dataLayer)

	// 初始化仓库
	jobRepo := data.NewJobRepo(dataLayer)
	taskRepo := data.NewTaskRepo(dataLayer)
	redisCache := data.NewRedisCache(&c.Data, dataLayer)

	// 初始化业务逻辑层
	migratorUC := biz.NewMigratorUseCase(&c.Data, jobRepo, taskRepo, redisCache)
	cronJobUC := biz.NewCronJobUseCase(&c.Data, jobRepo, taskRepo, redisCache, dataLayer, migratorUC)
	executorUC := biz.NewExecutorUseCase(&c.Data, jobRepo, taskRepo, utils.NewHttpClient())
	triggerUC := biz.NewTriggerUseCase(&c.Data, jobRepo, taskRepo, redisCache, executorUC)
	schedulerUC := biz.NewSchedulerUseCase(&c.Data, jobRepo, taskRepo, redisCache, triggerUC)

	// 初始化服务层
	service := service.NewCodewaveTimerService(cronJobUC, schedulerUC, migratorUC)

	// 初始化链路追踪
	tracerProvider, err := tracing.InitTracerProvider("http://localhost:14268/api/traces")
	if err != nil {
		log.Fatalf("Failed to initialize tracer provider: %v", err)
	}

	// 初始化系统自身的定时任务
	timerTask := utils.NewTimerTask()
	timerTask.Start()

	// 为定时任务生成 traceID 并创建带有 traceID 的 context
	traceID := uuid.New().String()
	ctx := context.WithValue(context.Background(), "Trace-ID", traceID)

	timerTask.AddStartupJob(func() {
		err := schedulerUC.Work(ctx)
		if err != nil {
			log.Printf("Scheduler work failed: %v", err)
		}
		log.Println("Startup job is running")
	})

	err = timerTask.AddRecurringJob("@every 30m", func() {
		migratorUC.BatchMigratorTimer(ctx)
		log.Println("Recurring job is running every 30 minutes")
	})

	if err != nil {
		log.Fatalf("Failed to add recurring job: %v", err)
	}

	return &ServiceContext{
		Config:         c,
		DB:             db,
		Cache:          cacheClient,
		JobRepo:        jobRepo,
		TaskRepo:       taskRepo,
		RedisCache:     redisCache,
		CronJobUC:      cronJobUC,
		SchedulerUC:    schedulerUC,
		MigratorUC:     migratorUC,
		TimerTask:      timerTask,
		TxManager:      txManager,
		Service:        service,
		TracerProvider: tracerProvider,
	}
}
