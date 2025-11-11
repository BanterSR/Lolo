package model

import (
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (s *Player) GetQuestDetail() *proto.QuestDetail {
	info := &proto.QuestDetail{
		Chapters:                make([]*proto.Chapter, 0),
		DailyQuestBonusDayLeft:  nil,
		DailyQuestBonusWeekLeft: nil,
		RandomQuestBonusLeft:    nil,
		Quests:                  make([]*proto.Quest, 0),
	}
	for chapterId, chapterInfo := range gdconf.GetStoryChapters() {
		chapter := &proto.Chapter{
			ChapterId:        chapterId,
			RewardedStoryIds: make([]uint32, 0),
		}
		for StoryId, _ := range chapterInfo.StoryList {
			chapter.RewardedStoryIds = append(chapter.RewardedStoryIds, StoryId)
		}
		alg.AddList(&info.Chapters, chapter)
	}
	for questId, questInfo := range gdconf.GetQuestInfos() {
		// if proto.EQuestType(questInfo.Config.NewType) == proto.EQuestType_EQuestType_MainQuest {
		// 	continue
		// }
		quest := &proto.Quest{
			QuestId:       questId,
			Conditions:    make([]*proto.Condition, 0),
			Status:        proto.QuestStatus_QuestStatus_Finish,
			CompleteCount: 1,
			BonusTimes:    1,
			ActivityId:    0,
		}
		for _, set := range questInfo.ConditionSetGroup.QuestConditionSet {
			for _, achieveConditionID := range set.AchieveConditionID {
				alg.AddList(&quest.Conditions, &proto.Condition{
					ConditionId: uint32(achieveConditionID),
					Progress:    1,
					Status:      proto.QuestStatus_QuestStatus_Finish,
				})
			}
		}
		alg.AddList(&info.Quests, quest)
	}

	return info
}

func (s *Player) GetPlayerQuestionnaireInfo() *proto.PlayerQuestionnaireInfo {
	return &proto.PlayerQuestionnaireInfo{
		ToFill: make([]*proto.QuestionnaireBrief, 0),
	}
}
