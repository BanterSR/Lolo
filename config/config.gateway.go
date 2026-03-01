package config

import (
	"github.com/gookit/slog"
)

type GateWay struct {
	Log          *Log     `json:"Log"`
	PacketLog    *Log     `json:"PacketLog"`
	OuterIp      string   `json:"OuterIp"`
	OuterPort    int      `json:"OuterPort"`
	OuterAddr    string   `json:"OuterAddr"`
	MaxPlayerNum int64    `json:"MaxPlayerNum"`
	BlackCmd     []string `json:"BlackCmd"`
	CheckToken   bool     `json:"CheckToken"`
	CheckUrl     string   `json:"CheckUrl"`
}

var defaultGateWay = &GateWay{
	Log: &Log{
		Level:   slog.InfoLevel,
		LogFile: false,
		AppName: "Gate",
	},
	PacketLog: &Log{
		Level:   slog.InfoLevel,
		LogFile: false,
		AppName: "Packet",
	},
	OuterIp:      "127.0.0.1",
	OuterPort:    11000,
	OuterAddr:    "0.0.0.0:11000",
	MaxPlayerNum: 0,
	BlackCmd: []string{
		"PlayerPingReq",
		"PlayerPingRsp",
		"PlayerSceneRecordReq",
		"PlayerSceneRecordRsp",
		"PlayerSceneSyncDataNotice",
	},
	CheckToken: true,
	CheckUrl:   "http://127.0.0.1:8080/gucooing/lolo/checkSdkToken",
}

func GetGateWay() *GateWay {
	return GetConfig().GateWay
}

func (x *GateWay) GetLog() *Log {
	return x.Log
}

func (x *GateWay) GetOuterIp() string {
	return x.OuterIp
}

func (x *GateWay) GetOuterPort() int {
	return x.OuterPort
}

func (x *GateWay) GetOuterAddr() string {
	return x.OuterAddr
}

func (x *GateWay) GetMaxPlayerNum() int64 {
	return x.MaxPlayerNum
}

func (x *GateWay) GetBlackCmd() []string {
	return x.BlackCmd
}

func (x *GateWay) GetPacketLog() *Log {
	return x.PacketLog
}

func (x *GateWay) GetCheckToken() bool {
	return x.CheckToken
}

func (x *GateWay) GetCheckUrl() string {
	return x.CheckUrl
}
