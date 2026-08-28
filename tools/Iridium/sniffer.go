package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/golang/snappy"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/google/gopacket/tcpassembly"
	"google.golang.org/protobuf/proto"

	pb "gucooing/lolo/protocol/proto"
)

type Packet struct {
	Time        int64  `json:"time"`
	FromServer  bool   `json:"fromServer"`
	PacketId    uint32 `json:"packetId"`
	PacketName  string `json:"packetName"`
	RequestID   uint32 `json:"requestId"`
	SequenceID  uint32 `json:"sequenceId"`
	DecodeError string `json:"decodeError,omitempty"`
	Object      any    `json:"object"`
	Raw         []byte `json:"raw"`
}

type streamFactory struct{}

type tcpStream struct {
	netFlow       gopacket.Flow
	transportFlow gopacket.Flow
	flowKey       string
	fromServer    bool
	decoder       packetDecoder
}

type packetDecoder struct {
	buffer     []byte
	recovering bool
	dropBytes  int
	flowKey    string
}

var packetFilter = make(map[string]bool)

const (
	tcpHeadSize         = 2
	liveCaptureSnapshot = 65535
	maxPacketHeadSize   = 64
	maxPacketBodySize   = 512000
)

type packetFrame struct {
	head      *pb.PacketHead
	totalSize int
	bodyStart int
	bodyEnd   int
}

type tcpFlowState struct {
	truncatedLogged bool
}

type packetAssembler struct {
	assembler      *tcpassembly.Assembler
	pcapWriter     *pcapgo.NgWriter
	truncatedFlows map[string]*tcpFlowState
}

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

func (f *streamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	flowKey := getSessionKey(net, transport)
	s := &tcpStream{
		netFlow:       net,
		transportFlow: transport,
		flowKey:       flowKey,
		fromServer:    isFromServer(transport),
		decoder: packetDecoder{
			flowKey: flowKey,
		},
	}
	return s
}

func (s *tcpStream) Reassembled(reassemblies []tcpassembly.Reassembly) {
	for _, reassembly := range reassemblies {
		if reassembly.Start {
			s.decoder.reset()
		}

		if reassembly.Skip != 0 {
			s.decoder.markLoss(reassembly.Skip)
		}

		if len(reassembly.Bytes) > 0 {
			s.decoder.append(reassembly.Bytes, s.fromServer, reassembly.Seen)
		}

		if reassembly.End {
			s.decoder.reset()
		}
	}
}

func (s *tcpStream) ReassemblyComplete() {
	s.decoder.reset()
}

func (d *packetDecoder) reset() {
	d.buffer = d.buffer[:0]
	d.recovering = false
	d.dropBytes = 0
}

func (d *packetDecoder) markLoss(skip int) {
	if skip < 0 {
		log.Printf("TCP stream %s starts mid-stream; trying to recover at next application frame boundary\n", d.flowKey)
		d.buffer = d.buffer[:0]
		d.dropBytes = 0
		d.recovering = true
		return
	}

	if d.dropBytes > 0 {
		if skip > d.dropBytes {
			log.Printf("TCP stream %s lost %d bytes beyond the packet being dropped; trying to recover at next frame boundary\n", d.flowKey, skip)
			d.buffer = d.buffer[:0]
			d.dropBytes = 0
			d.recovering = true
			return
		}
		d.dropBytes -= skip
		return
	}

	frame, complete, err := parsePacketFrame(d.buffer)
	if err != nil || complete || frame == nil {
		log.Printf("TCP stream %s lost %d bytes without a pending frame boundary; trying to recover at next frame boundary\n", d.flowKey, skip)
		d.buffer = d.buffer[:0]
		d.dropBytes = 0
		d.recovering = true
		return
	}

	missingInFrame := frame.totalSize - len(d.buffer)
	if skip > missingInFrame {
		log.Printf("TCP stream %s lost %d bytes beyond current packet; trying to recover at next frame boundary\n", d.flowKey, skip)
		d.buffer = d.buffer[:0]
		d.dropBytes = 0
		d.recovering = true
		return
	}

	d.dropBytes = missingInFrame - skip
	d.recovering = false
	log.Printf("TCP stream %s lost %d bytes inside packet; dropping current packet and %d following bytes\n",
		d.flowKey,
		skip,
		d.dropBytes,
	)
	d.buffer = d.buffer[:0]
}

