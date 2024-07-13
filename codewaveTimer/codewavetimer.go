package main

import (
	"codewave-timer/codewaveTimer/internal/config"
	"codewave-timer/codewaveTimer/internal/handler"
	"codewave-timer/codewaveTimer/internal/middleware"
	"codewave-timer/codewaveTimer/internal/svc"
	"flag"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "codewaveTimer/etc/codewavetimer-api.yaml", "the config file")

func main() {
	flag.Parse()

	c := config.GetGlobalConfig(*configFile)

	// 打印配置以进行调试
	//fmt.Printf("Loaded config: %+v\n", c)

	ctx := svc.NewServiceContext(*c)

	restConf := rest.RestConf{
		Host: c.ServerConfig.HTTP.Host,
		Port: c.ServerConfig.HTTP.Port,
	}
	server := rest.MustNewServer(restConf)
	defer server.Stop()

	// 添加 TraceMiddleware
	server.Use(middleware.TraceMiddleware)

	handler.RegisterHandlers(server, ctx)
	server.Start()

	// 启动 项目本身的 TimerTask
	//ctx.TimerTask.Start()
	//defer ctx.TimerTask.Stop()
}
