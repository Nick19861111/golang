package main

import (
	"common/config"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"user/app"
)

var configFile = flag.String("config", "application.yml", "config file")

func main() {
	//1.加载配置
	flag.Parse()
	config.InitConfig(*configFile)
	fmt.Println(config.Conf)
	//2.启动监控
	//启动携程
	go func() {
		err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
		fmt.Println("携程已经启动")
	}()
	//3.启动程序 grpc的服务
	err := app.Run(context.Background())
	if err != nil {
		log.Panicln("app run error:", err)
		os.Exit(-1)
	}
}
