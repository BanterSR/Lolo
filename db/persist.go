package db

import (
	"log"
	"sync"
)

// persistJob 一次持久化操作。key 用于分片:同一 key 串行(保证写序),不同 key 并行(吃多核)。
type persistJob struct {
	key uint32
	fn  func() error
}

type persister struct {
	shards []chan persistJob
	wg     sync.WaitGroup
}

var persist *persister

// StartPersist 启动 shardNum 条落库协程。
// 同一 userId 恒定落到同一分片→串行有序;不同 userId 分散→并行吃多核(gzip)与并发 DB 连接。
// 由 NewDB 调用,shardNum/bufSize 均来自配置。
func StartPersist(shardNum, bufSize int) {
	if shardNum < 1 {
		shardNum = 1
	}
	if bufSize < 1 {
		bufSize = 1
	}
	p := &persister{shards: make([]chan persistJob, shardNum)}
	for i := range p.shards {
		ch := make(chan persistJob, bufSize)
		p.shards[i] = ch
		p.wg.Add(1)
		go p.run(ch)
	}
	persist = p
}

func (p *persister) run(ch chan persistJob) {
	defer p.wg.Done()
	for job := range ch { // 通道关闭后自动排空剩余任务再退出
		if err := job.fn(); err != nil {
			log.Printf("持久化失败 key:%d err:%v", job.key, err)
		}
	}
}

// Persist 把一次针对某玩家的写操作投递到其分片异步执行(同玩家保证顺序)。
// fn 仅应包含纯 DB 写(不要在里面碰游戏内存态)。未启动时降级为同步写,保证不丢。
func Persist(key uint32, fn func() error) {
	p := persist
	if p == nil {
		_ = fn()
		return
	}
	p.shards[key%uint32(len(p.shards))] <- persistJob{key: key, fn: fn}
}

// StopPersist 关闭并排空所有分片(关服调用,确保落盘不丢)。调用前应确保生产者已停止投递。
func StopPersist() {
	p := persist
	if p == nil {
		return
	}
	persist = nil
	for _, ch := range p.shards {
		close(ch)
	}
	p.wg.Wait()
}
