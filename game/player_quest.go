package game

import (
	"math/rand"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func questRewards(s *model.Player, rewardIds ...int32) ([]rewardItem, bool) {
	rewards := make([]rewardItem, 0)
	for _, rewardId := range rewardIds {
		if rewardId == 0 {
			continue
		}
		for _, reward := range gdconf.GetRewardItemPoolByRewardId(uint32(rewardId)) {
			if reward == nil || reward.ItemID == 0 {
				return nil, false
			}
			count := int64(reward.ItemMaxCount)
			if count <= 0 {
				count = 1
			}
			if reward.ItemMinCount > 0 && reward.ItemMaxCount > reward.ItemMinCount {
				count = int64(reward.ItemMinCount) + rand.Int63n(int64(reward.ItemMaxCount-reward.ItemMinCount)+1)
			}
			rewards = append(rewards, rewardItem{id: uint32(reward.ItemID), num: count})
		}
	}
	return rewards, true
}

func grantQuestRewards(s *model.Player, rewardId int32) []*proto.ItemDetail {
	rewards, ok := questRewards(s, rewardId)
	if !ok {
		return nil
	}
	items, _ := grantPreparedRewards(s, rewards)
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
	rewards := make([]rewardItem, 0)
	if storyId != 0 {
		for _, chapter := range gdconf.GetStoryChapters() {
			if story := chapter.StoryList[storyId]; story != nil {
				var ok bool
				rewards, ok = questRewards(s, story.Reward)
				if !ok {
					rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
					return
				}
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
		ids := []int32{info.Config.Reward}
		if quest.CompleteCount == 0 {
			ids = append(ids, info.Config.FirstReward)
		}
		var ok bool
		rewards, ok = questRewards(s, ids...)
		if !ok {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
	}
	if status := prepareRewardItems(s, rewards); status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	rsp.Status = commitPlayerReward(s, func() proto.StatusCode {
		items, status := grantPreparedRewards(s, rewards)
		if status != proto.StatusCode_StatusCode_OK {
			return status
		}
		rsp.Items = items
		if storyId != 0 {
			for _, chapter := range gdconf.GetStoryChapters() {
				if story := chapter.StoryList[storyId]; story != nil {
					questModel.RewardStory(storyId)
					quest.Status = proto.QuestStatus_QuestStatus_Finish
					s.AcceptNextMainQuest(storyId)
					break
				}
			}
		} else {
			quest.CompleteCount++
			quest.Status = proto.QuestStatus_QuestStatus_Finish
		}
		return proto.StatusCode_StatusCode_OK
	})
	if rsp.Status != proto.StatusCode_StatusCode_OK {
		rsp.Items = nil
		return
	}
	if len(rsp.Items) > 0 {
		notice := &proto.PackNotice{Status: proto.StatusCode_StatusCode_OK, Items: rsp.Items}
		g.send(s, 0, notice)
	}
	s.SetActiveTime()
}
