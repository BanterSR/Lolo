package sdk

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gucooing/lolo/config"
	"gucooing/lolo/gdconf"
)

func (s *Server) Router() {
	s.router.Any("/", HandleDefault)

	dispatch := s.router.Group("/dispatch")
	{
		dispatch.POST("/region_info", regionInfo)
		dispatch.POST("/client_hot_update", clientHotUpdate)
	}
}

func HandleDefault(c *gin.Context) {
	c.String(200, "Lolo!")
}

type RegionInfoRequest struct {
	Version         string `form:"version" binding:"required"`
	Version2        string `form:"version2" binding:"required"`
	AccountType     string `form:"accountType" binding:"required"`
	OS              string `form:"os" binding:"required"`
	LastLoginSDKUID string `form:"lastloginsdkuid" binding:"required"`
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

func regionInfo(c *gin.Context) {
	var req RegionInfoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	conf := config.GetGateWay()
	info := &RegionInfo{
		Status:           true,
		Message:          "success",
		GateTcpIp:        conf.GetOuterIp(),
		GateTcpPort:      conf.GetOuterPort(),
		IsServerOpen:     true,
		Text:             "",
		ClientLogTcpIp:   config.GetLogServer().GetOuterIp(),
		ClientLogTcpPort: config.GetLogServer().GetOuterPort(),
		CurrentVersion:   gdconf.GetClientVersion(req.Version),
		PhotoShareCdnUrl: "https://cdn-photo-of.inutan.com/cn_prod_main",
	}

	c.JSONP(http.StatusOK, info)
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

func clientHotUpdate(c *gin.Context) {
	var req RegionInfoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	info := &GMClientConfig{
		Status:               true,
		Message:              "success",
		HotOssUrl:            "http://cdn-of.inutan.com/Resources;https://cdn-of.inutan.com/Resources",
		CurrentVersion:       gdconf.GetClientVersion(req.Version),
		Server:               "cn_prod_main",
		SsAppId:              "c969ebf346794cc797ed6eb6c3eac089",
		SsServerUrl:          "https://te-of.inutan.com",
		OpenGm:               true,
		OpenErrorLog:         true,
		OpenNetConnectingLog: true,
		IpAddress:            c.ClientIP(),
		PayUrl:               "http://api-callback-of.inutan.com:19701",
		IsTestServer:         true,
		ErrorLogLevel:        0,
		ServerId:             "10001",
		OpenCs:               true,
	}

	c.JSONP(http.StatusOK, info)
}
