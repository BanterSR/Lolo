package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func validItemBaseInstance(item *model.ItemBaseInfo, itemId uint32, tag proto.EBagItemTag) bool {
	if item == nil || item.ItemId != itemId || item.ItemType != tag {
		return false
	}
	conf := gdconf.GetItemConfigure(itemId)
	return conf != nil && proto.EBagItemTag(conf.GetNewBagItemTag()) == tag
}

func validWeaponInstance(s *model.Player, instanceId uint32) (*model.ItemWeaponInfo, *excel.WeaponConfigure, bool) {
	if s == nil || instanceId == 0 {
		return nil, nil, false
	}
	item := s.GetItemModel().GetItemWeaponInfo(instanceId)
	if item == nil || item.InstanceId != instanceId ||
		!validItemBaseInstance(item.ItemBaseInfo, item.ItemId, proto.EBagItemTag_EBagItemTag_Weapon) {
		return nil, nil, false
	}
	conf := gdconf.GetWeaponAllInfo(item.WeaponId)
	if conf == nil || conf.WeaponInfo == nil ||
		uint32(conf.WeaponInfo.GetID()) != item.WeaponId ||
		uint32(conf.WeaponInfo.GetItemID()) != item.ItemId ||
		proto.EWeaponSystemType(conf.WeaponInfo.GetNewWeaponSystemType()) != item.WeaponSystemType {
		return nil, nil, false
	}
	return item, conf.WeaponInfo, true
}

func validArmorInstance(s *model.Player, instanceId uint32) (*model.ItemArmorInfo, *excel.ArmorConfigure, bool) {
	if s == nil || instanceId == 0 {
		return nil, nil, false
	}
	item := s.GetItemModel().GetItemArmorInfo(instanceId)
	if item == nil || item.InstanceId != instanceId ||
		!validItemBaseInstance(item.ItemBaseInfo, item.ItemId, proto.EBagItemTag_EBagItemTag_Armor) {
		return nil, nil, false
	}
	conf := gdconf.GetArmorAllInfo(item.ArmorId)
	if conf == nil || conf.ArmorInfo == nil ||
		uint32(conf.ArmorInfo.GetID()) != item.ArmorId ||
		uint32(conf.ArmorInfo.GetItemID()) != item.ItemId ||
		proto.EWeaponSystemType(conf.ArmorInfo.GetNewWeaponSystemType()) != item.WeaponSystemType {
		return nil, nil, false
	}
	return item, conf.ArmorInfo, true
}

func validPosterInstance(s *model.Player, instanceId uint32) (*model.ItemPosterInfo, *excel.PosterConfigure, bool) {
	if s == nil || instanceId == 0 {
		return nil, nil, false
	}
	item := s.GetItemModel().GetItemPosterInfo(instanceId)
	if item == nil || item.InstanceId != instanceId ||
		!validItemBaseInstance(item.ItemBaseInfo, item.ItemId, proto.EBagItemTag_EBagItemTag_Poster) {
		return nil, nil, false
	}
	conf := gdconf.GetPosterAllInfo(item.PosterId)
	if conf == nil || conf.PosterInfo == nil ||
		uint32(conf.PosterInfo.GetID()) != item.PosterId ||
		uint32(conf.PosterInfo.GetItemID()) != item.ItemId {
		return nil, nil, false
	}
	return item, conf.PosterInfo, true
}

func validPresetIndex(index uint32) bool {
	constant := gdconf.GetConstant()
	return constant != nil && index < constant.EquipmentPresetNum
}

func validArmorSlot(slot proto.EEquipType) bool {
	_, ok := proto.EEquipType_name[int32(slot)]
	return ok
}

func validPosterSlot(slot proto.PosterIndex) bool {
	_, ok := proto.PosterIndex_name[int32(slot)]
	return ok
}

type equipmentKind uint8

const (
	equipmentWeapon equipmentKind = iota
	equipmentArmor
	equipmentPoster
)

type equipmentSlot struct {
	kind equipmentKind
	slot uint32
	old  uint32
	new  uint32
}

type equipmentLocation struct {
	character   *model.CharacterInfo
	preset      *model.EquipmentPreset
	kind        equipmentKind
	slot        uint32
	instanceId  uint32
	clearWearer bool
}

