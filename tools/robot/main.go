package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"gucooing/lolo/protocol/cmd"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置错误: %v\n", err)
		os.Exit(2)
	}
	load, err := buildLoad(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "构建负载曲线失败: %v\n", err)
		os.Exit(2)
	}

	_ = cmd.Get() // 预热 cmd 反射注册表

	hw := collectHardware()
	pprofBase := cfg.Pprof
	if pprofBase == "" {
		pprofBase = cfg.Sdk
	}
	m := NewMetrics()
	srv := newServerStat(pprofBase)
	hist := newHistory(600, hw, cfg, pprofBase+"/debug/pprof/")
	pool := NewPool(cfg, m, load)

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\n收到中断信号，正在优雅停止…")
		cancel()
	}()

	printBanner(cfg, hw, pprofBase)

	var web *http.Server
	if cfg.Web != "" {
		var werr error
		web, werr = startWeb(cfg.Web, hist)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "⚠ Web 仪表盘启动失败（%s）: %v —— 压测继续，仅无仪表盘\n\n", cfg.Web, werr)
		} else {
			fmt.Printf("Web 仪表盘: http://%s/\n\n", webDisplayAddr(cfg.Web))
		}
	}

	go srvScrapeLoop(ctx, srv, 3*time.Second)

	repDone := make(chan struct{})
	go reporter(ctx, m, cfg.Report.D(), load, srv, hist, repDone)

	start := time.Now()
	pool.Run(ctx)
	cancel()
	<-repDone
	if web != nil {
		shCtx, shCancel := context.WithTimeout(context.Background(), time.Second)
		_ = web.Shutdown(shCtx)
		shCancel()
	}

	printSummary(m, cfg, hw, srv, time.Since(start))
}

// srvScrapeLoop 以固定节奏异步抓取服务端 pprof，避免每个上报周期都命中较重的
// heap profile，也避免抓取延迟拖慢实况打印。
func srvScrapeLoop(ctx context.Context, srv *ServerStat, interval time.Duration) {
	srv.scrape()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			srv.scrape()
		}
	}
}

// reporter 每 interval 打印一行实况，并推入历史供 web 仪表盘使用。
func reporter(ctx context.Context, m *Metrics, interval time.Duration, load LoadFunc, srv *ServerStat, hist *History, done chan<- struct{}) {
	defer close(done)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start := time.Now()
	var prevLogin int64
	first := true
	secs := interval.Seconds()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			winCount := m.rttWin.count()
			p50 := m.rttWin.percentile(0.50)
			p95 := m.rttWin.percentile(0.95)
			p99 := m.rttWin.percentile(0.99)
			m.rttWin.reset()

			login := atomic.LoadInt64(&m.loginOK)
			dLogin := login - prevLogin
			prevLogin = login

			elapsed := time.Since(start)
			target, _ := load(elapsed)
			if winCount > 0 {
				if first {
					m.firstWinP95 = p95
					first = false
				}
				m.lastWinP95 = p95
			}

			sOK, sGr, sHeap, _, sGC := srv.snapshot()

			fmt.Printf("[%6s] active=%-4d/%-4d login=%d(+%d %.0f/s) fail(sdk=%d conn=%d login=%d) rps=%.0f rtt(p50=%s p95=%s p99=%s) disc=%d to=%d 服务端(gr=%d heap=%.0fMB)\n",
				elapsed.Truncate(time.Second), atomic.LoadInt64(&m.active), int64(target),
				login, dLogin, float64(dLogin)/secs,
				atomic.LoadInt64(&m.sdkLoginFail), atomic.LoadInt64(&m.connectFail), atomic.LoadInt64(&m.loginFail),
				float64(winCount)/secs, ms(p50), ms(p95), ms(p99),
				atomic.LoadInt64(&m.disconnects), atomic.LoadInt64(&m.rspTimeout), sGr, sHeap)

			hist.add(sample{
				T:            time.Now().UnixMilli(),
				Elapsed:      elapsed.Seconds(),
				Active:       atomic.LoadInt64(&m.active),
				Target:       target,
				RPS:          float64(winCount) / secs,
				LoginRate:    float64(dLogin) / secs,
				P50:          float64(p50.Microseconds()) / 1000,
				P95:          float64(p95.Microseconds()) / 1000,
				P99:          float64(p99.Microseconds()) / 1000,
				LoginOK:      login,
				LoginFail:    atomic.LoadInt64(&m.loginFail),
				Disconnects:  atomic.LoadInt64(&m.disconnects),
				Timeouts:     atomic.LoadInt64(&m.rspTimeout),
				SrvGoroutine: sGr,
				SrvHeapMB:    sHeap,
				SrvNumGC:     sGC,
				SrvOK:        sOK,
			})
		}
	}
}

