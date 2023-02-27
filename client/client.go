package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abc463774475/msglist"
	"github.com/abc463774475/timer/timewheel"

	"github.com/abc463774475/quickmq/qcmq/utils/snowflake"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/msg"
)

type (
	SUBFUN      func(data []byte, _msg *msg.MsgPub)
	SUBACKFUN   func(_msg *msg.MsgSubAck)
	PUBCALLBACK func(data []byte)

	Client struct {
		nc   net.Conn
		addr string

		clientType int
		msgRecv    *msglist.MsgList
		// msgSend    chan *msg.Msg
		msgSend *msglist.MsgList

		rttDuration time.Duration
		cquit       chan struct{}

		sfs     map[string]SUBFUN
		rwmuSFs sync.RWMutex

		sackfun  map[int64]SUBACKFUN
		rwmuSAck sync.RWMutex

		pubCallback map[int64]PUBCALLBACK
		rwmuPubCB   sync.RWMutex

		sid uint64

		tw *timewheel.TimeWheel
	}
)

func newClient(addr string, ct int) *Client {
	c := &Client{
		nc:         nil,
		addr:       addr,
		clientType: ct,
		// msgRecv:     make(chan *msg.Msg, 200),
		msgRecv: msglist.NewMsgList(),
		msgSend: msglist.NewMsgList(),
		// msgSend:     make(chan *msg.Msg, 200),
		cquit:       make(chan struct{}, 2),
		sfs:         make(map[string]SUBFUN),
		sackfun:     make(map[int64]SUBACKFUN),
		pubCallback: make(map[int64]PUBCALLBACK),
		tw:          timewheel.NewTimeWheel(1*time.Second, 100),
	}

	c.tw.Start()
	go c.run()

	return c
}

func NewClient(addr string) *Client {
	return newClient(addr, 0)
}

func (c *Client) readLoop() {
	nlog.Info("readLoop")
	defer func() {
		nlog.Info("readLoop end")
	}()
	for {
		headeBytes := make([]byte, msg.HeadSize)
		n, err := io.ReadFull(c.nc, headeBytes)
		if err != nil || n == 0 {
			nlog.Erro("readLoop: read header: %v", err)
			c.closeConnection()
			return
		}

		header := &msg.Head{}
		header.Load(headeBytes)

		if header.Length == 0 {
			//c.msgRecv <- &msg.Msg{
			//	Head: *header,
			//	Data: []byte{},
			//}
			c.msgRecv.Push(&msg.Msg{
				Head: *header,
				Data: []byte{},
			})
			continue
		}

		data := make([]byte, header.Length)
		n, err = io.ReadFull(c.nc, data)
		if err != nil || n == 0 {
			nlog.Erro("readLoop: read data: %v", err)
			c.closeConnection()
			return
		}

		//c.msgRecv <- &msg.Msg{
		//	Head: *header,
		//	Data: data,
		//}
		c.msgRecv.Push(&msg.Msg{
			Head: *header,
			Data: data,
		})
	}
}

func (c *Client) writeLoop() {
	nlog.Info("writeLoop")
	defer func() {
		nlog.Info("writeLoop end")
	}()
	for {
		//select {
		//case msg := <-c.msgSend:
		//	_ = c.writeMsg(msg)
		//case <-c.cquit:
		//	return
		//}

		msgs := c.msgSend.Pop()
		for _, _msg := range msgs {
			switch data := _msg.(type) {
			case *msg.Msg:
				_ = c.writeMsg(data)
			case nil:
				return
			}
		}
	}
}

func (c *Client) writeMsg(msg *msg.Msg) error {
	data := msg.Save()
	n, err := c.nc.Write(data)
	if err != nil {
		return err
	}

	if n == 0 {
		return errors.New("writeMsg: write 0 bytes")
	}

	if n != len(data) {
		time.Sleep(1 * time.Second)
		n1, err := c.nc.Write(data[n:])
		if err != nil {
			return err
		}

		if n1 != len(data[n:]) {
			return fmt.Errorf("writeMsg: write 0 bytes %v", n1)
		}

	}
	return nil
}

// client active close
func (c *Client) ActiveClose() {
	c.UnsubAll()
	// todo 简单实现
	time.AfterFunc(time.Second*2, func() {
		c.nc.Close()
	})
}

