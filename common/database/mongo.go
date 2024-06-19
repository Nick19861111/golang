package database

import (
	"common/config"
	"common/logs"
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type MongoManager struct {
	Cli *mongo.Client
	Db  *mongo.Database
}

func NewMongo() *MongoManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//链接数据库
	clientOptions := options.Client().ApplyURI(config.Conf.Database.MongoConf.Url)
	clientOptions.SetAuth(options.Credential{
		Username: config.Conf.Database.MongoConf.UserName,
		Password: config.Conf.Database.MongoConf.Password,
	})
	//内存池的大小
	clientOptions.SetMinPoolSize(uint64(config.Conf.Database.MongoConf.MinPoolSize))
	clientOptions.SetMaxPoolSize(uint64(config.Conf.Database.MongoConf.MaxPoolSize))
	//链接
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logs.Fatal("mongodb connect err: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		logs.Fatal("mongodb ping err: %v", err)
	}
	m := &MongoManager{
		Cli: client,
	}
	m.Db = m.Cli.Database(config.Conf.Database.MongoConf.Db)
	return m
}

// 关闭方法
func (m *MongoManager) Close() {
	err := m.Cli.Disconnect(context.TODO())
	if err != nil {
		logs.Fatal("mongodb close err: %v", err)
	}
}