func equipmentInstanceId(preset *model.EquipmentPreset, kind equipmentKind, slot uint32) uint32 {
	if preset == nil {
		return 0
	}
	switch kind {
	case equipmentWeapon:
		return preset.WeaponInstanceId
	case equipmentArmor:
		if armor := preset.Armors[proto.EEquipType(slot)]; armor != nil {
			return armor.InstanceId
		}
	case equipmentPoster:
		if poster := preset.Posters[proto.PosterIndex(slot)]; poster != nil {
			return poster.InstanceId
		}
	}
	return 0
}

func setEquipmentInstanceId(preset *model.EquipmentPreset, kind equipmentKind, slot, instanceId uint32) {
	switch kind {
	case equipmentWeapon:
		preset.WeaponInstanceId = instanceId
	case equipmentArmor:
		preset.GetArmor(proto.EEquipType(slot)).InstanceId = instanceId
	case equipmentPoster:
		preset.GetPoster(proto.PosterIndex(slot)).InstanceId = instanceId
	}
}

func equipmentInstance(s *model.Player, kind equipmentKind, instanceId uint32) (model.EBagItemTag, bool) {
	switch kind {
	case equipmentWeapon:
		item, _, ok := validWeaponInstance(s, instanceId)
		if !ok {
			return nil, false
		}
		return item, true
	case equipmentArmor:
		item, _, ok := validArmorInstance(s, instanceId)
		if !ok {
			return nil, false
		}
		return item, true
	case equipmentPoster:
		item, _, ok := validPosterInstance(s, instanceId)
		if !ok {
			return nil, false
		}
		return item, true
	default:
		return nil, false
	}
}

func setEquipmentWearer(s *model.Player, kind equipmentKind, instanceId, characterId, presetIndex uint32) bool {
	item, ok := equipmentInstance(s, kind, instanceId)
	if !ok {
		return false
	}
	switch value := item.(type) {
	case *model.ItemWeaponInfo:
		value.SetWearerId(characterId, presetIndex)
	case *model.ItemArmorInfo:
		value.SetWearer(characterId, presetIndex)
	case *model.ItemPosterInfo:
		value.SetWearer(characterId, presetIndex)
	default:
		return false
	}
	return true
}

func equipmentWearer(s *model.Player, kind equipmentKind, instanceId uint32) (uint32, uint32, bool) {
	item, ok := equipmentInstance(s, kind, instanceId)
	if !ok {
		return 0, 0, false
	}
	switch value := item.(type) {
	case *model.ItemWeaponInfo:
		return value.WearerId, value.WearerIndex, true
	case *model.ItemArmorInfo:
		return value.WearerId, value.WearerIndex, true
	case *model.ItemPosterInfo:
		return value.WearerId, value.WearerIndex, true
	default:
		return 0, 0, false
	}
}

// ownedEquipmentLocation 验证实例的 wearer 元数据确实指向玩家角色和对应槽位。
func ownedEquipmentLocation(s *model.Player, character *model.CharacterInfo, presetIndex uint32,
	kind equipmentKind, slot, instanceId uint32) (*equipmentLocation, bool) {
	if instanceId == 0 {
		return nil, true
	}
	wearerId, wearerIndex, ok := equipmentWearer(s, kind, instanceId)
	if !ok || wearerId != character.CharacterId || wearerIndex != presetIndex {
		return nil, false
	}
	preset := character.EquipmentPresetList[presetIndex]
	if preset == nil || preset.PresetIndex != presetIndex ||
		equipmentInstanceId(preset, kind, slot) != instanceId {
		return nil, false
	}
	return &equipmentLocation{
		character: character, preset: preset, kind: kind, slot: slot,
		instanceId: instanceId, clearWearer: true,
	}, true
}

func equipmentInstanceExists(s *model.Player, kind equipmentKind, instanceId uint32) bool {
	switch kind {
	case equipmentWeapon:
		return s.GetItemModel().GetItemWeaponInfo(instanceId) != nil
	case equipmentArmor:
		return s.GetItemModel().GetItemArmorInfo(instanceId) != nil
	case equipmentPoster:
		return s.GetItemModel().GetItemPosterInfo(instanceId) != nil
	default:
		return false
	}
}

