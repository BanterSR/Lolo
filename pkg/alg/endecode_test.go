package alg

import (
	"bytes"
	"testing"

	"github.com/golang/snappy"
)

func TestHandleFlagRoundTrip(t *testing.T) {
	plain := []byte("lolo-packet")
	got, err := HandleFlag(0, plain)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("plain packet = %q, %v", got, err)
	}
	compressed := snappy.Encode(nil, plain)
	got, err = HandleFlag(1, compressed)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("compressed packet = %q, %v", got, err)
	}
}

func TestHandleFlagRejectsMalformedAndOversizedPackets(t *testing.T) {
	if _, err := HandleFlag(2, nil); err == nil {
		t.Fatal("unsupported flag accepted")
	}
	if _, err := HandleFlag(1, []byte{0xff}); err == nil {
		t.Fatal("malformed snappy packet accepted")
	}
	compressed := snappy.Encode(nil, make([]byte, PacketMaxLen+1))
	if _, err := HandleFlag(1, compressed); err == nil {
		t.Fatal("oversized decoded packet accepted")
	}
}
