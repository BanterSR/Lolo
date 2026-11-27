package game

import (
	"sort"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetMails(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GetMailsRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Mails:  s.GetMailModel().Briefs(),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) OperateMails(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OperateMailsReq)
	rsp := &proto.OperateMailsRsp{
		Status:         proto.StatusCode_StatusCode_OK,
		Mails:          make([]*proto.MailBriefData, 0),
		ClaimResult:    make([]*proto.ItemDetail, 0),
		DeletedMailIds: make([]uint32, 0),
		OperateType:    req.OperateType,
	}
	switch req.OperateType {
	case proto.MailOperateType_MailOperateType_NONE:
		return
	case proto.MailOperateType_MailOperateType_READ:
		if !validMail(s, req.MailId) {
			rsp.Status = proto.StatusCode_StatusCode_MAIL_NOT_EXIST
			return
		}
		s.GetMailModel().GetState(req.MailId).Read = true
	case proto.MailOperateType_MailOperateType_DELETE_ALL_READED:
		rsp.DeletedMailIds = deleteReadMails(s)
	case proto.MailOperateType_MailOperateType_CLAIM:
		rsp.Status = commitPlayerReward(s, func() proto.StatusCode {
			items, status := claimMail(s, req.MailId)
			rsp.ClaimResult = items
			return status
		})
		if rsp.Status != proto.StatusCode_StatusCode_OK {
			rsp.ClaimResult = nil
		}
	case proto.MailOperateType_MailOperateType_CLAIM_ALL:
		rsp.Status = commitPlayerReward(s, func() proto.StatusCode {
			items, status := claimAllMails(s)
			rsp.ClaimResult = items
			return status
		})
		if rsp.Status != proto.StatusCode_StatusCode_OK {
			rsp.ClaimResult = nil
		}
	case proto.MailOperateType_MailOperateType_COLLECT:
		if !validMail(s, req.MailId) {
			rsp.Status = proto.StatusCode_StatusCode_MAIL_NOT_EXIST
			return
		}
		state := s.GetMailModel().GetState(req.MailId)
		if state.Collected {
			rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
			return
		}
		state.Collected = true
		state.Read = true
		rsp.CollectStatus = true
	default:
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
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
		return nil, proto.StatusCode_StatusCode_MAIL_NOT_EXIST
	}
	state := s.GetMailModel().GetState(mailId)
	if state.Claimed {
		return nil, proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	}
	rewards, status := mailAttachments(model.TestMails[mailId])
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	if status = prepareRewardItems(s, rewards); status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	items, status := grantPreparedRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	state.Claimed = true
	state.Read = true
	return items, proto.StatusCode_StatusCode_OK
}

func claimAllMails(s *model.Player) ([]*proto.ItemDetail, proto.StatusCode) {
	ids := make([]uint32, 0, len(model.TestMails))
	for id := range model.TestMails {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	items := make([]*proto.ItemDetail, 0)
	claimed := 0
	lastStatus := proto.StatusCode_StatusCode_MAIL_NO_REWARD
	for _, id := range ids {
		result, status := claimMail(s, id)
		if status == proto.StatusCode_StatusCode_OK {
			claimed++
			items = append(items, result...)
			continue
		}
		if status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD &&
			status != proto.StatusCode_StatusCode_MAIL_NO_REWARD {
			lastStatus = status
		}
	}
	if claimed == 0 {
		return nil, lastStatus
	}
	return items, proto.StatusCode_StatusCode_OK
}

func mailAttachments(mail *proto.MailBriefData) ([]rewardItem, proto.StatusCode) {
	if mail == nil {
		return nil, proto.StatusCode_StatusCode_MAIL_NOT_EXIST
	}
	rewards := make([]rewardItem, 0, len(mail.Items)+len(mail.Reward))
	if len(mail.Items) > 0 {
		for _, attachment := range mail.Items {
			if attachment == nil || attachment.ItemId == 0 || attachment.Num <= 0 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			rewards = append(rewards, rewardItem{id: attachment.ItemId, num: attachment.Num})
		}
	} else {
		for _, reward := range mail.Reward {
			if reward == nil || reward.MainItem == nil || reward.MainItem.ItemId == 0 {
				return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
			}
			num := int64(1)
			if base := reward.MainItem.GetBaseItem(); base != nil && base.Num > 0 {
				num = base.Num
			}
			rewards = append(rewards, rewardItem{id: reward.MainItem.ItemId, num: num})
		}
	}
	if len(rewards) == 0 {
		return nil, proto.StatusCode_StatusCode_MAIL_NO_REWARD
	}
	return rewards, proto.StatusCode_StatusCode_OK
}

func grantMailAttachments(s *model.Player, mail *proto.MailBriefData) ([]*proto.ItemDetail, proto.StatusCode) {
	rewards, status := mailAttachments(mail)
	if status != proto.StatusCode_StatusCode_OK {
		return nil, status
	}
	return grantPreparedRewards(s, rewards)
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
