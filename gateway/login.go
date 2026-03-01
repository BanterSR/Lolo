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
	verified := false
	defer func() {
		req.conn.Send(0, rsp)
		if !verified {
			req.conn.Close()
		}
		g.delLoginChan <- req.SdkUid
	}()

	sdkUid := alg.S2U32(req.SdkUid)
	if !g.GetToken(req.SdkUid, req.LoginToken) {
		log.Gate.Debugf("SdkUid:%s,token verification failed", req.SdkUid)
		return
	}

	ofUser, err := db.GetOFUserBySdkUid(sdkUid)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_AccountUnauth
		log.Gate.Debugf("SdkUid:%s,get account failed err:%s", req.SdkUid, err.Error())
		return
	}

	rsp.IsServerOpen = true
	rsp.UserId = ofUser.UserId

	if ofUser.Ban {
		rsp.Text = ofUser.BanText
		rsp.BanEndTime = ofUser.BanTime.Unix()
		log.Gate.Debugf("SdkUid:%s,ban login denied reason:%s", req.SdkUid, ofUser.BanText)
		return
	}

	req.conn.SetUID(ofUser.UserId)
	log.Gate.Infof("UserId:%v platform:%s verify success, logging in...", ofUser.UserId, proto.AccountType(req.AccountType).String())

	verified = true
	go g.receive(req.conn, ofUser.UserId)
}
