package gateway

import (
	"gucooing/lolo/db"
	"gucooing/lolo/game"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/proto"
)

type LoginInfo struct {
	*proto.VerifyLoginTokenReq
	conn ofnet.Conn
	uuid string
}

func (g *Gateway) loginSessionManagement() {
	loginMap := make(map[string]*LoginInfo)
	for {
		select {
		case login := <-g.loginChan:
			loginMap[login.uuid] = login
			g.VerifyLoginToken(login)
		case uuidS := <-g.delLoginChan:
			delete(loginMap, uuidS)
		}
	}
}

func (g *Gateway) VerifyLoginToken(login *LoginInfo) {
	rsp := &proto.VerifyLoginTokenRsp{
		AccountType: login.AccountType,
		SdkUid:      login.SdkUid,
		DeviceUuid:  login.DeviceUuid,
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
		login.conn.Send(0, rsp)
		if !verified {
			login.conn.Close()
		}
		g.delLoginChan <- login.uuid
	}()

	sdkUid := alg.S2U32(login.SdkUid)
	if !g.GetToken(login.SdkUid, login.LoginToken) {
		log.Gate.Debugf("SdkUid:%s,token verification failed", login.SdkUid)
		return
	}

	ofUser, err := db.GetOFUserBySdkUid(sdkUid)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_AccountUnauth
		log.Gate.Debugf("SdkUid:%s,get account failed err:%s", login.SdkUid, err.Error())
		return
	}

	rsp.IsServerOpen = true
	rsp.UserId = ofUser.UserId

	if ofUser.Ban {
		rsp.Text = ofUser.BanText
		rsp.BanEndTime = ofUser.BanTime.Unix()
		log.Gate.Debugf("SdkUid:%s,ban login denied reason:%s", login.SdkUid, ofUser.BanText)
		return
	}

	login.conn.SetUID(ofUser.UserId)
	log.Gate.Infof("UserId:%v platform:%s verify success, logging in...", ofUser.UserId, proto.AccountType(login.AccountType).String())

	verified = true
	se, ok := g.sessionMap.Get(ofUser.UserId)
	if ok {
		g.game.GetGateTask() <- &game.KillPlayer{
			UserId:     se.userId,
			UUID:       se.uuid,
			Reason:     proto.PlayerOfflineReason_PlayerOfflineReason_AnotherLogin,
			KillPlayer: false,
		}
		close(se.done)
		g.sessionMap.Del(ofUser.UserId)
	}
	se = &session{
		userId: ofUser.UserId,
		uuid:   login.uuid,
		conn:   login.conn,
		done:   make(chan struct{}),
	}
	g.sessionMap.Set(se.userId, se)

	go g.receive(se)
}
