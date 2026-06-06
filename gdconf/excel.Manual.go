package gdconf

import "gucooing/lolo/protocol/excel"

type Manual struct {
	all *excel.AllManualDatas

	ManualMap           map[uint32]*excel.ManualConfigure
	ManualDungeonMap    map[uint32]*excel.ManualDungeonConfigure
	ManualShowRewardMap map[uint32]*excel.ManualShowRewardConfigure

	ManualFlagMap         map[uint32]*excel.ManualItems
	ManualSceneMap        map[uint32][]*excel.ManualItems
	ManualDungeonFlagMap  map[uint32]*excel.ManualDungeonItems
	ManualDungeonSceneMap map[uint32][]*excel.ManualDungeonItems
}

func (g *GameConfig) loadManual() {
	info := &Manual{
		all:                   new(excel.AllManualDatas),
		ManualMap:             make(map[uint32]*excel.ManualConfigure),
		ManualDungeonMap:      make(map[uint32]*excel.ManualDungeonConfigure),
		ManualShowRewardMap:   make(map[uint32]*excel.ManualShowRewardConfigure),
		ManualFlagMap:         make(map[uint32]*excel.ManualItems),
		ManualSceneMap:        make(map[uint32][]*excel.ManualItems),
		ManualDungeonFlagMap:  make(map[uint32]*excel.ManualDungeonItems),
		ManualDungeonSceneMap: make(map[uint32][]*excel.ManualDungeonItems),
	}
	g.Excel.Manual = info
	name := "Manual.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetManual().GetDatas() {
		id := uint32(v.GetID())
		if id != 0 {
			info.ManualMap[id] = v
		}
		for _, item := range v.GetManualItems() {
			flagId := uint32(item.GetFlagID())
			if flagId != 0 {
				info.ManualFlagMap[flagId] = item
			}
			sceneId := uint32(item.GetTransferSceneID())
			if sceneId != 0 {
				info.ManualSceneMap[sceneId] = append(info.ManualSceneMap[sceneId], item)
			}
		}
	}

	for _, v := range info.all.GetManualDungeon().GetDatas() {
		id := uint32(v.GetID())
		if id != 0 {
			info.ManualDungeonMap[id] = v
		}
		for _, item := range v.GetManualDungeonItems() {
			flagId := uint32(item.GetFlagID())
			if flagId != 0 {
				info.ManualDungeonFlagMap[flagId] = item
			}
			sceneId := uint32(item.GetTransferSceneID())
			if sceneId != 0 {
				info.ManualDungeonSceneMap[sceneId] = append(info.ManualDungeonSceneMap[sceneId], item)
			}
		}
	}

	for _, v := range info.all.GetManualShowReward().GetDatas() {
		id := uint32(v.GetID())
		if id != 0 {
			info.ManualShowRewardMap[id] = v
		}
	}
}

func GetManualConfigureMap() map[uint32]*excel.ManualConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualMap
}

func GetManualConfigure(id uint32) *excel.ManualConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualMap[id]
}

func GetManualDungeonConfigure(id uint32) *excel.ManualDungeonConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualDungeonMap[id]
}

func GetManualShowRewardConfigure(id uint32) *excel.ManualShowRewardConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualShowRewardMap[id]
}

func GetManualItemMap() map[uint32]*excel.ManualItems {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualFlagMap
}

func GetManualItemByFlagId(flagId uint32) *excel.ManualItems {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualFlagMap[flagId]
}

func GetSceneManualItems(sceneId uint32) []*excel.ManualItems {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualSceneMap[sceneId]
}

func GetManualDungeonItemMap() map[uint32]*excel.ManualDungeonItems {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualDungeonFlagMap
}

func GetManualDungeonItemByFlagId(flagId uint32) *excel.ManualDungeonItems {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualDungeonFlagMap[flagId]
}

func GetSceneManualDungeonItems(sceneId uint32) []*excel.ManualDungeonItems {
	if cc == nil || cc.Excel == nil || cc.Excel.Manual == nil {
		return nil
	}
	return cc.Excel.Manual.ManualDungeonSceneMap[sceneId]
}
