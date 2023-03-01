package mock

import (
	"encoding/json"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/msg"
	"github.com/abc463774475/quickmq/qcmq/server/base"
)

type AcceptClient struct {
	ID   int64
	Name string

	sender base.AcceptClientSendHandler

	srv *Server
}

func (c *AcceptClient) SendMsg(msgid msg.MSGID, i interface{}) {
	c.sender.SendMsg(msgid, i)
}

// NewAcceptClient 创建客户端
func NewAcceptClient(id int64, server *Server) *AcceptClient {
	c := &AcceptClient{
		ID:  id,
		srv: server,
	}

	c.Init()
	return c
}

// Init 初始化
func (c *AcceptClient) Init() {
}

//func (c *AcceptClient) SetSender(sender base.AcceptClientSendHandler) {
//	c.sender = sender
//}

func (c *AcceptClient) HandleRegister(_msg *msg.Msg) {
	info := &ClientRegister{}
	err := json.Unmarshal(_msg.Data, info)
	if err != nil {
		nlog.Erro("json.Unmarshal err", err)
		return
	}

	c.Name = info.Name
}

func (c *AcceptClient) HandlePublish(_msg *msg.Msg) {
	info := &MsgPub{}
	err := json.Unmarshal(_msg.Data, info)
	if err != nil {
		nlog.Erro("json.Unmarshal err", err)
		return
	}

	// 这里也可以是srv name 规则，跳过 底层的sub name
	nlog.Info("HandlePublish  %v", info)
}

func (c *AcceptClient) HandleSubscribe(_msg *msg.Msg) {
	info := &MsgSub{}
	err := json.Unmarshal(_msg.Data, info)
	if err != nil {
		nlog.Erro("json.Unmarshal err", err)
		return
	}

	// 这里也可以是srv name 规则，跳过 底层的sub name
	// 如果要用默认的呢?
	nlog.Info("HandleSubscribe  %v", info)
}

func (c *AcceptClient) GetID() int64 {
	return c.ID
}
