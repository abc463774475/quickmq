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

	"github.com/abc463774475/quickmq/server/base"

	"github.com/abc463774475/msglist"
	"github.com/abc463774475/timer/timewheel"

	"github.com/abc463774475/quickmq/utils/snowflake"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/msg"
)

type (
	SUBFUN      = base.SUBFUN
	SUBACKFUN   = base.SUBACKFUN
	PUBCALLBACK = base.PUBCALLBACK

	Client struct {
		id   int64
		nc   net.Conn
		addr string

		clientType int
		readList   *msglist.MsgList
		writeList  *msglist.MsgList

		clienter base.ConnectClientHandler

		rttDuration time.Duration
		cquit       chan struct{}

		sfs     map[string]SUBFUN
		rwmuSFs sync.RWMutex

		sackfuns map[int64]SUBACKFUN
		rwmuSAck sync.RWMutex

		pubCallbacks map[int64]PUBCALLBACK
		rwmuPubCB    sync.RWMutex

		sid uint64

		tw *timewheel.TimeWheel
	}
)

func (c *Client) GetID() int64 {
	return c.id
}

func (c *Client) Register() {}

func newClient(options ...Option) *Client {
	id := snowflake.GetID()
	c := &Client{
		nc: nil,
		// addr:       addr,
		// clientType: ct,
		id: id,
	}
	//if handler == nil {
	//	handler = c
	//}
	//
	//c.clienter = handler

	for _, o := range options {
		o.apply(c)
	}
	if c.clienter == nil {
		c.clienter = c
	}

	c.Init()
	c.tw.Start()
	go c.run()

	return c
}

func NewClient(options ...Option) *Client {
	return newClient(options...)
}

// Init
func (c *Client) Init() {
	c.readList = msglist.NewMsgList()
	c.writeList = msglist.NewMsgList()
	c.cquit = make(chan struct{}, 2)
	c.sfs = make(map[string]SUBFUN)
	c.sackfuns = make(map[int64]SUBACKFUN)
	c.pubCallbacks = make(map[int64]PUBCALLBACK)
	c.tw = timewheel.NewTimeWheel(1*time.Second, 100)
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
			c.readList.Push(&msg.Msg{
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

		c.readList.Push(&msg.Msg{
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
		msgs := c.writeList.Pop()
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
		time.Sleep(100 * time.Millisecond)
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

	c.readList.Push(nil)
	c.writeList.Push(nil)
	// c.cquit <- struct{}{}
}

func (c *Client) UnsubAll() {
	c.rwmuSFs.RLock()
	defer c.rwmuSFs.RUnlock()
	subs := make([]string, 0, len(c.sfs))
	for k := range c.sfs {
		subs = append(subs, k)
	}

	c.SendMsg(msg.MSG_UNSUB, msg.MsgUnSub{Subs: subs})
}

func (c *Client) del() {
	c.readList.Clear()
	c.tw.Stop()
}

func (c *Client) processMsg() {
	nlog.Info("processMsg")

	defer func() {
		nlog.Info("processMsg end")
	}()

	type timerMsg struct{}

	c.tw.Add(5*time.Second, -1, func() {
		// c.SendMsg(msg.MSG_PING, nil)
		c.readList.Push(&timerMsg{})
	}, nil)

	for {
		_msgs := c.readList.Pop()

		for _, _msg := range _msgs {
			// recv nil, means the channel is closed
			switch msgData := _msg.(type) {
			case *msg.Msg:
				c.processMsgImpl(msgData)
			case *timerMsg:
				c.SendMsg(msg.MSG_PING, nil)
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

	c.clienter.Register()
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
	c.SendMsg(msg.MSG_HANDSHAKE, &msg.MsgHandshake{
		Type: int32(c.clientType),
		Name: "client test",
	})
}

func (c *Client) SendMsg(msgID msg.MSGID, i interface{}) {
	data, err := json.Marshal(i)
	if err != nil {
		nlog.Erro("SendMsg: json.Marshal: %v", err)
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

	c.writeList.Push(msg)
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
	c.SendMsg(msg.MSG_SUB, &msg.MsgSub{
		UniqueID: uid,
		Sub:      sub,
		SID:      strSID,
	})

	if subackf != nil {
		c.rwmuSAck.Lock()
		c.sackfuns[uid] = subackf
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

	c.SendMsg(msg.MSG_UNSUB, &msg.MsgUnSub{
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

	c.SendMsg(msg.MSG_PUB, &msg.MsgPub{
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

	c.Subscribe(fmt.Sprintf("%v", uid), func(data []byte, pub *msg.MsgPub) {
		pubCB(data)
	}, nil)

	c.SendMsg(msg.MSG_PUB, &msg.MsgPub{
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
	subackf, ok := c.sackfuns[suback.UniqueID]
	if !ok {
		c.rwmuSAck.Unlock()
		// nlog.Erro("processMsgSubAck: suback not found: %v", suback.UniqueID)
		return
	}

	delete(c.sackfuns, suback.UniqueID)

	c.rwmuSAck.Unlock()

	subackf(suback)
}
