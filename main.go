package main

import (
	"github.com/gin-gonic/gin"
	"qq-zone/routes"
	"qq-zone/setting"
)

func main() {
	// 设置全局配置
	setting.SetGlobalSetting()

	r := gin.Default()
	// 设置路由配置
	setting.SetRouteSetting(r)
	// 注册路由
	routes.InitRoutes(r)
	_ = r.Run("0.0.0.0:9000")
}
