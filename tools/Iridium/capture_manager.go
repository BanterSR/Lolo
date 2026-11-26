package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket/pcap"
)

const captureSchemaVersion = 1

var errCaptureActive = errors.New("a capture session is already active")

type CaptureStatus struct {
	SchemaVersion  int        `json:"schemaVersion"`
	State          string     `json:"state"`
	Mode           string     `json:"mode,omitempty"`
	Label          string     `json:"label,omitempty"`
	DeviceName     string     `json:"deviceName,omitempty"`
	InputPath      string     `json:"inputPath,omitempty"`
	MinPort        uint16     `json:"minPort"`
	MaxPort        uint16     `json:"maxPort"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	StoppedAt      *time.Time `json:"stoppedAt,omitempty"`
	LastPacketAt   *time.Time `json:"lastPacketAt,omitempty"`
	SessionDir     string     `json:"sessionDir,omitempty"`
	ManifestPath   string     `json:"manifestPath,omitempty"`
	PacketDumpPath string     `json:"packetDumpPath,omitempty"`
	PCAPPath       string     `json:"pcapPath,omitempty"`
	PacketCount    uint64     `json:"packetCount"`
	LastError      string     `json:"lastError,omitempty"`
}

type CapturedPacket struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            uint64 `json:"id"`
	Time          int64  `json:"time"`
	Timestamp     string `json:"timestamp"`
	Direction     string `json:"direction"`
	FromServer    bool   `json:"fromServer"`
	MessageID     uint32 `json:"messageId"`
	MessageName   string `json:"messageName"`
	RequestID     uint32 `json:"requestId"`
	SequenceID    uint32 `json:"sequenceId"`
	BodySize      int    `json:"bodySize"`
	Object        any    `json:"object,omitempty"`
	RawBase64     string `json:"rawBase64,omitempty"`
	DecodeError   string `json:"decodeError,omitempty"`
}

type captureStartOptions struct {
	Mode       string
	Label      string
	DeviceName string
	InputPath  string
	DumpJSON   bool
	IncludeRaw bool
	SavePCAP   bool
}

type packetQuery struct {
	AfterID   uint64
	Limit     int
	Name      string
	Direction string
}

type captureManager struct {
	mu             sync.RWMutex
	status         CaptureStatus
	handle         *pcap.Handle
	packetDumpFile *os.File
	includeRaw     bool
	packetBuffer   []CapturedPacket
	packetStart    int
	packetCapacity int
}

var captures = &captureManager{
	status: CaptureStatus{
		SchemaVersion: captureSchemaVersion,
		State:         "idle",
	},
}

func (m *captureManager) configure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if isActiveCaptureState(m.status.State) {
		return
	}
	m.status.MinPort = uint16(config.MinPort)
	m.status.MaxPort = uint16(config.MaxPort)
	m.packetCapacity = config.PacketBufferSize
	if m.packetBuffer == nil {
		m.packetBuffer = make([]CapturedPacket, 0, m.packetCapacity)
	}
}

func (m *captureManager) begin(opts captureStartOptions) (CaptureStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if isActiveCaptureState(m.status.State) {
		return m.status, errCaptureActive
	}

	startedAt := time.Now()
	sessionDir, err := createSessionDir(config.OutputDir, startedAt, opts.Label)
	if err != nil {
		return m.status, err
	}

	status := CaptureStatus{
		SchemaVersion: captureSchemaVersion,
		State:         "starting",
		Mode:          opts.Mode,
		Label:         opts.Label,
		DeviceName:    opts.DeviceName,
		InputPath:     opts.InputPath,
		MinPort:       uint16(config.MinPort),
		MaxPort:       uint16(config.MaxPort),
		StartedAt:     &startedAt,
		SessionDir:    sessionDir,
		ManifestPath:  filepath.Join(sessionDir, "manifest.json"),
	}
	if opts.DumpJSON {
		status.PacketDumpPath = filepath.Join(sessionDir, "packets.ndjson")
	}
	if opts.SavePCAP {
		status.PCAPPath = filepath.Join(sessionDir, "capture.pcapng")
	}

	var dumpFile *os.File
	if status.PacketDumpPath != "" {
		dumpFile, err = os.Create(status.PacketDumpPath)
		if err != nil {
			return m.status, fmt.Errorf("create packet dump: %w", err)
		}
	}

	m.status = status
	m.handle = nil
	m.packetDumpFile = dumpFile
	m.includeRaw = opts.IncludeRaw
	m.packetCapacity = config.PacketBufferSize
	m.packetBuffer = make([]CapturedPacket, 0, m.packetCapacity)
	m.packetStart = 0
	m.writeManifestLocked()
	return m.status, nil
}

func (m *captureManager) attachHandle(handle *pcap.Handle) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.State != "starting" {
		return false
	}
	m.handle = handle
	m.status.State = "capturing"
	m.writeManifestLocked()
	return true
}

func (m *captureManager) stop() (CaptureStatus, error) {
	m.mu.Lock()
	if !isActiveCaptureState(m.status.State) {
		status := m.status
		m.mu.Unlock()
		return status, errors.New("no capture session is active")
	}
	m.status.State = "stopping"
	handle := m.handle
	status := m.status
	m.writeManifestLocked()
	m.mu.Unlock()

	if handle != nil {
		handle.Close()
	}
	return status, nil
}

func (m *captureManager) finish(captureErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stoppedAt := time.Now()
	m.status.StoppedAt = &stoppedAt
	m.status.State = "stopped"
	m.status.LastError = ""
	if captureErr != nil {
		m.status.State = "error"
		m.status.LastError = captureErr.Error()
	}
	m.handle = nil
	if m.packetDumpFile != nil {
		if err := m.packetDumpFile.Close(); err != nil && m.status.LastError == "" {
			m.status.LastError = err.Error()
			m.status.State = "error"
		}
		m.packetDumpFile = nil
	}
	m.writeManifestLocked()
}

func (m *captureManager) statusSnapshot() CaptureStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *captureManager) record(packet CapturedPacket, raw []byte) CapturedPacket {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.PacketCount++
	packet.SchemaVersion = captureSchemaVersion
	packet.ID = m.status.PacketCount
	packet.BodySize = len(raw)
	if m.includeRaw {
		packet.RawBase64 = base64.StdEncoding.EncodeToString(raw)
	}
	packetTime := time.UnixMilli(packet.Time)
	packet.Timestamp = packetTime.Format(time.RFC3339Nano)
	m.status.LastPacketAt = &packetTime

	m.appendPacketLocked(packet)
	if m.packetDumpFile != nil {
		if err := json.NewEncoder(m.packetDumpFile).Encode(packet); err != nil {
			m.status.LastError = fmt.Sprintf("write packet dump: %v", err)
		}
	}
	return packet
}

func (m *captureManager) appendPacketLocked(packet CapturedPacket) {
	if m.packetCapacity <= 0 {
		return
	}
	if len(m.packetBuffer) < m.packetCapacity {
		m.packetBuffer = append(m.packetBuffer, packet)
		return
	}
	m.packetBuffer[m.packetStart] = packet
	m.packetStart = (m.packetStart + 1) % m.packetCapacity
}

func (m *captureManager) queryPackets(query packetQuery) []CapturedPacket {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limit := query.Limit
	if limit <= 0 || limit > m.packetCapacity {
		limit = m.packetCapacity
	}
	packets := make([]CapturedPacket, 0, min(limit, len(m.packetBuffer)))
	for offset := range len(m.packetBuffer) {
		index := (m.packetStart + offset) % len(m.packetBuffer)
		packet := m.packetBuffer[index]
		if packet.ID <= query.AfterID {
			continue
		}
		if query.Name != "" && !strings.EqualFold(packet.MessageName, query.Name) {
			continue
		}
		if query.Direction != "" && !strings.EqualFold(packet.Direction, query.Direction) {
			continue
		}
		packets = append(packets, packet)
		if len(packets) == limit {
			break
		}
	}
	return packets
}

func (m *captureManager) writeManifestLocked() {
	if m.status.ManifestPath == "" {
		return
	}
	data, err := json.MarshalIndent(m.status, "", "  ")
	if err != nil {
		m.status.LastError = fmt.Sprintf("encode manifest: %v", err)
		return
	}
	if err := os.WriteFile(m.status.ManifestPath, append(data, '\n'), 0o644); err != nil {
		m.status.LastError = fmt.Sprintf("write manifest: %v", err)
	}
}

func isActiveCaptureState(state string) bool {
	switch state {
	case "starting", "capturing", "stopping":
		return true
	default:
		return false
	}
}

var invalidLabelCharacters = regexp.MustCompile(`[^\p{L}\p{N}._-]+`)

func createSessionDir(baseDir string, startedAt time.Time, label string) (string, error) {
	cleanLabel := strings.Trim(invalidLabelCharacters.ReplaceAllString(label, "-"), "-.")
	dirName := startedAt.Format("20060102-150405.000")
	if cleanLabel != "" {
		dirName += "-" + cleanLabel
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("create capture output directory: %w", err)
	}

	var sessionDir string
	for suffix := 0; suffix < 1000; suffix++ {
		candidateName := dirName
		if suffix > 0 {
			candidateName = fmt.Sprintf("%s-%d", dirName, suffix)
		}
		candidate := filepath.Join(baseDir, candidateName)
		err := os.Mkdir(candidate, 0o755)
		if err == nil {
			sessionDir = candidate
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("create capture session directory: %w", err)
		}
	}
	if sessionDir == "" {
		return "", errors.New("could not allocate a unique capture session directory")
	}
	absoluteDir, err := filepath.Abs(sessionDir)
	if err != nil {
		return "", fmt.Errorf("resolve capture session directory: %w", err)
	}
	return absoluteDir, nil
}
