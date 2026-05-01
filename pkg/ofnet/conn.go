package ofnet

import (
	"gucooing/lolo/pkg/alg"
	"net"

	pb "google.golang.org/protobuf/proto"
)

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

func newConn() *conn {
	return &conn{}
}

type conn struct {
}
