package database

import (
	"common/config"
	"common/logs"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisManager struct {
	Cli        *redis.Client        //单机
	ClusterCli *redis.ClusterClient //集群
}

// 创建一个redis客户端
func NewRedis() *RedisManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var clusterCli *redis.ClusterClient
	var cli *redis.Client
	addrs := config.Conf.Database.RedisConf.ClusterAddrs

	if len(addrs) == 0 {
		//单节点
		cli = redis.NewClient(&redis.Options{
			Addr:         config.Conf.Database.RedisConf.Addr,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	} else {
		clusterCli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Conf.Database.RedisConf.ClusterAddrs,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	}

	//ping的操作
	if clusterCli == nil {
		if err := clusterCli.Ping(ctx).Err(); err != nil {
			logs.Fatal("redis cluster connect err%v", err)
			return nil
		}
	}

	if cli == nil {
		if err := cli.Ping(ctx).Err(); err != nil {
			logs.Fatal("redis connect err%v", err)
			return nil
		}
	}
	return &RedisManager{
		Cli:        cli,
		ClusterCli: clusterCli,
	}

}

// 关闭相关操作
func (r *RedisManager) Close() {
	//即成的关闭
	if r.ClusterCli != nil {
		if err := r.ClusterCli.Close(); err != nil {
			logs.Error("redis ClusterCli close err%v", err)
		}
	}

	//单机的
	if r.Cli != nil {
		if err := r.Cli.Close(); err != nil {
			logs.Error("redis cli close err%v", err)
		}
	}
}
