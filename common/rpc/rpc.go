package rpc

import (
	"common/config"
	"common/discovery"
	"common/logs"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"user/pb"
)

var (
	UserClient pb.UserServiceClient
)

func Init() {
	//etcd 解析器
	r := discovery.NewResolver(config.Conf.Etcd)
	resolver.Register(r)
	//end
	userDomain := config.Conf.Domain["user"]
	initClient(userDomain.Name, userDomain.LoadBalance, &UserClient)
}

func initClient(name string, LoadBalance bool, client interface{}) {
	//服务器的地址
	addr := fmt.Sprintf("etcd:///%s", name)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials())}
	if LoadBalance {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}
	conn, err := grpc.DialContext(context.TODO(), addr, opts...)
	if err != nil {
		logs.Fatal("rpc connect etcd err:%v", err)
	}
	switch c := client.(type) {
	case *pb.UserServiceClient:
		*c = pb.NewUserServiceClient(conn)
	default:
		logs.Fatal("rpc client type error")
	}
}
