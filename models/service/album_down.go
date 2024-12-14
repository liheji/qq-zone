package service

import (
	"qq-zone/models/dao/qq"
	"sync"
)

var (
	mutexAlbumDown   sync.Mutex
	albumDownService *AlbumDownService
)

// SingleAlbumDown 获取 AlbumDownService 实例
func SingleAlbumDown() *AlbumDownService {
	if albumDownService == nil {
		mutexAlbumDown.Lock()
		if albumDownService == nil {
			albumDownService = &AlbumDownService{
				clientMap: make(map[string]*qq.AlbumDown),
			}
		}
		mutexAlbumDown.Unlock()
	}
	return albumDownService
}

// AlbumDownService 服务封装
type AlbumDownService struct {
	mapMutex  sync.Mutex
	clientMap map[string]*qq.AlbumDown
}

// AddClient 添加新客户端连接
func (service *AlbumDownService) AddClient(handler *qq.AlbumDown) {
	// 将客户端存入 map
	service.mapMutex.Lock()
	service.clientMap[handler.ClientID] = handler
	service.mapMutex.Unlock()
}

// GetClient 获取客户端连接
func (service *AlbumDownService) GetClient(clientID string) *qq.AlbumDown {
	// 将客户端存入 map
	service.mapMutex.Lock()
	defer service.mapMutex.Unlock()
	if handler, exists := service.clientMap[clientID]; exists {
		return handler
	}
	return nil
}

// RemoveClient 移除客户端连接
func (service *AlbumDownService) RemoveClient(clientID string) {
	service.mapMutex.Lock()
	defer service.mapMutex.Unlock()
	if _, exists := service.clientMap[clientID]; exists {
		delete(service.clientMap, clientID)
	}
}

// ContainsClient 获取客户端连接
func (service *AlbumDownService) ContainsClient(clientID string) bool {
	if handler, exists := service.clientMap[clientID]; exists {
		return handler != nil
	}
	return false
}
