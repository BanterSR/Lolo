package gdconf

import "gucooing/lolo/protocol/excel"

type AchieveQuest struct {
	all      *excel.AllAchieveQuestDatas
	QuestMap map[uint32]*excel.AchieveQuestConfigure
	GroupMap map[uint32]*excel.AchieveQuestGroupConfigure
}

func (g *GameConfig) loadAchieveQuest() {
	info := &AchieveQuest{
		all:      new(excel.AllAchieveQuestDatas),
		QuestMap: make(map[uint32]*excel.AchieveQuestConfigure),
		GroupMap: make(map[uint32]*excel.AchieveQuestGroupConfigure),
	}
	g.Excel.AchieveQuest = info
	ReadJson(g.excelPath, "AchieveQuest.json", &info.all)
	for _, v := range info.all.GetAchieveQuest().GetDatas() {
		if v != nil && v.GetID() != 0 {
			info.QuestMap[uint32(v.GetID())] = v
		}
	}
	for _, v := range info.all.GetAchieveQuestGroup().GetDatas() {
		if v != nil && v.GetID() != 0 {
			info.GroupMap[uint32(v.GetID())] = v
		}
	}
}

func GetAchieveQuestGroup(id uint32) *excel.AchieveQuestGroupConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.AchieveQuest == nil {
		return nil
	}
	return cc.Excel.AchieveQuest.GroupMap[id]
}

func GetAchieveQuestGroups() map[uint32]*excel.AchieveQuestGroupConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.AchieveQuest == nil {
		return nil
	}
	return cc.Excel.AchieveQuest.GroupMap
}
