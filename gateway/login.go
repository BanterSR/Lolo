package gateway

import (
	"gucooing/lolo/db"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/proto"
)

type LoginInfo struct {
	*proto.VerifyLoginTokenReq
	conn ofnet.Conn
}

func (g *Gateway) loginSessionManagement() {
	loginMap := make(map[string]*LoginInfo)
	for {
		select {
		case login := <-g.loginChan:
			if _, ok := loginMap[login.SdkUid]; ok {
				// 重复的登录请求
				continue
			}
			loginMap[login.SdkUid] = login
			g.VerifyLoginToken(login)
		case sdkUid := <-g.delLoginChan:
			delete(loginMap, sdkUid)
		}
	}
}

func (g *Gateway) VerifyLoginToken(req *LoginInfo) {
	// 由于没有sdk 所以这里同意全部登录请求
	rsp := &proto.VerifyLoginTokenRsp{
		AccountType: req.AccountType,
		SdkUid:      req.SdkUid,
		DeviceUuid:  req.DeviceUuid,
		Status:      proto.StatusCode_StatusCode_Ok,
		TimeLeft:    4294967295,
		Text:        "",
		BanEndTime:  0,

		UserId:       0,
		IsServerOpen: false,
		Os:           0,
	}
	defer func() {
		req.conn.Send(0, rsp)
		g.delLoginChan <- req.SdkUid
	}()
	sdkUid := alg.S2U32(req.SdkUid)
	// token验证
	if !g.GetToken(req.SdkUid, req.LoginToken) {
		log.Gate.Debugf("SdkUid:%s,token验证失败", req.SdkUid)
		return
	}
	ofUser, err := db.GetOFUserBySdkUid(sdkUid)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_AccountUnauth
		log.Gate.Debugf("SdkUid:%s,拉取账号失败err:%s", req.SdkUid, err.Error())
		return
	}
	rsp.IsServerOpen = true
	rsp.UserId = ofUser.UserId

	// 验证是否被ban
	if ofUser.Ban {
		rsp.Text = ofUser.BanText
		rsp.BanEndTime = ofUser.BanTime.Unix()
		log.Gate.Debugf("SdkUid:%s,因被封禁被禁止登录,封禁原因:%s", req.SdkUid, ofUser.BanText)
		return
	}

	req.conn.SetUID(ofUser.UserId)
	log.Gate.Infof("UserId:%v 平台:%s 身份验证成功正在登录中...", ofUser.UserId, proto.AccountType(req.AccountType).String())

	go g.receive(req.conn, ofUser.UserId)
}
