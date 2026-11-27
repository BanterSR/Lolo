package gdconf

import "gucooing/lolo/protocol/excel"

type BlessingTree struct {
	all     *excel.AllBlessingTreeDatas
	TreeMap map[uint32]*excel.BlessingTreeConfigure
	InfoMap map[uint32]*excel.BlessingTreeInfoConfigure
}

func (g *GameConfig) loadBlessingTree() {
	info := &BlessingTree{
		all:     new(excel.AllBlessingTreeDatas),
		TreeMap: make(map[uint32]*excel.BlessingTreeConfigure),
		InfoMap: make(map[uint32]*excel.BlessingTreeInfoConfigure),
	}
	g.Excel.BlessingTree = info
	ReadJson(g.excelPath, "BlessingTree.json", &info.all)
	for _, v := range info.all.GetBlessingTree().GetDatas() {
		if v == nil || v.GetID() == 0 {
			continue
		}
		info.TreeMap[uint32(v.GetID())] = v
	}
	for _, v := range info.all.GetBlessingTreeInfo().GetDatas() {
		if v == nil || v.GetID() == 0 {
			continue
		}
		info.InfoMap[uint32(v.GetID())] = v
	}
}

func GetBlessingTreeConfigure(id uint32) *excel.BlessingTreeConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.BlessingTree == nil {
		return nil
	}
	return cc.Excel.BlessingTree.TreeMap[id]
}

func GetBlessingTreeInfoConfigure(id uint32) *excel.BlessingTreeInfoConfigure {
	if cc == nil || cc.Excel == nil || cc.Excel.BlessingTree == nil {
		return nil
	}
	return cc.Excel.BlessingTree.InfoMap[id]
}
