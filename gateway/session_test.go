package gateway

import (
	"errors"
	"net"
	"testing"

	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/cache"
)

type gatewayTestConn struct{}

func (*gatewayTestConn) Read() (*alg.GameMsg, error)    { return nil, errors.New("not implemented") }
func (*gatewayTestConn) SendBin(uint32, uint32, []byte) {}
func (*gatewayTestConn) Send(uint32, pb.Message)        {}
func (*gatewayTestConn) SetUID(uint32)                  {}
func (*gatewayTestConn) GetSeqId() uint32               { return 0 }
func (*gatewayTestConn) SetServerTag(string)            {}
func (*gatewayTestConn) Close()                         {}
func (*gatewayTestConn) LocalAddr() net.Addr            { return &net.TCPAddr{} }
func (*gatewayTestConn) RemoteAddr() net.Addr           { return &net.TCPAddr{} }

func TestDuplicateLoginReplacesSessionAndStaleCleanupCannotRemoveIt(t *testing.T) {
	gateway := &Gateway{sessionMap: cache.New[uint32, *session](0)}
	old := &session{userId: 7, uuid: "old", conn: &gatewayTestConn{}, done: make(chan struct{})}
	next := &session{userId: 7, uuid: "new", conn: &gatewayTestConn{}, done: make(chan struct{})}
	gateway.sessionMap.Set(old.userId, old)

	if previous := gateway.activateSession(next); previous != old {
		t.Fatalf("duplicate login did not return replaced session: %+v", previous)
	}
	select {
	case <-old.done:
	default:
		t.Fatal("duplicate login left the old receive loop running")
	}
	if current, ok := gateway.sessionMap.Get(7); !ok || current != next {
		t.Fatalf("session map was not replaced: current=%+v ok=%v", current, ok)
	}

	gateway.removeSessionIfCurrent(old)
	if current, ok := gateway.sessionMap.Get(7); !ok || current != next {
		t.Fatal("stale disconnect removed the replacement session")
	}
	gateway.removeSessionIfCurrent(next)
	if _, ok := gateway.sessionMap.Get(7); ok {
		t.Fatal("current disconnect did not remove its session")
	}
	next.stop()
	next.stop() // duplicate-login and network cleanup may both request stop.
}
