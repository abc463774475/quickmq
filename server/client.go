package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/abc463774475/msglist"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/msg"
	"github.com/abc463774475/quickmq/qcmq/utils/snowflake"
)

type ClientType int

//go:generate go run github.com/dmarkham/enumer -type=ClientType
const (
	CLIENT ClientType = iota
	ROUTER
	SYSTEM
	ACCOUNT
)

type msgDeny struct {
	reason string
	deny   *sublist
	dcache map[string]bool
}

const (
	//nolint:varcheck
	maxBuffersSize = 1024 * 1024 * 64
)

type clientFlag int

//go:generate go run github.com/dmarkham/enumer -type=clientFlag
const (
	handshake clientFlag = 1 << iota
	readOnly
	expectConnect
	closeConnection
)

func (cf *clientFlag) set(c clientFlag) {
	*cf |= c
}

func (cf *clientFlag) clear(c clientFlag) {
	*cf &= ^c
}

func (cf clientFlag) isSet(c clientFlag) bool {
	return cf&c != 0
}

func (cf *clientFlag) setIfNotSet(c clientFlag) bool {
	if *cf&c == 0 {
		*cf |= c
		return true
	}
	return false
}

type closeState int

const (
	//nolint:varcheck
	clientClosed closeState = iota + 1
	//nolint:varcheck
	writeError
	readError
)

type readCache struct {
	id      uint64
	results map[string]*sublistResult

	msgs  int64
	bytes int64
	subs  int32

	rsz int32
	srs int32
}

type routeTarget struct {
	sub *subscription
	qs  []byte
	_qs [32]byte
}

type outbound struct {
	p   []byte        // Primary write buffer
	s   []byte        // Secondary for use post flush
	nb  net.Buffers   // net.Buffers for writev IO
	sz  int32         // limit size per []byte, uses variable BufSize constants, start, min, max.
	sws int32         // Number of short writes, used for dynamic resizing.
	pb  int64         // Total pending/queued bytes.
	pm  int32         // Total pending/queued messages.
	fsp int32         // Flush signals that are pending per producer from readLoop's pcd.
	sg  *sync.Cond    // To signal writeLoop that there is data to flush.
	wdl time.Duration // Snapshot of write deadline.
	mp  int64         // Snapshot of max pending for client.
	lft time.Duration // Last flush time for Write.
	stc chan struct{} // Stall chan we create to slow down producers on overrun, e.g. fan-in.
}

type client struct {
	id             int64
	kind           ClientType
	isServerAccept bool

	name string

	stats
	srv *server
	acc *Account

	mpay  int32
	msubs int32

	nc net.Conn

	in  readCache
	out outbound

	addr string

	subs        map[string]*subscription
	subsWithSID map[string]*subscription
	mperms      *msgDeny

	rtt      time.Duration
	rttStart time.Time

	msgRecv *msglist.MsgList
	msgSend *msglist.MsgList

	cquit chan struct{}

	mu sync.Mutex
}

func newAcceptClient(id int64, conn net.Conn, s *server) *client {
	c := &client{}
	c.id = id
	c.srv = s
	c.nc = conn
	c.isServerAccept = true

	return c
}

func newConnectClient(addr string) *client {
	c := &client{}
	c.id = snowflake.GetID()
	c.addr = addr
	c.isServerAccept = false
	return c
}

func (c *client) init() {
	c.subs = make(map[string]*subscription, 100)
	c.subsWithSID = make(map[string]*subscription, 100)
	c.mperms = &msgDeny{}
	// c.msgRecv = make(chan *msg.Msg, 2000)
	c.msgRecv = msglist.NewMsgList()
	// c.msgSend = make(chan *msg.Msg, 2000)
	c.msgSend = msglist.NewMsgList()
	// c.cquit = make(chan struct{}, 2)
}

