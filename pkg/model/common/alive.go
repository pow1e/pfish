package common

import (
	"github.com/pow1e/pfish/api"
	"github.com/sirupsen/logrus"
	"sync"
)

var GlobalAliveUsers = NewAliveUsers()

type AliveUsers struct {
	Clients map[string]*ClientStream
	Mutex   sync.RWMutex
}

type ClientStream struct {
	Stream api.Fish_HeartbeatServer
	Active bool
}

// NewAliveUsers 创建一个新的 AliveUsers 实例
func NewAliveUsers() *AliveUsers {
	return &AliveUsers{
		Clients: make(map[string]*ClientStream),
	}
}

// AddClient 添加客户端到活跃列表
func (a *AliveUsers) AddClient(clientID string, stream api.Fish_HeartbeatServer) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	a.Clients[clientID] = &ClientStream{
		Stream: stream,
		Active: true,
	}
	logrus.Printf("客户端添加成功: client_id=%s\n", clientID)
}

// RemoveClient 从活跃列表中移除客户端
func (a *AliveUsers) RemoveClient(clientID string) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	delete(a.Clients, clientID)
	logrus.Printf("客户端移除成功: client_id=%s\n", clientID)
}

// ProbeClients 发送探活请求并移除无响应的客户端
func (a *AliveUsers) ProbeClients() {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	for clientID, client := range a.Clients {
		err := client.Stream.Send(&api.HeartbeatReply{Code: 200}) // 200 表示探活请求
		if err != nil {
			logrus.Infof("探活失败: client_id=%s, 错误=%v", clientID, err)
			client.Active = false
			delete(a.Clients, clientID)
		} else {
			client.Active = true
			logrus.Infof("探活成功: client_id=%s", clientID)
		}
	}
}

// GetClients 获取所有活跃客户端（线程安全）
func (a *AliveUsers) GetClients() map[string]*ClientStream {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()
	clients := make(map[string]*ClientStream)
	for k, v := range a.Clients {
		clients[k] = v
	}
	return clients
}
