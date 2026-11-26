package model

// ActivityModel 保存活动周期、奖励领取和服务器成就计数。
// 它挂在 ArchiveModel 下，随玩家 blob 一起持久化，不依赖客户端上报的进度。
type ActivityModel struct {
	Periods               map[string]*ActivityPeriodState `json:"periods,omitempty"`
	Achievements          map[uint32]uint64               `json:"achievements,omitempty"`
	RewardedAchievements  map[uint32]bool                 `json:"rewardedAchievements,omitempty"`
	RewardedAchieveGroups map[uint32]bool                 `json:"rewardedAchieveGroups,omitempty"`
	RewardedAbilityBadges map[uint32]bool                 `json:"rewardedAbilityBadges,omitempty"`
	SupplyBox             *SupplyBoxState                 `json:"supplyBox,omitempty"`
	MonthCard             *MonthCardState                 `json:"monthCard,omitempty"`
}

func (s *Player) GetActivityModel() *ActivityModel {
	archive := s.GetArchive()
	if archive.Activity == nil {
		archive.Activity = new(ActivityModel)
	}
	return archive.Activity
}

func (a *ActivityModel) GetPeriods() map[string]*ActivityPeriodState {
	if a.Periods == nil {
		a.Periods = make(map[string]*ActivityPeriodState)
	}
	return a.Periods
}

func (a *ActivityModel) GetPeriod(key string) *ActivityPeriodState {
	periods := a.GetPeriods()
	state := periods[key]
	if state == nil {
		state = new(ActivityPeriodState)
		periods[key] = state
	}
	return state
}

func (a *ActivityModel) GetAchievements() map[uint32]uint64 {
	if a.Achievements == nil {
		a.Achievements = make(map[uint32]uint64)
	}
	return a.Achievements
}

func (a *ActivityModel) AchievementProgress(id uint32) uint64 {
	return a.GetAchievements()[id]
}

func (a *ActivityModel) AddAchievementProgress(id uint32, count uint64) {
	if count == 0 {
		return
	}
	progress := a.GetAchievements()
	if ^uint64(0)-progress[id] < count {
		progress[id] = ^uint64(0)
		return
	}
	progress[id] += count
}

func (a *ActivityModel) GetRewardedAchievements() map[uint32]bool {
	if a.RewardedAchievements == nil {
		a.RewardedAchievements = make(map[uint32]bool)
	}
	return a.RewardedAchievements
}

func (a *ActivityModel) GetRewardedAchieveGroups() map[uint32]bool {
	if a.RewardedAchieveGroups == nil {
		a.RewardedAchieveGroups = make(map[uint32]bool)
	}
	return a.RewardedAchieveGroups
}

func (a *ActivityModel) GetRewardedAbilityBadges() map[uint32]bool {
	if a.RewardedAbilityBadges == nil {
		a.RewardedAbilityBadges = make(map[uint32]bool)
	}
	return a.RewardedAbilityBadges
}

func (a *ActivityModel) GetSupplyBox() *SupplyBoxState {
	if a.SupplyBox == nil {
		a.SupplyBox = new(SupplyBoxState)
	}
	return a.SupplyBox
}

func (a *ActivityModel) GetMonthCard() *MonthCardState {
	if a.MonthCard == nil {
		a.MonthCard = new(MonthCardState)
	}
	return a.MonthCard
}

type ActivityPeriodState struct {
	Claims     map[string]bool          `json:"claims,omitempty"`
	Counters   map[string]uint64        `json:"counters,omitempty"`
	SignIn     *ActivitySignInState     `json:"signIn,omitempty"`
	Gift       *ActivityGiftState       `json:"gift,omitempty"`
	BattlePass *ActivityBattlePassState `json:"battlePass,omitempty"`
	Turntable  *ActivityTurntableState  `json:"turntable,omitempty"`
}

func (p *ActivityPeriodState) GetClaims() map[string]bool {
	if p.Claims == nil {
		p.Claims = make(map[string]bool)
	}
	return p.Claims
}

func (p *ActivityPeriodState) IsClaimed(key string) bool {
	return p.GetClaims()[key]
}

func (p *ActivityPeriodState) Claim(key string) {
	p.GetClaims()[key] = true
}

func (p *ActivityPeriodState) GetCounters() map[string]uint64 {
	if p.Counters == nil {
		p.Counters = make(map[string]uint64)
	}
	return p.Counters
}

func (p *ActivityPeriodState) Counter(key string) uint64 {
	return p.GetCounters()[key]
}

func (p *ActivityPeriodState) AddCounter(key string, count uint64) {
	if count == 0 {
		return
	}
	counters := p.GetCounters()
	if ^uint64(0)-counters[key] < count {
		counters[key] = ^uint64(0)
		return
	}
	counters[key] += count
}

func (p *ActivityPeriodState) GetSignIn() *ActivitySignInState {
	if p.SignIn == nil {
		p.SignIn = new(ActivitySignInState)
	}
	return p.SignIn
}

func (p *ActivityPeriodState) GetGift() *ActivityGiftState {
	if p.Gift == nil {
		p.Gift = new(ActivityGiftState)
	}
	return p.Gift
}

func (p *ActivityPeriodState) GetBattlePass() *ActivityBattlePassState {
	if p.BattlePass == nil {
		p.BattlePass = new(ActivityBattlePassState)
	}
	return p.BattlePass
}

func (p *ActivityPeriodState) GetTurntable() *ActivityTurntableState {
	if p.Turntable == nil {
		p.Turntable = new(ActivityTurntableState)
	}
	return p.Turntable
}

type ActivitySignInState struct {
	LoginDay     uint32 `json:"loginDay,omitempty"`
	InitTime     int64  `json:"initTime,omitempty"`
	LastLoginDay string `json:"lastLoginDay,omitempty"`
}

type ActivityGiftState struct {
	HasBuy bool `json:"hasBuy,omitempty"`
}

type ActivityBattlePassState struct {
	Level      uint32 `json:"level,omitempty"`
	TotalExp   uint32 `json:"totalExp,omitempty"`
	CurrentExp uint32 `json:"currentExp,omitempty"`
	WeekKey    string `json:"weekKey,omitempty"`
	WeekExp    uint32 `json:"weekExp,omitempty"`
	HasBuy     bool   `json:"hasBuy,omitempty"`
}

type ActivityTurntableState struct {
	Used      map[uint32]uint32         `json:"used,omitempty"`
	DrawCount uint32                    `json:"drawCount,omitempty"`
	Records   []ActivityTurntableRecord `json:"records,omitempty"`
}

func (t *ActivityTurntableState) GetUsed() map[uint32]uint32 {
	if t.Used == nil {
		t.Used = make(map[uint32]uint32)
	}
	return t.Used
}

type ActivityTurntableRecord struct {
	Index    uint32 `json:"index,omitempty"`
	ItemID   uint32 `json:"itemId,omitempty"`
	ItemNum  uint32 `json:"itemNum,omitempty"`
	DrawTime int64  `json:"drawTime,omitempty"`
}

type SupplyBoxState struct {
	NextRewardTime int64 `json:"nextRewardTime,omitempty"`
}

type MonthCardState struct {
	OverDueTime  int64  `json:"overDueTime,omitempty"`
	RewardDays   uint32 `json:"rewardDays,omitempty"`
	LastClaimDay string `json:"lastClaimDay,omitempty"`
}
