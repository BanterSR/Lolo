package game

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/proto"
)

// VerifyLoginToken 在玩家对象创建前处理，用于快速确认本地 UID 可用。
func (g *Game) VerifyLoginToken(conn ofnet.Conn, msg *alg.GameMsg) {
	req := msg.Body.(*proto.VerifyLoginTokenReq)
	rsp := &proto.VerifyLoginTokenRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		AccountType:  req.AccountType,
		SdkUid:       req.SdkUid,
		IsServerOpen: true,
		Os:           req.Os,
		DeviceUuid:   req.DeviceUuid,
	}
	if value, err := strconv.ParseUint(req.SdkUid, 10, 32); err == nil && value > 0 {
		rsp.UserId = uint32(value)
	} else {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
	}
	conn.Send(msg.PacketId, rsp)
}

func (g *Game) PlayerOffline(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.PlayerOfflineRsp{
		Status: proto.StatusCode_StatusCode_OK,
	})
}

// 本文件补齐客户端会发送、但此前没有路由的通用请求。
// 有持久状态的请求写入 Archive/Scene 模型；纯表现请求只做协议确认。

func (g *Game) AccountNameBindingCancel(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.AccountNameBindingCancelRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		PlayerId: s.UserId,
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActionStudy(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActionStudyReq)
	if req.ActionId > 0 {
		s.GetArchive().SetArchiveMap("actionStudy:"+strconv.Itoa(int(req.ActionId)), "1")
	}
	g.send(s, msg.PacketId, &proto.ActionStudyRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) ClientLogAuth(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.ClientLogAuthRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) ClientLogMessage(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ClientLogMessageReq)
	rsp := &proto.ClientLogMessageRsp{
		Status:             proto.StatusCode_StatusCode_OK,
		Logs:               req.Logs,
		RemoveTemplateHash: req.RemoveTemplateHash,
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) Report(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.ReportRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) UnfreezePlayerAction(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.UnfreezePlayerActionRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) MultipleActionAccept(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.MultipleActionAcceptRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) GenericSceneA(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GenericSceneAReq)
	rsp := &proto.GenericSceneARsp{
		Status:       proto.StatusCode_StatusCode_OK,
		GenericMsgId: req.GenericMsgId,
		Params:       req.Params,
		ParamMap:     req.ParamMap,
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) SPOperate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SPOperateReq)
	archive := s.GetArchive()
	rsp := &proto.SPOperateRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		OperateType: req.OperateType,
	}
	hash := func(value string) string {
		sum := sha256.Sum256([]byte(value))
		return hex.EncodeToString(sum[:])
	}
	password := archive.GetArchiveValue("sp:password")
	switch req.OperateType {
	case proto.SPOperateType_SPOperateType_Open:
		if req.Password == "" {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			break
		}
		archive.SetArchiveMap("sp:password", hash(req.Password))
		archive.SetArchiveMap("sp:open", "1")
	case proto.SPOperateType_SPOperateType_Change:
		if password == "" || hash(req.OldPassword) != password || req.Password == "" {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			break
		}
		archive.SetArchiveMap("sp:password", hash(req.Password))
		archive.SetArchiveMap("sp:open", "1")
	case proto.SPOperateType_SPOperateType_Check:
		if password == "" || hash(req.Password) != password {
			archive.SetArchiveMap("sp:check", "0")
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			break
		}
		archive.SetArchiveMap("sp:check", "1")
	case proto.SPOperateType_SPOperateType_Close:
		archive.SetArchiveMap("sp:open", "0")
		archive.SetArchiveMap("sp:check", "0")
	default:
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
	}
	rsp.SpOpenStatus = archive.GetArchiveValue("sp:open") == "1"
	rsp.SpCheckStatus = archive.GetArchiveValue("sp:check") == "1"
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) DailyTaskExchange(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.DailyTaskExchangeRsp{
		Status:            proto.StatusCode_StatusCode_OK,
		Rewards:           make([]*proto.ItemDetail, 0),
		TodayConverted:    0,
		ExchangeTimesLeft: 0,
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) DoLikes(s *model.Player, msg *alg.GameMsg) {
	const dailyLimit = 10
	archive := s.GetArchive()
	key := "likes:" + strconv.FormatInt(time.Now().Unix()/86400, 10)
	used, _ := strconv.ParseInt(archive.GetArchiveValue(key), 10, 64)
	if used < dailyLimit {
		used++
		archive.SetArchiveMap(key, strconv.FormatInt(used, 10))
	}
	left := int64(dailyLimit) - used
	if left < 0 {
		left = 0
	}
	g.send(s, msg.PacketId, &proto.DoLikesRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		LeftLikeNums: uint32(left),
		LikeNums:     used,
	})
}

