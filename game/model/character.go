package model

import (
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type CharacterModel struct {
	CharacterMap     map[uint32]*CharacterInfo `json:"characterMap,omitempty"`
	PlacedCharacters []uint32                  `json:"placedCharacters"` // 摆放的角色
}

func DefaultCharacterModel() *CharacterModel {
	info := &CharacterModel{
		CharacterMap:     make(map[uint32]*CharacterInfo),
		PlacedCharacters: make([]uint32, 0),
	}
	return info
}

func (s *Player) AllCharacterModel() {
	for _, conf := range gdconf.GetCharacterAllMap() {
		characterInfo := s.AddCharacter(conf.CharacterId)
		if characterInfo == nil {
			log.Game.Errorf("添加角色:%v失败", conf.CharacterId)
			continue
		}
	}
}

func (s *Player) GetCharacterModel() *CharacterModel {
	if s == nil {
		return nil
	}
	if s.Character == nil {
		s.Character = DefaultCharacterModel()
	}
	return s.Character
}

func (c *CharacterModel) GetPlacedCharacters() []uint32 {
	if c.PlacedCharacters == nil {
		c.PlacedCharacters = make([]uint32, 0)
	}
	return c.PlacedCharacters
}

func (c *CharacterModel) SetPlacedCharacter(characterId uint32, isRemove bool) {
	if isRemove {
		for i := 0; i < len(c.PlacedCharacters); i++ {
			if c.PlacedCharacters[i] == characterId {
				c.PlacedCharacters = append(c.PlacedCharacters[:i], c.PlacedCharacters[i+1:]...)
				return
			}
		}
	} else {
		for _, id := range c.PlacedCharacters {
			if id == characterId {
				return
			}
		}
		c.PlacedCharacters = append(c.PlacedCharacters, characterId)
	}
}

type CharacterInfo struct {
	CharacterId               uint32                      `json:"characterId,omitempty"`               // 角色id
	IsNew                     bool                        `json:"isNew,omitempty"`                     // 是否新角色
	Level                     uint32                      `json:"level,omitempty"`                     // 角色等级
	MaxLevel                  uint32                      `json:"maxLevel,omitempty"`                  // 最大等级
	Exp                       uint32                      `json:"exp,omitempty"`                       // 角色经验
	BreakLevel                uint32                      `json:"breakLevel,omitempty"`                // 突破等级
	Star                      uint32                      `json:"star,omitempty"`                      // 角色星级
	CharacterAppearance       *CharacterAppearance        `json:"characterAppearance,omitempty"`       // 角色外貌
	CharacterSkillList        map[uint32]*CharacterSkill  `json:"characterSkillList,omitempty"`        // 角色技能
	InUseEquipmentPresetIndex uint32                      `json:"inUseEquipmentPresetIndex,omitempty"` // 当前装备表
	EquipmentPresetList       map[uint32]*EquipmentPreset `json:"equipmentPresets,omitempty"`          // 角色装备预设表
	InUseOutfitPresetIndex    uint32                      `json:"inUseOutfitPresetIndex,omitempty"`    // 当前服装表
	OutfitPresetList          map[uint32]*OutfitPreset    `json:"outfitPresets,omitempty"`             // 角色服装预设表
	GatherWeapon              uint32                      `json:"gatherWeapon,omitempty"`              // 手持道具
	IsUnlockPayment           bool                        `json:"isUnlockPayment,omitempty"`           // 已解锁购买
}

func newCharacterInfo(characterId uint32) *CharacterInfo {
	info := &CharacterInfo{
		CharacterId:               characterId,
		IsNew:                     true,
		Level:                     1,
		BreakLevel:                0,
		Exp:                       0,
		Star:                      0,
		CharacterAppearance:       newCharacterAppearance(characterId),
		CharacterSkillList:        newCharacterSkillList(characterId),
		InUseEquipmentPresetIndex: 0,
		EquipmentPresetList:       nil,
		InUseOutfitPresetIndex:    0,
		OutfitPresetList:          nil,
	}

	return info
}

func (c *CharacterModel) GetCharacterMap() map[uint32]*CharacterInfo {
	if c.CharacterMap == nil {
		c.CharacterMap = make(map[uint32]*CharacterInfo)
	}
	return c.CharacterMap
}

func (c *CharacterModel) GetCharacterInfo(characterId uint32) *CharacterInfo {
	list := c.GetCharacterMap()
	if list == nil {
		return nil
	}
	return list[characterId]
}

func (s *Player) AddCharacter(characterId uint32) *CharacterInfo {
	list := s.GetCharacterModel().GetCharacterMap()
	if list == nil {
		return nil
	}
	conf := gdconf.GetCharacterAll(characterId)
	if conf == nil ||
		!conf.CharacterInfo.IsShow {
		log.Game.Warnf("尝试添加不存在的角色:%v", characterId)
		return nil
	}
	if characterInfo, ok := list[characterId]; ok {
		characterInfo.IsNew = false
		log.Game.Debugf("重复添加角色:%v", characterId)
		return characterInfo
	}
	characterInfo := newCharacterInfo(characterId)
	characterInfo.UpMaxLevel()
	list[characterId] = characterInfo
	// 初始化装备
	preset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
	itemWeapon := s.GetItemModel().AddItemWeapon(uint32(conf.CharacterInfo.DefaultWeaponID))
	if itemWeapon == nil {
		log.Game.Warnf("角色:%v,添加默认武器:%v失败", characterId, conf.CharacterInfo.DefaultWeaponID)
		return nil
	}
	itemWeapon.SetWearerId(characterId, preset.PresetIndex)
	preset.WeaponInstanceId = itemWeapon.InstanceId
	// 初始化外观
	for index := uint32(0); index < gdconf.GetConstant().OutfitPresetNum; index++ {
		outfit := characterInfo.GetOutfitPreset(index)
		if hat := s.GetItemModel().AddItemFashion(
			uint32(conf.CharacterInfo.HatID)); hat != nil {
			outfit.Hat = uint32(conf.CharacterInfo.HatID)
		}
		if hair := s.GetItemModel().AddItemFashion(
			uint32(conf.CharacterInfo.HairID)); hair != nil {
			outfit.Hair = uint32(conf.CharacterInfo.HairID)
		}
		if cloth := s.GetItemModel().AddItemFashion(
			uint32(conf.CharacterInfo.ClothID)); cloth != nil {
			outfit.Clothes = uint32(conf.CharacterInfo.ClothID)
		}
	}

	return characterInfo
}

func (c *CharacterModel) GetAllPbCharacter() []*proto.Character {
	list := make([]*proto.Character, 0)
	for _, characterInfo := range c.GetCharacterMap() {
		alg.AddList(&list, characterInfo.Character())
	}
	return list
}

func (c *CharacterInfo) UpMaxLevel() {
	conf := gdconf.GetCharacterAll(c.CharacterId)
	if c.BreakLevel < 1 ||
		len(conf.LevelRules) < int(c.BreakLevel) {
		c.MaxLevel = 20
		return
	}
	levelRule := conf.LevelRules[c.BreakLevel-1]
	c.MaxLevel = uint32(levelRule.GetTopMaxLevel())
}

func (c *CharacterInfo) Character() *proto.Character {
	pbInfo := &proto.Character{
		CharacterId:               c.CharacterId,
		Level:                     c.Level,
		MaxLevel:                  c.MaxLevel,
		Exp:                       c.Exp,
		Star:                      c.Star,
		EquipmentPresets:          c.GetPbEquipmentPresets(),
		InUseEquipmentPresetIndex: c.InUseEquipmentPresetIndex,
		OutfitPresets:             c.GetPbOutfitPresets(),
		InUseOutfitPresetIndex:    c.InUseOutfitPresetIndex,
		GatherWeapon:              c.GatherWeapon,
		CharacterAppearance:       c.GetPbCharacterAppearance(),
		CharacterSkillList:        c.GetPbCharacterSkillList(),
		RewardedAchievementIdLst:  make([]uint32, 0),
		IsUnlockPayment:           c.IsUnlockPayment,
		RewardIndexLst:            make([]uint32, 0),
		MpGameWeapon:              0,
	}

	return pbInfo
}

func (c *CharacterInfo) ItemDetail() *proto.ItemDetail {
	return &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  c.CharacterId,
			ItemTag: proto.EBagItemTag_EBagItemTag_Card,
			IsNew:   c.IsNew,
			Item: &proto.ItemInfo_Character{
				Character: c.Character(),
			},
		},
		TransformedItem: make([]*proto.ItemInfo, 0),
		Extras:          make([]*proto.ItemInfo, 0),
		PackType:        0,
		ExtraQuality:    0,
	}
}

