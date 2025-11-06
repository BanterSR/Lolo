package config

type LogServer struct {
	OuterIp   string `json:"OuterIp"`
	OuterPort int    `json:"OuterPort"`
	OuterAddr string `json:"OuterAddr"`
}

var defaultLogServer = &LogServer{
	OuterIp:   "127.0.0.1",
	OuterPort: 12000,
	OuterAddr: "0.0.0.0:12000",
}

func GetLogServer() *LogServer {
	if GetConfig().LogServer == nil {
		return defaultLogServer
	}
	return GetConfig().LogServer
}

func (x *LogServer) GetOuterIp() string {
	return x.OuterIp
}

func (x *LogServer) GetOuterPort() int {
	return x.OuterPort
}

func (x *LogServer) GetOuterAddr() string {
	return x.OuterAddr
}
