package model

import "sort"

// ProfileModel 保存玩家档案中需要随玩家 blob 一起持久化的协议状态。
// 基础资料仍由 db.OFGameBasic 保存；这里只保存没有独立数据表的字段。
type ProfileModel struct {
	PhoneCase             uint32          `json:"phoneCase,omitempty"`
	EnergyLastRecoverAt   int64           `json:"energyLastRecoverAt,omitempty"`
	EnergyBuyDay          int64           `json:"energyBuyDay,omitempty"`
	EnergyBuyNum          uint32          `json:"energyBuyNum,omitempty"`
	PlayerLevelRewards    map[uint32]bool `json:"playerLevelRewards,omitempty"`
	UnlockedWorldLevels   map[uint32]bool `json:"unlockedWorldLevels,omitempty"`
	CompletedAchievements map[uint32]bool `json:"completedAchievements,omitempty"`
	CompletedFlags        map[uint32]bool `json:"completedFlags,omitempty"`
	LastPopEmojiAt        int64           `json:"lastPopEmojiAt,omitempty"`
	LastClapAt            int64           `json:"lastClapAt,omitempty"`
	LastThrowDiceAt       int64           `json:"lastThrowDiceAt,omitempty"`
}

func DefaultProfileModel() *ProfileModel {
	p := &ProfileModel{}
	p.EnsureMaps()
	return p
}

func (p *ProfileModel) EnsureMaps() {
	if p == nil {
		return
	}
	if p.PlayerLevelRewards == nil {
		p.PlayerLevelRewards = make(map[uint32]bool)
	}
	if p.UnlockedWorldLevels == nil {
		p.UnlockedWorldLevels = make(map[uint32]bool)
	}
	if p.CompletedAchievements == nil {
		p.CompletedAchievements = make(map[uint32]bool)
	}
	if p.CompletedFlags == nil {
		p.CompletedFlags = make(map[uint32]bool)
	}
}

func (p *ProfileModel) HasPlayerLevelReward(level uint32) bool {
	return p != nil && p.PlayerLevelRewards[level]
}

func (p *ProfileModel) MarkPlayerLevelReward(level uint32) {
	if p == nil || level == 0 {
		return
	}
	p.EnsureMaps()
	p.PlayerLevelRewards[level] = true
}

func (p *ProfileModel) IsWorldLevelUnlocked(level uint32) bool {
	return p != nil && p.UnlockedWorldLevels[level]
}

func (p *ProfileModel) MarkWorldLevelUnlocked(level uint32) {
	if p == nil || level == 0 {
		return
	}
	p.EnsureMaps()
	p.UnlockedWorldLevels[level] = true
}

func (p *ProfileModel) RewardedPlayerLevels() []uint32 {
	if p == nil {
		return nil
	}
	levels := make([]uint32, 0, len(p.PlayerLevelRewards))
	for level, rewarded := range p.PlayerLevelRewards {
		if rewarded {
			levels = append(levels, level)
		}
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })
	return levels
}

// ProfileAccessor 是玩家 blob 挂载点的最小接口。Player 接入 Profile 字段后
// 实现 GetProfileModel 即可，业务层不需要依赖 Player 的具体存储布局。
type ProfileAccessor interface {
	GetProfileModel() *ProfileModel
}

func GetPlayerProfile(player *Player) *ProfileModel {
	if player == nil {
		return nil
	}
	accessor, ok := any(player).(ProfileAccessor)
	if !ok {
		return nil
	}
	return accessor.GetProfileModel()
}
