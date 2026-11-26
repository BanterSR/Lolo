package model

import (
	"sort"

	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/proto"
)

type QuestModel struct {
	QuestMap        map[uint32]*proto.Quest `json:"questMap,omitempty"`
	RewardedStories map[uint32]bool         `json:"rewardedStories,omitempty"`
}

func (s *Player) GetQuestModel() *QuestModel {
	if s.Quest == nil {
		s.Quest = new(QuestModel)
	}
	return s.Quest
}

func (q *QuestModel) GetQuestMap() map[uint32]*proto.Quest {
	if q.QuestMap == nil {
		q.QuestMap = make(map[uint32]*proto.Quest)
	}
	return q.QuestMap
}

func (q *QuestModel) GetRewardedStories() map[uint32]bool {
	if q.RewardedStories == nil {
		q.RewardedStories = make(map[uint32]bool)
	}
	return q.RewardedStories
}

func NewQuest(questId uint32) *proto.Quest {
	info := gdconf.GetQuestInfos()[questId]
	if info == nil {
		return nil
	}
	quest := &proto.Quest{
		QuestId:       questId,
		Conditions:    make([]*proto.Condition, 0),
		Status:        proto.QuestStatus_QuestStatus_InProgress,
		CompleteCount: 0,
		BonusTimes:    0,
		ActivityId:    0,
	}
	if info.ConditionSetGroup != nil {
		for _, set := range info.ConditionSetGroup.QuestConditionSet {
			for _, conditionId := range set.AchieveConditionID {
				quest.Conditions = append(quest.Conditions, &proto.Condition{
					ConditionId: uint32(conditionId),
					Progress:    0,
					Status:      proto.QuestStatus_QuestStatus_InProgress,
				})
			}
		}
	}
	return quest
}

func (q *QuestModel) Accept(questId uint32) *proto.Quest {
	if quest := NewQuest(questId); quest != nil {
		q.GetQuestMap()[questId] = quest
		return quest
	}
	return nil
}

func (q *QuestModel) Complete(questId uint32) bool {
	quest := q.GetQuestMap()[questId]
	if quest == nil {
		return false
	}
	for _, condition := range quest.Conditions {
		condition.Status = proto.QuestStatus_QuestStatus_Finish
	}
	quest.Status = proto.QuestStatus_QuestStatus_Completed
	return true
}

func (q *QuestModel) IsStoryRewarded(storyId uint32) bool {
	return q.GetRewardedStories()[storyId]
}

func (q *QuestModel) RewardStory(storyId uint32) {
	q.GetRewardedStories()[storyId] = true
}

func (s *Player) AcceptFirstMainQuest() *proto.Quest {
	chapters := gdconf.GetStoryChapters()
	chapterIds := make([]uint32, 0, len(chapters))
	for chapterId := range chapters {
		chapterIds = append(chapterIds, chapterId)
	}
	sort.Slice(chapterIds, func(i, j int) bool { return chapterIds[i] < chapterIds[j] })
	for _, chapterId := range chapterIds {
		storyIds := make([]uint32, 0, len(chapters[chapterId].StoryList))
		for storyId := range chapters[chapterId].StoryList {
			storyIds = append(storyIds, storyId)
		}
		sort.Slice(storyIds, func(i, j int) bool { return storyIds[i] < storyIds[j] })
		for _, storyId := range storyIds {
			story := chapters[chapterId].StoryList[storyId]
			if story.QuestID == 0 || gdconf.GetQuestInfos()[uint32(story.QuestID)] == nil {
				continue
			}
			if proto.EQuestType(gdconf.GetQuestInfos()[uint32(story.QuestID)].Config.NewType) != proto.EQuestType_EQuestType_MainQuest {
				continue
			}
			if quest := s.GetQuestModel().Accept(uint32(story.QuestID)); quest != nil {
				return quest
			}
		}
	}
	return nil
}

func (s *Player) AcceptNextMainQuest(storyId uint32) *proto.Quest {
	for _, chapter := range gdconf.GetStoryChapters() {
		if _, ok := chapter.StoryList[storyId]; !ok {
			continue
		}
		ids := make([]uint32, 0, len(chapter.StoryList))
		for id := range chapter.StoryList {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for index, id := range ids {
			if id != storyId || index+1 >= len(ids) {
				continue
			}
			nextStory := chapter.StoryList[ids[index+1]]
			if nextStory.PreStoryUnlock != 0 && !s.GetQuestModel().IsStoryRewarded(uint32(nextStory.PreStoryUnlock)) {
				return nil
			}
			if nextStory.QuestID == 0 {
				return nil
			}
			return s.GetQuestModel().Accept(uint32(nextStory.QuestID))
		}
	}
	return nil
}

func (s *Player) GetPlayerQuestionnaireInfo() *proto.PlayerQuestionnaireInfo {
	return &proto.PlayerQuestionnaireInfo{ToFill: make([]*proto.QuestionnaireBrief, 0)}
}

func (s *Player) GetQuestDetail() *proto.QuestDetail {
	info := &proto.QuestDetail{
		Chapters:                make([]*proto.Chapter, 0),
		DailyQuestBonusDayLeft:  nil,
		DailyQuestBonusWeekLeft: nil,
		RandomQuestBonusLeft:    nil,
		Quests:                  make([]*proto.Quest, 0),
	}
	questModel := s.GetQuestModel()
	for chapterId, chapterInfo := range gdconf.GetStoryChapters() {
		chapter := &proto.Chapter{
			ChapterId:        chapterId,
			RewardedStoryIds: make([]uint32, 0),
		}
		for storyId := range chapterInfo.StoryList {
			if questModel.IsStoryRewarded(storyId) {
				chapter.RewardedStoryIds = append(chapter.RewardedStoryIds, storyId)
			}
		}
		info.Chapters = append(info.Chapters, chapter)
	}
	for _, quest := range questModel.GetQuestMap() {
		info.Quests = append(info.Quests, quest)
	}
	return info
}
