package setting

import (
	"github.com/gin-gonic/gin"
	"os"
)

func SetGlobalSetting() {
	// 设置Gin模式
	gin.SetMode(gin.DebugMode)
	// 设置日志输出
	gin.DefaultWriter = os.Stdout
}

func SetRouteSetting(r *gin.Engine) {
	// proxy
	_ = r.SetTrustedProxies([]string{"127.0.0.1"})
	// template
	r.LoadHTMLGlob("templates/*.html")
	// 静态文件
	r.Static("/assets", "templates/assets")
	// 允许跨域
	r.Use(CrosMiddleware())
}

func CrosMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域
		context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
		context.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma,token,openid,opentoken")
		context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar")
		context.Header("Access-Control-Allow-Credentials", "false")
		context.Next()
	}
}
