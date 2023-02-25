package server

import (
	"net"
	"sync"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/abc463774475/quickmq/qcmq/msg"
	"github.com/abc463774475/quickmq/qcmq/utils/snowflake"
)

type server struct {
	cfg config

	listener net.Listener

	stats

	running  bool
	shutdown bool

	accounts sync.Map
	gacc     *Account

	lock sync.RWMutex

	rwmClients sync.RWMutex
	clients    map[int64]*client
	routes     map[int64]*client
	remotes    map[string]*client

	totalClients uint64

	rwmRouter         sync.RWMutex
	allRouterConfInfo map[string]*RouterConfInfo

	// LameDuck mode
	// 后端服务正在监听端口，并且可以服务请求，但是已经明确要求客户端停止发送请求。
	// 当某个请求进入跛脚鸭状态时，它会将这个状态广播给所有已经连接的客户端。
	ldm   bool
	ldmCh chan bool

	shutdownComplete chan struct{}
}

func newServer(options ...Option) *server {
	s := &server{}
	for _, opt := range options {
		opt.apply(&s.cfg)
	}

	s.clients = make(map[int64]*client)
	s.routes = make(map[int64]*client)
	s.ldmCh = make(chan bool, 1)
	s.shutdownComplete = make(chan struct{})
	s.remotes = make(map[string]*client)
	s.allRouterConfInfo = make(map[string]*RouterConfInfo)

	s.gacc = NewAccount(globalAccountName)
	s.registerAccount(s.gacc)

	return s
}

func (s *server) startClientListener() {
	s.listener, _ = net.Listen("tcp", s.cfg.Addr)
	s.running = true
	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			nlog.Erro("err %v", err)
			continue
		}
		s.acceptOneConnection(conn, CLIENT)
	}
}

func (s *server) acceptOneConnection(conn net.Conn, kind ClientType) {
	// nlog.Info("acceptOneConnection %v", conn.RemoteAddr())
	id := snowflake.GetID()
	c := newAcceptClient(id, conn, s)
	c.kind = kind
	c.init()

	_ = c.registerWithAccount(s.globalAccount())

	s.rwmClients.Lock()

	go c.run()
	s.clients[id] = c
	s.rwmClients.Unlock()

	//if kind == ROUTER {
	//	// s.addRoute(c, c.name)
	//}
}

func (s *server) start() {
	nlog.Info("start server  %v %v", s.cfg.Addr, s.cfg.ClusterAddr)
	go s.startClientListener()

	if s.cfg.ClusterAddr != "" {
		go s.startRouterListener()
	}

	if s.cfg.ConnectRouterAddr != "" {
		s.connectToRoute("", s.cfg.ConnectRouterAddr)
	}

	s.WaitForShutdown()
}

func (s *server) WaitForShutdown() {
	<-s.shutdownComplete
}

func (s *server) globalAccount() *Account {
	s.lock.RLock()
	defer s.lock.RUnlock()
	rs := s.gacc
	return rs
}

func (s *server) addRoute(c *client, name string) bool {
	s.lock.Lock()
	if _, ok := s.routes[c.id]; ok {
		s.lock.Unlock()
		nlog.Erro("addRoute: client %v already in routes", c.id)
		return false
	}
	if _, ok := s.remotes[name]; ok {
		s.lock.Unlock()
		nlog.Erro("addRoute: name %v already in remotes", name)
		return false
	}
	s.routes[c.id] = c
	s.remotes[name] = c
	s.lock.Unlock()

	s.sendSubsToRoute(c)

	s.forwardNewRouteInfoToKnownServers(c)
	return true
}

func (s *server) sendSubsToRoute(route *client) {
	all := s.getAllAccountInfo()
	route.SendMsg(msg.MSG_SNAPSHOTSUBS, &msg.MsgSnapshotSubs{All: all})

	route.SendMsg(msg.MSG_REMOTEROUTEADDSUB, &msg.MsgRemoteRouteAddSub{
		Name: s.globalAccount().name,
		Subs: s.globalAccount().getAllCurSubs(),
	})
}

// 获取所有Account的信息
func (s *server) getAllAccountInfo() []*msg.Accounts {
	s.lock.RLock()
	defer s.lock.RUnlock()

	accs := make([]*msg.Accounts, 0, 32)
	s.accounts.Range(func(key, value interface{}) bool {
		acc := value.(*Account)
		acc.rwmu.RLock()
		itemp := acc.getMsgAccounts()
		accs = append(accs, &itemp)
		acc.rwmu.RUnlock()
		return true
	})
	return accs
}

// 告知其他路由，有新的路由加入
func (s *server) forwardNewRouteInfoToKnownServers(route2 *client) {
	s.lock.Lock()
	for _, route := range s.routes {
		if route.id == route2.id {
			continue
		}
		route.SendMsg(msg.MSG_NEWROUTE, &msg.MsgNewRoute{Name: route2.name})
	}
	s.lock.Unlock()
}

