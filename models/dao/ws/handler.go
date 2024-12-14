package ws

import (
	"github.com/gorilla/websocket"
	"sync"
)

// WebSocketHandler 基础处理器
type WebSocketHandler struct {
	Conn      *websocket.Conn
	ClientID  string
	Callbacks map[string]func(w *WebSocketHandler, request *Request) (any, error)

	writeMutex sync.Mutex
}

// NewWebSocketHandler 创建新的 WebSocketHandler
func NewWebSocketHandler(conn *websocket.Conn, clientID string) *WebSocketHandler {
	return &WebSocketHandler{
		Conn:      conn,
		ClientID:  clientID,
		Callbacks: make(map[string]func(w *WebSocketHandler, request *Request) (any, error)),
	}
}

// On 注册路径和对应的处理函数
func (w *WebSocketHandler) On(path string, handler func(w *WebSocketHandler, request *Request) (any, error)) {
	w.Callbacks[path] = handler
}

// AfterConnected 连接成功后的操作
func (w *WebSocketHandler) AfterConnected() {
	// 自定义连接成功逻辑
}

// AfterClosed 连接关闭后的操作
func (w *WebSocketHandler) AfterClosed() {
	// 自定义连接关闭逻辑
}

// OnError 处理错误
func (w *WebSocketHandler) OnError(err error) {
	// 自定义错误处理逻辑
}

// TextMessage 文本消息处理
func (w *WebSocketHandler) TextMessage(request *Request) {
	if callback, exists := w.Callbacks[request.Path]; exists {
		res, err := callback(w, request)
		if err == nil {
			w.Response(request, Ok(res))
		} else {
			w.Response(request, Error(err.Error()))
		}
	} else {
		w.Response(request, Error("未定义的路径"))
	}
}

func (w *WebSocketHandler) Send(path string, payload any) {
	response := Response{
		Path:    path,
		Payload: payload,
	}
	w.writeMutex.Lock()
	_ = w.Conn.WriteJSON(response)
	w.writeMutex.Unlock()
}

func (w *WebSocketHandler) Response(request *Request, payload any) {

	response := Response{
		Path:      request.Path,
		RequestId: request.RequestId,
		Payload:   payload,
	}
	w.writeMutex.Lock()
	_ = w.Conn.WriteJSON(response)
	w.writeMutex.Unlock()
}

// BinaryMessage 二进制消息处理
func (w *WebSocketHandler) BinaryMessage(payload []byte) {
	// 自定义二进制消息逻辑
}
