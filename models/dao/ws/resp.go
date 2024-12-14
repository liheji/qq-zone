package ws

type Response struct {
	Path      string `json:"path"`
	RequestId int    `json:"requestId"` // 可选，前端传输的请求ID
	Payload   any    `json:"payload"`
}

func Ok(data any) map[string]any {
	return map[string]any{
		"code": 0,
		"msg":  "success",
		"data": data,
	}
}

func Error(errMsg string) map[string]any {
	return map[string]any{
		"code": 1,
		"msg":  errMsg,
		"data": nil,
	}
}
