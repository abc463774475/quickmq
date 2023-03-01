package mock

import (
	"sync"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/server/base"
)

// 为什么要用写，，就是想给一个项目控制的机会。方便定制化操作！
type Server struct {
	ID      int64
	clients map[int64]*AcceptClient
	l       sync.RWMutex
}

// NewServer 创建服务
func NewServer() *Server {
	s := &Server{}
	s.Init()
	return s
}

// Init 初始化
func (s *Server) Init() {
	s.clients = make(map[int64]*AcceptClient)
}

func (s *Server) NewAcceptClienter(id int64) base.AcceptClienHandler {
	c := NewAcceptClient(id, s)

	s.l.Lock()
	if _, ok := s.clients[id]; ok {
		s.l.Unlock()
		nlog.Erro("client id is exist", id)
		return nil
	}
	s.clients[id] = c
	s.l.Unlock()

	return c
}

func (s *Server) DelClient(id int64) error {
	s.l.Lock()

	if _, ok := s.clients[id]; !ok {
		s.l.Unlock()
		nlog.Erro("client id is not exist", id)
		return nil
	}

	delete(s.clients, id)
	s.l.Unlock()
	return nil
}

func (s *Server) GetClient(id int64) base.AcceptClienHandler {
	s.l.RLock()
	defer s.l.RUnlock()

	if c, ok := s.clients[id]; ok {
		return c
	}
	return nil
}

func (s *Server) GetID() int64 {
	return s.ID
}
