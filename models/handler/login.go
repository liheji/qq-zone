package handler

import (
	"fmt"
	"os"
	"qq-zone/models/dao/qq"
	"qq-zone/models/dao/vo"
	ws2 "qq-zone/models/dao/ws"
	"qq-zone/models/service"
	"qq-zone/utils"
	"qq-zone/utils/filer"
	"time"
)

type LoginRequest struct {
	QQ       string `json:"qq"`
	Album    string `json:"album"`
	FriendQQ string `json:"friendQQ"`
}

func LoginHandler(w *ws2.WebSocketHandler, request *ws2.Request) (any, error) {
	req, err := utils.StringToObj[LoginRequest](request.Payload)
	if err != nil {
		return nil, err
	}

	// 前置处理
	oldQ := service.SingleQzone().GetClient(w.ClientID)
	if oldQ != nil && !oldQ.Drop {
		if oldQ.QQ == req.QQ {
			if oldQ.LoginStatus < 2 {
				return nil, fmt.Errorf("正在登录")
			}
			// 获取好友列表
			friendList, err := qq.NewAlbumDown(w, oldQ).GetFriends()
			if err != nil {
				return nil, err
			}
			resList := append([]*vo.Friend{oldQ.GetSelf()}, vo.GjsonToObj[*vo.Friend](friendList)...)
			w.Send("loginSuccess", map[string]any{"msg": "登录成功", "friendList": resList})
			return "", nil
		}
		oldQ.Drop = true
	}

	// 创建新客户端
	q := qq.NewQzone(w.ClientID, w)
	q.QQ = req.QQ
	service.SingleQzone().AddClient(q)

	q.WG.Add(1)
	go func(qzone *qq.Qzone, innerW *ws2.WebSocketHandler) {
		if err := qzone.Login(); err != nil {
			innerW.Send("loginError", err.Error())
			return
		}
		// 获取好友列表
		friendList, err := qq.NewAlbumDown(innerW, qzone).GetFriends()
		if err != nil {
			innerW.Send("loginError", err.Error())
			return
		}
		resList := append([]*vo.Friend{qzone.GetSelf()}, vo.GjsonToObj[*vo.Friend](friendList)...)
		innerW.Send("loginSuccess", map[string]any{"msg": "登录成功", "friendList": resList})
		defer func() {
			if filer.IsFile(qq.QRCODE_SAVE_PATH) {
				_ = os.Remove(qq.QRCODE_SAVE_PATH)
			}
			// 不删除
			// service.SingleQzone().RemoveClient(qzone.ClientID)
		}()
	}(q, w)
	q.WG.Wait()

	return fmt.Sprintf("/qrimage?t=%v", time.Now().Unix()), nil
}
