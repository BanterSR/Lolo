package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/snappy"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

// Client 是纯协议 TCP 客户端，帧格式镜像 pkg/ofnet/net-tcp.go：
//
//	[2字节 BigEndian headLen][headLen 字节 PacketHead][BodyLen 字节 body]
//
// body 在 > alg.SnappySize 时用 snappy 压缩（head.Flag=1），无加密。
type Client struct {
	conn   net.Conn
	buf    *bufio.Reader
	seqId  uint32
	m      *Metrics
	sendMu sync.Mutex
}

func Dial(addr string, timeout time.Duration, m *Metrics) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
		buf:  bufio.NewReaderSize(conn, alg.PacketMaxLen),
		m:    m,
	}, nil
}

// Send 用指定 packetId 发送一个 proto 消息。
func (x *Client) Send(packetId uint32, protoObj pb.Message) error {
	cmdId := cmd.Get().GetCmdIdByProtoObj(protoObj)
	bodyByte, err := pb.Marshal(protoObj)
	if err != nil {
		return err
	}

	x.sendMu.Lock()
	defer x.sendMu.Unlock()

	head := &proto.PacketHead{
		MsgId:    cmdId,
		Flag:     0,
		SeqId:    x.seqId,
		PacketId: packetId,
	}
	x.seqId++

	if len(bodyByte) > alg.SnappySize {
		bodyByte = snappy.Encode(nil, bodyByte)
		head.Flag = 1
	}
	head.BodyLen = uint32(len(bodyByte))
	headBytes, err := pb.Marshal(head)
	if err != nil {
		return err
	}

	bin := make([]byte, alg.TcpHeadSize+len(headBytes)+len(bodyByte))
	binary.BigEndian.PutUint16(bin[:alg.TcpHeadSize], uint16(len(headBytes)))
	copy(bin[alg.TcpHeadSize:], headBytes)
	copy(bin[alg.TcpHeadSize+len(headBytes):], bodyByte)

	n, err := x.conn.Write(bin)
	if x.m != nil {
		atomic.AddInt64(&x.m.bytesSent, int64(n))
	}
	return err
}

// Recv 阻塞读取一个完整消息并反序列化。
func (x *Client) Recv() (*alg.GameMsg, error) {
	for {
		x.conn.SetReadDeadline(time.Now().Add(70 * time.Second))

		headLenByte := make([]byte, alg.TcpHeadSize)
		if _, err := io.ReadFull(x.buf, headLenByte); err != nil {
			return nil, err
		}
		headLen := binary.BigEndian.Uint16(headLenByte)

		headByte := make([]byte, headLen)
		if _, err := io.ReadFull(x.buf, headByte); err != nil {
			return nil, err
		}
		head := new(proto.PacketHead)
		if err := pb.Unmarshal(headByte, head); err != nil {
			return nil, err
		}

		bodyByte := make([]byte, head.BodyLen)
		if _, err := io.ReadFull(x.buf, bodyByte); err != nil {
			return nil, err
		}
		if x.m != nil {
			atomic.AddInt64(&x.m.bytesRecv, int64(alg.TcpHeadSize+int(headLen)+int(head.BodyLen)))
		}

		bodyByte = alg.HandleFlag(head.Flag, bodyByte)
		protoObj := cmd.Get().GetProtoObjByCmdId(head.MsgId)
		if protoObj == nil {
			// 未知消息，跳过继续读
			continue
		}
		if err := pb.Unmarshal(bodyByte, protoObj); err != nil {
			return nil, err
		}
		return &alg.GameMsg{PacketHead: head, Body: protoObj}, nil
	}
}

func (x *Client) Close() error { return x.conn.Close() }
