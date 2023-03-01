package mock

import (
	"github.com/abc463774475/quickmq/qcmq/client"
	"github.com/abc463774475/quickmq/qcmq/msg"
	"github.com/abc463774475/quickmq/qcmq/server/base"
)

// 这里只会有一个客户端，所以暂时不做mgr管理
type ConnectClient struct {
	Addr string
	ID   int64
	Name string

	sender base.ConnectClientSendHandler
}

func (c *ConnectClient) GetID() int64 {
	return c.ID
}

func (c *ConnectClient) Register() {
	reqInfo := &ClientRegister{
		Name: "test",
		ID:   c.ID,
	}

	c.sender.SendMsg(msg.MSG_HANDSHAKE, reqInfo)
}

func (c *ConnectClient) Publish(sub string, i interface{}) {
	c.sender.Publish(sub, i)
}

func (c *ConnectClient) SetSender(sender base.ConnectClientSendHandler) {}

func (c *ConnectClient) SendMsg(msgid msg.MSGID, i interface{}) {
	c.sender.SendMsg(msgid, i)
}

// NewConnectClient 创建客户端
func NewConnectClient(addr string, name string,
	id int64,
) *ConnectClient {
	c := &ConnectClient{
		Addr: addr,
		ID:   id,
		Name: name,
	}

	c.Init()

	cl := client.NewClient(addr, c)

	c.sender = cl
	return c
}

// Init 初始化
func (c *ConnectClient) Init() {
}

// Publish 发布消息
func (c *ConnectClient) PublishMsg(sub string, i interface{}) {
	c.sender.Publish(sub, i)
}

// Subscribe 订阅消息
func (c *ConnectClient) Subscribe(sub string, subf base.SUBFUN,
	subackf base.SUBACKFUN,
) {
	c.sender.Subscribe(sub, subf, subackf)
}

// Req 请求消息
func (c *ConnectClient) Req(sub string, i interface{},
	pubCB base.PUBCALLBACK,
) {
	c.sender.Req(sub, i, pubCB)
}
