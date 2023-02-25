package server

import (
	"testing"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/utils/snowflake"
)

func TestServer_common(t *testing.T) {
	nlog.InitLog(nlog.WithCompressType(nlog.Quick))
	snowflake.Init(1)
	s := newServer(WithAddr(":8087"),
		WithClusterAddr(":18087"),
	)

	s.start()

	nlog.Info("end")
}

func TestServer_route1(t *testing.T) {
	nlog.InitLog(nlog.WithCompressType(nlog.Quick))
	snowflake.Init(2)
	s := newServer(WithName("rout1"),
		WithAddr(":8088"),
		WithClusterAddr(":18088"),
		WithConnectRouterAddr("127.0.0.1:18087"),
	)

	s.start()

	nlog.Info("end")
}

func TestServer_route2(t *testing.T) {
	nlog.InitLog(nlog.WithCompressType(nlog.Quick))

	snowflake.Init(3)
	s := newServer(WithName("rout2"),
		WithAddr(":8089"),
		WithClusterAddr(":18089"),
		WithConnectRouterAddr("127.0.0.1:18087"),
	)

	s.start()

	nlog.Info("end")
}

func TestLen(t *testing.T) {
	str := `{"sub":"haorena","data":"AQEBAQ=="}`
	t.Log(len(str))
}
