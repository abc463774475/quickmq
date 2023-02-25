package client

import (
	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/utils/snowflake"
	"testing"
	"time"

	"github.com/abc463774475/quickmq/qcmq/msg"
)

func TestClient_sub(t *testing.T) {
	snowflake.Init(1000)
	sub := "haorena"

	c := newClient("localhost:8087", 0)
	time.AfterFunc(1*time.Second, func() {
		c.Subscribe(sub, func(data []byte, pub *msg.MsgPub) {
			nlog.Erro("sub: %v", string(data))
		}, func(ack *msg.MsgSubAck) {
			nlog.Erro("sub ack: %v", ack)
		})
	})

	time.Sleep(1000 * time.Second)
}

func TestClient_pub(t *testing.T) {
	snowflake.Init(1001)
	c := newClient("localhost:8087", 0)
	data := []byte("hello world")
	time.AfterFunc(2*time.Second, func() {
		c.Publish("haorena", data)
	})

	time.Sleep(1000 * time.Second)
}
