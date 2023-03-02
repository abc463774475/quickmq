package client

import (
	"github.com/abc463774475/quickmq/msg"
)

func (c *Client) sendPing() {
	//c.SendMsg(msg.MSG_PING, msg.MsgPing{
	//	Time: time.Now(),
	//})
}

func (c *Client) processMsgPing(_msg *msg.Msg) {
	c.SendMsg(msg.MSG_PONG, msg.MsgPong{})
}

func (c *Client) processMsgPong(_msg *msg.Msg) {
}
