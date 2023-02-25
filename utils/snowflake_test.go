package utils

import (
	"sync"
	"testing"
	"time"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/bwmarrin/snowflake"
	_ "github.com/bwmarrin/snowflake"
	_ "github.com/sony/sonyflake"
)

func TestSnowFlake(t *testing.T) {
	start := time.Now()
	m := map[int64]struct{}{}
	l := sync.Mutex{}

	max := 20000

	go func() {
		node, _ := snowflake.NewNode(1)
		for i := 0; i < max; i++ {
			p := node.Generate().Int64()
			l.Lock()
			if _, ok := m[int64(p)]; ok {
				nlog.Panic("dup node  %v  %v", len(m), p)
			}
			m[int64(p)] = struct{}{}
			l.Unlock()
		}
	}()

	node, _ := snowflake.NewNode(1)
	for i := 0; i < max; i++ {
		p := node.Generate().Int64()
		l.Lock()
		if _, ok := m[int64(p)]; ok {
			nlog.Panic("dup node  %v", p)
		}
		m[int64(p)] = struct{}{}
		l.Unlock()
	}

	time.Sleep(3 * time.Second)
	nlog.Info("m %v  %v", time.Now().Sub(start), len(m))
}
