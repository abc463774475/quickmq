package msg

import (
	"bytes"
	"encoding/binary"
	"unsafe"

	nlog "github.com/abc463774475/my_tool/n_log"
)

var HeadSize uint32 = func() uint32 {
	//nolint: staticcheck
	return uint32(unsafe.Sizeof(Head{}))
}()

const (
	MAX_MSG_LENGTH = 1024 * 10
)

// 传输层添加的包头
type Head struct {
	ID      MSGID
	Length  uint32
	Crc32   uint32
	Encrypt uint32
}

func (h *Head) Load(data []byte) {
	_ = binary.Read(bytes.NewBuffer(data), binary.LittleEndian, h)
}

func (h *Head) Save() []byte {
	w := bytes.NewBuffer(nil)
	_ = binary.Write(w, binary.LittleEndian, h)
	return w.Bytes()
}

type Msg struct {
	Head
	Data []byte
}

func (m *Msg) Load(data []byte) {
	m.Head.Load(data[:HeadSize])
	left := data[HeadSize:]
	if m.Length+uint32(HeadSize) != uint32(len(data)) {
		panic("msg length error")
	}
	m.Data = left
}

func (m *Msg) Save() []byte {
	m.Length = uint32(len(m.Data))
	data := make([]byte, m.Length+HeadSize)
	copy(data[:HeadSize], m.Head.Save())
	copy(data[HeadSize:], m.Data)

	nlog.Erro("save head %v msg: %v  len  %v", m.Head, string(m.Data), len(data))
	return data
}