func (d *packetDecoder) append(data []byte, fromServer bool, timestamp time.Time) {
	if d.dropBytes > 0 {
		if len(data) <= d.dropBytes {
			d.dropBytes -= len(data)
			return
		}
		data = data[d.dropBytes:]
		d.dropBytes = 0
	}

	if d.recovering {
		d.buffer = d.buffer[:0]
		if _, _, err := parsePacketFrame(data); err != nil {
			log.Printf("TCP stream %s skipped %d bytes while recovering frame boundary: %v\n", d.flowKey, len(data), err)
			return
		}
		d.recovering = false
	}

	d.buffer = append(d.buffer, data...)

	for len(d.buffer) >= tcpHeadSize {
		frame, complete, err := parsePacketFrame(d.buffer)
		if err != nil {
			log.Printf("TCP stream %s lost frame sync: %v; trying to recover at next frame boundary\n", d.flowKey, err)
			d.buffer = d.buffer[:0]
			d.recovering = true
			return
		}
		if !complete {
			break
		}

		// 提取包体数据
		head := frame.head
		bodyBin := d.buffer[frame.bodyStart:frame.bodyEnd]

		// 处理压缩标志
		bodyBin = handleFlag(head.Flag, bodyBin)

		// 解析协议内容
		objectJSON, err := parseProtoToInterface(head.MsgId, bodyBin)
		if err != nil {
			// 尝试动态解析
			bodyPB, dynamicErr := DynamicParse(bodyBin)
			if dynamicErr != nil {
				buildPacketToSend(
					head,
					bodyBin,
					fromServer,
					timestamp,
					nil,
					fmt.Sprintf("known schema: %v; dynamic decode: %v", err, dynamicErr),
				)
			} else {
				buildPacketToSend(head, bodyBin, fromServer, timestamp, bodyPB, "")
			}
		} else {
			buildPacketToSend(head, bodyBin, fromServer, timestamp, objectJSON, "")
		}

		// 从缓冲区移除已处理的数据
		d.buffer = d.buffer[frame.totalSize:]
	}
}

func parsePacketFrame(data []byte) (*packetFrame, bool, error) {
	if len(data) < tcpHeadSize {
		return nil, false, nil
	}

	headLen := int(binary.BigEndian.Uint16(data[:tcpHeadSize]))
	if headLen <= 0 || headLen > maxPacketHeadSize {
		return nil, false, fmt.Errorf("invalid PacketHead length %d", headLen)
	}

	headEnd := tcpHeadSize + headLen
	if len(data) < headEnd {
		return nil, false, nil
	}

	head := new(pb.PacketHead)
	if err := proto.Unmarshal(data[tcpHeadSize:headEnd], head); err != nil {
		return nil, false, fmt.Errorf("cannot parse PacketHead: %w", err)
	}
	if err := validatePacketHead(head); err != nil {
		return nil, false, err
	}

	bodyStart := headEnd
	bodyEnd := bodyStart + int(head.BodyLen)
	if len(data) < bodyEnd {
		return &packetFrame{
			head:      head,
			totalSize: bodyEnd,
			bodyStart: bodyStart,
			bodyEnd:   bodyEnd,
		}, false, nil
	}

	return &packetFrame{
		head:      head,
		totalSize: bodyEnd,
		bodyStart: bodyStart,
		bodyEnd:   bodyEnd,
	}, true, nil
}

func validatePacketHead(head *pb.PacketHead) error {
	if head.MsgId == 0 {
		return fmt.Errorf("invalid msg_id %d", head.MsgId)
	}
	if head.Flag > 1 {
		return fmt.Errorf("unsupported PacketHead flag %d", head.Flag)
	}
	if head.BodyLen > maxPacketBodySize {
		return fmt.Errorf("body_len %d exceeds max %d", head.BodyLen, maxPacketBodySize)
	}
	return nil
}

func newPacketAssembler(linkType layers.LinkType, pcapFile *os.File) (*packetAssembler, error) {
	pa := &packetAssembler{
		assembler:      tcpassembly.NewAssembler(tcpassembly.NewStreamPool(&streamFactory{})),
		truncatedFlows: make(map[string]*tcpFlowState),
	}
	pa.assembler.MaxBufferedPagesTotal = 1000
	pa.assembler.MaxBufferedPagesPerConnection = 100

	if pcapFile == nil {
		return pa, nil
	}

	writer, err := pcapgo.NewNgWriter(pcapFile, linkType)
	if err != nil {
		return nil, err
	}
	pa.pcapWriter = writer
	return pa, nil
}