func (c *Client) closeConnection() {
	c.nc.Close()
	// just twice to make sure, since there have two goroutines
	// c.cquit <- struct{}{}

	c.msgRecv.Push(nil)
	c.msgSend.Push(nil)
	// c.cquit <- struct{}{}
}

func (c *Client) UnsubAll() {
	c.rwmuSFs.RLock()
	defer c.rwmuSFs.RUnlock()
	subs := make([]string, 0, len(c.sfs))
	for k := range c.sfs {
		subs = append(subs, k)
	}

	c.sendMsg(msg.MSG_UNSUB, msg.MsgUnSub{Subs: subs})
}

func (c *Client) del() {
	// close(c.msgRecv)
	// close(c.msgSend)
	// close(c.cquit)
	c.msgRecv.Clear()
	c.tw.Stop()
}

func (c *Client) processMsg() {
	nlog.Info("processMsg")

	defer func() {
		nlog.Info("processMsg end")
	}()

	type timerMsg struct{}

	c.tw.Add(5*time.Second, -1, func() {
		// c.sendMsg(msg.MSG_PING, nil)
		c.msgRecv.Push(&timerMsg{})
	}, nil)

	for {
		_msgs := c.msgRecv.Pop()

		for _, _msg := range _msgs {
			// recv nil, means the channel is closed
			switch msgData := _msg.(type) {
			case *msg.Msg:
				c.processMsgImpl(msgData)
			case *timerMsg:
				c.sendMsg(msg.MSG_PING, nil)
			case nil:
				return
			}
		}
	}
}

func (c *Client) run() {
	// Set the read deadline for the client.
	// Start reading.
	nlog.Debug("client run")
	defer func() {
		nlog.Debug("client del finish")
	}()

	c.connect()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		c.readLoop()
		wg.Done()
	}()
	go func() {
		c.writeLoop()
		wg.Done()
	}()

	c.processMsg()
	wg.Wait()
	c.del()
}

func (c *Client) connect() {
	var err error
	c.nc, err = net.DialTimeout("tcp", c.addr, time.Second*5)

	if err != nil {
		panic("err = " + err.Error())
	}

	nlog.Info("connect: %v", c.addr)
	// c.register()
}

func (c *Client) processMsgImpl(_msg *msg.Msg) {
	msgFilter := map[msg.MSGID]bool{
		msg.MSG_PONG: true,
	}

	if _, ok := msgFilter[_msg.ID]; !ok {
		// nlog.Info("processMsgImpl: %v  %v", _msg.Head.ID, string(_msg.Data))
	}

	switch _msg.ID {
	case msg.MSG_PING:
		c.processMsgPing(_msg)
	case msg.MSG_PONG:
		c.processMsgPong(_msg)
	case msg.MSG_PUB:
		c.processMsgPub(_msg)
	case msg.MSG_SUBACK:
		c.processMsgSubAck(_msg)
	}
}

func (c *Client) register() {
	nlog.Debug("register")
	c.sendMsg(msg.MSG_HANDSHAKE, &msg.MsgHandshake{
		Type: int32(c.clientType),
		Name: "client test",
	})
}

func (c *Client) sendMsg(msgID msg.MSGID, i interface{}) {
	data, err := json.Marshal(i)
	if err != nil {
		nlog.Erro("sendMsg: json.Marshal: %v", err)
		return
	}

	msg := &msg.Msg{
		Head: msg.Head{
			ID:      msgID,
			Length:  0,
			Crc32:   0,
			Encrypt: 0,
		},
		Data: data,
	}

	//if len(c.msgSend) < cap(c.msgSend) {
	//	c.msgSend <- msg
	//} else {
	//	nlog.Erro("sendMsg: msgSend is full")
	//}
	c.msgSend.Push(msg)
}

