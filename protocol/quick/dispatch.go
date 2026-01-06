package quick

type RegionInfoRequest struct {
	Version         string `form:"version" binding:"required"`
	Version2        string `form:"version2" binding:"required"`
	AccountType     string `form:"accountType" binding:"required"`
	OS              string `form:"os" binding:"required"`
	LastLoginSDKUID string `form:"lastloginsdkuid"`
}

type RegionInfo struct {
	Status           bool   `json:"status"`
	Message          string `json:"message"`
	GateTcpIp        string `json:"gate_tcp_ip"`
	GateTcpPort      int    `json:"gate_tcp_port"`
	IsServerOpen     bool   `json:"is_server_open"`
	Text             string `json:"text"`
	ClientLogTcpIp   string `json:"client_log_tcp_ip"`
	ClientLogTcpPort int    `json:"client_log_tcp_port"`
	CurrentVersion   string `json:"currentVersion"`
	PhotoShareCdnUrl string `json:"photo_share_cdn_url"`
}

type GMClientConfig struct {
	Status               bool   `json:"status"`
	Message              string `json:"message"`
	HotOssUrl            string `json:"hotOssUrl"`
	CurrentVersion       string `json:"currentVersion"`
	Server               string `json:"server"`
	SsAppId              string `json:"ssAppId"`
	SsServerUrl          string `json:"ssServerUrl"`
	OpenGm               bool   `json:"open_gm"`
	OpenErrorLog         bool   `json:"open_error_log"`
	OpenNetConnectingLog bool   `json:"open_netConnecting_log"`
	IpAddress            string `json:"ipAddress"`
	PayUrl               string `json:"payUrl"`
	IsTestServer         bool   `json:"isTestServer"`
	ErrorLogLevel        int    `json:"error_log_level"`
	ServerId             string `json:"server_id"`
	OpenCs               bool   `json:"open_cs"`
}

type ClientBlack struct {
	ID           int    `json:"ID"`
	MANUFACTURER string `json:"MANUFACTURER"`
	MODEL        string `json:"MODEL"`
}

type GetNoticeList struct {
	Data       interface{} `json:"data"`
	ServerTime int64       `json:"serverTime"`
}
