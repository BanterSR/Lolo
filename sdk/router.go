package sdk

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gucooing/lolo/pkg/alg"
)

func (s *Server) Router() {
	s.router.Any("/", HandleDefault)

	s.router.Any("/Resources/*path", resources)
	s.router.GET("/resolve", resolve)
	s.router.GET("/config", teConfig)
	s.router.POST("/sync", teConfig)

	dispatch := s.router.Group("/dispatch")
	{
		dispatch.Any("/get_notice_list", getNoticeList)
		dispatch.Any("/get_notice_url_list", getNoticeUrlList)
		dispatch.POST("/region_info", regionInfo)
		dispatch.HEAD("/region_info", HandleDefault)
		dispatch.POST("/client_hot_update", clientHotUpdate)
		dispatch.POST("/get_login_url_list", getLoginUrlList)
		dispatch.GET("/get_client_black_list", getClientBlackList)
	}
	s.router.POST("/v3/bind", bindTest)
	v1 := s.router.Group("/v1", alg.AutoCryptoMiddlewareV1())
	{
		system := v1.Group("/system")
		{
			system.POST("/init", systemInitV1)
			// system.POST("/getNotice", getNoticeV1)
		}
		user := v1.Group("/user")
		{
			user.POST("/loginByName", s.loginByNameV1)
			user.POST("/autoLogin", s.autoLoginV1)
		}
		auth := v1.Group("/auth")
		{
			auth.POST("/getUserInfo", s.getUserInfoV1)
			auth.POST("/asyUonline", s.asyUonlineV1)
		}
	}

	v2 := s.router.Group("/v2", alg.AutoCryptoMiddlewareV2())
	{
		system := v2.Group("/system")
		{
			system.POST("/init", systemInitV2)
		}
		user := v2.Group("/user")
		{
			user.POST("/loginByName", s.loginByNameV2)
			user.POST("/autoLogin", s.autoLoginV2)
		}
	}
}

func HandleDefault(c *gin.Context) {
	c.String(200, "Lolo!")
}

func resources(c *gin.Context) {
	path := c.Param("path")
	url := "http://cdn-of.inutan.com/Resources" + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()
	c.DataFromReader(resp.StatusCode, resp.ContentLength,
		resp.Header.Get("Content-Type"), resp.Body, nil)
}

type ResolveInfo struct {
	Host string   `json:"host"`
	Ttl  int      `json:"ttl"`
	Ips  []string `json:"ips"`
	Cip  string   `json:"cip"`
	Cl   []int    `json:"cl"`
}

func resolve(c *gin.Context) {
	domain := c.Query("domain")

	c.JSON(http.StatusOK, &ResolveInfo{
		Host: domain,
		Ttl:  60,
		Ips:  make([]string, 0),
		Cip:  c.ClientIP(),
		Cl:   []int{3},
	})
}

func teConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"server_timestamp": time.Now().UnixMilli(),
			"sync_batch_size":  100,
			"sync_interval":    90,
		},
		"msg": "",
	})
}

func bindTest(c *gin.Context) {
	// reqBase64, _ := io.ReadAll(c.Request.Body)
	// reqBytes, _ := base64.StdEncoding.DecodeString(string(reqBase64))
	// rsaLen := binary.BigEndian.Uint32(reqBytes[:4])
	// rsaBytes := reqBytes[4 : 4+rsaLen]
	//
	// aesLen := binary.BigEndian.Uint32(reqBytes[4+rsaLen : 4+rsaLen+4])
	// aesBytes := reqBytes[4+rsaLen+4 : 4+rsaLen+4+aesLen]
	//
	// log.Printf("rsaLen:%d rsaBytes:%s \n aesLen:%d aesBytes:%s\n",
	// 	rsaLen, hex.EncodeToString(rsaBytes), aesLen, hex.EncodeToString(aesBytes))
	//
	// rsa := flyrsa.NewFlyRSA(1024)
	// e := new(big.Int)
	// n := new(big.Int)
	// d := new(big.Int)
	// e.SetString("9cbd92ccef123be840deec0c6ed0547194c1e471d11b6f375e56038458fb18833e5bab2e1206b261495d7e2d1d9e5aa859e6d4b671a8ca5d78efede48e291a3f", 16)
	// n.SetString("a387f05b88acf4898fb76054412d552b80160e6947b00153046fab67d49d97274839358a9182c30f6df4e2cbc461ed1e3721922c4034ba2ac38fe2258ae0a9d14f032fe5068d35d097bafbb9d4c020fbf921ab0b723bcfbcb804e51a23305da0cb9112855f0f4658a69ea78106692107793e1537dc2636a014a83cbc442a709b", 16)
	// d.SetString("3da02c026e5f807f15782f5fc09ddf131a050a40c09b1e0897cde76a1183dc6950c84008cb825a426c49b0ab297b6ff3e1117a695d39d91ea49a15d93fd125d281fa64a940ef85cf2fdf9641f0fb9bb39d83b7090cb5f5c757524bd20902b4408ae86eb974a5dafaddaf8e08f552db63d3396b088608643bf88cd34adec004ff", 16)
	//
	// aesKey, err := rsa.Decode(rsaBytes, d, n)
	// if err != nil {
	// 	fmt.Printf("解密失败: %v\n", err)
	// } else {
	// 	fmt.Printf("解密结果AesKey: %s\n", string(aesKey))
	// }
	//
	// data := &flyrsa.Data{}
	// req, err := data.AES128Decode(aesKey, aesBytes)
	// if err != nil {
	// 	fmt.Printf("AES解密失败: %v\n", err)
	// } else {
	// 	fmt.Printf("AES解密结果req: %s\n", string(req))
	// }
}
