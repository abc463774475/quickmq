package mock

import (
	"testing"
	"time"

	"github.com/abc463774475/quickmq/msg"
	"github.com/abc463774475/quickmq/utils/snowflake"
)

// TestConnectClient 测试客户端
func TestConnectClient(t *testing.T) {
	snowflake.Init(1000)

	cl := NewConnectClient(":8081", "test", 1)

	time.Sleep(3 * time.Second)

	cl.Subscribe("haorena", func(data []byte, pub *msg.MsgPub) {
		t.Log("sub: ", string(data))
	}, func(ack *msg.MsgSubAck) {
		t.Log("sub ack: ", ack)
	})

	time.Sleep(3 * time.Second)

	cl.Publish("haorena", []byte("hello world"))

	time.Sleep(1000 * time.Second)
}