func printBanner(cfg *Config, hw HardwareInfo, pprofBase string) {
	fmt.Println("================ Lolo 协议机器人压测 ================")
	fmt.Printf("模式/场景 : %s / %s\n", cfg.Mode, cfg.Scenario)
	fmt.Printf("网关      : %s\n", cfg.Gate)
	fmt.Printf("SDK       : %s\n", cfg.Sdk)
	fmt.Printf("pprof     : %s/debug/pprof/\n", pprofBase)
	switch cfg.Mode {
	case "profile":
		if len(cfg.Stages) > 0 {
			fmt.Printf("负载曲线  : %d 个阶段%s\n", len(cfg.Stages), loopStr(cfg.Loop))
		} else {
			fmt.Printf("负载曲线  : pattern=%s base=%d peak=%d%s\n", cfg.Pattern, cfg.Base, orInt(cfg.Peak, cfg.CCU), loopStr(cfg.Loop))
		}
	default:
		fmt.Printf("负载曲线  : ccu=%d ramp=%s duration=%s\n", cfg.CCU, cfg.Ramp, cfg.Duration)
	}
	fmt.Printf("客户端    : %s/%s %d核 (GOMAXPROCS=%d) %s %s\n", hw.OS, hw.Arch, hw.NumCPU, hw.GOMAXPROCS, hw.Hostname, hw.GoVersion)
	fmt.Println("====================================================")
}

func printSummary(m *Metrics, cfg *Config, hw HardwareInfo, srv *ServerStat, elapsed time.Duration) {
	heap, sysMem := clientMemMB()
	sOK, sGr, sHeap, sSys, sGC := srv.snapshot()

	fmt.Println("\n==================== 压测结果汇总 ====================")
	fmt.Printf("模式/场景 : %s / %s\n", cfg.Mode, cfg.Scenario)
	fmt.Printf("总时长    : %s\n", elapsed.Truncate(time.Second))
	fmt.Printf("SDK 登录  : ok=%d fail=%d\n", m.sdkLoginOK, m.sdkLoginFail)
	fmt.Printf("网关连接  : ok=%d fail=%d\n", m.connectOK, m.connectFail)
	fmt.Printf("完整登录  : ok=%d fail=%d (成功率 %.1f%%)\n", m.loginOK, m.loginFail, pct(m.loginOK, m.loginOK+m.loginFail))
	fmt.Printf("请求/响应 : sent=%d recv=%d timeout=%d\n", m.reqSent, m.rspRecv, m.rspTimeout)
	fmt.Printf("断线      : %d\n", m.disconnects)
	fmt.Printf("流量      : 发送=%s 接收=%s\n", bytesH(m.bytesSent), bytesH(m.bytesRecv))
	fmt.Printf("RTT(累计) : p50=%s p95=%s p99=%s (样本 %d)\n",
		ms(m.rttAll.percentile(0.50)), ms(m.rttAll.percentile(0.95)), ms(m.rttAll.percentile(0.99)), m.rttAll.count())
	if m.firstWinP95 > 0 {
		fmt.Printf("延迟漂移  : 首窗 p95=%s -> 末窗 p95=%s\n", ms(m.firstWinP95), ms(m.lastWinP95))
	}
	fmt.Println("---- 硬件/资源 ----")
	fmt.Printf("客户端    : %s/%s %d核 (GOMAXPROCS=%d) %s %s\n", hw.OS, hw.Arch, hw.NumCPU, hw.GOMAXPROCS, hw.Hostname, hw.GoVersion)
	fmt.Printf("客户端内存: 堆=%.1fMB 申请=%.1fMB\n", heap, sysMem)
	if sOK {
		fmt.Printf("服务端    : goroutine=%d 堆=%.1fMB 申请=%.1fMB NumGC=%d (via pprof)\n", sGr, sHeap, sSys, sGC)
	} else {
		fmt.Printf("服务端    : pprof 不可达（需 lolo 处于 dev 模式）\n")
	}
	fmt.Println("=====================================================")
}

func loopStr(loop bool) string {
	if loop {
		return " (循环)"
	}
	return ""
}

func orInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func webDisplayAddr(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return "localhost" + addr
	}
	return addr
}
