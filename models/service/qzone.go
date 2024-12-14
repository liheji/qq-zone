package service

import (
	"qq-zone/models/dao/qq"
	"sync"
)

var (
	mutexQzone   sync.Mutex
	qzoneService *QzoneService
)

// SingleQzone 获取 QzoneService 实例
func SingleQzone() *QzoneService {
	if qzoneService == nil {
		mutexQzone.Lock()
		if qzoneService == nil {
			qzoneService = &QzoneService{
				clientMap: make(map[string]*qq.Qzone),
			}
		}
		mutexQzone.Unlock()
	}
	return qzoneService
}

// QzoneService 服务封装
type QzoneService struct {
	mapMutex  sync.Mutex
	clientMap map[string]*qq.Qzone
}

// AddClient 添加新客户端连接
func (service *QzoneService) AddClient(handler *qq.Qzone) {
	// 将客户端存入 map
	service.mapMutex.Lock()
	service.clientMap[handler.ClientID] = handler
	service.mapMutex.Unlock()
}

// GetClient 获取客户端连接
func (service *QzoneService) GetClient(clientID string) *qq.Qzone {
	// 将客户端存入 map
	service.mapMutex.Lock()
	defer service.mapMutex.Unlock()
	if handler, exists := service.clientMap[clientID]; exists {
		return handler
	}
	return nil
}

// RemoveClient 移除客户端连接
func (service *QzoneService) RemoveClient(clientID string) {
	service.mapMutex.Lock()
	defer service.mapMutex.Unlock()
	if _, exists := service.clientMap[clientID]; exists {
		delete(service.clientMap, clientID)
	}
}

// ContainsClient 获取客户端连接
func (service *QzoneService) ContainsClient(clientID string) bool {
	if handler, exists := service.clientMap[clientID]; exists {
		return handler != nil
	}
	return false
}
