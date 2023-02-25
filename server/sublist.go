package server

import (
	"sync"
	"sync/atomic"

	nlog "github.com/abc463774475/my_tool/n_log"
)

type subscription struct {
	client  *client
	subject string
	queue   []byte
	// qw represents the queue weight.
	qw     int32
	closed int32
}

type sublistResult struct {
	// subscriptions is a slice of subscriptions that match the subject.
	subs []*subscription
}

// node level 构建出一颗字典树,
// 前期暂时用不着，后期扩展的支持wildcard 的时候需要
type node struct {
	next  *level
	psubs map[*subscription]*subscription
	plist []*subscription
}

type level struct {
	nodes map[string]*node
}

func newNode() *node {
	n := &node{
		next:  nil,
		psubs: make(map[*subscription]*subscription),
		plist: nil,
	}
	return n
}

func newLevel() *level {
	l := &level{
		nodes: make(map[string]*node),
	}
	return l
}

type sublist struct {
	sync.RWMutex
	genid uint64
	// 暂时不管这里哦
	// cache map[string]*sublistResult

	// 根节点
	root *level

	count uint32
}

func newSublist() *sublist {
	s := &sublist{
		root:  newLevel(),
		count: 0,
	}

	return s
}

func (s *sublist) Insert(sub *subscription) error {
	s.Lock()
	defer s.Unlock()

	n := s.root.nodes[sub.subject]
	if n == nil {
		n = newNode()
		s.root.nodes[sub.subject] = n

		nlog.Debug("new node %v", sub.subject)
	}

	n.psubs[sub] = sub
	n.plist = append(n.plist, sub)

	s.count++
	atomic.AddUint64(&s.genid, 1)

	return nil
}

func (s *sublist) deleteSubs(subs []string) {
	s.Lock()
	defer s.Unlock()

	for _, sub := range subs {
		n := s.root.nodes[sub]
		if n == nil {
			continue
		}

		delete(s.root.nodes, sub)
		nlog.Erro("sublist.deleteSub: %v", sub)

		s.count--
	}
}

func (s *sublist) match(sub string) *sublistResult {
	s.RLock()
	defer s.RUnlock()

	n := s.root.nodes[sub]
	if n == nil {
		return nil
	}

	r := &sublistResult{}
	r.subs = append(r.subs, n.plist...)
	return r
}
