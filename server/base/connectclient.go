package base

import "github.com/abc463774475/quickmq/qcmq/msg"

type (
	SUBFUN      func(data []byte, _msg *msg.MsgPub)
	SUBACKFUN   func(_msg *msg.MsgSubAck)
	PUBCALLBACK func(data []byte)
)

type ConnectClientHandler interface {
	GetID() int64
	Register()
}

type ConnectClientSendHandler interface {
	SendMsg(msgid msg.MSGID, i interface{})
	Publish(sub string, i interface{})
	Subscribe(sub string, subf SUBFUN, subackf SUBACKFUN)
	Req(sub string, i interface{}, pubCB PUBCALLBACK)
}
