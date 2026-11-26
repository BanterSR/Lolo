package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type Config struct {
	DeviceName        string         `json:"deviceName"`
	PacketFilter      []string       `json:"packetFilter"`
	AutoSavePcapFiles bool           `json:"autoSavePcapFiles"`
	DumpJSON          bool           `json:"dumpJson"`
	IncludeRawInDump  bool           `json:"includeRawInDump"`
	MaxPort           layers.TCPPort `json:"maxPort"`
	MinPort           layers.TCPPort `json:"minPort"`
	ListenAddr        string         `json:"listenAddr"`
	OutputDir         string         `json:"outputDir"`
	PacketBufferSize  int            `json:"packetBufferSize"`
}

type cliOptions struct {
	configPath  string
	listDevices bool
	format      string
	ip          string
	listenAddr  string
	outputDir   string
	autoStart   bool
}

var config *Config

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	opts := parseFlags()
	if opts.listDevices {
		return printDevices(opts.format)
	}

	loadedConfig, err := loadConfig(opts.configPath)
	if err != nil {
		return err
	}
	if err := applyCLIOverrides(loadedConfig, opts); err != nil {
		return err
	}
	if err := normalizeConfig(loadedConfig, opts.configPath); err != nil {
		return err
	}
	config = loadedConfig
	captures.configure()

	packetFilter = make(map[string]bool, len(config.PacketFilter))
	for _, packetName := range config.PacketFilter {
		packetFilter[packetName] = true
	}

	return startServer(opts.autoStart)
}

func parseFlags() cliOptions {
	var opts cliOptions
	flag.StringVar(&opts.configPath, "config", "./config.json", "Path to the Iridium JSON config")
	flag.BoolVar(&opts.listDevices, "l", false, "List network devices and exit")
	flag.StringVar(&opts.format, "format", "human", "Device list format: human or json")
	flag.StringVar(&opts.ip, "ip", "", "Select the capture device that owns this IP address")
	flag.StringVar(&opts.listenAddr, "listen", "", "Override the HTTP listen address")
	flag.StringVar(&opts.outputDir, "output-dir", "", "Override the capture output directory")
	flag.BoolVar(&opts.autoStart, "auto-start", false, "Start live capture when the HTTP server starts")
	flag.Parse()
	return opts
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}

	loadedConfig := new(Config)
	if err := json.Unmarshal(data, loadedConfig); err != nil {
		return nil, fmt.Errorf("decode config %q: %w", path, err)
	}
	return loadedConfig, nil
}

func applyCLIOverrides(loadedConfig *Config, opts cliOptions) error {
	if opts.ip != "" {
		device, err := findDeviceByIP(opts.ip)
		if err != nil {
			return fmt.Errorf("select capture device by IP %q: %w", opts.ip, err)
		}
		loadedConfig.DeviceName = device.Name
	}
	if opts.listenAddr != "" {
		loadedConfig.ListenAddr = opts.listenAddr
	}
	if opts.outputDir != "" {
		loadedConfig.OutputDir = opts.outputDir
	}
	return nil
}

func normalizeConfig(loadedConfig *Config, configPath string) error {
	if loadedConfig.MinPort == 0 || loadedConfig.MaxPort == 0 {
		return errors.New("minPort and maxPort must both be greater than zero")
	}
	if loadedConfig.MinPort > loadedConfig.MaxPort {
		return errors.New("minPort must not be greater than maxPort")
	}
	if loadedConfig.ListenAddr == "" {
		loadedConfig.ListenAddr = "127.0.0.1:1984"
	}
	if loadedConfig.OutputDir == "" {
		loadedConfig.OutputDir = "./captures"
	}
	if loadedConfig.PacketBufferSize <= 0 {
		loadedConfig.PacketBufferSize = 2000
	}

	if !filepath.IsAbs(loadedConfig.OutputDir) {
		configDir, err := filepath.Abs(filepath.Dir(configPath))
		if err != nil {
			return fmt.Errorf("resolve config directory: %w", err)
		}
		loadedConfig.OutputDir = filepath.Join(configDir, loadedConfig.OutputDir)
	}
	loadedConfig.OutputDir = filepath.Clean(loadedConfig.OutputDir)
	return nil
}

func listDevices() ([]pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("list capture devices: %w", err)
	}
	return devices, nil
}

func findDeviceByIP(ip string) (*pcap.Interface, error) {
	wantedIP := net.ParseIP(ip)
	if wantedIP == nil {
		return nil, fmt.Errorf("invalid IP address %q", ip)
	}

	devices, err := listDevices()
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		for _, address := range device.Addresses {
			if address.IP.Equal(wantedIP) {
				matched := device
				return &matched, nil
			}
		}
	}
	return nil, fmt.Errorf("no capture device owns IP %q", ip)
}

func printDevices(format string) error {
	devices, err := listDevices()
	if err != nil {
		return err
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(devices)
	case "human":
		log.Println(color.RedString("Name"), "\tDescription\t", color.CyanString("IP address"), "\tSubnet mask")
		for _, device := range devices {
			log.Println(color.RedString(device.Name), "\t", device.Description, "\t")
			for _, address := range device.Addresses {
				log.Println("\t\t\t", color.CyanString(address.IP.String()), "\t", address.Netmask)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported format %q; use human or json", format)
	}
}
