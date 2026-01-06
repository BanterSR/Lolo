package sdk

import (
	"github.com/gin-gonic/gin"
	"gucooing/lolo/config"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/quick"
	"net/http"
	"time"
)

func getNoticeList(c *gin.Context) {
	c.JSON(http.StatusOK, &quick.GetNoticeList{
		Data:       gdconf.GetNoticeList(),
		ServerTime: time.Now().Unix(),
	})
}

func getNoticeUrlList(c *gin.Context) {
	urls := make([]string, 0)
	c.JSON(http.StatusOK, &quick.GetNoticeList{
		Data:       urls,
		ServerTime: time.Now().Unix(),
	})
}

func getLoginUrlList(c *gin.Context) {
	c.String(http.StatusOK, "")
}

func regionInfo(c *gin.Context) {
	var req quick.RegionInfoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	conf := config.GetGateWay()
	info := &quick.RegionInfo{
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

func clientHotUpdate(c *gin.Context) {
	// alg.ProxyGin(c, "http://dsp-prod-of.inutan.com:18881/dispatch/client_hot_update")
	var req quick.RegionInfoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	info := &quick.GMClientConfig{
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

func getClientBlackList(c *gin.Context) {
	c.JSON(http.StatusOK, []*quick.ClientBlack{
		{ID: 100, MANUFACTURER: "RETRY_LIMITATION", MODEL: "4"},
		{ID: 600, MANUFACTURER: "HUAWEI", MODEL: ""},
		{ID: 1000, MANUFACTURER: "Samsung", MODEL: ""},
		{ID: 2000, MANUFACTURER: "Sony", MODEL: ""},
	})
}
