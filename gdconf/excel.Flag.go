package gdconf

import "gucooing/lolo/protocol/excel"

type Flag struct {
	all          *excel.AllFlagDatas
	FlagMap      map[uint32]*excel.FlagConfigure
	SceneFlagMap map[uint32][]*excel.FlagConfigure
}

func (g *GameConfig) loadFlag() {
	info := &Flag{
		all:          new(excel.AllFlagDatas),
		FlagMap:      make(map[uint32]*excel.FlagConfigure),
		SceneFlagMap: make(map[uint32][]*excel.FlagConfigure),
	}
	g.Excel.Flag = info
	name := "Flag.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetFlag().GetDatas() {
		id := uint32(v.GetID())
		if id != 0 {
			info.FlagMap[id] = v
		}
		sceneId := uint32(v.GetSceneID())
		if sceneId != 0 {
			info.SceneFlagMap[sceneId] = append(info.SceneFlagMap[sceneId], v)
		}
	}
}

func GetFlagConfigureMap() map[uint32]*excel.FlagConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Flag == nil {
		return nil
	}
	return cc.Excel.Flag.FlagMap
}

func GetFlagConfigure(id uint32) *excel.FlagConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Flag == nil {
		return nil
	}
	return cc.Excel.Flag.FlagMap[id]
}

func GetSceneFlagConfigure(sceneId uint32) []*excel.FlagConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.Flag == nil {
		return nil
	}
	return cc.Excel.Flag.SceneFlagMap[sceneId]
}
