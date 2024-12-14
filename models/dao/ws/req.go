package ws

// Request 通用请求结构体
type Request struct {
	Path      string `json:"path"`
	RequestId int    `json:"requestId"` // 可选，前端传输的请求ID
	Payload   any    `json:"payload"`
	// 内部使用，是否已经响应
}
