package main

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ServerStat 周期性抓取服务端 pprof，读取 goroutine 数与堆内存，
// 用于把服务端资源表现与客户端施加的负载对齐观察。
// 依赖 lolo 在 dev 模式下注册的 /debug/pprof/*。
type ServerStat struct {
	base   string
	client *http.Client

	mu         sync.Mutex
	ok         bool
	goroutines int
	heapAlloc  uint64
	sys        uint64
	numGC      uint32
}

func newServerStat(base string) *ServerStat {
	return &ServerStat{
		base:   strings.TrimRight(base, "/"),
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (s *ServerStat) get(path string) (string, bool) {
	resp, err := s.client.Get(s.base + path)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// scrape 抓取一次，更新内部快照。失败时标记 ok=false。
func (s *ServerStat) scrape() {
	goroutines, gok := s.scrapeGoroutines()
	heap, sys, numGC, hok := s.scrapeHeap()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ok = gok || hok
	if gok {
		s.goroutines = goroutines
	}
	if hok {
		s.heapAlloc = heap
		s.sys = sys
		s.numGC = numGC
	}
}

func (s *ServerStat) scrapeGoroutines() (int, bool) {
	body, ok := s.get("/debug/pprof/goroutine?debug=1")
	if !ok {
		return 0, false
	}
	// 首行形如: "goroutine profile: total 1234"
	line, _, _ := strings.Cut(body, "\n")
	if i := strings.LastIndex(line, "total "); i >= 0 {
		if n, err := strconv.Atoi(strings.TrimSpace(line[i+len("total "):])); err == nil {
			return n, true
		}
	}
	return 0, false
}

func (s *ServerStat) scrapeHeap() (heap, sys uint64, numGC uint32, ok bool) {
	body, got := s.get("/debug/pprof/heap?debug=1")
	if !got {
		return 0, 0, 0, false
	}
	// 尾部含 runtime.MemStats，逐行形如: "# HeapAlloc = 12345"
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		k, v, found := strings.Cut(line, " = ")
		if !found || !strings.HasPrefix(k, "# ") {
			continue
		}
		switch strings.TrimPrefix(k, "# ") {
		case "HeapAlloc":
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				heap = n
				ok = true
			}
		case "Sys":
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				sys = n
			}
		case "NumGC":
			if n, err := strconv.ParseUint(v, 10, 32); err == nil {
				numGC = uint32(n)
			}
		}
	}
	return heap, sys, numGC, ok
}

// snapshot 返回最近一次抓取结果（堆/申请以 MB 为单位）。
func (s *ServerStat) snapshot() (ok bool, goroutines int, heapMB, sysMB float64, numGC uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ok, s.goroutines, float64(s.heapAlloc) / 1024 / 1024, float64(s.sys) / 1024 / 1024, s.numGC
}
