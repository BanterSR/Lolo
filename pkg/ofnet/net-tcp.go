package ofnet

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/golang/snappy"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type tcpNet struct {
	*netBase
	listener  net.Listener
	closeOnce sync.Once
}

func newTcpNet(addr string, base *netBase) (*tcpNet, error) {
	x := &tcpNet{
		netBase: base,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	x.listener = listener
	x.statsTag = listener.Addr().String()
	return x, nil
}

func (x *tcpNet) Accept() (Conn, error) {
	if x == nil {
		return nil, errors.New("tcpNet is nil")
	}
	conn, err := x.listener.Accept()
	if err != nil {
		return nil, err
	}
	tconn := &tcpConn{
		net:  x,
		conn: conn,
		buf:  bufio.NewReaderSize(conn, alg.PacketMaxLen),
	}
	x.onConnOpen()

	return tconn, nil
}

func (x *tcpNet) Close() error {
	if x == nil {
		return nil
	}
	var err error
	x.closeOnce.Do(func() {
		x.StopStatsLoop()
		err = x.listener.Close()
	})
	return err
}

type tcpConn struct {
	base      *conn
	net       *tcpNet
	conn      net.Conn
	buf       *bufio.Reader
	uid       uint32
	seqId     uint32
	serverTag string
	closed    int32
}

func (x *tcpConn) GetSeqId() uint32 {
	return x.seqId
}

func (x *tcpConn) Read() (*alg.GameMsg, error) {
	for {
		// head
		headLenByte := make([]byte, alg.TcpHeadSize)
		_, err := io.ReadFull(x.buf, headLenByte)
		if err != nil {
			return nil, err
		}
		headLen := binary.BigEndian.Uint16(headLenByte)

		headByte := make([]byte, headLen)
		_, err = io.ReadFull(x.buf, headByte)
		if err != nil {
			return nil, err
		}
		head := new(proto.PacketHead)
		err = pb.Unmarshal(headByte, head)
		if err != nil {
			x.net.log.Errorf("Could not parse PacketHead proto Error:%s\n", err)
			return nil, err
		}

		// body
		bodyByte := make([]byte, head.BodyLen)
		_, err = io.ReadFull(x.buf, bodyByte)
		if err != nil {
			return nil, err
		}
		x.net.recordRecvBytes(alg.TcpHeadSize + int(headLen) + int(head.BodyLen))
		bodyByte = alg.HandleFlag(head.Flag, bodyByte)
		protoObj := cmd.Get().GetProtoObjByCmdId(head.MsgId)
		if protoObj == nil {
			x.net.log.Errorf("protoObj by cmdId:%d\n", head.MsgId)
			continue
		}
		err = pb.Unmarshal(bodyByte, protoObj)
		if err != nil {
			x.net.log.Errorf("unmarshal proto data err: %v\n", err)
			return nil, err
		}
		x.net.logMag(ClientMsg,
			x.serverTag,
			x.uid,
			head,
			protoObj)
		gameMsg := &alg.GameMsg{
			PacketHead: head,
			Body:       protoObj,
		}
		x.net.recordRequest()

		return gameMsg, nil
	}
}

func (x *tcpConn) Send(packetId uint32, protoObj pb.Message) {
	if x == nil {
		return
	}

	cmdId := cmd.Get().GetCmdIdByProtoObj(protoObj)

	bodyByte, err := pb.Marshal(protoObj)
	if err != nil {
		log.Gate.Errorf("marshal proto data err: %v\n", err)
		return
	}

	head := &proto.PacketHead{
		MsgId:    cmdId,
		Flag:     0,
		BodyLen:  0,
		SeqId:    x.seqId,
		PacketId: packetId,

		TotalPackCount: 0,
	}
	x.seqId++

	if len(bodyByte) > alg.SnappySize {
		bodyByte = snappy.Encode(nil, bodyByte)
		head.Flag = 1
	}
	x.net.logMag(ServerMsg, x.serverTag, x.uid, head, protoObj)
	head.BodyLen = uint32(len(bodyByte))
	headBytes, err := pb.Marshal(head)
	if err != nil {
		x.net.log.Errorf("marshal proto data err: %v\n", err)
		return
	}
	bin := make([]byte, alg.TcpHeadSize+len(headBytes)+len(bodyByte))

	binary.BigEndian.PutUint16(bin[:alg.TcpHeadSize], uint16(len(headBytes)))
	// 头部数据
	copy(bin[alg.TcpHeadSize:], headBytes)
	// proto数据
	copy(bin[alg.TcpHeadSize+len(headBytes):], bodyByte)

	n, err := x.conn.Write(bin)
	x.net.recordSendBytes(n)
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		x.net.log.Errorf("tcpConn write error: %v", err)
		return
	}
}

func (x *tcpConn) SetUID(uid uint32) {
	x.uid = uid
}

func (x *tcpConn) SetServerTag(serverTag string) {
	x.serverTag = serverTag
}

func (x *tcpConn) Close() {
	if x == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&x.closed, 0, 1) {
		return
	}
	x.net.onConnClose()
	_ = x.conn.Close()
}

func (x *tcpConn) LocalAddr() net.Addr {
	return x.conn.LocalAddr()
}

func (x *tcpConn) RemoteAddr() net.Addr {
	return x.conn.RemoteAddr()
}