// targetEquipmentLocation 允许 0 卸下请求清理旧存档中的缺失/未佩戴实例引用；
// 若实例仍属于其他角色，则拒绝误改其他角色的 wearer。
func targetEquipmentLocation(s *model.Player, character *model.CharacterInfo, presetIndex uint32,
	kind equipmentKind, slot, instanceId uint32) (*equipmentLocation, bool) {
	if instanceId == 0 {
		return nil, true
	}
	preset := character.EquipmentPresetList[presetIndex]
	if preset == nil || preset.PresetIndex != presetIndex ||
		equipmentInstanceId(preset, kind, slot) != instanceId {
		return nil, false
	}
	location := &equipmentLocation{character: character, preset: preset, kind: kind, slot: slot, instanceId: instanceId}
	if !equipmentInstanceExists(s, kind, instanceId) {
		return location, true
	}
	wearerId, wearerIndex, ok := equipmentWearer(s, kind, instanceId)
	if !ok {
		return nil, false
	}
	if wearerId != 0 && (wearerId != character.CharacterId || wearerIndex != presetIndex) {
		return nil, false
	}
	location.clearWearer = true
	return location, true
}

func buildEquipmentUpdate(s *model.Player, req *proto.CharacterEquipUpdateReq) ([]equipmentSlot, []*model.CharacterInfo, bool) {
	if s == nil || req == nil || req.EquipmentPreset == nil || !validPresetIndex(req.EquipmentPreset.PresetIndex) {
		return nil, nil, false
	}
	character := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if character == nil {
		return nil, nil, false
	}
	preset := character.EquipmentPresetList[req.EquipmentPreset.PresetIndex]
	if preset != nil && preset.PresetIndex != req.EquipmentPreset.PresetIndex {
		return nil, nil, false
	}

	requested := make(map[uint64]struct{}, len(req.EquipmentPreset.Armors)+len(req.EquipmentPreset.Posters)+1)
	seenInstances := make(map[uint32]struct{}, len(req.EquipmentPreset.Armors)+len(req.EquipmentPreset.Posters)+1)
	addRequest := func(kind equipmentKind, slot, instanceId uint32) bool {
		key := uint64(kind)<<32 | uint64(slot)
		if _, ok := requested[key]; ok {
			return false
		}
		requested[key] = struct{}{}
		if instanceId != 0 {
			if _, ok := seenInstances[instanceId]; ok {
				return false
			}
			seenInstances[instanceId] = struct{}{}
			if _, ok := equipmentInstance(s, kind, instanceId); !ok {
				return false
			}
		}
		return true
	}

	weaponId := req.EquipmentPreset.Weapon
	if !addRequest(equipmentWeapon, 0, weaponId) {
		return nil, nil, false
	}
	for _, armor := range req.EquipmentPreset.Armors {
		if armor == nil || !validArmorSlot(armor.EquipType) || !addRequest(equipmentArmor, uint32(armor.EquipType), armor.ArmorId) {
			return nil, nil, false
		}
		if armor.ArmorId != 0 {
			_, conf, ok := validArmorInstance(s, armor.ArmorId)
			if !ok || proto.EEquipType(conf.GetNewEquipType()) != armor.EquipType {
				return nil, nil, false
			}
		}
	}
	for _, poster := range req.EquipmentPreset.Posters {
		if poster == nil || !validPosterSlot(poster.PosterIndex) || !addRequest(equipmentPoster, uint32(poster.PosterIndex), poster.PosterId) {
			return nil, nil, false
		}
	}

	finalIds := make(map[uint64]uint32)
	finalIds[uint64(equipmentWeapon)<<32] = equipmentInstanceId(preset, equipmentWeapon, 0)
	for _, slot := range proto.EEquipType_value {
		key := uint64(equipmentArmor)<<32 | uint64(slot)
		finalIds[key] = equipmentInstanceId(preset, equipmentArmor, uint32(slot))
	}
	for _, slot := range proto.PosterIndex_value {
		key := uint64(equipmentPoster)<<32 | uint64(slot)
		finalIds[key] = equipmentInstanceId(preset, equipmentPoster, uint32(slot))
	}
	finalIds[uint64(equipmentWeapon)<<32] = weaponId
	for _, armor := range req.EquipmentPreset.Armors {
		finalIds[uint64(equipmentArmor)<<32|uint64(armor.EquipType)] = armor.ArmorId
	}
	for _, poster := range req.EquipmentPreset.Posters {
		finalIds[uint64(equipmentPoster)<<32|uint64(poster.PosterIndex)] = poster.PosterId
	}
	seenFinal := make(map[uint32]struct{}, len(finalIds))
	for _, instanceId := range finalIds {
		if instanceId == 0 {
			continue
		}
		if _, ok := seenFinal[instanceId]; ok {
			return nil, nil, false
		}
		seenFinal[instanceId] = struct{}{}
	}

	updates := make([]equipmentSlot, 0, len(requested))
	for key := range requested {
		kind := equipmentKind(key >> 32)
		slot := uint32(key)
		oldId := equipmentInstanceId(preset, kind, slot)
		newId := finalIds[key]
		if oldId != newId {
			updates = append(updates, equipmentSlot{kind: kind, slot: slot, old: oldId, new: newId})
		}
	}
	return updates, []*model.CharacterInfo{character}, true
}

