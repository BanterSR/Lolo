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
	DungeonID               uint32                  `json:"dungeon_id"`                  // 副本id
	TeamInfo                TeamInfo                `json:"team_info"`                   // 队伍信息
	AllTaskFinished         bool                    `json:"all_task_finished"`           // 是否完成全部任务
	EnterTimes              uint32                  `json:"enter_times,omitempty"`       // 累计进入次数
	ExitTimes               uint32                  `json:"exit_times,omitempty"`        // 累计退出次数
	FinishTimes             uint32                  `json:"finish_times,omitempty"`      // 累计完成次数
	DailyEnterCount         uint32                  `json:"daily_enter_count,omitempty"` // 当日进入次数
	DailySweepCount         uint32                  `json:"daily_sweep_count,omitempty"` // 当日扫荡次数
	LastEnterDay            int64                   `json:"last_enter_day,omitempty"`    // 进入次数所属服务器日
	LastSweepDay            int64                   `json:"last_sweep_day,omitempty"`    // 扫荡次数所属服务器日
	LastEnterTime           int64                   `json:"last_enter_time"`             // 上次进入时间
	LastFinishTime          int64                   `json:"last_finish_time"`            // 上次完成耗时
	Stars                   uint32                  `json:"stars,omitempty"`             // 历史最佳星级
	StarRewardClaimed       uint32                  `json:"star_reward_claimed,omitempty"`
	FirstClearRewardClaimed bool                    `json:"first_clear_reward_claimed,omitempty"`
	LastRunToken            uint64                  `json:"last_run_token,omitempty"`
	RewardClaimedToken      uint64                  `json:"reward_claimed_token,omitempty"`
	Pos                     *proto.Vector3          `json:"pos"`
	Rot                     *proto.Vector3          `json:"rot"`
	TreasureBoxes           map[uint32]*TreasureBox `json:"treasure_boxes"` // 宝箱状态
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
			if cti == nil || cti.Index < 0 || d.TreasureBoxes[uint32(cti.Index)] != nil {
				continue
			}
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
			d.TreasureBoxes[info.Index] = info
		}
	}
	// 旧存档按 BoxId 建键；迁移到配置索引，避免重复 BoxId 互相覆盖。
	for key, box := range d.TreasureBoxes {
		if box == nil {
			delete(d.TreasureBoxes, key)
			continue
		}
		if key != box.Index {
			if _, exists := d.TreasureBoxes[box.Index]; !exists {
				d.TreasureBoxes[box.Index] = box
			}
			delete(d.TreasureBoxes, key)
		}
	}
	return d.TreasureBoxes
}

func (d *DungeonInfo) ResetTreasureBoxes() {
	for _, box := range d.GetTreasureBoxes() {
		box.State = proto.TreasureBoxState_TreasureBoxState_Close
		box.NextRefreshTime = 0
	}
}

func (d *DungeonInfo) DungeonData() *proto.DungeonData {
	info := &proto.DungeonData{
		DungeonId:        d.DungeonID,
		AllTaskFinished:  d.AllTaskFinished,
		EnterTimes:       d.EnterTimes,
		ExitTimes:        d.ExitTimes,
		FinishTimes:      d.FinishTimes,
		Coins:            make([]uint32, 0), // TODO 代币
		LastFinishTime:   d.LastFinishTime,  // 上次完成时间
		TaskFinishReward: proto.RewardStatus_NotReward,
		StarReward:       proto.RewardStatus_NotReward,
		Char1:            d.TeamInfo.Char1,
		Char2:            d.TeamInfo.Char2,
		Char3:            d.TeamInfo.Char3,
		LastEnterTime:    d.LastEnterTime, // 上次进入时间
		Pos:              d.Pos,
		Rot:              d.Rot,
		IsOpenSecretBox:  false, // 是否已开启宝箱
	}
	if d.FirstClearRewardClaimed {
		info.TaskFinishReward = proto.RewardStatus_Reward
	}
	if d.Stars > 0 && d.StarRewardClaimed&starRewardMask(d.Stars) == starRewardMask(d.Stars) {
		info.StarReward = proto.RewardStatus_Reward
	}
	return info
}

func starRewardMask(stars uint32) uint32 {
	if stars >= 3 {
		return 0b111
	}
	return (1 << stars) - 1
}

type DungeonRuntimeState uint8

const (
	DungeonRuntimeEntered DungeonRuntimeState = iota
	DungeonRuntimeRunning
	DungeonRuntimeFinished
	DungeonRuntimeClaimed
)

// DungeonRuntime 只属于当前场景中的一轮副本，不写入玩家存档。
type DungeonRuntime struct {
	Token           uint64
	DungeonID       uint32
	PlayerLevel     uint32
	State           DungeonRuntimeState
	StartedAt       int64
	EndedAt         int64
	Stars           uint32
	MonstersByIndex bool
	Monsters        map[uint32]DungeonMonsterRuntime
	DeadMonsters    map[uint32]bool
	Encounters      map[uint32]proto.BattleState
	Drops           map[uint32]*DropItem
	Treasures       map[uint32]*DungeonTreasureRuntime
}

type DungeonMonsterRuntime struct {
	UnitID       uint32
	RewardPoolID uint32
}

type DungeonTreasureRuntime struct {
	Drop   *DropItem
	Picked bool
}

type SceneDungeon struct {
	*ScenePlayerInfo
	Info    *DungeonInfo
	Runtime *DungeonRuntime
}
