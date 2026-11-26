package game

import (
	"math/rand"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func grantQuestRewards(s *model.Player, rewardId int32) []*proto.ItemDetail {
	items := make([]*proto.ItemDetail, 0)
	if rewardId == 0 {
		return items
	}
	for _, reward := range gdconf.GetRewardItemPoolByRewardId(uint32(rewardId)) {
		if reward == nil || reward.ItemID == 0 {
			continue
		}
		count := int64(reward.ItemMaxCount)
		if count <= 0 {
			count = 1
		}
		if reward.ItemMinCount > 0 && reward.ItemMaxCount > reward.ItemMinCount {
			count = int64(reward.ItemMinCount) + rand.Int63n(int64(reward.ItemMaxCount-reward.ItemMinCount)+1)
		}
		if ctx := s.AddAllTypeItem(uint32(reward.ItemID), count); ctx != nil {
			if item := ctx.AddItemDetail(); item != nil {
				items = append(items, item)
			}
		}
	}
	return items
}

func (g *Game) AcceptQuest(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AcceptQuestReq)
	rsp := &proto.AcceptQuestRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)
	questId := req.QuestId
	if req.QuestType == proto.EQuestType_EQuestType_MainQuest {
		for _, chapter := range gdconf.GetStoryChapters() {
			if story := chapter.StoryList[req.QuestId]; story != nil {
				questId = uint32(story.QuestID)
				if story.PreStoryUnlock != 0 && !s.GetQuestModel().IsStoryRewarded(uint32(story.PreStoryUnlock)) {
					rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
					return
				}
				break
			}
		}
	}
	info := gdconf.GetQuestInfos()[questId]
	if info == nil || (req.QuestType != proto.EQuestType_EQuestType_MainQuest && proto.EQuestType(info.Config.NewType) != req.QuestType) {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	if s.GetQuestModel().GetQuestMap()[questId] != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.Quest = s.GetQuestModel().Accept(questId)
	if rsp.Quest == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		log.Game.Errorf("UserId:%v 接取任务落库失败:%s", s.UserId, err.Error())
	}
}

func (g *Game) ClaimQuestReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ClaimQuestRewardReq)
	rsp := &proto.ClaimQuestRewardRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)
	questModel := s.GetQuestModel()
	questId := req.QuestId
	storyId := uint32(0)
	if req.QuestType == proto.EQuestType_EQuestType_MainQuest {
		storyId = req.QuestId
		for _, chapter := range gdconf.GetStoryChapters() {
			if story := chapter.StoryList[storyId]; story != nil {
				questId = uint32(story.QuestID)
				break
			}
		}
		if questModel.IsStoryRewarded(storyId) {
			rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
			return
		}
	}
	quest := questModel.GetQuestMap()[questId]
	if quest == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if quest.Status != proto.QuestStatus_QuestStatus_Completed && quest.Status != proto.QuestStatus_QuestStatus_Finish {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.Quest = quest
	if storyId != 0 {
		for _, chapter := range gdconf.GetStoryChapters() {
			if story := chapter.StoryList[storyId]; story != nil {
				rsp.Items = append(rsp.Items, grantQuestRewards(s, story.Reward)...)
				questModel.RewardStory(storyId)
				quest.Status = proto.QuestStatus_QuestStatus_Finish
				s.AcceptNextMainQuest(storyId)
				break
			}
		}
	} else {
		info := gdconf.GetQuestInfos()[questId]
		if info == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
		if info.Config.LimitNum > 0 && quest.CompleteCount >= uint32(info.Config.LimitNum) {
			rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
			return
		}
		rsp.Items = append(rsp.Items, grantQuestRewards(s, info.Config.Reward)...)
		if quest.CompleteCount == 0 {
			rsp.Items = append(rsp.Items, grantQuestRewards(s, info.Config.FirstReward)...)
		}
		quest.CompleteCount++
		quest.Status = proto.QuestStatus_QuestStatus_Finish
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		log.Game.Errorf("UserId:%v 任务奖励落库失败:%s", s.UserId, err.Error())
		return
	}
	if len(rsp.Items) > 0 {
		notice := &proto.PackNotice{Status: proto.StatusCode_StatusCode_OK, Items: rsp.Items}
		g.send(s, 0, notice)
	}
	s.SetActiveTime()
}
