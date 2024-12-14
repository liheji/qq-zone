package handler

import (
	"fmt"
	ws2 "qq-zone/models/dao/ws"
	"qq-zone/models/service"
	"qq-zone/utils"
)

type CancelRequest struct {
	CancelType string `json:"cancelType"`
}

func CancelHandler(w *ws2.WebSocketHandler, request *ws2.Request) (any, error) {
	req, err := utils.StringToObj[CancelRequest](request.Payload)
	if err != nil {
		return nil, err
	}

	switch req.CancelType {
	case "login":
		q := service.SingleQzone().GetClient(w.ClientID)
		if q == nil {
			return nil, fmt.Errorf("未找到客户端")
		}
		if q.Drop {
			return nil, fmt.Errorf("重复取消")
		}
		q.Drop = true
	default:
		a := service.SingleAlbumDown().GetClient(w.ClientID)
		if a == nil {
			return nil, fmt.Errorf("未找到客户端")
		}
		if a.Drop {
			return nil, fmt.Errorf("重复取消")
		}
		a.Drop = true
	}

	return "", nil
}
