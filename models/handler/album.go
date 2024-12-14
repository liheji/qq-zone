package handler

import (
	"fmt"
	"qq-zone/models/dao/qq"
	ws2 "qq-zone/models/dao/ws"
	"qq-zone/models/service"
	"qq-zone/utils"
)

type AlbumRequest struct {
	QQ       string `json:"qq"`
	FriendQQ string `json:"friendQQ"`
	Album    string `json:"album"`
}

func AlbumHandler(w *ws2.WebSocketHandler, request *ws2.Request) (any, error) {
	req, err := utils.StringToObj[AlbumRequest](request.Payload)
	if err != nil {
		return nil, err
	}

	// 前置处理
	q := service.SingleQzone().GetClient(w.ClientID)
	if q == nil || q.Drop {
		return nil, fmt.Errorf("未登录")
	}
	if q.LoginStatus < 2 {
		return nil, fmt.Errorf("正在登录")
	}
	if q.QQ != req.QQ {
		return nil, fmt.Errorf("当前账号未登录")
	}

	q.Album = req.Album
	q.FriendQQ = req.FriendQQ
	// 获取好友列表
	friendAlbumMap, err := qq.NewAlbumDown(w, q).GetAlbumList()
	if err != nil {
		return nil, err
	}

	return friendAlbumMap, nil
}