func (s *server) removeRoute(c *client) {
	s.lock.Lock()
	if _, ok := s.routes[c.id]; !ok {
		s.lock.Unlock()
		nlog.Erro("removeRoute: client %v not in routes", c.id)
		return
	}

	if _, ok := s.remotes[c.name]; !ok {
		s.lock.Unlock()
		nlog.Erro("removeRoute: name %v not in remotes", c.name)
		return
	}

	delete(s.routes, c.id)
	delete(s.remotes, c.name)
	s.lock.Unlock()

	nlog.Erro("removeRoute: %v", c.name)
}

func (s *server) snapshotSubs(c *client, snapShot *msg.MsgSnapshotSubs) {
	s.lock.Lock()
	defer s.lock.Unlock()

	//for _, v := range snapShot.All {
	//	atemp, ok := s.accounts.Load(v.Name)
	//	if !ok {
	//		nlog.Erro("snapshotSubs: account %v not found", v.Name)
	//		continue
	//	}
	//
	//	nlog.Erro("snapshotSubs: account %+v", v)
	//	acc := atemp.(*Account)
	//	for k1, v1 := range v.RM {
	//		acc.rm[k1] = v1
	//	}
	//}
}

func (s *server) registerAccount(account *Account) {
	s.accounts.Store(account.name, account)
}

func (s *server) updateRouteSubscriptionMap(acc *Account, sub *subscription) {
	nlog.Debug("updateRouteSubscriptionMap: %v %v", acc.name, sub.subject)
	for _, route := range s.routes {
		route.sendRemoteNewSub(sub)
	}
}

func (s *server) updateRouteUnSubscriptionMap(subs []string) {
	for _, route := range s.routes {
		route.sendRemoteUnSub(subs)
	}
}

func (s *server) startRouterListener() {
	l, err := net.Listen("tcp", s.cfg.ClusterAddr)
	if err != nil {
		nlog.Erro("startRouterListener: %v", err)
		return
	}

	nlog.Info("startRouterListener: %v", s.cfg.ClusterAddr)

	for s.running {
		conn, err := l.Accept()
		if err != nil {
			nlog.Erro("err  %v", err)
			continue
		}
		s.acceptOneConnection(conn, ROUTER)
	}
}

func (s *server) addRemoteName(c *client, rname string) {
	nlog.Debug("addRemoteName: %v %v", c.id, rname)
	s.lock.Lock()
	if _, ok := s.routes[c.id]; !ok {
		nlog.Erro("addRemoteName: client %v not in routes", c.id)
		s.lock.Unlock()
		return
	}
	s.remotes[rname] = c
	s.lock.Unlock()
}

func (s *server) connectToRoute(name string, addr string) {
	c := newConnectClient(addr)
	if !c.connect() {
		nlog.Erro("connect error")
		return
	}
	c.init()
	c.srv = s
	c.acc = s.globalAccount()
	c.kind = ROUTER

	_msg := &msg.MsgRegisterRouter{
		RouterInfo: msg.RouterInfo{
			Name:        s.cfg.Name,
			ClientAddr:  s.cfg.Addr,
			ClusterAddr: s.cfg.ClusterAddr,
		},
	}

	c.SendMsg(msg.MSG_REGISTERROUTER, _msg)

	go c.run()

	// 把client 加入 server
	s.lock.Lock()
	s.routes[c.id] = c
	if name != "" {
		s.remotes[name] = c
	}
	s.lock.Unlock()
}

func (s *server) addRouterConfInfo(c *client, msg *msg.MsgRegisterRouter) {
	s.rwmRouter.Lock()
	defer s.rwmRouter.Unlock()
	if _, ok := s.allRouterConfInfo[msg.Name]; ok {
		nlog.Debug("addRouterConfInfo: router %v already in allRouterConfInfo", msg.Name)
		// return
	}

	s.allRouterConfInfo[msg.Name] = &RouterConfInfo{
		ID:          msg.Name,
		ListenAddr:  msg.ClientAddr,
		ClusterAddr: msg.ClusterAddr,
	}

	s.addRoute(c, msg.Name)

	nlog.Debug("addRouterInfo: %+v", msg)
}

func (s *server) getAllRouteInfos() []*RouterConfInfo {
	s.rwmRouter.RLock()
	defer s.rwmRouter.RUnlock()
	ret := make([]*RouterConfInfo, 0, len(s.allRouterConfInfo))
	for _, v := range s.allRouterConfInfo {
		tmp := *v
		ret = append(ret, &tmp)
	}
	return ret
}

func (s *server) addRouterInfos(all []*msg.RouterInfo) {
	for _, v := range all {
		s.connectToRoute(v.Name, v.ClusterAddr)
	}
}
