package sdk

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gucooing/lolo/config"
)

func (s *Server) Router() {
	s.router.Any("/", HandleDefault)

	dispatch := s.router.Group("/dispatch")
	{
		dispatch.POST("/region_info", regionInfo)
	}
}

func HandleDefault(c *gin.Context) {
	c.String(200, "BanGK!")
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
		CurrentVersion:   "2025-09-19-17-04-53_2025-11-04-14-13-58",
		PhotoShareCdnUrl: "https://cdn-photo-of.inutan.com/cn_prod_main",
	}

	c.JSONP(http.StatusOK, info)
}