func (g *Game) AreaLevelUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AreaLevelUpReq)
	rsp := &proto.AreaLevelUpRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		Items:    make([]*proto.ItemDetail, 0),
		Achieves: make([]*proto.Achieve, 0),
	}
	if req.AreaId == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		g.send(s, msg.PacketId, rsp)
		return
	}
	archive := s.GetArchive()
	key := "areaLevel:" + strconv.Itoa(int(req.AreaId))
	level, _ := strconv.ParseUint(archive.GetArchiveValue(key), 10, 32)
	level++
	archive.SetArchiveMap(key, strconv.Itoa(int(level)))
	rsp.Area = &proto.AreaData{
		AreaId:    req.AreaId,
		AreaState: proto.AreaData_Unlock,
		Level:     uint32(level),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) ChallengeStateUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChallengeStateUpdateReq)
	rsp := &proto.ChallengeStateUpdateRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	if req.ChallengeId == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		g.send(s, msg.PacketId, rsp)
		return
	}
	s.GetArchive().SetArchiveMap(
		"challenge:"+strconv.Itoa(int(req.ChallengeId)),
		strconv.Itoa(int(req.BattleState))+":"+strconv.Itoa(int(req.UseTime)),
	)
	rsp.ChallengeData = &proto.ChallengeData{
		ChallengeId: req.ChallengeId,
		UseTime:     req.UseTime,
		State:       req.BattleState,
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) RiddleStateUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.RiddleStateUpdateReq)
	rsp := &proto.RiddleStateUpdateRsp{
		Status: proto.StatusCode_StatusCode_OK,
		RiddleData: &proto.RiddleData{
			RiddleId: req.RiddleId,
			IsWin:    req.IsWin,
			State:    req.BattleState,
		},
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) SceneTransmit(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.SceneTransmitRsp{Status: proto.StatusCode_StatusCode_OK}
	if scenePlayer := g.getWordInfo().getScenePlayer(s); scenePlayer != nil && scenePlayer.CurScene != nil {
		rsp.SceneId = scenePlayer.CurScene.GetSceneId()
		rsp.ChannelId = scenePlayer.ChannelId
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) TransmitScene(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TransmitSceneReq)
	rsp := &proto.TransmitSceneRsp{Status: proto.StatusCode_StatusCode_OK}
	target := g.getWordInfo().getScenePlayerByUserId(req.FriendId)
	if target == nil || target.CurScene == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		g.send(s, msg.PacketId, rsp)
		return
	}
	rsp.SceneId = target.CurScene.GetSceneId()
	rsp.ChannelId = target.ChannelId
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) CollectCoin(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CollectCoinReq)
	rsp := &proto.CollectCoinRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		CoinIndex: req.CoinIndex,
		Rewards:   make([]*proto.ItemDetail, 0),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) CollectVoiceRegion(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CollectVoiceRegionReq)
	g.send(s, msg.PacketId, &proto.CollectVoiceRegionRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		VoiceId: req.VoiceId,
	})
}

func (g *Game) PlaceCampFire(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlaceCampFireReq)
	campFire := &proto.CampFire{
		Pos:      req.Pos,
		Rot:      req.Rot,
		Id:       s.UserId,
		PlayerId: s.UserId,
		ActionId: req.ActionId,
	}
	g.send(s, msg.PacketId, &proto.PlaceCampFireRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		CampFire: campFire,
	})
}

