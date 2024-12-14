package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"qq-zone/models/dao/ws"
	"qq-zone/models/handler"
	service2 "qq-zone/models/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocket 处理 WebSocket 连接
func WebSocket(c *gin.Context) {
	clientID := c.Param("clientID")
	if !service2.CheckCache(clientID) {
		c.String(http.StatusBadRequest, "Client ID Invalid")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusBadRequest, "Failed to upgrade connection")
		return
	}
	defer conn.Close()

	wsHandler := ws.NewWebSocketHandler(conn, clientID)
	wsHandler.On("login", handler.LoginHandler)
	wsHandler.On("album", handler.AlbumHandler)
	wsHandler.On("download", handler.DownloadHandler)
	wsHandler.On("cancel", handler.CancelHandler)

	defer service2.SingleWebSocket().RemoveClient(clientID)
	service2.SingleWebSocket().AddClient(wsHandler)
}
