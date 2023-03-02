package client

import (
	"sync/atomic"
	"testing"
	"time"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/utils/snowflake"

	"github.com/abc463774475/quickmq/msg"
	"github.com/abc463774475/timer/timewheel"
)

func TestClient_sub(t *testing.T) {
	snowflake.Init(1000)
	sub := "haorena"

	c := NewClient(WithAddr("localhost:8087"))
	var count int32 = 0
	time.AfterFunc(1*time.Second, func() {
		c.Subscribe(sub, func(data []byte, pub *msg.MsgPub) {
			atomic.AddInt32(&count, 1)
			// nlog.Erro("sub: %v %v", string(data), atomic.LoadInt32(&count))
			v := atomic.LoadInt32(&count)
			if v%10000 == 0 {
				nlog.Erro("sub: %v %v", string(data), atomic.LoadInt32(&count))
			}
		}, func(ack *msg.MsgSubAck) {
			nlog.Erro("sub ack: %v", ack)
		})
	})

	time.Sleep(1000 * time.Second)
}

func TestClient_pub(t *testing.T) {
	snowflake.Init(1001)
	c := NewClient(WithAddr("localhost:8087"))
	data := []byte("hello world")
	time.AfterFunc(2*time.Second, func() {
		for i := 0; i < 100000; i++ {
			c.Publish("haorena", data)
		}
	})

	time.Sleep(1000 * time.Second)

	tw := timewheel.NewTimeWheel(1*time.Second, 100)
	tw.Start()

	tw.Add(10*time.Millisecond, 1, func() {
		nlog.Erro("xxxxxxxxx")
	}, nil)
}
