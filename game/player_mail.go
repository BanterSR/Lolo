package game

import (
	"sort"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

// mailGrantItem 将邮件附件发放到玩家背包。单独抽出便于在无配置表的
// 单元测试中替换为内存发放。
var mailGrantItem = func(s *model.Player, itemID uint32, num int64) *proto.ItemDetail {
	if s == nil || itemID == 0 || num <= 0 {
		return nil
	}
	ctx := s.AddAllTypeItem(itemID, num)
	if ctx == nil {
		return nil
	}
	return ctx.AddItemDetail()
}

func (g *Game) GetMails(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GetMailsRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Mails:  s.GetMailModel().Briefs(),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) OperateMails(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OperateMailsReq)
	rsp := &proto.OperateMailsRsp{
		Status:         proto.StatusCode_StatusCode_Ok,
		Mails:          make([]*proto.MailBriefData, 0),
		ClaimResult:    make([]*proto.ItemDetail, 0),
		DeletedMailIds: make([]uint32, 0),
		OperateType:    req.OperateType,
	}
	switch req.OperateType {
	case proto.MailOperateType_MailOperateType_None:
		return
	case proto.MailOperateType_MailOperateType_Read:
		if !validMail(s, req.MailId) {
			rsp.Status = proto.StatusCode_StatusCode_MailNotExist
			return
		}
		s.GetMailModel().GetState(req.MailId).Read = true
	case proto.MailOperateType_MailOperateType_DeleteAllReaded:
		rsp.DeletedMailIds = deleteReadMails(s)
	case proto.MailOperateType_MailOperateType_Claim:
		items, status := claimMail(s, req.MailId)
		rsp.Status = status
		rsp.ClaimResult = items
	case proto.MailOperateType_MailOperateType_ClaimAll:
		items, status := claimAllMails(s)
		rsp.Status = status
		rsp.ClaimResult = items
	case proto.MailOperateType_MailOperateType_Collect:
		if !validMail(s, req.MailId) {
			rsp.Status = proto.StatusCode_StatusCode_MailNotExist
			return
		}
		state := s.GetMailModel().GetState(req.MailId)
		if state.Collected {
			rsp.Status = proto.StatusCode_StatusCode_RepeatGetReward
			return
		}
		state.Collected = true
		state.Read = true
		rsp.CollectStatus = true
	default:
		rsp.Status = proto.StatusCode_StatusCode_BadReq
	}

	rsp.Mails = s.GetMailModel().Briefs()
	g.send(s, msg.PacketId, rsp)
	if len(rsp.ClaimResult) > 0 {
		g.PackNoticeByItems(s, rsp.ClaimResult)
	}
}

func validMail(s *model.Player, mailId uint32) bool {
	if s == nil || mailId == 0 {
		return false
	}
	if model.TestMails[mailId] == nil {
		return false
	}
	return !s.GetMailModel().IsDeleted(mailId)
}

func claimMail(s *model.Player, mailId uint32) ([]*proto.ItemDetail, proto.StatusCode) {
	if !validMail(s, mailId) {
		return nil, proto.StatusCode_StatusCode_MailNotExist
	}
	state := s.GetMailModel().GetState(mailId)
	if state.Claimed {
		return nil, proto.StatusCode_StatusCode_RepeatGetReward
	}
	items, status := grantMailAttachments(s, model.TestMails[mailId])
	if status != proto.StatusCode_StatusCode_Ok {
		return nil, status
	}
	state.Claimed = true
	state.Read = true
	return items, proto.StatusCode_StatusCode_Ok
}

func claimAllMails(s *model.Player) ([]*proto.ItemDetail, proto.StatusCode) {
	ids := make([]uint32, 0, len(model.TestMails))
	for id := range model.TestMails {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	items := make([]*proto.ItemDetail, 0)
	claimed := 0
	lastStatus := proto.StatusCode_StatusCode_MailNoReward
	for _, id := range ids {
		result, status := claimMail(s, id)
		if status == proto.StatusCode_StatusCode_Ok {
			claimed++
			items = append(items, result...)
			continue
		}
		if status != proto.StatusCode_StatusCode_RepeatGetReward &&
			status != proto.StatusCode_StatusCode_MailNoReward {
			lastStatus = status
		}
	}
	if claimed == 0 {
		return nil, lastStatus
	}
	return items, proto.StatusCode_StatusCode_Ok
}

func grantMailAttachments(s *model.Player, mail *proto.MailBriefData) ([]*proto.ItemDetail, proto.StatusCode) {
	if mail == nil {
		return nil, proto.StatusCode_StatusCode_MailNotExist
	}
	items := make([]*proto.ItemDetail, 0, len(mail.Items)+len(mail.Reward))
	if len(mail.Items) > 0 {
		for _, attachment := range mail.Items {
			if attachment == nil || attachment.ItemId == 0 || attachment.Num <= 0 {
				continue
			}
			item := mailGrantItem(s, attachment.ItemId, attachment.Num)
			if item == nil {
				return nil, proto.StatusCode_StatusCode_CfgNotFound
			}
			items = append(items, item)
		}
	} else {
		for _, reward := range mail.Reward {
			if reward == nil || reward.MainItem == nil || reward.MainItem.ItemId == 0 {
				continue
			}
			num := int64(1)
			if base := reward.MainItem.GetBaseItem(); base != nil && base.Num > 0 {
				num = base.Num
			}
			item := mailGrantItem(s, reward.MainItem.ItemId, num)
			if item == nil {
				return nil, proto.StatusCode_StatusCode_CfgNotFound
			}
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return nil, proto.StatusCode_StatusCode_MailNoReward
	}
	return items, proto.StatusCode_StatusCode_Ok
}

func deleteReadMails(s *model.Player) []uint32 {
	mailModel := s.GetMailModel()
	states := mailModel.States
	deleted := make([]uint32, 0, len(states))
	for id, state := range states {
		if state == nil || (!state.Read && !state.Claimed) {
			continue
		}
		if model.TestMails[id] == nil || mailModel.IsDeleted(id) {
			continue
		}
		mailModel.Delete(id)
		deleted = append(deleted, id)
	}
	sort.Slice(deleted, func(i, j int) bool { return deleted[i] < deleted[j] })
	return deleted
}