func (c *Client) processMsgPub(_msg *msg.Msg) {
	pub := &msg.MsgPub{}
	err := json.Unmarshal(_msg.Data, pub)
	if err != nil {
		nlog.Erro("processMsgPub: json.Unmarshal: %v", err)
		return
	}
	// nlog.Debug("processMsgPub: %v  %v", pub.Sub, string(pub.Data))

	c.rwmuSFs.RLock()
	sf, ok := c.sfs[pub.Sub]
	if !ok {
		c.rwmuSFs.RUnlock()
		nlog.Erro("processMsgPub: sub not found: %v", pub.Sub)
		return
	}
	c.rwmuSFs.RUnlock()

	sf(pub.Data, pub)
}

func (c *Client) Subscribe(sub string, subf SUBFUN, subackf SUBACKFUN) {
	c.rwmuSFs.RLock()
	_, ok := c.sfs[sub]
	if ok {
		c.rwmuSFs.RUnlock()
		nlog.Erro("Subscribe: sub already exists: %v", sub)
		return
	}

	c.rwmuSFs.RUnlock()

	c.rwmuSFs.Lock()
	c.sfs[sub] = subf
	atomic.AddUint64(&c.sid, 1)
	sid := atomic.LoadUint64(&c.sid)
	strSID := strconv.FormatUint(sid, 20)
	uid := snowflake.GetID()
	c.sendMsg(msg.MSG_SUB, &msg.MsgSub{
		UniqueID: uid,
		Sub:      sub,
		SID:      strSID,
	})

	if subackf != nil {
		c.rwmuSAck.Lock()
		c.sackfun[uid] = subackf
		c.rwmuSAck.Unlock()
	}

	c.rwmuSFs.Unlock()
}

func (c *Client) UnSubscribe(sub string) {
	c.rwmuSFs.Lock()
	if _, ok := c.sfs[sub]; !ok {
		nlog.Erro("UnSubscribe: %v not exist", sub)
		c.rwmuSFs.Unlock()
		return
	}

	c.sendMsg(msg.MSG_UNSUB, &msg.MsgUnSub{
		Subs: []string{sub},
	})
	delete(c.sfs, sub)
	c.rwmuSFs.Unlock()
}

func (c *Client) Publish(sub string, i interface{}) {
	var data []byte
	var err error
	switch i.(type) {
	case []byte:
		data = i.([]byte)
	case *[]byte:
		data = *i.(*[]byte)
	case string:
		data = []byte(i.(string))
	case *string:
		data = []byte(*i.(*string))
	default:
		data, err = json.Marshal(i)
		if err != nil {
			nlog.Erro("Publish: json.Marshal: %v", err)
			return
		}
	}

	c.sendMsg(msg.MSG_PUB, &msg.MsgPub{
		UniqueID: snowflake.GetID(),
		Sub:      sub,
		Data:     data,
	})
}

func (c *Client) Req(sub string, i interface{}, pubCB PUBCALLBACK) {
	var data []byte
	var err error
	switch i.(type) {
	case []byte:
		data = i.([]byte)
	case *[]byte:
		data = *i.(*[]byte)
	case string:
		data = []byte(i.(string))
	case *string:
		data = []byte(*i.(*string))
	default:
		data, err = json.Marshal(i)
		if err != nil {
			nlog.Erro("Publish: json.Marshal: %v", err)
			return
		}
	}
	uid := snowflake.GetID()

	// c.rwmuPubCB.Lock()
	// c.pubCallback[uid] = pubCB
	// c.rwmuPubCB.Unlock()

	c.Subscribe(fmt.Sprintf("%v", uid), func(data []byte, pub *msg.MsgPub) {
		pubCB(data)
	}, nil)

	c.sendMsg(msg.MSG_PUB, &msg.MsgPub{
		UniqueID: uid,
		Sub:      sub,
		Data:     data,
	})
}

func (c *Client) processMsgSubAck(_msg *msg.Msg) {
	suback := &msg.MsgSubAck{}
	err := json.Unmarshal(_msg.Data, suback)
	if err != nil {
		nlog.Erro("processMsgSubAck: json.Unmarshal: %v", err)
		return
	}

	c.rwmuSAck.Lock()
	subackf, ok := c.sackfun[suback.UniqueID]
	if !ok {
		c.rwmuSAck.Unlock()
		// nlog.Erro("processMsgSubAck: suback not found: %v", suback.UniqueID)
		return
	}

	delete(c.sackfun, suback.UniqueID)

	c.rwmuSAck.Unlock()

	subackf(suback)
}