func (pa *packetAssembler) flushOlderThan(t time.Time) {
	pa.assembler.FlushOlderThan(t)
}

func (pa *packetAssembler) flushAll() {
	pa.assembler.FlushAll()
	if pa.pcapWriter != nil {
		pa.pcapWriter.Flush()
	}
}

func (pa *packetAssembler) handlePacket(packet gopacket.Packet) {
	if packet == nil {
		return
	}

	captureInfo := packet.Metadata().CaptureInfo
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return
	}
	tcp := tcpLayer.(*layers.TCP)
	dstPort := tcp.DstPort
	srcPort := tcp.SrcPort

	if (srcPort < config.MinPort || srcPort > config.MaxPort) &&
		(dstPort < config.MinPort || dstPort > config.MaxPort) {
		return
	}

	netLayer := packet.NetworkLayer()
	if netLayer == nil {
		return
	}

	flowKey := getSessionKey(netLayer.NetworkFlow(), tcp.TransportFlow())
	if captureInfo.CaptureLength < captureInfo.Length {
		state := pa.truncatedFlows[flowKey]
		if state == nil {
			state = &tcpFlowState{}
			pa.truncatedFlows[flowKey] = state
		}
		if !state.truncatedLogged {
			log.Printf("TCP stream %s has truncated packets; TCP reassembly will report the loss caplen=%d len=%d\n",
				flowKey,
				captureInfo.CaptureLength,
				captureInfo.Length,
			)
			state.truncatedLogged = true
		}
	}

	if pa.pcapWriter != nil {
		if err := pa.pcapWriter.WritePacket(captureInfo, packet.Data()); err != nil {
			log.Println("Could not write packet to pcap file", err)
		}
	}

	pa.assembler.AssembleWithTimestamp(
		netLayer.NetworkFlow(),
		tcp,
		captureInfo.Timestamp,
	)
}

func startSniffer(captureHandler *pcap.Handle, pcapFile *os.File) error {
	// expr := fmt.Sprintf("tcp portrange %v-%v", int64(config.MinPort), int64(config.MaxPort))
	// expr = "tcp"
	// err := captureHandler.SetBPFFilter(expr)
	// if err != nil {
	// 	log.Println("Could not set the filter of capture:", err)
	// 	return
	// }

	packetSource := gopacket.NewPacketSource(captureHandler, captureHandler.LinkType())
	packetSource.NoCopy = true

	pa, err := newPacketAssembler(captureHandler.LinkType(), pcapFile)
	if err != nil {
		return fmt.Errorf("create packet assembler: %w", err)
	}
	defer pa.flushAll()

	flushTicker := time.NewTicker(1 * time.Minute)
	defer flushTicker.Stop()

	log.Println("Starting packet capture...")

	for {
		select {
		case packet, ok := <-packetSource.Packets():
			if !ok {
				log.Println("Packet channel closed")
				return nil
			}
			pa.handlePacket(packet)

		case <-flushTicker.C:
			pa.flushOlderThan(time.Now().Add(-5 * time.Minute))
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

func buildPacketToSend(
	head *pb.PacketHead,
	data []byte,
	fromServer bool,
	timestamp time.Time,
	objectJSON any,
	decodeError string,
) {
	if _, ok := packetFilter[GetProtoNameById(head.MsgId)]; ok {
		return
	}

	direction := "client_to_server"
	if fromServer {
		direction = "server_to_client"
	}
	packetName := GetProtoNameById(head.MsgId)
	captures.record(CapturedPacket{
		Time:        timestamp.UnixMilli(),
		Direction:   direction,
		FromServer:  fromServer,
		MessageID:   head.MsgId,
		MessageName: packetName,
		RequestID:   head.PacketId,
		SequenceID:  head.SeqId,
		Object:      objectJSON,
		DecodeError: decodeError,
	}, data)

	packet := &Packet{
		Time:        timestamp.UnixMilli(),
		FromServer:  fromServer,
		PacketId:    head.MsgId,
		PacketName:  packetName,
		RequestID:   head.PacketId,
		SequenceID:  head.SeqId,
		DecodeError: decodeError,
		Object:      objectJSON,
		Raw:         data,
	}

	jsonResult, err := json.Marshal(packet)
	if err != nil {
		log.Println("Json marshal error", err)
		return
	}
	logPacket(packet, head)
	sendStreamMsg(string(jsonResult))
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
