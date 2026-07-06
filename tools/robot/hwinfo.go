package main

import (
	"os"
	"runtime"
)

// HardwareInfo 描述运行压测客户端（负载生成机）的软硬件环境，
// 用于解读结果——比如判断客户端 CPU 是否成为瓶颈。
type HardwareInfo struct {
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	NumCPU     int    `json:"numCPU"`
	GOMAXPROCS int    `json:"gomaxprocs"`
	GoVersion  string `json:"goVersion"`
}

func collectHardware() HardwareInfo {
	host, _ := os.Hostname()
	return HardwareInfo{
		Hostname:   host,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		GoVersion:  runtime.Version(),
	}
}

// clientMemMB 返回压测进程自身的堆占用与向 OS 申请的内存（MB）。
func clientMemMB() (heap, sys float64) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapAlloc) / 1024 / 1024, float64(ms.Sys) / 1024 / 1024
}
