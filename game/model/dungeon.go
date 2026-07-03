package model

import (
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type DungeonModel struct {
	DungeonMap map[uint32]*DungeonInfo // 副本存档数据
}

func (s *Player) GetDungeonModel() *DungeonModel {
	if s.Dungeon == nil {
		s.Dungeon = new(DungeonModel)
	}
	return s.Dungeon
}

func (m *DungeonModel) GetDungeonMap() map[uint32]*DungeonInfo {
	if m.DungeonMap == nil {
		m.DungeonMap = make(map[uint32]*DungeonInfo)
	}
	return m.DungeonMap
}

func (m *DungeonModel) GetDungeonInfo(dungeonId uint32) *DungeonInfo {
	list := m.GetDungeonMap()
	info, ok := list[dungeonId]
	if !ok {
		info = &DungeonInfo{
			DungeonID: dungeonId,
		}
		list[dungeonId] = info
	}
	return info
}

// 副本数据
type DungeonInfo struct {
	DungeonID       uint32                  `json:"dungeon_id"`        // 副本id
	TeamInfo        TeamInfo                `json:"team_info"`         // 队伍信息
	AllTaskFinished bool                    `json:"all_task_finished"` // 是否完成全部任务
	LastEnterTime   int64                   `json:"last_enter_time"`   // 上次进入时间
	LastFinishTime  int64                   `json:"last_finish_time"`  // 上次完成时间
	Pos             *proto.Vector3          `json:"pos"`
	Rot             *proto.Vector3          `json:"rot"`
	TreasureBoxes   map[uint32]*TreasureBox `json:"treasure_boxes"` // 宝箱状态
}

func (d *DungeonInfo) GetTreasureBoxes() map[uint32]*TreasureBox {
	if d.TreasureBoxes == nil {
		d.TreasureBoxes = make(map[uint32]*TreasureBox)
		conf := gdconf.GetDungeonSceneInfo(d.DungeonID)
		if conf == nil {
			log.Game.Warnf("[DungeonID:%d]未找到有效的副本场景配置", d.DungeonID)
			return d.TreasureBoxes
		}
		for _, cti := range conf.Info.GetCollectionTreasureInfos() {
			info := &TreasureBox{
				Index:           uint32(cti.Index),
				BoxId:           uint32(cti.ID),
				Type:            0,
				State:           proto.TreasureBoxState_TreasureBoxState_Close,
				NextRefreshTime: 0,
			}
			tbc := gdconf.GetTreasureBoxConfigure(info.BoxId)
			if tbc != nil {
				info.Type = proto.ETreasureBoxType(tbc.NewTreasureBoxType)
			}
			d.TreasureBoxes[uint32(cti.ID)] = info
		}
	}
	return d.TreasureBoxes
}

func (d *DungeonInfo) DungeonData() *proto.DungeonData {
	info := &proto.DungeonData{
		DungeonId:        d.DungeonID,
		AllTaskFinished:  d.AllTaskFinished,
		EnterTimes:       0,                 // 进入次数
		ExitTimes:        0,                 // 退出次数
		FinishTimes:      0,                 // 完成次数
		Coins:            make([]uint32, 0), // TODO 代币
		LastFinishTime:   d.LastFinishTime,  // 上次完成时间
		TaskFinishReward: 0,
		StarReward:       0,
		Monsters:         make([]uint32, 0), // TODO 怪物
		Char1:            d.TeamInfo.Char1,
		Char2:            d.TeamInfo.Char2,
		Char3:            d.TeamInfo.Char3,
		LastEnterTime:    d.LastEnterTime, // 上次进入时间
		Pos:              d.Pos,
		Rot:              d.Rot,
		IsOpenSecretBox:  false, // 是否已开启宝箱
	}

	return info
}

type SceneDungeon struct {
	*ScenePlayerInfo
	Info *DungeonInfo
}
