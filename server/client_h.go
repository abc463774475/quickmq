package server

import nlog "github.com/abc463774475/my_tool/n_log"

func (c *client) del() {
	nlog.Info("del client prepare")
	c.acc.delClient(c)

	if c.kind == CLIENT {
		subs := []string{}
		for k := range c.subs {
			subs = append(subs, k)
		}

		c.UnSub(subs, true)
	} else if c.kind == ROUTER {
		// workflow:
		// 1. unsubscribe from all subscribers
		// 2. unsubscribe from all routes
		// 3. client lib redirect, reconnect TODO: 这里需要更新路由的订阅关系，主要工作集成在 client lib 中。需要保存自己所有的订阅关系
		subs := []string{}
		for k := range c.subs {
			subs = append(subs, k)
		}

		// 本机unsub，不通知其他router
		c.UnSub(subs, false)
		c.srv.removeRoute(c)
	}
}

func (c *client) UnSub(subs []string, isNotifyRoute bool) {
	acc := c.acc
	for _, sub := range subs {
		c.mu.Lock()
		if _, ok := c.subs[sub]; !ok {
			nlog.Erro("processMsgUnSub: sub %v not found", sub)
			c.mu.Unlock()
			continue
		}

		delete(c.subs, sub)
		if isNotifyRoute {
			bret := acc.delCurSubs(sub)
			if !bret {
				c.mu.Unlock()
				return
			}
		}

		acc.delRM(sub)
		c.in.subs--
		c.mu.Unlock()
	}

	srv := c.srv
	if isNotifyRoute {
		srv.updateRouteUnSubscriptionMap(subs)
	}
}
