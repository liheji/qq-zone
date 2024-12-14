package service

import (
	"encoding/json"
	ws2 "qq-zone/models/dao/ws"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	mutexWebsocket   sync.Mutex
	websocketService *WebSocketService
)

// SingleWebSocket 获取 WebSocketService 实例
func SingleWebSocket() *WebSocketService {
	if websocketService == nil {
		mutexWebsocket.Lock()
		if websocketService == nil {
			websocketService = &WebSocketService{
				clientMap: make(map[string]*ws2.WebSocketHandler),
			}
		}
		mutexWebsocket.Unlock()
	}
	return websocketService
}

// WebSocketService 服务封装
type WebSocketService struct {
	mapMutex  sync.Mutex
	clientMap map[string]*ws2.WebSocketHandler
}

// AddClient 添加新客户端连接
func (service *WebSocketService) AddClient(handler *ws2.WebSocketHandler) {
	// 将客户端存入 map
	service.mapMutex.Lock()
	service.clientMap[handler.ClientID] = handler
	service.mapMutex.Unlock()

	handler.AfterConnected()

	service.handleMessages(handler)
}

// GetClient 获取客户端连接
func (service *WebSocketService) GetClient(clientID string) *ws2.WebSocketHandler {
	// 将客户端存入 map
	service.mapMutex.Lock()
	defer service.mapMutex.Unlock()
	if handler, exists := service.clientMap[clientID]; exists {
		return handler
	}
	return nil
}

// RemoveClient 移除客户端连接
func (service *WebSocketService) RemoveClient(clientID string) {
	service.mapMutex.Lock()
	if handler, exists := service.clientMap[clientID]; exists {
		handler.AfterClosed()
		delete(service.clientMap, clientID)
	}
	defer service.mapMutex.Unlock()
}

// handleMessages 处理客户端消息
func (service *WebSocketService) handleMessages(handler *ws2.WebSocketHandler) {
	defer service.RemoveClient(handler.ClientID)

	for {
		messageType, payload, err := handler.Conn.ReadMessage()
		if err != nil {
			handler.OnError(err)
			return
		}

		switch messageType {
		case websocket.TextMessage:
			var request = new(ws2.Request)
			if err := json.Unmarshal(payload, request); err != nil {
				handler.OnError(err)
				continue
			}
			handler.TextMessage(request)
		case websocket.BinaryMessage:
			handler.BinaryMessage(payload)
		case websocket.PingMessage:
			if err := handler.Conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				handler.OnError(err)
				return
			}
		case websocket.PongMessage:
			if err := handler.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				handler.OnError(err)
				return
			}
		case websocket.CloseMessage:
			return
		}
	}
}