type CharacterAppearance struct {
	Badge                      uint32 `json:"badge,omitempty"`      // 徽章
	UmbrellaId                 uint32 `json:"umbrellaId,omitempty"` // 伞？
	InsectNetInstanceId        uint32 `json:"insectNetInstanceId,omitempty"`
	LoggingAxeInstanceId       uint32 `json:"loggingAxeInstanceId,omitempty"`
	WaterBottleInstanceId      uint32 `json:"waterBottleInstanceId,omitempty"`
	MiningHammerInstanceId     uint32 `json:"miningHammerInstanceId,omitempty"`
	CollectionGlovesInstanceId uint32 `json:"collectionGlovesInstanceId,omitempty"`
	FishingRodInstanceId       uint32 `json:"fishingRodInstanceId,omitempty"`
	VehicleInstanceId          uint32 `json:"vehicleInstanceId,omitempty"` // 车
}

func newCharacterAppearance(characterId uint32) *CharacterAppearance {
	info := &CharacterAppearance{
		Badge:                      gdconf.GetConstant().DefaultBadge,
		UmbrellaId:                 gdconf.GetConstant().DefaultUmbrellaId,
		InsectNetInstanceId:        0,
		LoggingAxeInstanceId:       0,
		WaterBottleInstanceId:      0,
		MiningHammerInstanceId:     0,
		CollectionGlovesInstanceId: 0,
		FishingRodInstanceId:       0,
	}

	return info
}

