package game

import (
	"strings"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

// 社交动作没有资源表冷却字段，使用服务端固定的防刷间隔。时间戳随玩家
// blob 持久化，重连不能绕过频率限制。
const socialActionCooldown = time.Second

func findSocialAction(actionID uint32) (*excel.ActionInfoConfigure, bool) {
	conf, err := profileActionConfigProvider()
	if err != nil || conf == nil || conf.GetActionInfo() == nil {
		return nil, false
	}
	for _, row := range conf.GetActionInfo().GetDatas() {
		if row != nil && row.ID > 0 && uint32(row.ID) == actionID {
			return row, true
		}
	}
	return nil, true
}

func validSocialVector(v *proto.Vector3, requireNonZero bool) bool {
	if v == nil || v.DecimalPlaces > 6 || !validSceneVector(v) {
		return false
	}
	return !requireNonZero || v.X != 0 || v.Y != 0 || v.Z != 0
}

func validPopEmojiAction(emojiID uint32) bool {
	action, configured := findSocialAction(emojiID)
	return configured && action != nil && strings.HasPrefix(action.AnimPrefab, "Action/Emoji/")
}

func validClapRequest(req *proto.ClapReq) bool {
	if req == nil {
		return false
	}
	action, configured := findSocialAction(req.MovementId)
	if !configured || action == nil || !strings.HasPrefix(action.AnimPrefab, "Action/Motion/拍手/") {
		return false
	}
	if req.IsCancel {
		return true
	}
	return validSocialVector(req.Position, false) && validSocialVector(req.Forward, true)
}

func validThrowDiceRequest(s *model.Player, req *proto.ThrowDiceReq) bool {
	if req == nil || req.DiceId == 0 || req.Pips < 1 || req.Pips > 6 || !validSocialVector(req.Position, false) {
		return false
	}
	itemConf := profileItemConfigure(req.DiceId)
	if itemConf == nil || proto.EBagItemTag(itemConf.GetNewBagItemTag()) != proto.EBagItemTag_EBagItemTag_Item {
		return false
	}
	owned := s.GetItemModel().GetItemBaseInfo(req.DiceId)
	if owned == nil || owned.ItemId != req.DiceId || owned.Num <= 0 ||
		owned.ItemType != proto.EBagItemTag_EBagItemTag_Item {
		return false
	}
	expressions, err := profileExpressionProvider()
	if err != nil || expressions == nil || expressions.GetExpression() == nil {
		return false
	}
	for _, expression := range expressions.GetExpression().GetDatas() {
		if expression == nil {
			continue
		}
		for _, group := range expression.GetExpressionGroupInfo() {
			if group == nil || group.NeedItemID != int32(req.DiceId) || group.ActionID <= 0 {
				continue
			}
			action, configured := findSocialAction(uint32(group.ActionID))
			if configured && action != nil && strings.Contains(action.AnimPrefab, "丢骰子") {
				return true
			}
		}
	}
	return false
}

func socialActionReady(profile *model.ProfileModel, last int64, now time.Time) bool {
	if profile == nil {
		return false
	}
	nowMs := now.UnixMilli()
	return last > nowMs || nowMs-last >= socialActionCooldown.Milliseconds()
}

func socialSceneChannel(g *Game, s *model.Player) *ChannelInfo {
	if g == nil || s == nil {
		return nil
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.channelInfo == nil || scenePlayer.CurScene == nil {
		return nil
	}
	return scenePlayer.channelInfo
}

func (g *Game) PopEmoji(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PopEmojiReq)
	rsp := &proto.PopEmojiRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	channel := socialSceneChannel(g, s)
	if channel == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	if !validPopEmojiAction(req.EmojiId) {
		rsp.Status = proto.StatusCode_StatusCode_ActionNotUnlock
		return
	}
	now := profileNow()
	if !socialActionReady(profile, profile.LastPopEmojiAt, now) {
		rsp.Status = proto.StatusCode_StatusCode_RequestsTooFrequent
		return
	}
	old := profile.LastPopEmojiAt
	profile.LastPopEmojiAt = now.UnixMilli()
	if err := profileSavePlayer(s); err != nil {
		profile.LastPopEmojiAt = old
		rsp.Status = proto.StatusCode_StatusCode_SysErr
		return
	}
	channel.sendAllPlayer(0, &proto.PopEmojiNotice{
		Status:       proto.StatusCode_StatusCode_Ok,
		EmojiId:      req.EmojiId,
		FromPlayerId: s.UserId,
	})
}

func (g *Game) Clap(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ClapReq)
	rsp := &proto.ClapRsp{Status: proto.StatusCode_StatusCode_Ok, MovementId: req.MovementId, IsCancel: req.IsCancel}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	channel := socialSceneChannel(g, s)
	if channel == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	if !validClapRequest(req) {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	now := profileNow()
	if !socialActionReady(profile, profile.LastClapAt, now) {
		rsp.Status = proto.StatusCode_StatusCode_RequestsTooFrequent
		return
	}
	old := profile.LastClapAt
	profile.LastClapAt = now.UnixMilli()
	if err := profileSavePlayer(s); err != nil {
		profile.LastClapAt = old
		rsp.Status = proto.StatusCode_StatusCode_SysErr
		return
	}
	if req.IsCancel {
		return
	}
	channel.sendAllPlayer(0, &proto.ClapResultNotice{
		Status:             proto.StatusCode_StatusCode_Ok,
		MatchedPlayerIdLst: []uint32{s.UserId},
		MovementId:         req.MovementId,
	})
}

func (g *Game) ThrowDice(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ThrowDiceReq)
	rsp := &proto.ThrowDiceRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	channel := socialSceneChannel(g, s)
	if channel == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	if !validThrowDiceRequest(s, req) {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	now := profileNow()
	if !socialActionReady(profile, profile.LastThrowDiceAt, now) {
		rsp.Status = proto.StatusCode_StatusCode_RequestsTooFrequent
		return
	}
	old := profile.LastThrowDiceAt
	profile.LastThrowDiceAt = now.UnixMilli()
	if err := profileSavePlayer(s); err != nil {
		profile.LastThrowDiceAt = old
		rsp.Status = proto.StatusCode_StatusCode_SysErr
		return
	}
	channel.sendAllPlayer(0, &proto.ThrowDiceNotice{
		Status:     proto.StatusCode_StatusCode_Ok,
		Position:   model.CopyVector3(req.Position),
		Pips:       req.Pips,
		PlayerName: s.NickName,
		DiceId:     req.DiceId,
	})
	channel.sendAllPlayer(0, &proto.ThrowDiceResultNotice{
		Status: proto.StatusCode_StatusCode_Ok,
		ZoneResultLst: []*proto.ThrowDiceResultListByZone{{ResultLst: []*proto.ThrowDiceResult{{
			DiceId:     req.DiceId,
			Pips:       req.Pips,
			PlayerName: s.NickName,
			WonResult:  proto.DiceWonResult_DiceWonResult_Win,
			PlayerId:   s.UserId,
		}}}},
	})
}
