package server

// Make sure all are 64bits for atomic use
type stats struct {
	inMsgs        int64
	outMsgs       int64
	inBytes       int64
	outBytes      int64
	slowConsumers int64
}

type RouterConfInfo struct {
	ID          string `json:"id"`
	ListenAddr  string `json:"listenAddr"`
	ClusterAddr string `json:"clusterAddr"`
}
