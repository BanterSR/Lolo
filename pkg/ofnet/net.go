package ofnet

import (
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gookit/slog"

	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type Net interface {
	Accept() (Conn, error)
	Close() error
	SetBlackPackId(packIdList map[uint32]struct{})
	SetFileLog(log *slog.SugaredLogger)
	Stats() NetStats
	StartStatsLoop()
	StopStatsLoop()
}

type NetStats struct {
	MinuteSendBytes int64   // sent bytes in last minute
	MinuteRecvBytes int64   // received bytes in last minute
	TotalSendBytes  int64   // total sent bytes since startup
	TotalRecvBytes  int64   // total received bytes since startup
	AvgSendRateBps  float64 // average send rate in recent window (bytes/sec)
	AvgRecvRateBps  float64 // average recv rate in recent window (bytes/sec)
	ConnNum         int64   // current connections
	QPS             float64 // average request qps in current window
}

func (s NetStats) String() string {
	return fmt.Sprintf("Conn:%d QPS:%.2f In1m:%s Out1m:%s InRate:%s OutRate:%s InTotal:%s OutTotal:%s",
		s.ConnNum,
		s.QPS,
		formatBytes(s.MinuteRecvBytes),
		formatBytes(s.MinuteSendBytes),
		formatRate(s.AvgRecvRateBps),
		formatRate(s.AvgSendRateBps),
		formatBytes(s.TotalRecvBytes),
		formatBytes(s.TotalSendBytes),
	)
}

func formatBytes(n int64) string {
	return formatSize(float64(n), false)
}

func formatRate(v float64) string {
	return formatSize(v, true)
}

func formatSize(value float64, withPerSecond bool) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	if value < 0 {
		value = 0
	}
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}

	suffix := units[idx]
	if withPerSecond {
		suffix += "/s"
	}
	if idx == 0 {
		return fmt.Sprintf("%d%s", int64(math.Round(value)), suffix)
	}
	return fmt.Sprintf("%.2f%s", value, suffix)
}

func NewNet(network, addr string, log *slog.SugaredLogger) (Net, error) {
	log.Infof("协议:%s,启动在%s 上", network, addr)
	switch network {
	case "tcp":
		return newTcpNet(addr, log)
	}
	return nil, errors.New("network not support")
}

type netBase struct {
	log         *slog.SugaredLogger
	fileLog     *slog.SugaredLogger
	blackPackId map[uint32]struct{}
	connNum     int64 // current connections
	maxConnNum  int64 // max connections
	stat        netStats
	statsTag    string
	statsDone   chan struct{}
	statsStop   sync.Once
}

type netStats struct {
	startUnix int64
	sendTotal int64
	recvTotal int64
	mu        sync.Mutex
	bucket    [60]minuteBucket
}

type minuteBucket struct {
	sec      int64
	send     int64
	recv     int64
	requests int64
}

func (c *netBase) SetFileLog(log *slog.SugaredLogger) {
	c.fileLog = log
}

func (c *netBase) SetBlackPackId(packIdList map[uint32]struct{}) {
	c.blackPackId = packIdList
}

func (c *netBase) logPack(packId uint32) bool {
	_, ok := c.blackPackId[packId]
	return !ok
}

func (c *netBase) SetMaxConnNum(maxConnNum int64) {
	c.maxConnNum = maxConnNum
}

func (c *netBase) GetConnNum() int64 {
	return atomic.LoadInt64(&c.connNum)
}

func (c *netBase) onConnOpen() {
	atomic.AddInt64(&c.connNum, 1)
}

func (c *netBase) onConnClose() {
	atomic.AddInt64(&c.connNum, -1)
}

func (c *netBase) recordSendBytes(n int) {
	if n <= 0 {
		return
	}
	size := int64(n)
	atomic.AddInt64(&c.stat.sendTotal, size)
	c.recordMinute(0, size, 0)
}

