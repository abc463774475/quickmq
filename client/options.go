package client

import "github.com/abc463774475/quickmq/server/base"

type Option interface {
	apply(client *Client)
}

type OptionFun func(client *Client)

func (f OptionFun) apply(client *Client) {
	f(client)
}

// WithAddr 设置连接的地址
func WithAddr(addr string) Option {
	return OptionFun(func(client *Client) {
		client.addr = addr
	})
}

// WithClientType 设置客户端类型
func WithClientType(clientType int) Option {
	return OptionFun(func(client *Client) {
		client.clientType = clientType
	})
}

// WithConnectClientHandler 设置连接成功后的回调
func WithConnectClientHandler(handler base.ConnectClientHandler) Option {
	return OptionFun(func(client *Client) {
		client.clienter = handler
	})
}
