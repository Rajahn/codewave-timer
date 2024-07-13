package config

import (
	"github.com/spf13/viper"
	"sync"
)

var (
	globalConfig GlobalConfig
	once         sync.Once
)

type GlobalConfig struct {
	ServerConfig ServerConfig `yaml:"server" mapstructure:"server"`
	Data         Data         `yaml:"data" mapstructure:"data"`
}

type ServerConfig struct {
	HTTP HTTPConfig `yaml:"http" mapstructure:"http"`
}

type HTTPConfig struct {
	Host    string `yaml:"host" mapstructure:"host"`
	Port    int    `yaml:"port" mapstructure:"port"`
	Timeout string `yaml:"timeout" mapstructure:"timeout"`
}

type Data struct {
	Database  DatabaseConfig  `yaml:"database" mapstructure:"database"`
	Redis     RedisConfig     `yaml:"redis" mapstructure:"redis"`
	Scheduler SchedulerConfig `yaml:"scheduler" mapstructure:"scheduler"`
	Trigger   TriggerConfig   `yaml:"trigger" mapstructure:"trigger"`
	Migrator  MigratorConfig  `yaml:"migrator" mapstructure:"migrator"`
}

type DatabaseConfig struct {
	Addr                     string `yaml:"addr" mapstructure:"addr"`
	User                     string `yaml:"user" mapstructure:"user"`
	Password                 string `yaml:"password" mapstructure:"password"`
	Database                 string `yaml:"database" mapstructure:"database"`
	MaxIdleConn              int    `yaml:"max_idle_conn" mapstructure:"max_idle_conn"`
	MaxOpenConn              int    `yaml:"max_open_conn" mapstructure:"max_open_conn"`
	MaxIdleTime              int    `yaml:"max_idle_time" mapstructure:"max_idle_time"`
	SlowThresholdMillisecond int    `yaml:"slow_threshold_millisecond" mapstructure:"slow_threshold_millisecond"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr" mapstructure:"addr"`
	Password string `yaml:"password" mapstructure:"password"`
	Db       int    `yaml:"db" mapstructure:"db"`
	PoolSize int    `yaml:"pool_size" mapstructure:"pool_size"`
}

type SchedulerConfig struct {
	WorkersNum             int `yaml:"workersNum" mapstructure:"workersNum"`
	BucketsNum             int `yaml:"bucketsNum" mapstructure:"bucketsNum"`
	TryLockSeconds         int `yaml:"tryLockSeconds" mapstructure:"tryLockSeconds"`
	TryLockGapMilliSeconds int `yaml:"tryLockGapMilliSeconds" mapstructure:"tryLockGapMilliSeconds"`
	SuccessExpireSeconds   int `yaml:"successExpireSeconds" mapstructure:"successExpireSeconds"`
}

type TriggerConfig struct {
	ZrangeGapSeconds int `yaml:"zrangeGapSeconds" mapstructure:"zrangeGapSeconds"`
	WorkersNum       int `yaml:"workersNum" mapstructure:"workersNum"`
	SlowWorkersNum   int `yaml:"slowWorkersNum" mapstructure:"slowWorkersNum"`
}

type MigratorConfig struct {
	WorkersNum         int `yaml:"workersNum" mapstructure:"workersNum"`
	MigrateStepMinutes int `yaml:"migrateStepMinutes" mapstructure:"migrateStepMinutes"`
}

func GetGlobalConfig(configFile string) *GlobalConfig {
	once.Do(func() {
		readConf(configFile)
	})
	return &globalConfig
}

func readConf(configFile string) {
	viper.SetConfigFile(configFile)

	err := viper.ReadInConfig()
	if err != nil {
		panic("read config file err: " + err.Error())
	}

	err = viper.Unmarshal(&globalConfig)
	if err != nil {
		panic("unmarshal config file err: " + err.Error())
	}
}