func (c *CharacterInfo) GetPbCharacterAppearance() *proto.CharacterAppearance {
	info := c.CharacterAppearance
	if info == nil {
		return nil
	}
	pbInfo := &proto.CharacterAppearance{
		Badge:                      info.Badge,
		UmbrellaId:                 info.UmbrellaId,
		InsectNetInstanceId:        info.InsectNetInstanceId,
		LoggingAxeInstanceId:       info.LoggingAxeInstanceId,
		WaterBottleInstanceId:      info.WaterBottleInstanceId,
		MiningHammerInstanceId:     info.MiningHammerInstanceId,
		CollectionGlovesInstanceId: info.CollectionGlovesInstanceId,
		FishingRodInstanceId:       info.FishingRodInstanceId,
		VehicleInstanceId:          info.VehicleInstanceId,
	}

	return pbInfo
}

type CharacterSkill struct {
	SkillId    uint32 `json:"skillId,omitempty"`
	SkillLevel uint32 `json:"skillLevel,omitempty"`
}

func (s *CharacterSkill) CharacterSkill() *proto.CharacterSkill {
	return &proto.CharacterSkill{
		SkillId:    s.SkillId,
		SkillLevel: s.SkillLevel,
	}
}

func newCharacterSkillList(id uint32) map[uint32]*CharacterSkill {
	conf := gdconf.GetCharacterAll(id)
	if conf == nil {
		log.Game.Warnf("添加技能的角色不存在id:%v", id)
		return nil
	}
	list := make(map[uint32]*CharacterSkill)
	addSkill := func(skillId uint32) {
		list[skillId] = &CharacterSkill{
			SkillId:    skillId,
			SkillLevel: 1,
		}
	}
	for _, skillId := range conf.CharacterInfo.GetSpellIDs() {
		addSkill(uint32(skillId))
	}
	for _, skillId := range conf.CharacterInfo.GetExSpellIDs() {
		addSkill(uint32(skillId))
	}
	return list
}

