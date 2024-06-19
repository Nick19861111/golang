package router

import (
	"common/config"
	"gate/api"
	"github.com/gin-gonic/gin"
)

// 注册路由 RegisterRouter
func RegisterRouter() *gin.Engine {
	//发布和测试版
	if config.Conf.Log.Level == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	//初始化grpc的客户端
	
	r := gin.Default()
	userhandler := api.NewUserHandler()

	r.POST("/register", userhandler.Register)
	return r
}
