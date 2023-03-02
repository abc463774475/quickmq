package base

import "github.com/abc463774475/quickmq/msg"

type AcceptClienHandler interface {
	GetID() int64

	HandleRegister(_msg *msg.Msg)
	HandlePublish(_msg *msg.Msg)
	HandleSubscribe(_msg *msg.Msg)
}

type AcceptClientSendHandler interface {
	SendMsg(msgid msg.MSGID, i interface{})
}
