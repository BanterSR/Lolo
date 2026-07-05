package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Pool 按 LoadFunc 曲线维持在线 bot 数量：不足则起动、超过则回收、断线则由重建补足。
// 账号 id 复用，使账号集合有界（= 峰值并发）。
type Pool struct {
	cfg  *Config
	m    *Metrics
	load LoadFunc
}

func NewPool(cfg *Config, m *Metrics, load LoadFunc) *Pool {
	return &Pool{cfg: cfg, m: m, load: load}
}

type running struct {
	cancel  context.CancelFunc
	account int
}

// Run 阻塞运行，直到负载曲线结束或 ctx 取消。
func (p *Pool) Run(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()
	bots := make(map[int]*running) // runSeq -> running
	exitCh := make(chan int, 4096)
	poolDone := make(chan struct{})
	var free []int // 可复用账号 id
	nextAccount := 0
	runSeq := 0
	var wg sync.WaitGroup

	launch := func() {
		var account int
		if n := len(free); n > 0 {
			account = free[n-1]
			free = free[:n-1]
		} else {
			account = nextAccount
			nextAccount++
		}
		rs := runSeq
		runSeq++
		bctx, cancel := context.WithCancel(ctx)
		bots[rs] = &running{cancel: cancel, account: account}
		b := newBot(account, p.cfg, p.m)
		wg.Go(func() {
			b.Run(bctx)
			select {
			case exitCh <- rs:
			case <-poolDone:
			}
		})
	}

	remove := func(rs int) {
		if r, ok := bots[rs]; ok {
			r.cancel()
			delete(bots, rs)
			free = append(free, r.account)
		}
	}

	reconcile := func(target int) {
		for len(bots) < target {
			launch()
		}
		for len(bots) > target {
			for rs := range bots {
				remove(rs)
				break
			}
		}
		atomic.StoreInt64(&p.m.active, int64(len(bots)))
	}

	shutdown := func() {
		close(poolDone)
		for _, r := range bots {
			r.cancel()
		}
		wg.Wait()
		atomic.StoreInt64(&p.m.active, 0)
	}

	for {
		select {
		case <-ctx.Done():
			shutdown()
			return
		case rs := <-exitCh:
			remove(rs)
			atomic.StoreInt64(&p.m.active, int64(len(bots)))
		case <-ticker.C:
			target, done := p.load(time.Since(start))
			if done {
				shutdown()
				return
			}
			if target < 0 {
				target = 0
			}
			reconcile(target)
		}
	}
}
