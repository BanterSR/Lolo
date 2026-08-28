package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestCaptureManagerWritesCompletePacketDump(t *testing.T) {
	originalConfig := config
	config = &Config{
		MinPort:          11000,
		MaxPort:          11100,
		OutputDir:        t.TempDir(),
		PacketBufferSize: 2,
	}
	t.Cleanup(func() {
		config = originalConfig
	})

	manager := &captureManager{
		status: CaptureStatus{
			SchemaVersion: captureSchemaVersion,
			State:         "idle",
		},
	}
	status, err := manager.begin(captureStartOptions{
		Mode:       "live",
		Label:      "test session",
		DeviceName: "test-device",
		DumpJSON:   true,
		IncludeRaw: true,
	})
	if err != nil {
		t.Fatalf("begin capture: %v", err)
	}

	raw := []byte{1, 2, 3, 4}
	packet := manager.record(CapturedPacket{
		Time:        time.Now().UnixMilli(),
		Direction:   "server_to_client",
		FromServer:  true,
		MessageID:   1006,
		MessageName: "PlayerMainDataRsp",
		RequestID:   7,
		SequenceID:  9,
		Object: map[string]any{
			"login_token": "preserved-as-requested",
		},
	}, raw)
	manager.finish(nil)

	if packet.RawBase64 != base64.StdEncoding.EncodeToString(raw) {
		t.Fatalf("raw body was not retained: %q", packet.RawBase64)
	}
	if status.PacketDumpPath == "" {
		t.Fatal("packet dump path is empty")
	}

	file, err := os.Open(status.PacketDumpPath)
	if err != nil {
		t.Fatalf("open packet dump: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("read packet dump: %v", scanner.Err())
	}
	var dumped CapturedPacket
	if err := json.Unmarshal(scanner.Bytes(), &dumped); err != nil {
		t.Fatalf("decode packet dump: %v", err)
	}
	dumpedObject, ok := dumped.Object.(map[string]any)
	if !ok || dumpedObject["login_token"] != "preserved-as-requested" {
		t.Fatalf("packet object was changed: %#v", dumped.Object)
	}
	if dumped.RawBase64 != packet.RawBase64 {
		t.Fatalf("dump raw body = %q, want %q", dumped.RawBase64, packet.RawBase64)
	}
}

func TestCaptureManagerRingBufferAndFilters(t *testing.T) {
	manager := &captureManager{
		status: CaptureStatus{
			SchemaVersion: captureSchemaVersion,
			State:         "capturing",
		},
		packetBuffer:   make([]CapturedPacket, 0, 2),
		packetCapacity: 2,
	}

	for id, name := range []string{"FirstRsp", "SecondRsp", "ThirdNotice"} {
		manager.record(CapturedPacket{
			Time:        time.Now().UnixMilli(),
			Direction:   "server_to_client",
			FromServer:  true,
			MessageID:   uint32(id + 1),
			MessageName: name,
		}, nil)
	}

	packets := manager.queryPackets(packetQuery{Limit: 10})
	if len(packets) != 2 || packets[0].MessageName != "SecondRsp" || packets[1].MessageName != "ThirdNotice" {
		t.Fatalf("unexpected ring contents: %#v", packets)
	}
	filtered := manager.queryPackets(packetQuery{
		AfterID:   2,
		Limit:     10,
		Name:      "thirdnotice",
		Direction: "server_to_client",
	})
	if len(filtered) != 1 || filtered[0].ID != 3 {
		t.Fatalf("unexpected filtered packets: %#v", filtered)
	}
}
