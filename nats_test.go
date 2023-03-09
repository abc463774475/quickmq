package quickmq

import (
	"testing"
	"time"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/client"
	"github.com/abc463774475/quickmq/msg"
	"github.com/abc463774475/quickmq/server"
	"github.com/abc463774475/quickmq/utils/snowflake"
)

func TestNats(t *testing.T) {
	snowflake.Init(1)
	server.NewServer(server.WithAddr(":8087")).Start()
}

func TestSub(t *testing.T) {
	snowflake.Init(1)
	c := client.NewClient(client.WithAddr(":8081"))

	c.Subscribe("test", func(data []byte, _msg *msg.MsgPub) {
		nlog.Info("test %v", _msg)
	}, nil)

	time.Sleep(time.Second * 100)
}

func TestPub(t *testing.T) {
	snowflake.Init(1)
	c := client.NewClient(client.WithAddr(":8081"))

	c.Publish("test", []byte("hello world"))

	time.Sleep(time.Second * 100)
}
