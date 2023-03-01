package mock

type ClientRegister struct {
	ID   int64
	Name string
	// SubName 不是在客户端注册的么？ 这里应该不需要了吧
	// SubName         string
	// WildcardSubName string
}

type MsgPub struct {
	UniqueID int64
	To       string
	Data     []byte
}

type MsgSub struct {
	UniqueID int64
	To       string

	// SID sequence id
	SID string
}