func (c *CharacterInfo) GetPbCharacterSkillList() []*proto.CharacterSkill {
	pbInfoList := make([]*proto.CharacterSkill, 0)
	for _, info := range c.CharacterSkillList {
		alg.AddList(&pbInfoList, info.CharacterSkill())
	}

	return pbInfoList
}

type EquipmentPreset struct {
	PresetIndex      uint32                            `json:"presetIndex,omitempty"`
	WeaponInstanceId uint32                            `json:"weaponInstanceId,omitempty"`
	Armors           map[proto.EEquipType]*ArmorInfo   `json:"armors,omitempty"`
	Posters          map[proto.PosterIndex]*PosterInfo `json:"posters,omitempty"`
}

type ArmorInfo struct {
	EquipType  proto.EEquipType `json:"equipType,omitempty"`
	InstanceId uint32           `json:"instanceId,omitempty"`
}

func (a *ArmorInfo) ArmorInfo() *proto.ArmorInfo {
	return &proto.ArmorInfo{
		EquipType: a.EquipType,
		ArmorId:   a.InstanceId,
	}
}

type PosterInfo struct {
	PosterIndex proto.PosterIndex `json:"posterIndex,omitempty"`
	InstanceId  uint32            `json:"instanceId,omitempty"`
}

func (a *PosterInfo) PosterInfo() *proto.PosterInfo {
	return &proto.PosterInfo{
		PosterIndex: a.PosterIndex,
		PosterId:    a.InstanceId,
	}
}

func newEquipmentPreset(presetIndex uint32) *EquipmentPreset {
	info := &EquipmentPreset{
		PresetIndex:      presetIndex,
		WeaponInstanceId: 0,
		Armors:           make(map[proto.EEquipType]*ArmorInfo),
		Posters:          make(map[proto.PosterIndex]*PosterInfo),
	}
	// 添加盔甲
	for _, tag := range proto.EEquipType_value {
		info.Armors[proto.EEquipType(tag)] = &ArmorInfo{
			EquipType:  proto.EEquipType(tag),
			InstanceId: 0,
		}
	}
	// 添加海报
	for _, index := range proto.PosterIndex_value {
		info.Posters[proto.PosterIndex(index)] = &PosterInfo{
			PosterIndex: proto.PosterIndex(index),
			InstanceId:  0,
		}
	}
	return info
}

func (e *EquipmentPreset) GetArmor(t proto.EEquipType) *ArmorInfo {
	if e.Armors == nil {
		e.Armors = make(map[proto.EEquipType]*ArmorInfo)
	}
	if e.Armors[t] == nil {
		e.Armors[t] = &ArmorInfo{
			EquipType:  t,
			InstanceId: 0,
		}
	}
	return e.Armors[t]
}

func (e *EquipmentPreset) GetPoster(index proto.PosterIndex) *PosterInfo {
	if e.Posters == nil {
		e.Posters = make(map[proto.PosterIndex]*PosterInfo)
	}
	if e.Posters[index] == nil {
		e.Posters[index] = &PosterInfo{
			PosterIndex: index,
			InstanceId:  0,
		}
	}
	return e.Posters[index]
}

func (c *CharacterInfo) GetEquipmentPresetList() map[uint32]*EquipmentPreset {
	if c.EquipmentPresetList == nil {
		c.EquipmentPresetList = make(map[uint32]*EquipmentPreset)
	}
	return c.EquipmentPresetList
}

func (c *CharacterInfo) GetEquipmentPreset(index uint32) *EquipmentPreset {
	list := c.GetEquipmentPresetList()
	info, ok := list[index]
	if !ok {
		info = newEquipmentPreset(index)
		list[index] = info
	}
	return info
}

func (e *EquipmentPreset) EquipmentPreset() *proto.EquipmentPreset {
	info := &proto.EquipmentPreset{
		PresetIndex: e.PresetIndex,
		Weapon:      e.WeaponInstanceId,
		Armors:      make([]*proto.ArmorInfo, 0),
		Posters:     make([]*proto.PosterInfo, 0),
	}
	for _, armor := range e.Armors {
		alg.AddList(&info.Armors, armor.ArmorInfo())
	}
	for _, poster := range e.Posters {
		alg.AddList(&info.Posters, poster.PosterInfo())
	}
	return info
}

