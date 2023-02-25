package server

import (
	"encoding/binary"
	"net"
	"time"

	nlog "github.com/abc463774475/my_tool/n_log"
)

type route struct {
	remoteID   int64
	remoteName string

	replySubs map[*subscription]*time.Timer
}

// buf not json just fro quick
func (c *client) addRouteSubOrUnsubProtoToBuf(buf []byte,
	accName string, sub *subscription, isSub bool,
) []byte {
	if isSub {
		buf = append(buf, '+')
	} else {
		buf = append(buf, '-')
	}
	buf = append(buf, accName...)
	buf = append(buf, ' ')
	buf = append(buf, sub.subject...)
	buf = append(buf, ' ')
	buf = append(buf, sub.queue...)
	buf = append(buf, ' ')

	binary.LittleEndian.PutUint32(buf[len(buf):], uint32(sub.qw))
	buf = append(buf, ' ')
	return buf
}

func (c *client) connect() bool {
	var err error
	addr := c.addr
	c.nc, err = net.DialTimeout("tcp", addr, time.Second*5)
	if err != nil {
		return false
	}

	nlog.Info("connect: %v", addr)

	return true
}

//func (c *client) routerRegister() {
//	c.sendMsg(msg.MSG_HANDSHAKE, &msg.MsgHandshake{
//		Type: int32(c.kind),
//		Name: utils.GetSessionIDByTimer(),
//	})
//}
