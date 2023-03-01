package mock

import "github.com/abc463774475/quickmq/qcmq/msg"

type MSGID = msg.MSGID

const (
	MSGID_MSG_REGISTER MSGID = 1
	MSGID_MSG_PUBLISH  MSGID = 2

	MSGID_MSG_SUBSCRIBE MSGID = 3
)
