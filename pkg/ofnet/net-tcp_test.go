package ofnet

import (
	"bufio"
	"encoding/binary"
	"net"
	"strings"
	"testing"

	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

func readFrameWithHead(t *testing.T, head *proto.PacketHead, body []byte) error {
	t.Helper()
	server, client := net.Pipe()
	defer server.Close()
	headBytes, err := pb.Marshal(head)
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, alg.TcpHeadSize+len(headBytes)+len(body))
	binary.BigEndian.PutUint16(frame[:alg.TcpHeadSize], uint16(len(headBytes)))
	copy(frame[alg.TcpHeadSize:], headBytes)
	copy(frame[alg.TcpHeadSize+len(headBytes):], body)
	go func() {
		_, _ = client.Write(frame)
		_ = client.Close()
	}()
	conn := &tcpConn{
		net:  &tcpNet{netBase: &netBase{}},
		conn: server,
		buf:  bufio.NewReader(server),
	}
	_, err = conn.Read()
	return err
}

func TestTCPReadRejectsOversizedBodyBeforeAllocation(t *testing.T) {
	err := readFrameWithHead(t, &proto.PacketHead{
		MsgId:   cmd.Get().GetCmdIdByProtoObj(&proto.PlayerMainDataReq{}),
		BodyLen: alg.PacketMaxLen + 1,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds max") {
		t.Fatalf("oversized packet error = %v", err)
	}
}

func TestTCPReadRejectsUnsupportedFlag(t *testing.T) {
	err := readFrameWithHead(t, &proto.PacketHead{
		MsgId: cmd.Get().GetCmdIdByProtoObj(&proto.PlayerMainDataReq{}),
		Flag:  2,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported packet flag") {
		t.Fatalf("unsupported flag error = %v", err)
	}
}

func TestTCPReadRejectsMalformedSnappy(t *testing.T) {
	err := readFrameWithHead(t, &proto.PacketHead{
		MsgId:   cmd.Get().GetCmdIdByProtoObj(&proto.PlayerMainDataReq{}),
		Flag:    1,
		BodyLen: 1,
	}, []byte{0xff})
	if err == nil {
		t.Fatal("malformed snappy packet accepted")
	}
}
