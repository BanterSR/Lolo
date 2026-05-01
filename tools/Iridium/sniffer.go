package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/golang/snappy"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"google.golang.org/protobuf/proto"

	pb "gucooing/lolo/protocol/proto"
)

type Packet struct {
	Time       int64       `json:"time"`
	FromServer bool        `json:"fromServer"`
	PacketId   uint32      `json:"packetId"`
	PacketName string      `json:"packetName"`
	Object     interface{} `json:"object"`
	Raw        []byte      `json:"raw"`
}

type session struct {
	netFlow       gopacket.Flow
	transportFlow gopacket.Flow
	fromServer    bool
	data          []byte
	lastSeen      time.Time
	mutex         sync.Mutex
}

type streamFactory struct {
	sessions    map[string]*session
	sessionLock sync.RWMutex
}

type tcpStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
	factory        *streamFactory
	session        *session
}

var (
	captureHandler        *pcap.Handle
	packetFilter          = make(map[string]bool)
	pcapFile              *os.File
	assembler             *tcpassembly.Assembler
	packetDumpFile        *os.File
	packetDumpFilePath    = "packet_dump.ndjson"
	packetDumpFileMutex   sync.Mutex
	streamFactoryInstance = &streamFactory{
		sessions: make(map[string]*session),
	}
)

const tcpHeadSize = 2

func getSessionKey(netFlow, transportFlow gopacket.Flow) string {
	return fmt.Sprintf("%s:%s->%s:%s",
		netFlow.Src(), transportFlow.Src(),
		netFlow.Dst(), transportFlow.Dst())
}

func getPortFromFlow(flow gopacket.Flow) int64 {
	ms, _ := strconv.ParseInt(flow.Src().String(), 10, 32)
	return ms
}

func isFromServer(transportFlow gopacket.Flow) bool {
	srcPort := getPortFromFlow(transportFlow)
	return (int64(config.MinPort) <= srcPort) && (srcPort <= int64(config.MaxPort))
}

func (f *streamFactory) getOrCreateSession(netFlow, transportFlow gopacket.Flow) *session {
	key := getSessionKey(netFlow, transportFlow)

	f.sessionLock.Lock()
	defer f.sessionLock.Unlock()

	if s, exists := f.sessions[key]; exists {
		return s
	}

	s := &session{
		netFlow:       netFlow,
		transportFlow: transportFlow,
		fromServer:    isFromServer(transportFlow),
		data:          make([]byte, 0),
		lastSeen:      time.Now(),
	}
	f.sessions[key] = s
	return s
}

func (f *streamFactory) cleanupOldSessions(timeout time.Duration) {
	f.sessionLock.Lock()
	defer f.sessionLock.Unlock()

	now := time.Now()
	for key, sess := range f.sessions {
		if now.Sub(sess.lastSeen) > timeout {
			delete(f.sessions, key)
		}
	}
}

func (f *streamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	s := &tcpStream{
		net:       net,
		transport: transport,
		factory:   f,
	}
	s.r = tcpreader.NewReaderStream()
	s.session = f.getOrCreateSession(net, transport)

	go s.processStream()

	return &s.r
}

func (s *tcpStream) processStream() {
	buf := make([]byte, math.MaxUint32)

	for {
		n, err := s.r.Read(buf)
		if n > 0 {
			s.session.mutex.Lock()
			s.session.data = append(s.session.data, buf[:n]...)
			s.session.lastSeen = time.Now()
			s.processSessionData()

			s.session.mutex.Unlock()
		}
		if err != nil {
			break
		}
	}
}

