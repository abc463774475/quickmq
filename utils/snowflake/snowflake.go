package snowflake

import (
	"fmt"

	"github.com/bwmarrin/snowflake"
)

const (
	MaxNodeID = 1024
)

var (
	NodeID int64 = 1
	Node   *snowflake.Node
)

func Init(nodeID int64) {
	NodeID = nodeID
	var err error
	Node, err = snowflake.NewNode(NodeID)
	if err != nil {
		panic(fmt.Sprintf("err  %v", err))
	}
}

func GetID() int64 {
	if Node == nil {
		panic("node nil")
	}
	return Node.Generate().Int64()
}
