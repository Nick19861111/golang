package app

import (
	"common/config"
	"common/logs"
	"context"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 启动程序
func Run(ctx context.Context) error {
	//1.日志库
	logs.InitLog(config.Conf.AppName)
	//2.etcd

	//启动grpc
	server := grpc.NewServer()

	go func() {
		listen, err := net.Listen("tcp", config.Conf.Grpc.Addr)
		if err != nil {
			logs.Fatal("user grpc listen err:%v", err)
		}

		//注册grpc server到ertc

		err = server.Serve(listen)
		if err != nil {
			logs.Fatal("user grpc listen server err:%v", err)
		}
	}()

	stop := func() {
		server.Stop()
		time.Sleep(3 * time.Second)
		logs.Info("user grpc server stop")
	}

	//期望有一个优雅启停 遇到中断 退出 终止 挂断
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	for {
		select {
		case <-ctx.Done():
			stop()
			//time out
			return nil
		case s := <-c:
			switch s {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				stop()
				logs.Info("user app quit")
				return nil
			case syscall.SIGHUP:
				stop()
				logs.Info("hang up!! user app quit")
				return nil
			default:
				return nil
			}
		}
	}
}