// 处理会话数据
func (s *tcpStream) processSessionData() {
	se := s.session

	for len(se.data) >= tcpHeadSize {
		// 解析头部长度
		headLen := binary.BigEndian.Uint16(se.data[:tcpHeadSize])

		// 检查是否收到完整的头部
		if len(se.data) < int(headLen)+tcpHeadSize {
			break
		}

		headBin := se.data[tcpHeadSize : int(headLen)+tcpHeadSize]

		// 解析PacketHead
		head := new(pb.PacketHead)
		err := proto.Unmarshal(headBin, head)
		if err != nil {
			log.Printf("Could not parse PacketHead proto Error:%s\n", err)
			continue
		}

		// 检查是否收到完整的包
		totalSize := uint32(headLen) + head.BodyLen + tcpHeadSize
		if uint32(len(se.data)) < totalSize {
			break
		}

		// 提取包体数据
		bodyStart := uint32(headLen) + tcpHeadSize
		bodyEnd := bodyStart + head.BodyLen
		bodyBin := se.data[bodyStart:bodyEnd]

		// 处理压缩标志
		bodyBin = handleFlag(head.Flag, bodyBin)

		// 解析协议内容
		timestamp := time.Now()

		objectJson, err := parseProtoToInterface(head.MsgId, bodyBin)
		if err != nil {
			// 尝试动态解析
			bodyPb, err := DynamicParse(bodyBin)
			if err != nil {
				log.Printf("Failed to parse body:%s\n", base64.StdEncoding.EncodeToString(bodyBin))
			} else {
				buildPacketToSend(head, bodyBin, se.fromServer, timestamp, bodyPb)
			}
		} else {
			buildPacketToSend(head, bodyBin, se.fromServer, timestamp, objectJson)
		}

		// 从缓冲区移除已处理的数据
		se.data = se.data[totalSize:]
	}
}

func openPcap(fileName string) {
	var err error
	captureHandler, err = pcap.OpenOffline(fileName)
	if err != nil {
		log.Println("Could not open pcap file", err)
		return
	}
	startSniffer()
}

func openCapture() {
	var err error
	captureHandler, err = pcap.OpenLive(config.DeviceName, 1500, true, pcap.BlockForever)
	if err != nil {
		log.Println("Could not open capture", err)
		return
	}

	if config.AutoSavePcapFiles {
		pcapFile, err = os.Create(time.Now().Format("2006-01-02_15-04-05") + ".pcapng")
		if err != nil {
			log.Println("Could not create pcapng file", err)
		}
	}

	startSniffer()
}

func closeHandle() {
	if captureHandler != nil {
		captureHandler.Close()
		captureHandler = nil
	}
	if pcapFile != nil {
		pcapFile.Close()
		pcapFile = nil
	}
	if assembler != nil {
		assembler.FlushAll()
	}
	packetDumpFileMutex.Lock()
	defer packetDumpFileMutex.Unlock()
	if packetDumpFile != nil {
		packetDumpFile.Close()
		packetDumpFile = nil
	}
}

func startSniffer() {
	defer closeHandle()

	var err error
	// expr := fmt.Sprintf("tcp portrange %v-%v", int64(config.MinPort), int64(config.MaxPort))
	// expr = "tcp"
	// err := captureHandler.SetBPFFilter(expr)
	// if err != nil {
	// 	log.Println("Could not set the filter of capture:", err)
	// 	return
	// }

	assembler = tcpassembly.NewAssembler(tcpassembly.NewStreamPool(streamFactoryInstance))

	assembler.MaxBufferedPagesTotal = 1000
	assembler.MaxBufferedPagesPerConnection = 100

	packetSource := gopacket.NewPacketSource(captureHandler, captureHandler.LinkType())
	packetSource.NoCopy = true

	var pcapWriter *pcapgo.NgWriter
	if pcapFile != nil {
		pcapWriter, err = pcapgo.NewNgWriter(pcapFile, captureHandler.LinkType())
		if err != nil {
			log.Println("Could not create pcapng writer", err)
		}
		defer pcapWriter.Flush()
	}

	cleanupTicker := time.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	flushTicker := time.NewTicker(1 * time.Minute)
	defer flushTicker.Stop()

	log.Println("Starting packet capture...")

	for {
		select {
		case packet, ok := <-packetSource.Packets():
			if !ok {
				log.Println("Packet channel closed")
				return
			}

			if packet == nil {
				continue
			}
			switch packet.TransportLayer().(type) {
			case *layers.TCP:
				dstPort := packet.TransportLayer().(*layers.TCP).DstPort
				srcPort := packet.TransportLayer().(*layers.TCP).SrcPort

				if (srcPort < config.MinPort || srcPort > config.MaxPort) &&
					(dstPort < config.MinPort || dstPort > config.MaxPort) {
					continue
				}

				if pcapWriter != nil {
					err := pcapWriter.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
					if err != nil {
						log.Println("Could not write packet to pcap file", err)
					}
				}

				if netLayer := packet.NetworkLayer(); netLayer != nil {
					if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
						tcp := tcpLayer.(*layers.TCP)
						assembler.AssembleWithTimestamp(
							netLayer.NetworkFlow(),
							tcp,
							packet.Metadata().Timestamp,
						)
					}
				}
			}

		case <-cleanupTicker.C:
			streamFactoryInstance.cleanupOldSessions(2 * time.Minute)

		case <-flushTicker.C:
			assembler.FlushOlderThan(time.Now().Add(-1 * time.Minute))
		}
	}
}

