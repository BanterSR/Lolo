package model

import (
	"gucooing/lolo/pkg/log"
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func CopyVector3(rot *proto.Vector3) *proto.Vector3 {
	return &proto.Vector3{
		X: rot.X,
		Y: rot.Y,
		Z: rot.Z,
	}
}

type SceneModel struct {
	SceneMap map[uint32]*SceneInfo `json:"sceneMap,omitempty"`
}

func (s *Player) GetSceneModel() *SceneModel {
	if s.Scene == nil {
		s.Scene = new(SceneModel)
	}
	return s.Scene
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
	if si.GatherLimits == nil {
		si.GatherLimits = make(map[uint32]*GatherLimit)
	}
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
	AreaId    uint32          `json:"areaId,omitempty"`
	AreaState proto.AreaState `json:"areaState,omitempty"`
	Level     uint32          `json:"level,omitempty"`
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
		info = &GatherLimit{
			GatherType:          t,
			GatherNum:           0,
			GatherLimitNum:      0,
			LuckyGatherLimitNum: 0,
		}
		list[t] = info
	}
	return info
}

type TreasureBox struct {
	Index           uint32                 `json:"index,omitempty"`           // 序号
	BoxId           uint32                 `json:"boxId,omitempty"`           // id
	Type            proto.ETreasureBoxType `json:"type,omitempty"`            // 类型
	State           proto.TreasureBoxState `json:"state,omitempty"`           // 状态
	NextRefreshTime int64                  `json:"nextRefreshTime,omitempty"` // 下次更新时间
}

func (t *TreasureBox) TreasureBoxData() *proto.TreasureBoxData {
	info := &proto.TreasureBoxData{
		Index:           t.Index,
		BoxId:           t.BoxId,
		Type:            t.Type,
		State:           t.State,
		NextRefreshTime: t.NextRefreshTime,
		Rewards:         make([]*proto.ItemDetail, 0),
	}

	return info
}

// 队伍场景信息
type SceneTeamInfo struct {
	*Player // 玩家主体
	Pos     *proto.Vector3
	Rot     *proto.Vector3
	Team    *TeamInfo
}

func (t *SceneTeamInfo) GetPbSceneTeam() (info *proto.SceneTeam) {
	info = &proto.SceneTeam{
		Char1: t.GetPbSceneCharacter(t.Team.Char1),
		Char2: t.GetPbSceneCharacter(t.Team.Char2),
		Char3: t.GetPbSceneCharacter(t.Team.Char3),
	}
	return
}

func (t *SceneTeamInfo) GetPbSceneCharacter(characterId uint32) (info *proto.SceneCharacter) {
	characterInfo := t.GetCharacterModel().GetCharacterInfo(characterId)
	if characterInfo == nil {
		log.Game.Debugf("玩家:%v队伍角色:%v不存在", t.UserId, characterId)
		return nil
	}
	info = &proto.SceneCharacter{
		Pos:                 t.Pos,
		Rot:                 t.Rot,
		CharId:              characterInfo.CharacterId,
		CharLv:              0, // ok
		CharBreakLv:         0, // ok
		CharStar:            0, // ok
		CharacterAppearance: characterInfo.GetPbCharacterAppearance(),
		OutfitPreset:        t.GetPbSceneCharacterOutfitPreset(characterInfo),
		WeaponId:            0, // ok
		WeaponStar:          0, // ok
		Armors:              make([]*proto.BaseArmor, 0),
		Posters:             make([]*proto.BasePoster, 0),
		GatherWeapon:        0, // ok
		IsDead:              false,
		InscriptionId:       0,
		InscriptionLv:       0,
		MpGameWeapon:        0,
	}
	// 基础
	t.UpdateCharacterBasic(info)
	// 装备
	t.UpdateCharacterEquip(info)
	// 装备
	{
		equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
		if equipmentPreset == nil {
			log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
				t.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
		} else {
			// 盔甲
			for _, armor := range equipmentPreset.Armors {
				item := t.GetItemModel().GetItemArmorInfo(armor.InstanceId)
				alg.AddList(&info.Armors, item.BaseArmor())
			}
			// 海报
			for _, poster := range equipmentPreset.Posters {
				item := t.GetItemModel().GetItemPosterInfo(poster.InstanceId)
				alg.AddList(&info.Posters, item.BasePoster())
			}
		}

	}

	return
}

func (t *SceneTeamInfo) GetPbSceneCharacterOutfitPreset(characterInfo *CharacterInfo) *proto.SceneCharacterOutfitPreset {
	outfit := characterInfo.GetOutfitPreset(characterInfo.InUseOutfitPresetIndex)
	if outfit == nil {
		log.Game.Warnf("玩家:%v角色:%v外貌序号:%v缺少",
			t.UserId, characterInfo.CharacterId, characterInfo.InUseOutfitPresetIndex)
		return nil
	}
	getODS := func(id, index uint32) *proto.OutfitDyeScheme {
		fashionInfo := t.GetItemModel().GetItemFashionInfo(id)
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

func (t *SceneTeamInfo) UpdateCharacterBasic(info *proto.SceneCharacter) {
	characterInfo := t.GetCharacterModel().GetCharacterInfo(info.GetCharId())
	if characterInfo == nil || info == nil {
		return
	}
	info.CharLv = characterInfo.Level
	info.CharBreakLv = characterInfo.BreakLevel
	info.CharStar = characterInfo.Star
	info.CharLv = characterInfo.Level
}

func (t *SceneTeamInfo) UpdateCharacterEquip(info *proto.SceneCharacter) {
	characterInfo := t.GetCharacterModel().GetCharacterInfo(info.GetCharId())
	if characterInfo == nil || info == nil {
		return
	}
	info.GatherWeapon = characterInfo.GatherWeapon
	equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
	if equipmentPreset == nil {
		log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
			t.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
	} else {
		// 武器
		weaponInfo := t.GetItemModel().GetItemWeaponInfo(equipmentPreset.WeaponInstanceId)
		if weaponInfo == nil {
			log.Game.Warnf("玩家:%v角色:%v装备-武器:%v缺少",
				t.UserId, characterInfo.CharacterId, equipmentPreset.WeaponInstanceId)
		} else {
			info.WeaponStar = weaponInfo.Star
			info.WeaponId = weaponInfo.WeaponId
		}
	}
}
