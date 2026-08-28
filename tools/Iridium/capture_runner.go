package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket/pcap"
)

type CaptureStartRequest struct {
	Label      string `json:"label"`
	DeviceName string `json:"deviceName"`
	IP         string `json:"ip"`
	DumpJSON   *bool  `json:"dumpJson"`
	IncludeRaw *bool  `json:"includeRaw"`
}

func startLiveCapture(request CaptureStartRequest) (CaptureStatus, error) {
	deviceName := request.DeviceName
	if request.IP != "" {
		device, err := findDeviceByIP(request.IP)
		if err != nil {
			return captures.statusSnapshot(), err
		}
		deviceName = device.Name
	}
	if deviceName == "" {
		deviceName = config.DeviceName
	}
	if deviceName == "" {
		return captures.statusSnapshot(), fmt.Errorf("deviceName or ip is required")
	}

	status, err := captures.begin(captureStartOptions{
		Mode:       "live",
		Label:      request.Label,
		DeviceName: deviceName,
		DumpJSON:   boolValue(request.DumpJSON, config.DumpJSON),
		IncludeRaw: boolValue(request.IncludeRaw, config.IncludeRawInDump),
		SavePCAP:   config.AutoSavePcapFiles,
	})
	if err != nil {
		return status, err
	}

	go runLiveCapture(deviceName)
	return status, nil
}

func startOfflineCapture(inputPath, label string, removeAfterCapture bool) (CaptureStatus, error) {
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return captures.statusSnapshot(), fmt.Errorf("read capture input: %w", err)
	}
	if inputInfo.IsDir() {
		return captures.statusSnapshot(), fmt.Errorf("capture input %q is a directory", inputPath)
	}

	status, err := captures.begin(captureStartOptions{
		Mode:       "offline",
		Label:      label,
		InputPath:  inputPath,
		DumpJSON:   true,
		IncludeRaw: config.IncludeRawInDump,
	})
	if err != nil {
		return status, err
	}

	go runOfflineCapture(inputPath, removeAfterCapture)
	return status, nil
}

func runLiveCapture(deviceName string) {
	log.Printf("opening live capture device=%s snaplen=%d", deviceName, liveCaptureSnapshot)
	handle, err := pcap.OpenLive(deviceName, liveCaptureSnapshot, true, pcap.BlockForever)
	if err != nil {
		captures.finish(fmt.Errorf("open live capture: %w", err))
		return
	}
	defer handle.Close()

	if !captures.attachHandle(handle) {
		captures.finish(nil)
		return
	}

	status := captures.statusSnapshot()
	var pcapFile *os.File
	if status.PCAPPath != "" {
		pcapFile, err = os.Create(status.PCAPPath)
		if err != nil {
			captures.finish(fmt.Errorf("create pcapng output: %w", err))
			return
		}
		defer pcapFile.Close()
		log.Printf("saving live capture to %s", status.PCAPPath)
	}

	err = startSniffer(handle, pcapFile)
	captures.finish(err)
}

func runOfflineCapture(inputPath string, removeAfterCapture bool) {
	if removeAfterCapture {
		defer os.Remove(inputPath)
	}
	log.Printf("opening pcap file %s", inputPath)
	handle, err := pcap.OpenOffline(inputPath)
	if err != nil {
		captures.finish(fmt.Errorf("open pcap file: %w", err))
		return
	}
	defer handle.Close()

	if !captures.attachHandle(handle) {
		captures.finish(nil)
		return
	}

	err = startSniffer(handle, nil)
	captures.finish(err)
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
