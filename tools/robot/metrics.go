package main

import (
	"fmt"
	"math/bits"
	"sync/atomic"
	"time"
)

// Metrics 是全局压测指标，全部通过原子操作更新，供多个 bot 协程并发写入。
type Metrics struct {
	sdkLoginOK   int64
	sdkLoginFail int64
	connectOK    int64
	connectFail  int64
	loginOK      int64 // 完整登录成功（VerifyLoginToken + PlayerLogin）
	loginFail    int64
	reqSent      int64
	rspRecv      int64
	rspTimeout   int64
	disconnects  int64
	bytesSent    int64
	bytesRecv    int64

	active int64 // 当前在线 bot 数（由 Pool 维护）

	rttAll *hist // 累计 RTT 直方图
	rttWin *hist // 窗口 RTT 直方图（每次上报后重置）

	// 延迟漂移（由上报协程单独写入，无并发）
	firstWinP95 time.Duration
	lastWinP95  time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{rttAll: newHist(), rttWin: newHist()}
}

func (m *Metrics) observeRTT(d time.Duration) {
	m.rttAll.add(d)
	m.rttWin.add(d)
}

// hist 是一个无锁的指数分桶直方图：桶 i 覆盖 [2^(i-1), 2^i) 微秒。
type hist struct {
	buckets [64]int64
}

func newHist() *hist { return &hist{} }

func (h *hist) add(d time.Duration) {
	us := max(d.Microseconds(), 1)
	i := bits.Len64(uint64(us)) // 1..64
	if i >= len(h.buckets) {
		i = len(h.buckets) - 1
	}
	atomic.AddInt64(&h.buckets[i], 1)
}

func (h *hist) count() int64 {
	var n int64
	for i := range h.buckets {
		n += atomic.LoadInt64(&h.buckets[i])
	}
	return n
}

func (h *hist) percentile(p float64) time.Duration {
	total := h.count()
	if total == 0 {
		return 0
	}
	target := int64(float64(total) * p)
	var cum int64
	for i := range h.buckets {
		cum += atomic.LoadInt64(&h.buckets[i])
		if cum >= target {
			// 取桶上界 2^i 微秒作近似
			return time.Duration(int64(1)<<uint(i)) * time.Microsecond
		}
	}
	return 0
}

func (h *hist) reset() {
	for i := range h.buckets {
		atomic.StoreInt64(&h.buckets[i], 0)
	}
}

// ms 把时长格式化为毫秒；<=0 显示 "-"。
func ms(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
}

func bytesH(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGT"[exp])
}

func pct(ok, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(ok) / float64(total) * 100
}
