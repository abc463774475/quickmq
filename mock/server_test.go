package mock

import (
	"testing"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/server"
	"github.com/abc463774475/quickmq/utils/snowflake"
)

func TestServer(t *testing.T) {
	nlog.Info("TestServer start")
	snowflake.Init(1)

	srv := server.NewServer(NewServer(), server.WithAddr(":8081"))

	srv.Start()
}