func handleFlag(flag uint32, body []byte) []byte {
	switch flag {
	case 0:
		return body
	case 1:
		dst, err := snappy.Decode(nil, body)
		if err != nil {
			log.Printf("Snappy decode error: %v\n", err)
			return body
		}
		return dst
	default:
		log.Printf("Unknown flag:%d\n", flag)
		return body
	}
}

func buildPacketToSend(head *pb.PacketHead, data []byte, fromServer bool, timestamp time.Time, objectJson interface{}) {
	if _, ok := packetFilter[GetProtoNameById(head.MsgId)]; ok {
		return
	}
	packet := &Packet{
		Time:       timestamp.UnixMilli(),
		FromServer: fromServer,
		PacketId:   head.MsgId,
		PacketName: GetProtoNameById(head.MsgId),
		Object:     objectJson,
		Raw:        data,
	}

	jsonResult, err := json.Marshal(packet)
	if err != nil {
		log.Println("Json marshal error", err)
		return
	}
	// logPacket(packet, head)

	log.Printf("name:%s time:%s b64:%s\n", GetProtoNameById(head.MsgId), timestamp.String(), base64.StdEncoding.EncodeToString(data))
	// writePacketDump(jsonResult)

	sendStreamMsg(string(jsonResult))
}

func writePacketDump(jsonResult []byte) {
	packetDumpFileMutex.Lock()
	defer packetDumpFileMutex.Unlock()

	if packetDumpFile == nil {
		file, err := os.OpenFile(packetDumpFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Println("Could not open packet dump file", err)
			return
		}
		packetDumpFile = file
	}

	if _, err := packetDumpFile.Write(jsonResult); err != nil {
		log.Println("Could not write packet dump data", err)
		return
	}
	if _, err := packetDumpFile.Write([]byte("\n")); err != nil {
		log.Println("Could not write packet dump newline", err)
	}
}

func logPacket(packet *Packet, head *pb.PacketHead) {
	from := "[Client]"
	if packet.FromServer {
		from = "[Server]"
	}
	forward := ""
	if strings.Contains(packet.PacketName, "Rsp") {
		forward = "<--"
	} else if strings.Contains(packet.PacketName, "Req") {
		forward = "-->"
	} else if strings.Contains(packet.PacketName, "Notice") && packet.FromServer {
		forward = "<-i"
	}

	log.Printf("%s\t%s\t%s%s\t%d bytes\tPacketId:%v SeqId:%v\n",
		color.GreenString(from),
		color.CyanString(forward),
		color.RedString(packet.PacketName),
		color.YellowString("#"+strconv.Itoa(int(packet.PacketId))),
		len(packet.Raw),
		head.PacketId,
		head.SeqId,
	)
}
