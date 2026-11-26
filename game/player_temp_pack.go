package game

import (
	"errors"
	"math"
	"reflect"
	"sort"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

const tempPackMaxSize = 30

var (
	errTempPackFull  = errors.New("temporary pack is full")
	errTempPackIndex = errors.New("invalid temporary pack index")
	errTempPackType  = errors.New("invalid temporary pack item type")
	errTempPackOwner = errors.New("temporary pack does not belong to player")
	errTempPackItem  = errors.New("invalid temporary pack item")
	errTempPackEquip = errors.New("equipment type does not match character")
)

type tempPackDeps struct {
	characterWeaponType func(uint32) (proto.EWeaponSystemType, bool)
	armorEquipType      func(uint32) (proto.EEquipType, bool)
}

var productionTempPackDeps = tempPackDeps{
	characterWeaponType: configuredCharacterWeaponType,
	armorEquipType:      configuredArmorEquipType,
}

func configuredCharacterWeaponType(id uint32) (result proto.EWeaponSystemType, ok bool) {
	defer func() {
		if recover() != nil {
			result, ok = 0, false
		}
	}()
	conf := gdconf.GetCharacterAll(id)
	if conf == nil || conf.CharacterInfo == nil {
		return 0, false
	}
	result = proto.EWeaponSystemType(conf.CharacterInfo.NewWeaponSystemType)
	return result, result > proto.EWeaponSystemType_EWeaponSystemType_None
}

func configuredArmorEquipType(id uint32) (result proto.EEquipType, ok bool) {
	defer func() {
		if recover() != nil {
			result, ok = 0, false
		}
	}()
	conf := gdconf.GetArmorAllInfo(id)
	if conf == nil || conf.ArmorInfo == nil {
		return 0, false
	}
	result = proto.EEquipType(conf.ArmorInfo.NewEquipType)
	return result, result >= proto.EEquipType_EEquipType_Head && result <= proto.EEquipType_EEquipType_Shoes
}

func tempPackErrorStatus(err error) proto.StatusCode {
	switch {
	case errors.Is(err, errTempPackFull):
		return proto.StatusCode_StatusCode_TempPackIsFull
	case errors.Is(err, errTempPackIndex), errors.Is(err, errTempPackType),
		errors.Is(err, errTempPackOwner), errors.Is(err, errTempPackEquip):
		return proto.StatusCode_StatusCode_BadReq
	default:
		return proto.StatusCode_StatusCode_ItemGetFail
	}
}

func (g *Game) currentTempPack(s *model.Player) (*model.SceneTempPack, error) {
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.CurScene == nil || scenePlayer.Player == nil ||
		scenePlayer.Player.UserId != s.UserId {
		return nil, errTempPackOwner
	}
	pack := scenePlayer.CurScene.GetTempPack()
	if pack == nil {
		return nil, errTempPackOwner
	}
	return pack, nil
}

func nilBagItem(item model.EBagItemTag) bool {
	if item == nil {
		return true
	}
	value := reflect.ValueOf(item)
	return value.Kind() == reflect.Ptr && value.IsNil()
}

func validateTempPackItem(item model.EBagItemTag) error {
	if nilBagItem(item) {
		return errTempPackItem
	}
	detail := item.ItemDetail()
	if detail == nil || detail.MainItem == nil || detail.MainItem.ItemId == 0 {
		return errTempPackItem
	}
	main := detail.MainItem
	switch typed := item.(type) {
	case *model.ItemBaseInfo:
		if typed.ItemId == 0 || typed.Num <= 0 || main.GetBaseItem() == nil || main.GetBaseItem().Num != typed.Num {
			return errTempPackItem
		}
	case *model.ItemWeaponInfo:
		if typed.WeaponId == 0 || typed.ItemBaseInfo == nil || main.GetWeapon() == nil || main.ItemTag != proto.EBagItemTag_EBagItemTag_Weapon {
			return errTempPackItem
		}
	case *model.ItemArmorInfo:
		if typed.ArmorId == 0 || typed.ItemBaseInfo == nil || main.GetArmor() == nil || main.ItemTag != proto.EBagItemTag_EBagItemTag_Armor {
			return errTempPackItem
		}
	case *model.ItemPosterInfo:
		if typed.PosterId == 0 || typed.ItemBaseInfo == nil || main.GetPoster() == nil || main.ItemTag != proto.EBagItemTag_EBagItemTag_Poster {
			return errTempPackItem
		}
	case *model.ItemFashionInfo:
		if typed.OutfitId == 0 || typed.ItemBaseInfo == nil || main.GetOutfit() == nil || main.ItemTag != proto.EBagItemTag_EBagItemTag_Fashion {
			return errTempPackItem
		}
	case *model.ItemInscriptionInfo:
		if typed.InscriptionId == 0 || typed.ItemBaseInfo == nil || main.GetInscription() == nil || main.ItemTag != proto.EBagItemTag_EBagItemTag_Inscription {
			return errTempPackItem
		}
	case *model.ItemPetInfo:
		if typed.ItemId == 0 || typed.ItemBaseInfo == nil || main.GetPet() == nil || main.ItemTag != proto.EBagItemTag_EBagItemTag_Pet {
			return errTempPackItem
		}
	default:
		return errTempPackType
	}
	return nil
}

func tempPackSnapshot(pack *model.SceneTempPack) (*proto.TempPack, error) {
	if pack == nil {
		return nil, errTempPackOwner
	}
	if len(pack.TempPack) > tempPackMaxSize {
		return nil, errTempPackFull
	}
	result := &proto.TempPack{MaxSize: tempPackMaxSize, ItemInfos: make([]*proto.ItemInfo, 0, len(pack.TempPack))}
	seen := make(map[model.EBagItemTag]struct{}, len(pack.TempPack))
	for index, item := range pack.TempPack {
		if err := validateTempPackItem(item); err != nil {
			return nil, err
		}
		if _, duplicate := seen[item]; duplicate {
			return nil, errTempPackItem
		}
		seen[item] = struct{}{}
		item.SetPackType(proto.PackType_PackType_TempPack)
		detail := item.ItemDetail()
		main := detail.MainItem
		main.TempPackIndex = uint32(index + 1)
		alg.AddList(&result.ItemInfos, main)
	}
	return result, nil
}

func pickupTempPackItems(pack *model.SceneTempPack, dropIndex uint32, pickIndex int32) ([]*proto.ItemDetail, error) {
	if pack == nil {
		return nil, errTempPackOwner
	}
	drop := pack.GetTempAreaPack(dropIndex)
	if drop == nil || len(drop.Items) == 0 {
		return nil, errTempPackIndex
	}
	indices := make([]int, 0, len(drop.Items))
	switch {
	case pickIndex == -1:
		for index := range drop.Items {
			indices = append(indices, index)
		}
	case pickIndex >= 1 && int(pickIndex) <= len(drop.Items):
		indices = append(indices, int(pickIndex)-1)
	default:
		return nil, errTempPackIndex
	}
	if len(pack.TempPack)+len(indices) > tempPackMaxSize {
		return nil, errTempPackFull
	}
	for _, index := range indices {
		if err := validateTempPackItem(drop.Items[index]); err != nil {
			return nil, err
		}
	}
	selected := make([]model.EBagItemTag, 0, len(indices))
	for _, index := range indices {
		selected = append(selected, drop.Items[index])
	}
	if pickIndex == -1 {
		delete(pack.TempAreaPack, dropIndex)
	} else {
		index := int(pickIndex) - 1
		copy(drop.Items[index:], drop.Items[index+1:])
		drop.Items[len(drop.Items)-1] = nil
		drop.Items = drop.Items[:len(drop.Items)-1]
		if len(drop.Items) == 0 {
			delete(pack.TempAreaPack, dropIndex)
		}
	}
	for _, item := range selected {
		item.SetPackType(proto.PackType_PackType_TempPack)
		pack.TempPack = append(pack.TempPack, item)
	}
	items := make([]*proto.ItemDetail, 0, len(selected))
	for _, item := range selected {
		alg.AddList(&items, item.ItemDetail())
	}
	return items, nil
}

func dropTempPackItem(pack *model.SceneTempPack, index uint32) (*proto.TempPack, error) {
	if pack == nil || index < 1 || int(index) > len(pack.TempPack) {
		return nil, errTempPackIndex
	}
	position := int(index) - 1
	if err := validateTempPackItem(pack.TempPack[position]); err != nil {
		return nil, err
	}
	copy(pack.TempPack[position:], pack.TempPack[position+1:])
	pack.TempPack[len(pack.TempPack)-1] = nil
	pack.TempPack = pack.TempPack[:len(pack.TempPack)-1]
	return tempPackSnapshot(pack)
}

func sortTempPack(pack *model.SceneTempPack) (*proto.TempPack, error) {
	if pack == nil {
		return nil, errTempPackOwner
	}
	for _, item := range pack.TempPack {
		if err := validateTempPackItem(item); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(pack.TempPack, func(i, j int) bool {
		left := pack.TempPack[i].ItemDetail().MainItem
		right := pack.TempPack[j].ItemDetail().MainItem
		if left.ItemId != right.ItemId {
			return left.ItemId < right.ItemId
		}
		if left.ItemTag != right.ItemTag {
			return left.ItemTag < right.ItemTag
		}
		return tempPackInstanceID(pack.TempPack[i]) < tempPackInstanceID(pack.TempPack[j])
	})
	return tempPackSnapshot(pack)
}

func tempPackInstanceID(item model.EBagItemTag) uint32 {
	switch typed := item.(type) {
	case *model.ItemWeaponInfo:
		return typed.InstanceId
	case *model.ItemArmorInfo:
		return typed.InstanceId
	case *model.ItemPosterInfo:
		return typed.InstanceId
	case *model.ItemPetInfo:
		return typed.InstanceId
	default:
		return 0
	}
}

func itemInstanceIDUsed(items *model.ItemModel, id uint32) bool {
	if id == 0 {
		return true
	}
	if items.GetItemWeaponInfo(id) != nil || items.GetItemArmorInfo(id) != nil ||
		items.GetItemPosterInfo(id) != nil || items.GetItemPetInfoMap()[id] != nil {
		return true
	}
	return false
}

func nextTempPackInstanceID(items *model.ItemModel) uint32 {
	for {
		id := items.NextInstanceIndex()
		if !itemInstanceIDUsed(items, id) {
			return id
		}
	}
}

func tempPackItemOwned(s *model.Player, item model.EBagItemTag) bool {
	items := s.GetItemModel()
	switch typed := item.(type) {
	case *model.ItemBaseInfo:
		return items.GetItemBaseMap()[typed.ItemId] == typed
	case *model.ItemWeaponInfo:
		return items.GetItemWeaponInfo(typed.InstanceId) == typed
	case *model.ItemArmorInfo:
		return items.GetItemArmorInfo(typed.InstanceId) == typed
	case *model.ItemPosterInfo:
		return items.GetItemPosterInfo(typed.InstanceId) == typed
	case *model.ItemFashionInfo:
		return items.GetItemFashionInfo(typed.OutfitId) == typed
	case *model.ItemInscriptionInfo:
		return items.GetItemInscriptionMap()[typed.InscriptionId] == typed
	case *model.ItemPetInfo:
		return items.GetItemPetInfoMap()[typed.InstanceId] == typed
	default:
		return false
	}
}

// commitTempPackItem moves the server-owned object into the inventory. It does
// not call AddAllTypeItem, which would create a second equipment instance.
func commitTempPackItem(s *model.Player, item model.EBagItemTag) (model.EBagItemTag, error) {
	if err := validateTempPackItem(item); err != nil {
		return nil, err
	}
	if tempPackItemOwned(s, item) {
		return nil, errTempPackItem
	}
	items := s.GetItemModel()
	switch typed := item.(type) {
	case *model.ItemBaseInfo:
		if existing := items.GetItemBaseInfo(typed.ItemId); existing != nil {
			if existing.Num < 0 || typed.Num > math.MaxInt64-existing.Num {
				return nil, errTempPackItem
			}
			existing.Num += typed.Num
			typed.SetPackType(proto.PackType_PackType_Inventory)
			return existing, nil
		}
		items.GetItemBaseMap()[typed.ItemId] = typed
	case *model.ItemWeaponInfo:
		typed.InstanceId = nextTempPackInstanceID(items)
		items.GetItemWeaponMap()[typed.InstanceId] = typed
	case *model.ItemArmorInfo:
		typed.InstanceId = nextTempPackInstanceID(items)
		items.GetItemArmorMap()[typed.InstanceId] = typed
	case *model.ItemPosterInfo:
		typed.InstanceId = nextTempPackInstanceID(items)
		items.GetItemPosterMap()[typed.InstanceId] = typed
	case *model.ItemPetInfo:
		typed.InstanceId = nextTempPackInstanceID(items)
		items.GetItemPetInfoMap()[typed.InstanceId] = typed
	case *model.ItemFashionInfo:
		if existing := items.GetItemFashionInfo(typed.OutfitId); existing != nil {
			return existing, nil
		}
		items.GetItemFashionMap()[typed.OutfitId] = typed
	case *model.ItemInscriptionInfo:
		if existing := items.GetItemInscriptionMap()[typed.InscriptionId]; existing != nil {
			return existing, nil
		}
		items.GetItemInscriptionMap()[typed.InscriptionId] = typed
	default:
		return nil, errTempPackType
	}
	typed := item
	typed.SetPackType(proto.PackType_PackType_Inventory)
	return typed, nil
}

func storeTempPackItems(s *model.Player, pack *model.SceneTempPack, ownerID uint32) ([]*proto.ItemDetail, error) {
	if pack == nil || ownerID == 0 || s.UserId != ownerID {
		return nil, errTempPackOwner
	}
	seen := make(map[model.EBagItemTag]struct{}, len(pack.TempPack))
	baseTotals := make(map[uint32]int64)
	for _, item := range pack.TempPack {
		if err := validateTempPackItem(item); err != nil {
			return nil, err
		}
		if _, duplicate := seen[item]; duplicate {
			return nil, errTempPackItem
		}
		if tempPackItemOwned(s, item) {
			return nil, errTempPackItem
		}
		if base, ok := item.(*model.ItemBaseInfo); ok {
			if base.Num > math.MaxInt64-baseTotals[base.ItemId] {
				return nil, errTempPackItem
			}
			baseTotals[base.ItemId] += base.Num
			if existing := s.GetItemModel().GetItemBaseInfo(base.ItemId); existing != nil &&
				(existing.Num < 0 || baseTotals[base.ItemId] > math.MaxInt64-existing.Num) {
				return nil, errTempPackItem
			}
		}
		seen[item] = struct{}{}
	}
	items := make([]*proto.ItemDetail, 0, len(pack.TempPack))
	for _, item := range pack.TempPack {
		stored, err := commitTempPackItem(s, item)
		if err != nil {
			return nil, err
		}
		alg.AddList(&items, stored.ItemDetail())
	}
	pack.TempPack = pack.TempPack[:0]
	return items, nil
}

func removeTempPackPosition(pack *model.SceneTempPack, position int) {
	copy(pack.TempPack[position:], pack.TempPack[position+1:])
	pack.TempPack[len(pack.TempPack)-1] = nil
	pack.TempPack = pack.TempPack[:len(pack.TempPack)-1]
}

func wearTempPackItem(s *model.Player, pack *model.SceneTempPack, ownerID, index, charID uint32,
	deps tempPackDeps) (*proto.TempPack, []*proto.ItemDetail, error) {
	if pack == nil || ownerID == 0 || s.UserId != ownerID {
		return nil, nil, errTempPackOwner
	}
	if index < 1 || int(index) > len(pack.TempPack) {
		return nil, nil, errTempPackIndex
	}
	character := s.GetCharacterModel().GetCharacterInfo(charID)
	if character == nil || character.InUseEquipmentPresetIndex >= 3 {
		return nil, nil, errTempPackEquip
	}
	item := pack.TempPack[index-1]
	if err := validateTempPackItem(item); err != nil {
		return nil, nil, err
	}
	if tempPackItemOwned(s, item) {
		return nil, nil, errTempPackItem
	}
	charType, ok := deps.characterWeaponType(charID)
	if !ok {
		return nil, nil, errTempPackEquip
	}
	preset := character.GetEquipmentPreset(character.InUseEquipmentPresetIndex)
	items := s.GetItemModel()
	changed := make([]*proto.ItemDetail, 0, 2)
	switch typed := item.(type) {
	case *model.ItemWeaponInfo:
		if typed.WeaponSystemType != charType {
			return nil, nil, errTempPackEquip
		}
		if typed.WearerId != 0 {
			oldCharacter := s.GetCharacterModel().GetCharacterInfo(typed.WearerId)
			if oldCharacter == nil || typed.InstanceId == 0 || typed.WearerIndex >= 3 {
				return nil, nil, errTempPackItem
			}
			oldPreset := oldCharacter.GetEquipmentPreset(typed.WearerIndex)
			if oldPreset.WeaponInstanceId != typed.InstanceId {
				return nil, nil, errTempPackItem
			}
			oldPreset.WeaponInstanceId = 0
		}
		stored, err := commitTempPackItem(s, typed)
		if err != nil {
			return nil, nil, err
		}
		weapon := stored.(*model.ItemWeaponInfo)
		old := items.GetItemWeaponInfo(preset.WeaponInstanceId)
		if old != nil {
			old.SetWearerId(0, 0)
			alg.AddList(&changed, old.ItemDetail())
		}
		preset.WeaponInstanceId = weapon.InstanceId
		weapon.SetWearerId(charID, preset.PresetIndex)
		alg.AddList(&changed, weapon.ItemDetail())
	case *model.ItemArmorInfo:
		if typed.WeaponSystemType != charType {
			return nil, nil, errTempPackEquip
		}
		equipType, ok := deps.armorEquipType(typed.ArmorId)
		if !ok {
			return nil, nil, errTempPackEquip
		}
		if typed.WearerId != 0 {
			oldCharacter := s.GetCharacterModel().GetCharacterInfo(typed.WearerId)
			if oldCharacter == nil || typed.InstanceId == 0 || typed.WearerIndex >= 3 {
				return nil, nil, errTempPackItem
			}
			oldPreset := oldCharacter.GetEquipmentPreset(typed.WearerIndex)
			if oldPreset.GetArmor(equipType).InstanceId != typed.InstanceId {
				return nil, nil, errTempPackItem
			}
			oldPreset.GetArmor(equipType).InstanceId = 0
		}
		stored, err := commitTempPackItem(s, typed)
		if err != nil {
			return nil, nil, err
		}
		armor := stored.(*model.ItemArmorInfo)
		slot := preset.GetArmor(equipType)
		old := items.GetItemArmorInfo(slot.InstanceId)
		if old != nil {
			old.SetWearer(0, 0)
			alg.AddList(&changed, old.ItemDetail())
		}
		slot.InstanceId = armor.InstanceId
		armor.SetWearer(charID, preset.PresetIndex)
		alg.AddList(&changed, armor.ItemDetail())
	default:
		return nil, nil, errTempPackType
	}
	removeTempPackPosition(pack, int(index)-1)
	snapshot, err := tempPackSnapshot(pack)
	if err != nil {
		return nil, nil, err
	}
	return snapshot, changed, nil
}

func configuredMonsterDrop(s *model.Player, pack *model.SceneTempPack, monsterID, playerLevel uint32) (*model.DropItem, error) {
	monster, ok := safeMonsterConfigure(monsterID)
	if !ok {
		return nil, errors.New("monster configuration is missing")
	}
	if monster.RewardPoolID <= 0 {
		return nil, nil
	}
	rewards, err := model.ResolveDropRewards(uint32(monster.RewardPoolID), s.GetSceneModel().GetWorldLevel(), playerLevel)
	if err != nil {
		return nil, err
	}
	// Generate into an isolated model so drops never enter the player inventory
	// before pickup. The resulting object is moved into the temp pack later.
	generator := &model.Player{}
	items := make([]model.EBagItemTag, 0, len(rewards))
	seen := make(map[model.EBagItemTag]struct{}, len(rewards))
	for _, reward := range rewards {
		ctx := generator.AddAllTypeItem(reward.ItemID, reward.Count)
		if ctx == nil {
			return nil, errors.New("drop item configuration is missing")
		}
		item := ctx.EBagItemTag
		if err := validateTempPackItem(item); err != nil {
			return nil, err
		}
		if base, stackable := item.(*model.ItemBaseInfo); stackable {
			copy := *base
			item = &copy
		} else if reward.Count != 1 {
			return nil, errors.New("equipment drop count must be one")
		}
		if _, duplicate := seen[item]; duplicate {
			return nil, errors.New("duplicate drop object")
		}
		seen[item] = struct{}{}
		item.SetPackType(proto.PackType_PackType_TempStorageArea)
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, errors.New("reward pool produced no drops")
	}
	return pack.AddDropItems(items), nil
}

func safeMonsterConfigure(id uint32) (result *excel.MonsterCharacterConfigure, ok bool) {
	defer func() {
		if recover() != nil {
			result, ok = nil, false
		}
	}()
	conf := gdconf.GetMonsterCharacterConfigure(id)
	if conf == nil {
		return nil, false
	}
	return conf, true
}

func (g *Game) pickupTempPack(s *model.Player, req *proto.PickupReq) (proto.StatusCode, []*proto.ItemDetail) {
	pack, err := g.currentTempPack(s)
	if err != nil {
		return tempPackErrorStatus(err), nil
	}
	items, err := pickupTempPackItems(pack, req.DropItemIndex, req.PickIndex)
	if err != nil {
		return tempPackErrorStatus(err), nil
	}
	if len(items) > 0 {
		g.PackNoticeByItems(s, items)
	}
	return proto.StatusCode_StatusCode_Ok, items
}

func (g *Game) TempPackItemDrop(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TempPackItemDropReq)
	rsp := &proto.TempPackItemDropRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	pack, err := g.currentTempPack(s)
	if err == nil {
		rsp.TempPack, err = dropTempPackItem(pack, req.Index)
	}
	if err != nil {
		rsp.Status = tempPackErrorStatus(err)
	}
}

func (g *Game) TempPackItemStore(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.TempPackItemStoreRsp{Status: proto.StatusCode_StatusCode_Ok, Items: make([]*proto.ItemDetail, 0)}
	defer g.send(s, msg.PacketId, rsp)
	pack, err := g.currentTempPack(s)
	if err == nil {
		rsp.Items, err = storeTempPackItems(s, pack, s.UserId)
	}
	if err != nil {
		rsp.Status = tempPackErrorStatus(err)
		return
	}
	if len(rsp.Items) > 0 {
		g.PackNoticeByItems(s, rsp.Items)
	}
}

func (g *Game) TempPackSort(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.TempPackSortRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	pack, err := g.currentTempPack(s)
	if err == nil {
		rsp.TempPack, err = sortTempPack(pack)
	}
	if err != nil {
		rsp.Status = tempPackErrorStatus(err)
	}
}

func (g *Game) TempPackWearEquip(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TempPackWearEquipReq)
	rsp := &proto.TempPackWearEquipRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	pack, err := g.currentTempPack(s)
	var items []*proto.ItemDetail
	if err == nil {
		rsp.TempPack, items, err = wearTempPackItem(s, pack, s.UserId, req.Index, req.CharId, productionTempPackDeps)
	}
	if err != nil {
		rsp.Status = tempPackErrorStatus(err)
		return
	}
	if len(items) > 0 {
		g.PackNoticeByItems(s, items)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdateEquip, req.CharId)
	}
}