func (c *client) run() {
	nlog.Info("client run")
	defer func() {
		nlog.Info("client delete finish")
	}()

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

func (c *client) readLoop() {
	nlog.Info("client readLoop")
	defer func() {
		nlog.Info("client readLoop end")
	}()
	for {
		headeBytes := make([]byte, msg.HeadSize)
		n, err := io.ReadFull(c.nc, headeBytes)
		if err != nil || n == 0 {
			nlog.Erro("readLoop: read header: %v", err)
			c.closeConnection(readError)
			return
		}

		header := &msg.Head{}
		header.Load(headeBytes)

		if header.Length == 0 {
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
			c.closeConnection(readError)
			return
		}

		c.msgRecv.Push(&msg.Msg{
			Head: *header,
			Data: data,
		})
	}
}

func (c *client) writeLoop() {
	nlog.Info("client writeLoop")
	defer func() {
		nlog.Info("client writeLoop end")
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

func (c *client) writeMsg(msg *msg.Msg) error {
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

func (c *client) closeConnection(state closeState) {
	nlog.Info("closeConnection: %v", state)
	c.nc.Close()
	// close(c.cquit)

	c.msgRecv.Push(nil)
	c.msgSend.Push(nil)
}

func (c *client) processMsg() {
	nlog.Info("client processMsg")
	defer func() {
		nlog.Info("client processMsg end")
	}()

	for {
		//select {
		//case msg := <-c.msgRecv:
		//	c.processMsgImpl(msg)
		//case <-c.cquit:
		//	return
		//}

		msgs := c.msgRecv.Pop()
		for _, _msg := range msgs {
			switch data := _msg.(type) {
			case *msg.Msg:
				c.processMsgImpl(data)
			case nil:
				return
			}
		}
	}
}

func (c *client) SendMsg(msgID msg.MSGID, i interface{}) {
	var data []byte
	var err error
	if i == nil {
	} else if d, ok := i.([]byte); ok {
		data = d
	} else if d, ok := i.(string); ok {
		data = []byte(d)
	} else {
		data, err = json.Marshal(i)
		if err != nil {
			nlog.Erro("sendMsg: json.Marshal: %v", err)
			return
		}
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

func (c *client) Ping() {
	c.rttStart = time.Now()
	c.SendMsg(msg.MSG_PING, msg.MsgPing{})
}

func (c *client) processMsgHandshake(_msg *msg.Msg) {
	handshake := &msg.MsgHandshake{}
	err := json.Unmarshal(_msg.Data, handshake)
	if err != nil {
		nlog.Erro("processMsgHandshake: json.Unmarshal: %v", err)
		return
	}

	c.name = handshake.Name
	c.kind = ClientType(handshake.Type)
	if handshake.Type == int32(CLIENT) {
	} else if handshake.Type == int32(ROUTER) {
		c.srv.addRoute(c, handshake.Name)
	} else if handshake.Type == int32(SYSTEM) {
	} else if handshake.Type == int32(ACCOUNT) {
	} else {
		nlog.Erro("processMsgHandshake: unknown type: %v", handshake.Type)
		return
	}
}

func (c *client) processMsgImpl(_msg *msg.Msg) {
	// nlog.Erro("processMsgImpl: %v  data %v", _msg.Head.ID, string(_msg.Data))
	switch _msg.ID {
	case msg.MSG_PING:
		c.SendMsg(msg.MSG_PONG, msg.MsgPong{})
	case msg.MSG_PONG:
		c.rtt = time.Since(c.rttStart)
		nlog.Info("processMsgImpl: %v  rtt %v", _msg.Head.ID, c.rtt)
	// case msg.MSG_HANDSHAKE:
	//	c.processMsgHandshake(_msg)
	case msg.MSG_SNAPSHOTSUBS:
		c.processMsgSnapshotSubs(_msg)
	case msg.MSG_SUB:
		c.processMsgSub(_msg)
	case msg.MSG_UNSUB:
		c.processMsgUnSub(_msg)
	case msg.MSG_PUB:
		c.processMsgPub(_msg)
	case msg.MSG_REMOTEROUTEADDSUB:
		c.processMsgRemoteRouteAddSub(_msg)
	case msg.MSG_REMOTEROUTEADDUNSUB:
		c.processMsgRemoteRouteUnSub(_msg)
	case msg.MSG_ROUTEPUB:
		c.processMsgRoutePub(_msg)
	case msg.MSG_REGISTERROUTER:
		c.processMsgRegisterRouter(_msg)
	case msg.MSG_CURALLROUTES:
		c.processMsgCurAllRoutes(_msg)
	}
}

func (c *client) processMsgSnapshotSubs(_msg *msg.Msg) {
	snapshotSubs := &msg.MsgSnapshotSubs{}
	err := json.Unmarshal(_msg.Data, snapshotSubs)
	if err != nil {
		nlog.Erro("processMsgSnapshotSubs: json.Unmarshal: %v", err)
		return
	}

	c.srv.snapshotSubs(c, snapshotSubs)
}

func (c *client) processMsgRoutePub(_msg *msg.Msg) {
	routePub := &msg.MsgRoutePub{}
	err := json.Unmarshal(_msg.Data, routePub)
	if err != nil {
		nlog.Erro("processMsgRoutePub: json.Unmarshal: %v", err)
		return
	}

	c.msgPub(&msg.MsgPub{
		Sub:  routePub.Sub,
		Data: routePub.Data,
	})
}

func (c *client) processMsgSub(_msg *msg.Msg) {
	msub := &msg.MsgSub{}
	err := json.Unmarshal(_msg.Data, msub)
	if err != nil {
		nlog.Erro("processMsgSub: json.Unmarshal: %v", err)
		return
	}

	sub := &subscription{
		client:  c,
		subject: msub.Sub,
		queue:   nil,
		qw:      0,
		closed:  0,
	}
	acc := c.acc
	c.mu.Lock()
	c.in.subs++
	sid := msub.SID

	ts := c.subsWithSID[sid]
	// 这是序列号的重复检查
	if ts == nil {
		c.subsWithSID[sid] = sub
		acc.addRM(sub.subject, 1)
		acc.addCurSubs(sub.subject, sub)
		err := acc.sl.Insert(sub)
		if err != nil {
			acc.delRM(sub.subject)
			delete(c.subsWithSID, sid)
			delete(c.subs, sub.subject)
			c.mu.Unlock()
			return
		}

		c.subs[sub.subject] = sub
	}

	srv := c.srv
	c.mu.Unlock()

	kind := c.kind
	if kind == CLIENT {
		srv.updateRouteSubscriptionMap(acc, sub)
	}

	c.SendMsg(msg.MSG_SUBACK, msg.MsgSubAck{
		Code:     msg.RspCode_Success,
		UniqueID: msub.UniqueID,
		Sub:      msub.Sub,
	})
}

func (c *client) processMsgUnSub(_msg *msg.Msg) {
	usub := &msg.MsgUnSub{}
	err := json.Unmarshal(_msg.Data, usub)
	if err != nil {
		nlog.Erro("processMsgUnSub: json.Unmarshal: %v", err)
		return
	}

	c.UnSub(usub.Subs, true)
}

func (c *client) processMsgPub(_msg *msg.Msg) {
	pub := &msg.MsgPub{}
	err := json.Unmarshal(_msg.Data, pub)
	if err != nil {
		nlog.Erro("processMsgPub: json.Unmarshal: %v", err)
		return
	}

	c.msgPub(pub)
}

func (c *client) msgPub(pub *msg.MsgPub) {
	r := c.acc.sl.match(pub.Sub)
	if r == nil {
		nlog.Erro("processMsgPub: match not exist: %v", pub.Sub)
		return
	}

	for _, sub := range r.subs {
		if sub.client.kind == CLIENT {
			sub.client.SendMsg(msg.MSG_PUB, pub)
		} else if sub.client.kind == ROUTER {
			sub.client.SendMsg(msg.MSG_ROUTEPUB, &msg.MsgRoutePub{
				Sub:  pub.Sub,
				Data: pub.Data,
			})
		}
	}
}

func (c *client) registerWithAccount(acc *Account) error {
	nlog.Debug("client id %v registerWithAccount: %v", c.id, acc.name)
	c.acc = acc
	acc.addClient(c)
	return nil
}

func (c *client) sendRemoteNewSub(sub *subscription) {
	nlog.Debug("sendRemoteNewSub: %v", sub.subject)
	c.SendMsg(msg.MSG_REMOTEROUTEADDSUB, &msg.MsgRemoteRouteAddSub{
		Name: c.name,
		Subs: []string{sub.subject},
	})
}

func (c *client) sendRemoteUnSub(subs []string) {
	nlog.Debug("sendRemoteNewSub: %v", subs)
	c.SendMsg(msg.MSG_REMOTEROUTEADDUNSUB, &msg.MsgRemoteRouteAddUnsub{
		Subs: subs,
	})
}

func (c *client) processMsgRemoteRouteAddSub(_msg *msg.Msg) {
	remoteNewSub := &msg.MsgRemoteRouteAddSub{}
	err := json.Unmarshal(_msg.Data, remoteNewSub)
	if err != nil {
		nlog.Erro("processRemoteNewSub: json.Unmarshal: %v", err)
		return
	}

	acc := c.acc
	if acc == nil {
		nlog.Erro("processRemoteNewSub: acc is nil")
		return
	}

	srv := c.srv
	srv.lock.Lock()
	for _, sub := range remoteNewSub.Subs {
		_sub := &subscription{
			client:  c,
			subject: sub,
			queue:   nil,
			qw:      0,
			closed:  0,
		}
		c.subs[sub] = _sub
		acc.addRM(sub, 1)
		err = acc.sl.Insert(_sub)
		if err != nil {
			nlog.Erro("111111")
			continue
		}
	}
	srv.lock.Unlock()
}

func (c *client) processMsgRegisterRouter(_msg *msg.Msg) {
	registerRouter := &msg.MsgRegisterRouter{}
	err := json.Unmarshal(_msg.Data, registerRouter)
	if err != nil {
		nlog.Erro("processMsgRegisterRouter: json.Unmarshal: %v", err)
		return
	}
	c.name = registerRouter.Name
	c.srv.addRouterConfInfo(c, registerRouter)

	all := c.srv.getAllRouteInfos()
	retMsg := &msg.MsgCurAllRoutes{
		RemoteName: c.srv.cfg.Name,
	}
	for _, route := range all {
		if route.ID == registerRouter.Name {
			continue
		}
		retMsg.All = append(retMsg.All, &msg.RouterInfo{
			Name:        route.ID,
			ClientAddr:  route.ListenAddr,
			ClusterAddr: route.ClusterAddr,
		})
	}
	// 同时要回信息
	c.SendMsg(msg.MSG_CURALLROUTES, retMsg)
}

func (c *client) processMsgCurAllRoutes(_msg *msg.Msg) {
	curAllRoutes := &msg.MsgCurAllRoutes{}
	err := json.Unmarshal(_msg.Data, curAllRoutes)
	if err != nil {
		nlog.Erro("processMsgCurAllRoutes: json.Unmarshal: %v", err)
		return
	}
	c.srv.addRemoteName(c, curAllRoutes.RemoteName)
	c.srv.addRouterInfos(curAllRoutes.All)
}

func (c *client) processMsgRemoteRouteUnSub(_msg *msg.Msg) {
	remoteUnSub := &msg.MsgRemoteRouteAddUnsub{}
	err := json.Unmarshal(_msg.Data, remoteUnSub)
	if err != nil {
		nlog.Erro("processRemoteUnSub: json.Unmarshal: %v", err)
		return
	}

	acc := c.acc
	if acc == nil {
		nlog.Erro("processRemoteUnSub: acc is nil")
		return
	}

	for _, sub := range remoteUnSub.Subs {
		acc.delRM(sub)
	}
}
