package sdk

import (
	"github.com/gin-gonic/gin"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/quick"
)

func systemInitV1(c *gin.Context) {
	req := new(quick.SystemInitRequest)
	rsp := quick.NewResponse()
	defer c.JSON(200, rsp)
	if err := alg.DecryptedData(c, &req); err != nil {
		rsp.SetError("解密失败")
		return
	}
	rsp.SetData(&quick.SystemInitResultV1{
		PayTypes: []*quick.PayType{
			{
				PayTypeId: "226",
				Sort:      "0",
				BackupGid: "",
				PayName:   "微信支付",
				Rebate: &quick.Rebate{
					Rate:       1,
					RateConfig: make([]interface{}, 0),
				},
			},
			{
				PayTypeId: "1",
				Sort:      "1",
				BackupGid: "",
				PayName:   "支付宝快捷",
				Rebate: &quick.Rebate{
					Rate:       1,
					RateConfig: make([]interface{}, 0),
				},
			},
		},
		Version: &quick.PtVer{
			VersionName: "empty",
			VersionNo:   0,
			VersionUrl:  "empty",
			UpdateTime:  "empty",
			IsMust:      "empty",
			UpdateTips:  "empty",
		},
		RealnameNode: "2",
		ProductConfig: &quick.PtConfig{
			UseServiceCenter:  "2",
			Logo:              "",
			UseSms:            "1",
			UseBBS:            "",
			Gift:              "",
			IsShowFloat:       "0",
			AutoOpenAgreement: "1",
			MainLoginType:     "3",
			UcentUrl:          "http://sdkapi-of.inutan.com/userCenter/play",
			UseCpLogin:        "0",
			FloatLogo:         "",
			FcmTips: &quick.FcmTips{
				NoAdultLogoutTip: "根据法规管控，当前为防沉迷管控时间，您将被强制下线。",
				GuestLoginTip:    "根据国家新闻出版署下发《关于进一步严格管理 切实防止未成年人沉迷网络游戏的通知》，严格限制向未成年人提供网络游戏服务的时间，所有网络游戏企业仅可在周五、周六、周日和法定节假日每日20时至21时向未成年人提供1小时服务，其他时间均不得以任何形式向未成年人提供网络游戏服务。",
				MinorLoginTip:    "根据国家新闻出版署下发《关于进一步严格管理 切实防止未成年人沉迷网络游戏的通知》，严格限制向未成年人提供网络游戏服务的时间，所有网络游戏企业仅可在周五、周六、周日和法定节假日每日20时至21时向未成年人提供1小时服务，其他时间均不得以任何形式向未成年人提供网络游戏服务。",
				GuestTimeTip:     "",
				MinorTimeTip:     "",
			},
			Theme:           "FF7E0C",
			UseAppAuth:      "0",
			SwitchWxAppPlug: "1",
			BanshuSwitch:    "0",
			RmAccountLg:     "0",
			RegVerifyCode:   "1",
			JoinQQGroup:     new(quick.JoinQQGroup),
			DisFastReg:      "1",
			NoPassWallet:    "0",
			HideMyFunc: &quick.HideMyFunc{
				HideRegBtn: 1,
			},
			SkinStyle: "0",
			RmGuestLg: 0,
		},
		UseEWallet: "0",
		AppAuthInfo: &quick.AppAuthInfo{
			Theme: "FF7E0C",
		},
		ClientIp: c.ClientIP(),
	})
}

func systemInitV2(c *gin.Context) {
	req := new(quick.SystemInitRequest)
	rsp := quick.NewResponse()
	defer c.JSON(200, rsp)
	if err := alg.DecryptedData(c, &req); err != nil {
		rsp.SetError("解密失败")
		return
	}
	rsp.SetData(&quick.SystemInitResultV2{
		ClientIp: c.ClientIP(),
		PtConfig: &quick.PtConfig{
			UseSms: "1",
			FcmTips: &quick.FcmTips{
				NoAdultLogoutTip: "根据法规管控，当前为防沉迷管控时间，您将被强制下线。",
			},
			JoinQQGroup: new(quick.JoinQQGroup),
		},
		PtVer: &quick.PtVer{
			VersionName: "empty",
			VersionNo:   0,
			VersionUrl:  "empty",
			UpdateTime:  "empty",
			IsMust:      "empty",
			UpdateTips:  "empty",
		},
		RealnameNode: "2",
	})
}
