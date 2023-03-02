package server

import "github.com/abc463774475/quickmq/server/base"

type config struct {
	Addr string `json:"addr"`
	Name string `json:"name"`
	// ClusterAddr 监听的集群节点
	ClusterAddr string `json:"clusterAddr"`
	// ConectRouterAddr 连接的路由节点Ip
	ConnectRouterAddr string `json:"connectRouterAddr"`
	MaxConn           int    `json:"maxConn"`
	MaxSubs           int    `json:"maxSubs"`
}

type Option interface {
	apply(s *server)
}

type OptionFun func(s *server)

func (f OptionFun) apply(s *server) {
	f(s)
}

func WithAddr(addr string) Option {
	return OptionFun(func(s *server) {
		s.cfg.Addr = addr
	})
}

func WithName(name string) Option {
	return OptionFun(func(s *server) {
		s.cfg.Name = name
	})
}

func WithClusterAddr(addr string) Option {
	return OptionFun(func(s *server) {
		s.cfg.ClusterAddr = addr
	})
}

func WithMaxConn(maxConn int) Option {
	return OptionFun(func(s *server) {
		s.cfg.MaxConn = maxConn
	})
}

func WithMaxSubs(maxSubs int) Option {
	return OptionFun(func(s *server) {
		s.cfg.MaxSubs = maxSubs
	})
}

func WithConnectRouterAddr(addr string) Option {
	return OptionFun(func(s *server) {
		s.cfg.ConnectRouterAddr = addr
	})
}

// WithService
func WithService(service base.Service) Option {
	return OptionFun(func(s *server) {
		s.service = service
	})
}