// applyCharacterEquipment 的全部校验在修改任何预设或 wearer 之前完成。
func applyCharacterEquipment(s *model.Player, req *proto.CharacterEquipUpdateReq, rsp *proto.CharacterEquipUpdateRsp) []uint32 {
	if rsp == nil {
		return nil
	}
	rsp.Status = proto.StatusCode_StatusCode_BadReq
	if s == nil || req == nil {
		return nil
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil
	}
	defer tx.Rollback()

	updates, characters, ok := buildEquipmentUpdate(s, req)
	if !ok {
		return nil
	}
	character := characters[0]
	presetIndex := req.EquipmentPreset.PresetIndex
	preset := character.EquipmentPresetList[presetIndex]

	clearLocations := make(map[equipmentLocation]struct{})
	affectedCharacters := map[uint32]*model.CharacterInfo{character.CharacterId: character}
	affectedItems := make(map[uint32]struct{})
	for _, update := range updates {
		if update.old != 0 {
			location, valid := targetEquipmentLocation(s, character, presetIndex, update.kind, update.slot, update.old)
			if !valid {
				return nil
			}
			clearLocations[*location] = struct{}{}
			affectedItems[update.old] = struct{}{}
		}
		if update.new == 0 {
			continue
		}
		wearerId, wearerIndex, valid := equipmentWearer(s, update.kind, update.new)
		if !valid {
			return nil
		}
		affectedItems[update.new] = struct{}{}
		if wearerId == 0 {
			if wearerIndex != 0 {
				return nil
			}
			continue
		}
		oldCharacter := s.GetCharacterModel().GetCharacterInfo(wearerId)
		if oldCharacter == nil || !validPresetIndex(wearerIndex) {
			return nil
		}
		location, valid := ownedEquipmentLocation(s, oldCharacter, wearerIndex, update.kind, update.slot, update.new)
		if !valid {
			return nil
		}
		clearLocations[*location] = struct{}{}
		affectedCharacters[oldCharacter.CharacterId] = oldCharacter
	}

	// 先清理所有旧槽位，再统一写入，兼容同一请求内的装备交换和跨角色换装。
	for location := range clearLocations {
		setEquipmentInstanceId(location.preset, location.kind, location.slot, 0)
		if location.clearWearer && !setEquipmentWearer(s, location.kind, location.instanceId, 0, 0) {
			return nil
		}
	}
	if preset == nil {
		preset = character.GetEquipmentPreset(presetIndex)
	}
	for _, update := range updates {
		setEquipmentInstanceId(preset, update.kind, update.slot, update.new)
		if update.new != 0 && !setEquipmentWearer(s, update.kind, update.new, character.CharacterId, presetIndex) {
			return nil
		}
	}

	rsp.Status = proto.StatusCode_StatusCode_Ok
	for _, affected := range affectedCharacters {
		rsp.Character = append(rsp.Character, affected.Character())
	}
	for instanceId := range affectedItems {
		kind := equipmentWeapon
		if s.GetItemModel().GetItemArmorInfo(instanceId) != nil {
			kind = equipmentArmor
		} else if s.GetItemModel().GetItemPosterInfo(instanceId) != nil {
			kind = equipmentPoster
		}
		item, ok := equipmentInstance(s, kind, instanceId)
		if ok {
			rsp.Items = append(rsp.Items, item.ItemDetail())
		}
	}
	tx.Commit()
	return affectedCharacterIds(affectedCharacters)
}

