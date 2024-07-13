package data

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/pkg/cache"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"sync"
	"time"
)

var (
	db        *gorm.DB
	rdb       *cache.Client
	onceDB    sync.Once
	onceCache sync.Once
)

func GetDataBase(conf *config.DatabaseConfig) *gorm.DB {
	onceDB.Do(func() {
		var err error
		db, err = NewDatabase(conf)
		if err != nil {
			//log.Fatalf("Failed to initialize database: %v", err)
		}
	})
	return db
}

func GetCache(conf *config.RedisConfig) *cache.Client {
	onceCache.Do(func() {
		var err error
		rdb, err = NewCache(conf)
		if err != nil {
			log.Fatalf("Failed to initialize cache: %v", err)
		}
	})
	return rdb
}

func NewDatabase(conf *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := conf.User + ":" + conf.Password + "@tcp(" + conf.Addr + ")/" + conf.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(conf.MaxIdleConn)
	sqlDB.SetMaxOpenConns(conf.MaxOpenConn)
	sqlDB.SetConnMaxIdleTime(time.Duration(conf.MaxIdleTime) * time.Second)

	return db, nil
}

func NewCache(conf *config.RedisConfig) (*cache.Client, error) {
	cache.Init(
		cache.WithAddr(conf.Addr),
		cache.WithPassWord(conf.Password),
		cache.WithDB(conf.Db),
		cache.WithPoolSize(conf.PoolSize))

	return cache.GetRedisCli(), nil
}
