package ofnet

import (
	"errors"
	"github.com/gookit/slog"
	"net"
	"sync/atomic"

	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/cmd"
)

var CLIENT_CONN_NUM int64 = 0 // 当前客户端连接数
var TPS int64

type Net interface {
	Accept() (Conn, error)
	Close() error
	SetBlackPackId(packIdList map[uint32]struct{})
	SetLogMsg(logMsg bool)
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
	logMsg      bool
	blackPackId map[uint32]struct{}
	connNum     int64 // 连接数
	maxConnNum  int64 // 最大连接数
}

func (c *netBase) SetBlackPackId(packIdList map[uint32]struct{}) {
	c.blackPackId = packIdList
}

func (c *netBase) SetLogMsg(logMsg bool) {
	c.logMsg = logMsg
}

func (c *netBase) logPack(packId uint32) bool {
	if !c.logMsg {
		return false
	}
	_, ok := c.blackPackId[packId]
	return !ok
}

func (c *netBase) SetMaxConnNum(maxConnNum int64) {
	c.maxConnNum = maxConnNum
}

func (c *netBase) GetConnNum() int64 {
	return atomic.LoadInt64(&c.connNum)
}

type Conn interface {
	Read() (*alg.GameMsg, error)
	Send(cmdId, packetId uint32, protoObj pb.Message)
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

func (c *netBase) logMag(tp int, serverTag string, isLog bool, uid, cmdId uint32, payloadMsg pb.Message) {
	if !isLog {
		return
	}
	var s string
	switch tp {
	case ClientMsg:
		s = "c -> s"
	case ServerMsg:
		s = "s -> c"
	}
	c.log.Debugf("%s[Server:%s][UID:%v][CMD:%s]Pack:%s",
		s,
		serverTag,
		uid,
		cmd.Get().GetCmdNameByCmdId(cmdId),
		protojson.Format(payloadMsg),
	)
}
