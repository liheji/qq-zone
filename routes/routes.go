package routes

import (
	"github.com/gin-gonic/gin"
	"qq-zone/controllers"
)

func InitRoutes(r *gin.Engine) {
	r.GET("/", controllers.Index)
	r.GET("/ws/:clientID", controllers.WebSocket)
	r.GET("/token", controllers.GetToken)
	r.GET("/qrimage", controllers.GetQrImage)
	r.GET("/avatar/:burl", controllers.GetAvatar)
	r.GET("/result", controllers.GetResult)
	// 默认路由
	//r.NoRoute(controllers.Index)
}
