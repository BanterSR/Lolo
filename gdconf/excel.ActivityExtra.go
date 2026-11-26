package gdconf

import "gucooing/lolo/protocol/excel"

type ActivityInvite struct {
	all               *excel.AllActivityInviteDatas
	InviteMap         map[uint32]*excel.ActivityInviteConfigure
	QuestMap          map[uint32]*excel.ActivityInviteQuestConfigure
	RewardByCondition map[uint32][]*excel.ActivityInvitePoolRewardItems
}

func (g *GameConfig) loadActivityInvite() {
	info := &ActivityInvite{
		all:               new(excel.AllActivityInviteDatas),
		InviteMap:         make(map[uint32]*excel.ActivityInviteConfigure),
		QuestMap:          make(map[uint32]*excel.ActivityInviteQuestConfigure),
		RewardByCondition: make(map[uint32][]*excel.ActivityInvitePoolRewardItems),
	}
	g.Excel.ActivityInvite = info
	ReadJson(g.excelPath, "ActivityInvite.json", &info.all)
	for _, v := range info.all.GetActivityInvite().GetDatas() {
		if v != nil && v.GetID() != 0 {
			info.InviteMap[uint32(v.GetID())] = v
		}
	}
	for _, v := range info.all.GetActivityInviteQuest().GetDatas() {
		if v == nil || v.GetID() == 0 {
			continue
		}
		info.QuestMap[uint32(v.GetID())] = v
		for _, item := range v.GetActivityInviteQuestPoolItem() {
			if item == nil || item.GetAchieveID() == 0 {
				continue
			}
			info.RewardByCondition[uint32(item.GetAchieveID())] = item.GetActivityInvitePoolRewardItem()
		}
	}
}

func GetActivityInviteQuestReward(conditionID uint32) []*excel.ActivityInvitePoolRewardItems {
	if cc == nil || cc.Excel == nil || cc.Excel.ActivityInvite == nil {
		return nil
	}
	return cc.Excel.ActivityInvite.RewardByCondition[conditionID]
}

type InteractionActivity struct {
	all                     *excel.AllInteractionActivityDatas
	ActivityMap             map[uint32]*excel.ActivityInteractionConfigure
	RecallMap               map[uint32]*excel.RecallActivityConfigure
	RecallRewardByCondition map[uint32][]*excel.RecallActivityRewardItems
}

func (g *GameConfig) loadInteractionActivity() {
	info := &InteractionActivity{
		all:                     new(excel.AllInteractionActivityDatas),
		ActivityMap:             make(map[uint32]*excel.ActivityInteractionConfigure),
		RecallMap:               make(map[uint32]*excel.RecallActivityConfigure),
		RecallRewardByCondition: make(map[uint32][]*excel.RecallActivityRewardItems),
	}
	g.Excel.InteractionActivity = info
	ReadJson(g.excelPath, "InteractionActivity.json", &info.all)
	for _, v := range info.all.GetActivityInteraction().GetDatas() {
		if v != nil && v.GetID() != 0 {
			info.ActivityMap[uint32(v.GetID())] = v
		}
	}
	for _, v := range info.all.GetRecallActivity().GetDatas() {
		if v == nil || v.GetID() == 0 {
			continue
		}
		info.RecallMap[uint32(v.GetID())] = v
		for _, item := range v.GetRecallActivityItem() {
			if item == nil || item.GetAchieveConditionID() == 0 {
				continue
			}
			info.RecallRewardByCondition[uint32(item.GetAchieveConditionID())] = item.GetRecallActivityRewardItem()
		}
	}
}

func GetActivityInteractionConfigure(id uint32) *excel.ActivityInteractionConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.InteractionActivity == nil {
		return nil
	}
	return cc.Excel.InteractionActivity.ActivityMap[id]
}

func GetRecallActivityReward(conditionID uint32) []*excel.RecallActivityRewardItems {
	if cc == nil || cc.Excel == nil || cc.Excel.InteractionActivity == nil {
		return nil
	}
	return cc.Excel.InteractionActivity.RecallRewardByCondition[conditionID]
}
