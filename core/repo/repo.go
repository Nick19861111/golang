package repo

import "common/database"

// 统一管理相关操作
type Manager struct {
	Mongo *database.MongoManager
	Redis *database.RedisManager
}

func New() *Manager {
	return &Manager{
		Mongo: database.NewMongo(),
		Redis: database.NewRedis(),
	}
}
