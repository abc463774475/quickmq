package server

import (
	"strconv"
	"sync"
	"time"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/msg"
	"github.com/abc463774475/quickmq/qcmq/utils/snowflake"
)

const (
	globalAccountName = "global"
)

// Account based limits.
type limits struct {
	mpay   int32
	msubs  int32
	mconns int32
}

type Account struct {
	stats
	limits
	srv *server

	name     string
	uniqueID int64
	updated  time.Time
	created  time.Time

	client *Client

	sl *sublist

	rm         map[string]int32
	curAllSubs map[string]*subscription

	rwmu sync.RWMutex
	csmu sync.RWMutex

	clients  map[*Client]struct{}
	clientMu sync.Mutex
}

func NewAccount(name string) *Account {
	a := &Account{}
	a.init()
	a.name = name
	a.uniqueID = snowflake.GetID()
	a.created = time.Now()
	a.updated = a.created
	a.limits = limits{
		mpay:   -1,
		msubs:  -1,
		mconns: -1,
	}
	a.sl = newSublist()
	return a
}

func (a *Account) init() {
	a.clients = make(map[*Client]struct{})
	a.rm = make(map[string]int32)
	a.curAllSubs = make(map[string]*subscription, 10000)
}

func (a *Account) String() string {
	return a.name + strconv.FormatUint(uint64(a.uniqueID), 36)
}

func (a *Account) setCurClient(client2 *Client) {
	a.client = client2
}

func (a *Account) addClient(c *Client) {
	a.clientMu.Lock()
	if _, ok := a.clients[c]; ok {
		nlog.Erro("Account.addClient: client already added %v", c)
		a.clientMu.Unlock()
		return
	}
	a.clients[c] = struct{}{}
	a.clientMu.Unlock()
}

func (a *Account) delClient(c *Client) {
	a.clientMu.Lock()
	if _, ok := a.clients[c]; !ok {
		nlog.Erro("Account.delClient: client not found", c)
		a.clientMu.Unlock()
		return
	}
	delete(a.clients, c)
	a.clientMu.Unlock()
}

func (a *Account) addRM(name string, value int32) {
	a.rwmu.Lock()
	if _, ok := a.rm[name]; ok {
		nlog.Erro("Account.addRM: name already added", name)
		a.rwmu.Unlock()
		return
	}
	a.rm[name] = value
	a.rwmu.Unlock()
}

func (a *Account) addCurSubs(name string, sub *subscription) {
	a.csmu.Lock()
	if _, ok := a.curAllSubs[name]; ok {
		nlog.Erro("Account.addCurSubs: name already added", name)
		a.csmu.Unlock()
		return
	}
	a.curAllSubs[name] = sub
	a.csmu.Unlock()
}

func (a *Account) getAllCurSubs() []string {
	a.csmu.RLock()
	defer a.csmu.RUnlock()

	ret := make([]string, 0, len(a.curAllSubs))
	for name := range a.curAllSubs {
		ret = append(ret, name)
	}
	return ret
}

func (a *Account) delRM(name string) {
	a.rwmu.Lock()
	if _, ok := a.rm[name]; !ok {
		nlog.Erro("Account.delRM: name not found", name)
		a.rwmu.Unlock()
		return
	}
	delete(a.rm, name)
	a.sl.deleteSubs([]string{name})

	a.rwmu.Unlock()
}

func (a *Account) delCurSubs(sub string) bool {
	a.csmu.Lock()
	if _, ok := a.curAllSubs[sub]; !ok {
		nlog.Erro("Account.delCurSubs: subname not found", sub)
		a.csmu.Unlock()
		return false
	}
	delete(a.curAllSubs, sub)
	a.csmu.Unlock()

	return true
}

func (a *Account) getMsgAccounts() msg.Accounts {
	a.rwmu.RLock()
	defer a.rwmu.RUnlock()
	rs := msg.Accounts{
		RM:   make(map[string]int32, len(a.rm)),
		Name: a.name,
	}
	for name, value := range a.rm {
		rs.RM[name] = value
	}
	return rs
}
