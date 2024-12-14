package handler

import (
	"fmt"
	"qq-zone/models/dao/qq"
	ws2 "qq-zone/models/dao/ws"
	"qq-zone/models/service"
	"qq-zone/utils"
)

type DownloadRequest struct {
	QQ       string `json:"qq"`
	Album    string `json:"album"`
	FriendQQ string `json:"friendQQ"`
}

func DownloadHandler(w *ws2.WebSocketHandler, request *ws2.Request) (any, error) {
	req, err := utils.StringToObj[DownloadRequest](request.Payload)
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

	if service.SingleAlbumDown().ContainsClient(w.ClientID) {
		return nil, fmt.Errorf("正在下载")
	}

	q.Album = req.Album
	q.FriendQQ = req.FriendQQ
	down := qq.NewAlbumDown(w, q)
	service.SingleAlbumDown().AddClient(down)
	go func(album *qq.AlbumDown, innerW *ws2.WebSocketHandler) {
		err := album.SpiderAlbumAll()
		if err != nil {
			innerW.Send("download", err.Error())
		}

		// 执行完成后删除
		defer service.SingleAlbumDown().RemoveClient(innerW.ClientID)
	}(down, w)

	return "开始下载", nil
}
