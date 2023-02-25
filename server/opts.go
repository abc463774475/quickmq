package server

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
	apply(cfg *config)
}

type OptionFun func(cfg *config)

func (f OptionFun) apply(cfg *config) {
	f(cfg)
}

func WithAddr(addr string) Option {
	return OptionFun(func(cfg *config) {
		cfg.Addr = addr
	})
}

func WithName(name string) Option {
	return OptionFun(func(cfg *config) {
		cfg.Name = name
	})
}

func WithClusterAddr(addr string) Option {
	return OptionFun(func(cfg *config) {
		cfg.ClusterAddr = addr
	})
}

func WithMaxConn(maxConn int) Option {
	return OptionFun(func(cfg *config) {
		cfg.MaxConn = maxConn
	})
}

func WithMaxSubs(maxSubs int) Option {
	return OptionFun(func(cfg *config) {
		cfg.MaxSubs = maxSubs
	})
}

func WithConnectRouterAddr(addr string) Option {
	return OptionFun(func(cfg *config) {
		cfg.ConnectRouterAddr = addr
	})
}