func (g *Game) RemoveCampFire(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.RemoveCampFireRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) PlaceMoonSpot(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.PlaceMoonSpotRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) EmotionMoonInteract(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.EmotionMoonInteractReq)
	s.GetArchive().SetArchiveMap("emotionMoon:"+strconv.Itoa(int(req.EmotionMoonId)), "1")
	g.send(s, msg.PacketId, &proto.EmotionMoonInteractRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Info:   &proto.EmotionMoonInfo{MoonId: req.EmotionMoonId, Interacted: true},
	})
}

func (g *Game) SetRoomDecor(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SetRoomDecorReq)
	for _, decor := range req.DecorList {
		s.GetArchive().SetArchiveMap(
			"roomDecor:"+strconv.Itoa(int(decor.DecorId)),
			strconv.Itoa(int(decor.PosId))+":"+strconv.Itoa(int(decor.ItemId)),
		)
	}
	g.send(s, msg.PacketId, &proto.SetRoomDecorRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) CharacterDead(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.CharacterDeadRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) CharacterRevive(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.CharacterReviveRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) BuyGameCoin(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.BuyGameCoinRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		ItemDetails: make([]*proto.ItemDetail, 0),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) useCodeOnce(s *model.Player, code, namespace string) bool {
	if code == "" {
		return false
	}
	key := namespace + ":" + code
	if s.GetArchive().GetArchiveValue(key) != "" {
		return false
	}
	s.GetArchive().SetArchiveMap(key, "1")
	return true
}

func (g *Game) UseRedemptionCode(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UseRedemptionCodeReq)
	rsp := &proto.UseRedemptionCodeRsp{Status: proto.StatusCode_StatusCode_OK, Code: req.Code}
	if !g.useCodeOnce(s, req.Code, "redeem") {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) InviteCodeUse(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.InviteCodeUseReq)
	rsp := &proto.InviteCodeUseRsp{Status: proto.StatusCode_StatusCode_OK}
	if !g.useCodeOnce(s, req.Code, "invite") {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) RecallCodeUse(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.RecallCodeUseReq)
	rsp := &proto.RecallCodeUseRsp{Status: proto.StatusCode_StatusCode_OK}
	if !g.useCodeOnce(s, req.Code, "recall") {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) InheritCodeUse(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.InheritCodeUseReq)
	rsp := &proto.InheritCodeUseRsp{Status: proto.StatusCode_StatusCode_OK, CodeStr: req.Code}
	if !g.useCodeOnce(s, req.Code, "inherit") {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetExpressAddress(s *model.Player, msg *alg.GameMsg) {
	archive := s.GetArchive()
	rsp := &proto.GetExpressAddressRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		Name:    archive.GetArchiveValue("express:name"),
		Phone:   archive.GetArchiveValue("express:phone"),
		Address: archive.GetArchiveValue("express:address"),
		OrderNo: archive.GetArchiveValue("express:order"),
		Items:   make([]*proto.BaseItem, 0),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) UploadExpressAddress(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UploadExpressAddressReq)
	archive := s.GetArchive()
	archive.SetArchiveMap("express:name", req.Name)
	archive.SetArchiveMap("express:phone", req.Phone)
	archive.SetArchiveMap("express:address", req.Address)
	rsp := &proto.UploadExpressAddressRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		Items:   req.Items,
		OrderNo: archive.GetArchiveValue("express:order"),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) UploadFGBytes(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UploadFGBytesReq)
	if len(req.Info) > 0 {
		s.GetArchive().SetArchiveMap("fg:bytes", strconv.Itoa(len(req.Info)))
	}
	g.send(s, msg.PacketId, &proto.UploadFGBytesRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) PlayerVitalityBuy(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PlayerVitalityBuyRsp{
		Status:         proto.StatusCode_StatusCode_OK,
		VitalityBuyNum: 0,
		Items:          make([]*proto.ItemDetail, 0),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) UseItemFriendIntimacy(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UseItemFriendIntimacyReq)
	g.send(s, msg.PacketId, &proto.UseItemFriendIntimacyRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		FriendId: req.FriendId,
		Intimacy: 0,
	})
}