func affectedCharacterIds(characters map[uint32]*model.CharacterInfo) []uint32 {
	ids := make([]uint32, 0, len(characters))
	for id := range characters {
		ids = append(ids, id)
	}
	return ids
}

func validateCharacterAppearance(s *model.Player, req *proto.UpdateCharacterAppearanceReq) (*model.CharacterAppearance, bool) {
	if s == nil || req == nil || req.Appearance == nil ||
		s.GetCharacterModel().GetCharacterInfo(req.CharId) == nil {
		return nil, false
	}
	appearance := req.Appearance
	if appearance.Badge != 0 && !validOwnedBaseItem(s, appearance.Badge, proto.EBagItemTag_EBagItemTag_Badge) {
		return nil, false
	}
	if appearance.UmbrellaId != 0 && !validOwnedBaseItem(s, appearance.UmbrellaId, proto.EBagItemTag_EBagItemTag_Umbrella) {
		return nil, false
	}
	tools := []struct {
		instanceId uint32
		weaponType proto.EWeaponType
	}{
		{appearance.InsectNetInstanceId, proto.EWeaponType_EWeaponType_InsectNet},
		{appearance.LoggingAxeInstanceId, proto.EWeaponType_EWeaponType_LoggingAxe},
		{appearance.WaterBottleInstanceId, proto.EWeaponType_EWeaponType_WaterBottle},
		{appearance.MiningHammerInstanceId, proto.EWeaponType_EWeaponType_MiningHammer},
		{appearance.CollectionGlovesInstanceId, proto.EWeaponType_EWeaponType_CollectionGloves},
		{appearance.FishingRodInstanceId, proto.EWeaponType_EWeaponType_FishingRod},
		{appearance.VehicleInstanceId, proto.EWeaponType_EWeaponType_Vehicle},
		{appearance.ShovelInstanceId, proto.EWeaponType_EWeaponType_Shovel},
	}
	seen := make(map[uint32]struct{}, len(tools))
	for _, tool := range tools {
		if tool.instanceId == 0 {
			continue
		}
		if _, ok := seen[tool.instanceId]; ok {
			return nil, false
		}
		seen[tool.instanceId] = struct{}{}
		item, conf, ok := validWeaponInstance(s, tool.instanceId)
		if !ok || item == nil || proto.EWeaponType(conf.GetNewWeaponType()) != tool.weaponType {
			return nil, false
		}
	}
	return &model.CharacterAppearance{
		Badge:                      appearance.Badge,
		UmbrellaId:                 appearance.UmbrellaId,
		InsectNetInstanceId:        appearance.InsectNetInstanceId,
		LoggingAxeInstanceId:       appearance.LoggingAxeInstanceId,
		WaterBottleInstanceId:      appearance.WaterBottleInstanceId,
		MiningHammerInstanceId:     appearance.MiningHammerInstanceId,
		CollectionGlovesInstanceId: appearance.CollectionGlovesInstanceId,
		FishingRodInstanceId:       appearance.FishingRodInstanceId,
		VehicleInstanceId:          appearance.VehicleInstanceId,
		ShovelInstanceId:           appearance.ShovelInstanceId,
	}, true
}

func validOwnedBaseItem(s *model.Player, itemId uint32, tag proto.EBagItemTag) bool {
	item := s.GetItemModel().GetItemBaseInfo(itemId)
	return item != nil && item.Num > 0 && validItemBaseInstance(item, itemId, tag)
}

func validGatherWeapon(s *model.Player, instanceId uint32) (*model.ItemWeaponInfo, bool) {
	if instanceId == 0 {
		return nil, true
	}
	item, conf, ok := validWeaponInstance(s, instanceId)
	if !ok {
		return nil, false
	}
	switch proto.EWeaponType(conf.GetNewWeaponType()) {
	case proto.EWeaponType_EWeaponType_CollectionGloves,
		proto.EWeaponType_EWeaponType_LoggingAxe,
		proto.EWeaponType_EWeaponType_MiningHammer,
		proto.EWeaponType_EWeaponType_InsectNet,
		proto.EWeaponType_EWeaponType_WaterBottle,
		proto.EWeaponType_EWeaponType_All:
		return item, true
	default:
		return nil, false
	}
}