func (c *CharacterInfo) GetPbEquipmentPresets() []*proto.EquipmentPreset {
	pbInfoList := make([]*proto.EquipmentPreset, 0)
	for i := uint32(0); i < gdconf.GetConstant().EquipmentPresetNum; i++ {
		e := c.GetEquipmentPreset(i)
		alg.AddList(&pbInfoList, e.EquipmentPreset())
	}
	return pbInfoList
}

type OutfitPreset struct {
	PresetIndex                 uint32          `json:"presetIndex,omitempty"`
	Hat                         uint32          `json:"hat,omitempty"`
	HatDyeSchemeIndex           uint32          `json:"hatDyeSchemeIndex,omitempty"`
	Hair                        uint32          `json:"hair,omitempty"`
	HairDyeSchemeIndex          uint32          `json:"hairDyeSchemeIndex,omitempty"`
	Clothes                     uint32          `json:"clothes,omitempty"`
	ClothesDyeSchemeIndex       uint32          `json:"clothesDyeSchemeIndex,omitempty"`
	Ornament                    uint32          `json:"ornament,omitempty"`
	OrnamentDyeSchemeIndex      uint32          `json:"ornamentDyeSchemeIndex,omitempty"`
	OutfitHideInfo              *OutfitHideInfo `json:"outfitHideInfo,omitempty"`
	PendTop                     uint32          `json:"pendTop,omitempty"`
	PendTopDyeSchemeIndex       uint32          `json:"pendTopDyeSchemeIndex,omitempty"`
	PendChest                   uint32          `json:"pendChest,omitempty"`
	PendChestDyeSchemeIndex     uint32          `json:"pendChestDyeSchemeIndex,omitempty"`
	PendPelvis                  uint32          `json:"pendPelvis,omitempty"`
	PendPelvisDyeSchemeIndex    uint32          `json:"pendPelvisDyeSchemeIndex,omitempty"`
	PendUpFace                  uint32          `json:"pendUpFace,omitempty"`
	PendUpFaceDyeSchemeIndex    uint32          `json:"pendUpFaceDyeSchemeIndex,omitempty"`
	PendDownFace                uint32          `json:"pendDownFace,omitempty"`
	PendDownFaceDyeSchemeIndex  uint32          `json:"pendDownFaceDyeSchemeIndex,omitempty"`
	PendLeftHand                uint32          `json:"pendLeftHand,omitempty"`
	PendLeftHandDyeSchemeIndex  uint32          `json:"pendLeftHandDyeSchemeIndex,omitempty"`
	PendRightHand               uint32          `json:"pendRightHand,omitempty"`
	PendRightHandDyeSchemeIndex uint32          `json:"pendRightHandDyeSchemeIndex,omitempty"`
	PendLeftFoot                uint32          `json:"pendLeftFoot,omitempty"`
	PendLeftFootDyeSchemeIndex  uint32          `json:"pendLeftFootDyeSchemeIndex,omitempty"`
	PendRightFoot               uint32          `json:"pendRightFoot,omitempty"`
	PendRightFootDyeSchemeIndex uint32          `json:"pendRightFootDyeSchemeIndex,omitempty"`
}

type OutfitHideInfo struct {
	HideOrn   bool `json:"hideOrn,omitempty"`
	HideBraid bool `json:"hideBraid,omitempty"`
}

func (o *OutfitHideInfo) OutfitHideInfo() *proto.OutfitHideInfo {
	return &proto.OutfitHideInfo{
		HideOrn:   o.HideOrn,
		HideBraid: o.HideBraid,
	}
}

