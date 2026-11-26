package model

import (
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/world"
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

var (
	ChannelTick   = 50 * time.Millisecond // 50 毫秒
	OneSTickCount int64
)

// 全服配置
type WorldTask struct {
	seed        int64
	tickCount   int64
	curDay      int64 // 当前天数
	weatherType proto.WeatherType
}

func NewWorldTask() *WorldTask {
	return &WorldTask{
		seed:        time.Now().Unix(),
		tickCount:   0,
		curDay:      0,
		weatherType: proto.WeatherType_WeatherType_SUNNY,
	}
}

func (w *WorldTask) Tick() {
	w.tickCount++
	// 时间帧递进
	if w.tickCount != 0 && w.tickCount%(OneSTickCount*61) == 0 {
		if w.Hour() == 0 {
			w.curDay++
		}
		w.weatherType = world.CalcWeather(w.seed, w.curDay, w.Hour())
	}
}

func (w *WorldTask) TodTime() uint32 {
	return uint32(w.tickCount/OneSTickCount*48) % (28 * 61 * 48)
}

func (w *WorldTask) Hour() int64 {
	return (w.tickCount / OneSTickCount / 61) % 24
}

func (w *WorldTask) Day(h int64) int64 {
	return w.curDay + (w.Hour()+h)/24
}

func (w *WorldTask) Weather() proto.WeatherType {
	return w.weatherType
}

func (w *WorldTask) CalcWeather(day, h int64) proto.WeatherType {
	return world.CalcWeather(w.seed, day, h)
}

func (w *WorldTask) TodSeq() int64 {
	return w.tickCount / (OneSTickCount * 61)
}

func CopyVector3(rot *proto.Vector3) *proto.Vector3 {
	return &proto.Vector3{
		X: rot.X,
		Y: rot.Y,
		Z: rot.Z,
	}
}

type SceneModel struct {
	WorldLevel       uint32                `json:"worldLevel,omitempty"`          // 世界等级
	CurPetInstanceId uint32                `json:"cur_pet_instance_id,omitempty"` // 当前携带的宠物索引
	SceneMap         map[uint32]*SceneInfo `json:"sceneMap,omitempty"`
}

func (s *Player) GetSceneModel() *SceneModel {
	if s.Scene == nil {
		s.Scene = new(SceneModel)
	}
	return s.Scene
}

func (sm *SceneModel) GetWorldLevel() uint32 {
	return sm.WorldLevel
}

func (sm *SceneModel) GetCurPetInstanceId() uint32 {
	return sm.CurPetInstanceId
}

func (sm *SceneModel) GetSceneMap() map[uint32]*SceneInfo {
	if sm.SceneMap == nil {
		sm.SceneMap = make(map[uint32]*SceneInfo)
	}
	return sm.SceneMap
}

func (sm *SceneModel) GetSceneInfo(sceneId uint32) *SceneInfo {
	list := sm.GetSceneMap()
	info, ok := list[sceneId]
	if !ok {
		info = &SceneInfo{
			SceneId:     sceneId,
			Collections: make(map[proto.ECollectionType]*CollectionInfo),
		}
		list[sceneId] = info
	}
	// 新建和旧存档恢复都立即补齐规则上限；后续访问还会按服务器日刷新。
	info.RefreshGatherLimits(GatherServerDay(time.Now()))
	return info
}

type SceneInfo struct {
	SceneId      uint32                                    `json:"sceneId,omitempty"`
	Collections  map[proto.ECollectionType]*CollectionInfo `json:"collections,omitempty"`  // 收集
	AreaDatas    map[uint32]*AreaData                      `json:"areaDatas,omitempty"`    // 锚点
	GatherLimits map[uint32]*GatherLimit                   `json:"gatherLimits,omitempty"` // 资源点
	TreasureBoxs map[uint32]*TreasureBox                   `json:"treasureBoxs,omitempty"` // 宝箱
}

func (si *SceneInfo) GetCollections() map[proto.ECollectionType]*CollectionInfo {
	if si.Collections == nil {
		si.Collections = make(map[proto.ECollectionType]*CollectionInfo)
	}
	return si.Collections
}

func (si *SceneInfo) GetAreaDatas() map[uint32]*AreaData {
	if si.AreaDatas == nil {
		si.AreaDatas = make(map[uint32]*AreaData)
	}
	return si.AreaDatas
}

func (si *SceneInfo) GetGatherLimits() map[uint32]*GatherLimit {
	si.RefreshGatherLimits(GatherServerDay(time.Now()))
	return si.GatherLimits
}

func (si *SceneInfo) GetTreasurBoxs() map[uint32]*TreasureBox {
	if si.TreasureBoxs == nil {
		si.TreasureBoxs = make(map[uint32]*TreasureBox)
	}
	return si.TreasureBoxs
}

type CollectionInfo struct {
	Type             uint32                             `json:"type,omitempty"`
	ItemMap          map[uint32]*PBCollectionRewardData `json:"itemMap,omitempty"`
	Level            uint32                             `json:"level,omitempty"`
	Exp              uint32                             `json:"exp,omitempty"`
	LastRefreshTime  time.Time                          `json:"lastRefreshTime,omitempty"`  // 上次刷新时间
	CollectedMoonIds []uint32                           `json:"collectedMoonIds,omitempty"` // 收集的月亮
}

func (si *SceneInfo) GetCollectionInfo(t proto.ECollectionType) *CollectionInfo {
	list := si.GetCollections()
	info, ok := list[t]
	if !ok {
		info = &CollectionInfo{
			Type:             uint32(t),
			ItemMap:          make(map[uint32]*PBCollectionRewardData),
			Level:            0,
			Exp:              0,
			LastRefreshTime:  time.Now(),
			CollectedMoonIds: make([]uint32, 0),
		}
		list[t] = info
	}
	if time.Now().Add(-4 * time.Minute).After(info.LastRefreshTime) {
		switch t {
		case proto.ECollectionType_ECollectionType_CollectMoonPiece:
			info.LastRefreshTime = time.Now()
			info.CollectedMoonIds = make([]uint32, 0)
		}
	}

	return info
}

func (c *CollectionInfo) CollectionData() *proto.CollectionData {
	info := &proto.CollectionData{
		Type:    c.Type,
		ItemMap: make(map[uint32]*proto.PBCollectionRewardData),
		Level:   c.Level,
		Exp:     c.Exp,
	}
	for k, v := range c.ItemMap {
		info.ItemMap[k] = v.PBCollectionRewardData()
	}

	return info
}

type PBCollectionRewardData struct {
	ItemId uint32             `json:"itemId,omitempty"`
	Status proto.RewardStatus `json:"status,omitempty"`
}

func (p *PBCollectionRewardData) PBCollectionRewardData() *proto.PBCollectionRewardData {
	return &proto.PBCollectionRewardData{
		ItemId: p.ItemId,
		Status: p.Status,
	}
}

type AreaData struct {
	AreaId    uint32                   `json:"areaId,omitempty"`
	AreaState proto.AreaData_AreaState `json:"areaState,omitempty"`
	Level     uint32                   `json:"level,omitempty"`
}

func (a *AreaData) AreaData() *proto.AreaData {
	return &proto.AreaData{
		AreaId:    a.AreaId,
		AreaState: a.AreaState,
		Level:     a.Level,
		Items:     make([]*proto.BaseItem, 0),
	}
}

type GatherLimit struct {
	GatherType          uint32 `json:"gatherType,omitempty"`
	GatherNum           uint32 `json:"gatherNum,omitempty"`
	GatherLimitNum      uint32 `json:"gatherLimitNum,omitempty"`
	LuckyGatherLimitNum uint32 `json:"luckyGatherLimitNum,omitempty"`
	LastGatherDay       int64  `json:"lastGatherDay,omitempty"` // 计数所属服务器日，0 表示尚未记录服务器日的旧存档
}

// GatherServerDay 与副本每日刷新一致，使用 UTC+8 服务器日。
func GatherServerDay(t time.Time) int64 {
	const serverOffset = int64(8 * 60 * 60)
	return (t.Unix() + serverOffset) / (24 * 60 * 60)
}

// RefreshGatherLimits 用当前场景规则覆盖存档中的上限，并仅在新服务器日清零累计次数。
// 旧存档没有 LastGatherDay 时只补记日期，保留累计次数，避免恢复存档时绕过收紧后的上限。
func (si *SceneInfo) RefreshGatherLimits(day int64) {
	if si == nil {
		return
	}
	if si.GatherLimits == nil {
		si.GatherLimits = make(map[uint32]*GatherLimit)
	}
	rule := gdconf.GetSceneGatherRuleConfigure(si.SceneId)
	for gatherType := uint32(0); gatherType < gdconf.GatherTypeCount; gatherType++ {
		normal, lucky, _ := gdconf.GatherRuleLimits(rule, gatherType)
		limit := si.GatherLimits[gatherType]
		if limit == nil {
			limit = &GatherLimit{GatherType: gatherType, LastGatherDay: day}
			si.GatherLimits[gatherType] = limit
		}
		limit.GatherLimitNum = normal
		limit.LuckyGatherLimitNum = lucky
		if limit.LastGatherDay == 0 {
			limit.LastGatherDay = day
		} else if limit.LastGatherDay != day {
			limit.GatherNum = 0
			limit.LastGatherDay = day
		}
	}
	// 损坏或越界的旧条目也只保留计数展示，上限为 0，采集校验会拒绝。
	for gatherType, limit := range si.GatherLimits {
		if gatherType < gdconf.GatherTypeCount {
			continue
		}
		if limit == nil {
			limit = &GatherLimit{GatherType: gatherType, LastGatherDay: day}
			si.GatherLimits[gatherType] = limit
		}
		limit.GatherLimitNum = 0
		limit.LuckyGatherLimitNum = 0
	}
}

func (g *GatherLimit) GatherLimit() *proto.GatherLimit {
	return &proto.GatherLimit{
		GatherType:          g.GatherType,
		GatherNum:           g.GatherNum,
		GatherLimitNum:      g.GatherLimitNum,
		LuckyGatherLimitNum: g.LuckyGatherLimitNum,
	}
}

func (si *SceneInfo) SceneGatherLimit() *proto.SceneGatherLimit {
	info := &proto.SceneGatherLimit{
		SceneId:      si.SceneId,
		GatherLimits: make([]*proto.GatherLimit, 0, len(si.GatherLimits)),
	}
	for _, v := range si.GetGatherLimits() {
		alg.AddList(&info.GatherLimits, v.GatherLimit())
	}

	return info
}

func (si *SceneInfo) GetGatherLimit(t uint32) *GatherLimit {
	list := si.GetGatherLimits()
	info, ok := list[t]
	if !ok {
		// 类型越界时仍返回零上限，调用方不能将其视为无限。
		info = &GatherLimit{GatherType: t, LastGatherDay: GatherServerDay(time.Now())}
		list[t] = info
	}
	return info
}

// 队伍场景信息
type ScenePlayerInfo struct {
	*Player  // 玩家主体
	Pos      *proto.Vector3
	Rot      *proto.Vector3
	Team     *TeamInfo
	SceneId  uint32
	TempPack *SceneTempPack // 临时背包
}

func (p *ScenePlayerInfo) GetScenePlayerInfo() *ScenePlayerInfo {
	return p
}

func (p *ScenePlayerInfo) GetSceneId() uint32 {
	return p.SceneId
}

func (p *ScenePlayerInfo) SetSceneId(id uint32) {
	if id != 0 {
		p.SceneId = id
	}
}

func (p *ScenePlayerInfo) GetPos() *proto.Vector3 {
	return p.Pos
}

func (p *ScenePlayerInfo) SetPos(pos *proto.Vector3) {
	p.Pos = pos
}

func (p *ScenePlayerInfo) GetRot() *proto.Vector3 {
	return p.Rot
}

func (p *ScenePlayerInfo) SetRot(rot *proto.Vector3) {
	p.Rot = rot
}

func (p *ScenePlayerInfo) GetTeam() *TeamInfo {
	return p.Team
}

func (p *ScenePlayerInfo) SetTeam(team TeamInfo) {
	p.Team = &team
}

func (p *ScenePlayerInfo) GetTempPack() *SceneTempPack {
	return p.TempPack
}

func NewScenePlayerInfo(p *Player, sceneID *uint32, team *TeamInfo, pos, rot *proto.Vector3) *ScenePlayerInfo {
	if sceneID == nil {
		sceneID = new(gdconf.GetConstant().DefaultSceneId)
	}
	if team == nil {
		team = p.GetTeamModel().GetTeamInfo()
	}
	if pos == nil || rot == nil {
		cfg := gdconf.GetSceneInfo(*sceneID)
		if cfg == nil {
			log.Game.Errorf("sceneId:%v ScenesConfigAsset.json配置文件中没有该场景", *sceneID)
			return nil
		}
		poss, rots := gdconf.GetSceneInfoRandomBorn(cfg.Info.GetBorn())
		pos = gdconf.ConfigVector3ToProtoVector3(poss)
		rot = gdconf.ConfigVector4ToProtoVector3(rots)
	}

	sp := &ScenePlayerInfo{
		Player:   p,
		Pos:      pos,
		Rot:      rot,
		Team:     team,
		SceneId:  *sceneID,
		TempPack: newSceneTempPack(),
	}
	return sp
}

func (p *ScenePlayerInfo) GetPbSceneTeam() (info *proto.SceneTeam) {
	info = &proto.SceneTeam{
		Char1: p.GetPbSceneCharacter(p.Team.Char1),
		Char2: p.GetPbSceneCharacter(p.Team.Char2),
		Char3: p.GetPbSceneCharacter(p.Team.Char3),
	}
	return
}

func (p *ScenePlayerInfo) GetPbSceneCharacter(characterId uint32) (info *proto.SceneCharacter) {
	if characterId == 0 {
		return nil
	}
	characterInfo := p.GetCharacterModel().GetCharacterInfo(characterId)
	if characterInfo == nil {
		log.Game.Debugf("玩家:%v队伍角色:%v不存在", p.UserId, characterId)
		return nil
	}
	info = &proto.SceneCharacter{
		Pos:                 p.Pos,
		Rot:                 p.Rot,
		CharId:              characterInfo.CharacterId,
		CharLv:              0, // ok
		CharBreakLv:         0, // ok
		CharStar:            0, // ok
		CharacterAppearance: characterInfo.GetPbCharacterAppearance(),
		OutfitPreset:        p.GetPbSceneCharacterOutfitPreset(characterInfo),
		WeaponId:            0, // ok
		WeaponStar:          0, // ok
		Armors:              make([]*proto.BaseArmor, 0),
		Posters:             make([]*proto.BasePoster, 0),
		GatherWeapon:        0, // ok
		IsDead:              false,
		InscriptionId:       0,
		InscriptionLv:       0,
		MPGameWeapon:        0,
	}
	// 基础
	p.UpdateCharacterBasic(info)
	// 装备
	p.UpdateCharacterEquip(info)
	// 装备
	{
		equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
		if equipmentPreset == nil {
			log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
				p.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
		} else {
			// 盔甲
			for _, armor := range equipmentPreset.Armors {
				item := p.GetItemModel().GetItemArmorInfo(armor.InstanceId)
				alg.AddList(&info.Armors, item.BaseArmor())
			}
			// 海报
			for _, poster := range equipmentPreset.Posters {
				item := p.GetItemModel().GetItemPosterInfo(poster.InstanceId)
				alg.AddList(&info.Posters, item.BasePoster())
			}
		}

	}

	return
}

func (p *ScenePlayerInfo) GetPbSceneCharacterOutfitPreset(characterInfo *CharacterInfo) *proto.SceneCharacterOutfitPreset {
	outfit := characterInfo.GetOutfitPreset(characterInfo.InUseOutfitPresetIndex)
	if outfit == nil {
		log.Game.Warnf("玩家:%v角色:%v外貌序号:%v缺少",
			p.UserId, characterInfo.CharacterId, characterInfo.InUseOutfitPresetIndex)
		return nil
	}
	getODS := func(id, index uint32) *proto.OutfitDyeScheme {
		fashionInfo := p.GetItemModel().GetItemFashionInfo(id)
		if fashionInfo == nil ||
			fashionInfo.GetDyeScheme(index) == nil {
			return &proto.OutfitDyeScheme{
				SchemeIndex: 0,
				Colors:      make([]*proto.PosColor, 0),
				IsUnLock:    false,
			}
		}
		return fashionInfo.GetDyeScheme(index).OutfitDyeScheme()
	}
	info := &proto.SceneCharacterOutfitPreset{
		Hat:                    outfit.Hat,
		HatDyeScheme:           getODS(outfit.Hat, outfit.HatDyeSchemeIndex),
		Hair:                   outfit.Hair,
		HairDyeScheme:          getODS(outfit.Hair, outfit.HairDyeSchemeIndex),
		Clothes:                outfit.Clothes,
		ClothDyeScheme:         getODS(outfit.Clothes, outfit.ClothesDyeSchemeIndex),
		Ornament:               outfit.Ornament,
		OrnDyeScheme:           getODS(outfit.Ornament, outfit.OrnamentDyeSchemeIndex),
		HideInfo:               outfit.OutfitHideInfo.OutfitHideInfo(),
		PendTop:                outfit.PendTop,
		PendTopDyeScheme:       getODS(outfit.PendTop, outfit.PendTopDyeSchemeIndex),
		PendChest:              outfit.PendChest,
		PendChestDyeScheme:     getODS(outfit.PendChest, outfit.PendChestDyeSchemeIndex),
		PendPelvis:             outfit.PendPelvis,
		PendPelvisDyeScheme:    getODS(outfit.PendPelvis, outfit.PendPelvisDyeSchemeIndex),
		PendUpFace:             outfit.PendUpFace,
		PendUpFaceDyeScheme:    getODS(outfit.PendUpFace, outfit.PendUpFaceDyeSchemeIndex),
		PendDownFace:           outfit.PendDownFace,
		PendDownFaceDyeScheme:  getODS(outfit.PendDownFace, outfit.PendDownFaceDyeSchemeIndex),
		PendLeftHand:           outfit.PendLeftHand,
		PendLeftHandDyeScheme:  getODS(outfit.PendLeftHand, outfit.PendLeftHandDyeSchemeIndex),
		PendRightHand:          outfit.PendRightHand,
		PendRightHandDyeScheme: getODS(outfit.PendRightHand, outfit.PendRightHandDyeSchemeIndex),
		PendLeftFoot:           outfit.PendLeftFoot,
		PendLeftFootDyeScheme:  getODS(outfit.PendLeftFoot, outfit.PendLeftFootDyeSchemeIndex),
		PendRightFoot:          outfit.PendRightFoot,
		PendRightFootDyeScheme: getODS(outfit.PendRightFoot, outfit.PendRightFootDyeSchemeIndex),
	}

	return info
}

func (p *ScenePlayerInfo) UpdateCharacterBasic(info *proto.SceneCharacter) {
	characterInfo := p.GetCharacterModel().GetCharacterInfo(info.GetCharId())
	if characterInfo == nil || info == nil {
		return
	}
	info.CharLv = characterInfo.Level
	info.CharBreakLv = characterInfo.BreakLevel
	info.CharStar = characterInfo.Star
	info.CharLv = characterInfo.Level
}

func (p *ScenePlayerInfo) UpdateCharacterEquip(info *proto.SceneCharacter) {
	characterInfo := p.GetCharacterModel().GetCharacterInfo(info.GetCharId())
	if characterInfo == nil || info == nil {
		return
	}
	info.GatherWeapon = characterInfo.GatherWeapon
	equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
	if equipmentPreset == nil {
		log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
			p.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
	} else {
		// 武器
		weaponInfo := p.GetItemModel().GetItemWeaponInfo(equipmentPreset.WeaponInstanceId)
		if weaponInfo == nil {
			log.Game.Warnf("玩家:%v角色:%v装备-武器:%v缺少",
				p.UserId, characterInfo.CharacterId, equipmentPreset.WeaponInstanceId)
		} else {
			info.WeaponStar = weaponInfo.Star
			info.WeaponId = weaponInfo.WeaponId
		}
	}
}
