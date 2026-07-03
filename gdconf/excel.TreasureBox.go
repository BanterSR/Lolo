package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type TreasureBox struct {
	all                      *excel.AllTreasureBoxDatas
	TreasureBoxMap           map[uint32]*excel.TreasureBoxConfigure
	CollectionTreasureBoxMap map[uint32]*excel.CollectionTreasureBoxConfigure
	EncounterTreasureBoxMap  map[uint32]*excel.EncounterTreasureBoxConfigure
	RiddleTreasureBoxMap     map[uint32]*excel.RiddleTreasureBoxConfigure
	GameTreasureBoxMap       map[uint32]*excel.GameTreasureBoxConfigure
}

func (g *GameConfig) loadTreasureBox() *TreasureBox {
	info := &TreasureBox{
		all:                      new(excel.AllTreasureBoxDatas),
		TreasureBoxMap:           make(map[uint32]*excel.TreasureBoxConfigure),
		CollectionTreasureBoxMap: make(map[uint32]*excel.CollectionTreasureBoxConfigure),
		EncounterTreasureBoxMap:  make(map[uint32]*excel.EncounterTreasureBoxConfigure),
		RiddleTreasureBoxMap:     make(map[uint32]*excel.RiddleTreasureBoxConfigure),
		GameTreasureBoxMap:       make(map[uint32]*excel.GameTreasureBoxConfigure),
	}
	g.Excel.TreasureBox = info
	name := "TreasureBox.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetTreasureBox().GetDatas() {
		info.TreasureBoxMap[uint32(v.GetID())] = v
	}
	for _, v := range info.all.GetCollectionTreasureBox().GetDatas() {
		info.CollectionTreasureBoxMap[uint32(v.GetID())] = v
	}
	for _, v := range info.all.GetEncounterTreasureBox().GetDatas() {
		info.EncounterTreasureBoxMap[uint32(v.GetID())] = v
	}
	for _, v := range info.all.GetRiddleTreasureBox().GetDatas() {
		info.RiddleTreasureBoxMap[uint32(v.GetID())] = v
	}
	for _, v := range info.all.GetGameTreasureBox().GetDatas() {
		info.GameTreasureBoxMap[uint32(v.GetID())] = v
	}

	return info
}

func GetTreasureBoxConfigure(id uint32) *excel.TreasureBoxConfigure {
	info := cc.Excel.TreasureBox
	if info == nil {
		return nil
	}
	return info.TreasureBoxMap[id]
}

func GetCollectionTreasureBoxConfigure(id uint32) *excel.CollectionTreasureBoxConfigure {
	info := cc.Excel.TreasureBox
	if info == nil {
		return nil
	}
	return info.CollectionTreasureBoxMap[id]
}

func GetEncounterTreasureBoxConfigure(id uint32) *excel.EncounterTreasureBoxConfigure {
	info := cc.Excel.TreasureBox
	if info == nil {
		return nil
	}
	return info.EncounterTreasureBoxMap[id]
}

func GetRiddleTreasureBoxConfigure(id uint32) *excel.RiddleTreasureBoxConfigure {
	info := cc.Excel.TreasureBox
	if info == nil {
		return nil
	}
	return info.RiddleTreasureBoxMap[id]
}

func GetGameTreasureBoxConfigure(id uint32) *excel.GameTreasureBoxConfigure {
	info := cc.Excel.TreasureBox
	if info == nil {
		return nil
	}
	return info.GameTreasureBoxMap[id]
}