func (o *OutfitPreset) OutfitPreset() *proto.OutfitPreset {
	info := &proto.OutfitPreset{
		PresetIndex:                 o.PresetIndex,
		Hat:                         o.Hat,
		Hair:                        o.Hair,
		Clothes:                     o.Clothes,
		Ornament:                    o.Ornament,
		HatDyeSchemeIndex:           o.HatDyeSchemeIndex,
		HairDyeSchemeIndex:          o.HairDyeSchemeIndex,
		ClothesDyeSchemeIndex:       o.ClothesDyeSchemeIndex,
		OrnamentDyeSchemeIndex:      o.OrnamentDyeSchemeIndex,
		OutfitHideInfo:              o.OutfitHideInfo.OutfitHideInfo(),
		PendTop:                     o.PendTop,
		PendChest:                   o.PendChest,
		PendPelvis:                  o.PendPelvis,
		PendUpFace:                  o.PendUpFace,
		PendDownFace:                o.PendDownFace,
		PendLeftHand:                o.PendLeftHand,
		PendRightHand:               o.PendRightHand,
		PendLeftFoot:                o.PendLeftFoot,
		PendRightFoot:               o.PendRightFoot,
		PendTopDyeSchemeIndex:       o.PendTopDyeSchemeIndex,
		PendChestDyeSchemeIndex:     o.PendChestDyeSchemeIndex,
		PendPelvisDyeSchemeIndex:    o.PendPelvisDyeSchemeIndex,
		PendUpFaceDyeSchemeIndex:    o.PendUpFaceDyeSchemeIndex,
		PendDownFaceDyeSchemeIndex:  o.PendDownFaceDyeSchemeIndex,
		PendLeftHandDyeSchemeIndex:  o.PendLeftHandDyeSchemeIndex,
		PendRightHandDyeSchemeIndex: o.PendRightHandDyeSchemeIndex,
		PendLeftFootDyeSchemeIndex:  o.PendLeftFootDyeSchemeIndex,
		PendRightFootDyeSchemeIndex: o.PendRightFootDyeSchemeIndex,
	}

	return info
}

func newOutfitPreset(index uint32) *OutfitPreset {
	return &OutfitPreset{
		PresetIndex:            index,
		Hat:                    0,
		HatDyeSchemeIndex:      0,
		Hair:                   0,
		HairDyeSchemeIndex:     0,
		Clothes:                0,
		ClothesDyeSchemeIndex:  0,
		Ornament:               0,
		OrnamentDyeSchemeIndex: 0,
		OutfitHideInfo: &OutfitHideInfo{
			HideOrn:   false,
			HideBraid: false,
		},
		PendTop:                     0,
		PendTopDyeSchemeIndex:       0,
		PendChest:                   0,
		PendChestDyeSchemeIndex:     0,
		PendPelvis:                  0,
		PendPelvisDyeSchemeIndex:    0,
		PendUpFace:                  0,
		PendUpFaceDyeSchemeIndex:    0,
		PendDownFace:                0,
		PendDownFaceDyeSchemeIndex:  0,
		PendLeftHand:                0,
		PendLeftHandDyeSchemeIndex:  0,
		PendRightHand:               0,
		PendRightHandDyeSchemeIndex: 0,
		PendLeftFoot:                0,
		PendLeftFootDyeSchemeIndex:  0,
		PendRightFoot:               0,
		PendRightFootDyeSchemeIndex: 0,
	}
}

func (c *CharacterInfo) GetOutfitPresetList() map[uint32]*OutfitPreset {
	if c.OutfitPresetList == nil {
		c.OutfitPresetList = make(map[uint32]*OutfitPreset)
	}
	return c.OutfitPresetList
}

func (c *CharacterInfo) GetOutfitPreset(index uint32) *OutfitPreset {
	list := c.GetOutfitPresetList()
	info, ok := list[index]
	if !ok {
		info = newOutfitPreset(index)
		list[index] = info
	}
	return info
}

func (c *CharacterInfo) GetPbOutfitPresets() []*proto.OutfitPreset {
	pbInfoList := make([]*proto.OutfitPreset, 0)

	for i := uint32(0); i < gdconf.GetConstant().OutfitPresetNum; i++ {
		o := c.GetOutfitPreset(i)
		alg.AddList(&pbInfoList, o.OutfitPreset())
	}

	return pbInfoList
}