func (c *netBase) recordRecvBytes(n int) {
	if n <= 0 {
		return
	}
	size := int64(n)
	atomic.AddInt64(&c.stat.recvTotal, size)
	c.recordMinute(size, 0, 0)
}

func (c *netBase) recordRequest() {
	c.recordMinute(0, 0, 1)
}

func (c *netBase) recordMinute(recv, send, requests int64) {
	now := time.Now().Unix()
	idx := int(now % int64(len(c.stat.bucket)))

	c.stat.mu.Lock()
	b := &c.stat.bucket[idx]
	if b.sec != now {
		*b = minuteBucket{sec: now}
	}
	b.recv += recv
	b.send += send
	b.requests += requests
	c.stat.mu.Unlock()
}

func (c *netBase) GetStats() NetStats {
	now := time.Now().Unix()
	connNum := atomic.LoadInt64(&c.connNum)

	var minuteSend, minuteRecv, minuteReq int64
	c.stat.mu.Lock()
	for i := range c.stat.bucket {
		b := c.stat.bucket[i]
		if b.sec == 0 || now-b.sec >= int64(len(c.stat.bucket)) {
			continue
		}
		minuteSend += b.send
		minuteRecv += b.recv
		minuteReq += b.requests
	}
	c.stat.mu.Unlock()

	startUnix := atomic.LoadInt64(&c.stat.startUnix)
	windowSec := int64(len(c.stat.bucket))
	if startUnix > 0 {
		elapsed := now - startUnix + 1
		if elapsed > 0 && elapsed < windowSec {
			windowSec = elapsed
		}
	}
	if windowSec <= 0 {
		windowSec = 1
	}

	qps := float64(minuteReq) / float64(windowSec)
	avgSendRate := float64(minuteSend) / float64(windowSec)
	avgRecvRate := float64(minuteRecv) / float64(windowSec)

	return NetStats{
		MinuteSendBytes: minuteSend,
		MinuteRecvBytes: minuteRecv,
		TotalSendBytes:  atomic.LoadInt64(&c.stat.sendTotal),
		TotalRecvBytes:  atomic.LoadInt64(&c.stat.recvTotal),
		AvgSendRateBps:  avgSendRate,
		AvgRecvRateBps:  avgRecvRate,
		ConnNum:         connNum,
		QPS:             qps,
	}
}

func (c *netBase) Stats() NetStats {
	return c.GetStats()
}

func (c *netBase) StartStatsLoop() {
	if c == nil || c.statsDone != nil {
		return
	}
	c.statsDone = make(chan struct{})
	go c.netStatsLoop()
}

func (c *netBase) StopStatsLoop() {
	if c == nil || c.statsDone == nil {
		return
	}
	c.statsStop.Do(func() {
		close(c.statsDone)
	})
}

func (c *netBase) netStatsLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.statsDone:
			return
		case <-ticker.C:
			if c.log == nil {
				continue
			}
			c.log.Infof("[NetStat] %s", c.Stats())
		}
	}
}

type Conn interface {
	Read() (*alg.GameMsg, error)
	Send(packetId uint32, protoObj pb.Message)
	SetUID(uint32)
	GetSeqId() uint32
	SetServerTag(serverTag string)
	Close()
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

const (
	ClientMsg = iota
	ServerMsg
)

func (c *netBase) logMag(tp int, serverTag string, uid uint32, head *proto.PacketHead, payloadMsg pb.Message) {
	var s string
	switch tp {
	case ClientMsg:
		s = "c -> s"
	case ServerMsg:
		s = "s -> c"
	}
	if _, ok := c.blackPackId[head.MsgId]; ok {
		return
	}
	if c.fileLog == nil {
		return
	}
	c.fileLog.Debugf("%s[Server:%s][UID:%v][PacketId:%v][CMD:%s]Pack:%s",
		s,
		serverTag,
		uid,
		head.PacketId,
		cmd.Get().GetCmdNameByCmdId(head.MsgId),
		protojson.Format(payloadMsg),
	)
}
