package model

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func installTestEquipmentGrowthConfig(t *testing.T) *equipmentGrowthConfig {
	t.Helper()
	old := equipmentGrowthConfigOverride
	cfg := &equipmentGrowthConfig{
		weapons:             make(map[uint32]*excel.WeaponConfigure),
		weaponProperties:    make(map[int32]*excel.WeaponPropertyConfigure),
		weaponStrengthen:    make(map[int32]*excel.WeaponStrengthenConfigure),
		weaponDisassemble:   make(map[int32]*excel.WeaponDisassembleConfigure),
		armors:              make(map[uint32]*excel.ArmorConfigure),
		armorProperties:     make(map[int32]*excel.ArmorPropertyConfigure),
		armorStrengthen:     make(map[int32]*excel.ArmorStrengthenConfigure),
		armorDisassemble:    make(map[int32]*excel.ArmorDisassembleConfigure),
		posters:             make(map[uint32]*excel.PosterConfigure),
		posterProperties:    make(map[int32]*excel.PropertyConfigure),
		posterIllustrations: make(map[uint32]*excel.PosterIllustrationConfigure),
		inscriptions:        make(map[uint32]*excel.InscriptionConfigure),
		items:               make(map[uint32]*excel.ItemConfigure),
		cookAffects:         make(map[int32]*excel.CookAffectConfigure),
		buffs:               make(map[int32]*excel.BuffConfigure),
		giftBags:            make(map[int32]*excel.GiftBagConfigure),
		rewardPools:         make(map[int32]*excel.RewardPoolConfigure),
		rewardItemPools:     make(map[int32]*excel.RewardItemPoolConfigure),
		globalConfig:        make(map[string]*excel.GlobalConfigConfigure),
	}
	cfg.items[101] = &excel.ItemConfigure{ID: 101, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)}
	cfg.items[270] = &excel.ItemConfigure{ID: 270, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)}
	cfg.items[250] = &excel.ItemConfigure{ID: 250, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)}
	cfg.items[6000010] = &excel.ItemConfigure{ID: 6000010, RelateID: 6000010, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Gift)}
	cfg.items[900] = &excel.ItemConfigure{ID: 900, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)}
	cfg.items[5013000] = &excel.ItemConfigure{ID: 5013000, RelateID: 5013000, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Food)}

	cfg.weapons[1] = &excel.WeaponConfigure{ID: 1, ItemID: 1, WeaponPropertyID: 10}
	cfg.weaponProperties[10] = &excel.WeaponPropertyConfigure{
		ID: 10,
		WeaponPropertyGroupInfo: []*excel.WeaponPropertyGroupInfo{{
			MinLevel: 1, MaxLevel: 10, WeaponLevel: 1, StrengthenID: 100, DisassembleID: 200,
		}},
	}
	cfg.weaponStrengthen[100] = &excel.WeaponStrengthenConfigure{
		ID: 100,
		WeaponStrengthenGroupInfo: []*excel.WeaponStrengthenGroupInfo{{
			Level: 1, AddAttack: 2, NeedGoldCount: 5,
			WeaponStrengthNeedItem: []*excel.WeaponStrengthNeedItems{{NeedItemID: 270, NeedItemCount: 2}},
		}},
	}
	cfg.weaponDisassemble[200] = &excel.WeaponDisassembleConfigure{
		ID:                    200,
		WeaponDisassembleItem: []*excel.WeaponDisassembleItem{{ItemID: 250, ItemMinCount: 1, ItemMaxCount: 1, ItemRate: 1}},
	}

	cfg.armors[2] = &excel.ArmorConfigure{ID: 2, ItemID: 2, ArmorPropertyID: 20}
	cfg.armorProperties[20] = &excel.ArmorPropertyConfigure{
		ID: 20,
		ArmorPropertyGroupInfo: []*excel.ArmorPropertyGroupInfo{{
			MinLevel: 1, MaxLevel: 10, ArmorLevel: 1, StrengthenID: 300, DisassembleID: 400,
		}},
	}
	cfg.armorStrengthen[300] = &excel.ArmorStrengthenConfigure{
		ID: 300,
		ArmorStrengthenGroupInfo: []*excel.ArmorStrengthenGroupInfo{{
			Level: 1, NeedGoldCount: 7,
			ArmorStrengthNeedItem: []*excel.ArmorStrengthNeedItems{{NeedItemID: 250, NeedItemCount: 3}},
		}},
	}
	cfg.armorDisassemble[400] = &excel.ArmorDisassembleConfigure{
		ID:                   400,
		ArmorDisassembleItem: []*excel.ArmorDisassembleItem{{ItemID: 250, ItemMinCount: 2, ItemMaxCount: 2, ItemRate: 1}},
	}

	cfg.posters[3] = &excel.PosterConfigure{ID: 3, ItemID: 3, IllustrationID: 3, PropertyID: 30}
	cfg.posterIllustrations[3] = &excel.PosterIllustrationConfigure{ID: 3, IsShow: true}
	cfg.posterProperties[30] = &excel.PropertyConfigure{
		ID:                      30,
		PosterPropertyGroupInfo: []*excel.PosterPropertyGroupInfo{{Star: 1}, {Star: 2}, {Star: 3}},
	}
	cfg.inscriptions[4] = &excel.InscriptionConfigure{ID: 4, ItemID: 4}

	cfg.giftBags[6000010] = &excel.GiftBagConfigure{
		ID: 6000010, PlayerLv: 1,
		GiftBagGroup: []*excel.GiftBagGroupInfo{{RewardPoolGroupID: 700}},
	}
	cfg.rewardPools[700] = &excel.RewardPoolConfigure{
		ID:              700,
		RewardPoolGroup: []*excel.RewardPoolGroupInfo{{RewardPoolGroupID: 701}},
	}
	cfg.rewardItemPools[701] = &excel.RewardItemPoolConfigure{
		ID:                  701,
		RewardItemPoolGroup: []*excel.RewardItemPoolGroupInfo{{ItemID: 900, ItemMinCount: 2, ItemMaxCount: 2, MinLevelLimit: 1, MaxLevelLimit: 100}},
	}
	cfg.cookAffects[5013000] = &excel.CookAffectConfigure{ID: 5013000, FoodEffectTypeParam: 60, BuffID: 5013000, NewCookFoodEffectType: 1}
	cfg.buffs[5013000] = &excel.BuffConfigure{ID: 5013000, ScriptID: 1}

	cfg.globalConfig["Equip_Disassemble_MaxCount"] = &excel.GlobalConfigConfigure{Key: "Equip_Disassemble_MaxCount", Values: []string{"100"}}
	cfg.globalConfig["UpgradeStarNeedItem"] = &excel.GlobalConfigConfigure{Key: "UpgradeStarNeedItem", Values: []string{"101:10"}}
	cfg.globalConfig["ItemUseCooldown"] = &excel.GlobalConfigConfigure{Key: "ItemUseCooldown", Values: []string{"60"}}
	equipmentGrowthConfigOverride = cfg
	t.Cleanup(func() { equipmentGrowthConfigOverride = old })
	return cfg
}

func newEquipmentTestPlayer() *Player {
	return &Player{
		Item: &ItemModel{
			ItemBaseInfo:       make(map[uint32]*ItemBaseInfo),
			ItemWeaponMap:      make(map[uint32]*ItemWeaponInfo),
			ItemArmorMap:       make(map[uint32]*ItemArmorInfo),
			ItemPosterMap:      make(map[uint32]*ItemPosterInfo),
			ItemInscriptionMap: make(map[uint32]*ItemInscriptionInfo),
		},
		Character: &CharacterModel{CharacterMap: make(map[uint32]*CharacterInfo)},
	}
}

func addEquipmentTestBase(t *testing.T, p *Player, cfg *equipmentGrowthConfig, itemID uint32, count int64) *ItemBaseInfo {
	t.Helper()
	conf := cfg.items[itemID]
	if conf == nil {
		t.Fatalf("missing test item config %d", itemID)
	}
	info := &ItemBaseInfo{
		ItemId: itemID, Num: count, ItemType: proto.EBagItemTag(conf.NewBagItemTag), PackType: proto.ItemDetail_PackType_Inventory,
	}
	p.Item.ItemBaseInfo[itemID] = info
	return info
}

func addEquipmentTestWeapon(p *Player, instanceID uint32) *ItemWeaponInfo {
	info := &ItemWeaponInfo{
		ItemBaseInfo: &ItemBaseInfo{ItemId: 1, ItemType: proto.EBagItemTag_EBagItemTag_Weapon, PackType: proto.ItemDetail_PackType_Inventory},
		WeaponId:     1, InstanceId: instanceID, Level: 1, Attack: 10, Star: 1,
	}
	p.Item.ItemWeaponMap[instanceID] = info
	return info
}

func addEquipmentTestArmor(p *Player, instanceID uint32) *ItemArmorInfo {
	info := &ItemArmorInfo{
		ItemBaseInfo: &ItemBaseInfo{ItemId: 2, ItemType: proto.EBagItemTag_EBagItemTag_Armor, PackType: proto.ItemDetail_PackType_Inventory},
		ArmorId:      2, InstanceId: instanceID, Level: 1, Star: 1,
	}
	p.Item.ItemArmorMap[instanceID] = info
	return info
}

func addEquipmentTestPoster(p *Player, instanceID uint32) *ItemPosterInfo {
	info := &ItemPosterInfo{
		ItemBaseInfo: &ItemBaseInfo{ItemId: 3, ItemType: proto.EBagItemTag_EBagItemTag_Poster, PackType: proto.ItemDetail_PackType_Inventory},
		PosterId:     3, InstanceId: instanceID, Star: 1,
	}
	p.Item.ItemPosterMap[instanceID] = info
	return info
}

func addEquipmentTestInscription(p *Player, inscriptionID uint32) *ItemInscriptionInfo {
	info := &ItemInscriptionInfo{
		ItemBaseInfo:  &ItemBaseInfo{ItemId: inscriptionID, ItemType: proto.EBagItemTag_EBagItemTag_Inscription, PackType: proto.ItemDetail_PackType_Inventory},
		InscriptionId: inscriptionID, Level: 1,
	}
	p.Item.ItemInscriptionMap[inscriptionID] = info
	return info
}

func requireGrowthStatus(t *testing.T, err error, expected proto.StatusCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected status %s, got success", expected)
	}
	if actual := EquipmentErrorStatus(err); actual != expected {
		t.Fatalf("expected status %s, got %s (%v)", expected, actual, err)
	}
}

func TestEquipmentGrowthConfigAcceptsSnakeCaseResourceKeys(t *testing.T) {
	normalized := normalizeEquipmentGrowthJSONKeys(map[string]any{
		"i_d":              101,
		"new_bag_item_tag": 9,
	})
	data, err := sonic.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	var conf excel.ItemConfigure
	if err := sonic.Unmarshal(data, &conf); err != nil {
		t.Fatal(err)
	}
	if conf.ID != 101 || conf.NewBagItemTag != 9 {
		t.Fatalf("snake_case resource keys were not decoded: id=%d tag=%d", conf.ID, conf.NewBagItemTag)
	}
}

func TestEquipmentGrowthRejectsGhostAndWrongType(t *testing.T) {
	installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	armor := addEquipmentTestArmor(p, 2)

	if _, _, err := sStrengthenWeaponForTest(p, 999); err == nil {
		t.Fatal("ghost weapon strengthening succeeded")
	}
	if _, _, err := sStrengthenWeaponForTest(p, armor.InstanceId); err == nil {
		t.Fatal("armor instance was accepted as a weapon")
	}
	if _, err := p.ToggleEquipmentLock(armor.InstanceId, proto.LockEquipReq_LockEquip_EquipType_Weapon); err == nil {
		t.Fatal("armor instance was locked as a weapon")
	}
	if _, _, err := p.InscribeWeapon(armor.InstanceId, 4); err == nil {
		t.Fatal("armor instance accepted an inscription")
	}
	if _, err := p.DisassembleEquipment([]uint32{999}, nil); err == nil {
		t.Fatal("ghost weapon disassembly succeeded")
	}
	if _, _, err := p.StarUpPoster(999, []uint32{998}); err == nil {
		t.Fatal("ghost poster star-up succeeded")
	}
	if p.Item.GetItemArmorInfo(2) == nil {
		t.Fatal("failed requests mutated the armor instance")
	}
}

func sStrengthenWeaponForTest(p *Player, id uint32) (*proto.PackNotice, uint32, error) {
	return p.StrengthenWeapon(id)
}

func TestWeaponStrengthCostsAreAtomicAndLevelCapped(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	weapon := addEquipmentTestWeapon(p, 1)
	gold := addEquipmentTestBase(t, p, cfg, 101, 10)
	material := addEquipmentTestBase(t, p, cfg, 270, 1)

	_, _, err := p.StrengthenWeapon(1)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH)
	if weapon.StrengthLevel != 0 || weapon.Attack != 10 || gold.Num != 10 || material.Num != 1 {
		t.Fatal("failed strengthening partially changed player state")
	}

	material.Num = 2
	notice, level, err := p.StrengthenWeapon(1)
	if err != nil {
		t.Fatalf("strengthening failed: %v", err)
	}
	if level != 1 || weapon.StrengthLevel != 1 || weapon.Attack != 12 || gold.Num != 5 || material.Num != 0 {
		t.Fatalf("unexpected strengthening result: level=%d attack=%d gold=%d material=%d", level, weapon.Attack, gold.Num, material.Num)
	}
	if notice == nil || len(notice.Items) != 2 {
		t.Fatal("strengthening did not report both consumed items")
	}

	gold.Num = 100
	material.Num = 100
	_, _, err = p.StrengthenWeapon(1)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_WEAPON_MAX_STRENGTH_LEVEL)
	if gold.Num != 100 || material.Num != 100 || weapon.StrengthLevel != 1 {
		t.Fatal("max-level strengthening consumed resources")
	}
}

func TestArmorStrengthRejectsInsufficientMaterials(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	armor := addEquipmentTestArmor(p, 2)
	gold := addEquipmentTestBase(t, p, cfg, 101, 7)
	material := addEquipmentTestBase(t, p, cfg, 250, 2)

	_, _, err := p.StrengthenArmor(2)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH)
	if armor.StrengthLevel != 0 || gold.Num != 7 || material.Num != 2 {
		t.Fatal("failed armor strengthening changed player state")
	}
	material.Num = 3
	_, level, err := p.StrengthenArmor(2)
	if err != nil || level != 1 || gold.Num != 0 || material.Num != 0 {
		t.Fatalf("armor strengthening failed: level=%d gold=%d material=%d err=%v", level, gold.Num, material.Num, err)
	}
	_, _, err = p.StrengthenArmor(2)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_ARMOR_MAX_STRENGTH_LEVEL)
}

func TestWeaponInscriptionMovesWithoutCopy(t *testing.T) {
	installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	weapon1 := addEquipmentTestWeapon(p, 1)
	weapon2 := addEquipmentTestWeapon(p, 2)
	inscription := addEquipmentTestInscription(p, 4)

	weapons, _, err := p.InscribeWeapon(1, 4)
	if err != nil || len(weapons) != 1 || weapon1.Inscription1 != 4 || inscription.WeaponInstanceId != 1 {
		t.Fatalf("initial inscription failed: %v", err)
	}
	if _, _, err := p.InscribeWeapon(1, 4); err == nil {
		t.Fatal("duplicate inscription installation succeeded")
	}
	weapons, _, err = p.InscribeWeapon(2, 4)
	if err != nil || len(weapons) != 2 || weapon1.Inscription1 != 0 || weapon2.Inscription1 != 4 || inscription.WeaponInstanceId != 2 {
		t.Fatalf("inscription move failed: %v", err)
	}
	if len(p.Item.ItemInscriptionMap) != 1 {
		t.Fatal("inscription move duplicated the inscription")
	}
	removedWeapon, removedInscription, err := p.RemoveWeaponInscription(2)
	if err != nil || removedWeapon.Inscription1 != 0 || removedInscription.WeaponInstanceId != 0 {
		t.Fatalf("inscription removal failed: %v", err)
	}
	if _, _, err := p.RemoveWeaponInscription(2); err == nil {
		t.Fatal("removing an empty inscription slot succeeded")
	}
}

func TestDisassembleRejectsDuplicatesAndEquippedInstances(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	weapon := addEquipmentTestWeapon(p, 1)
	addEquipmentTestBase(t, p, cfg, 250, 0)

	if _, err := p.DisassembleEquipment([]uint32{1, 1}, nil); err == nil {
		t.Fatal("duplicate disassembly succeeded")
	}
	if p.Item.GetItemWeaponInfo(1) == nil {
		t.Fatal("duplicate disassembly destroyed the instance")
	}
	weapon.WearerId = 10
	weapon.WearerIndex = 1
	if _, err := p.DisassembleEquipment([]uint32{1}, nil); err == nil {
		t.Fatal("equipped weapon disassembly succeeded")
	}
	if p.Item.GetItemWeaponInfo(1) == nil {
		t.Fatal("equipped disassembly destroyed the instance")
	}
	weapon.WearerId = 0
	weapon.WearerIndex = 0
	items, err := p.DisassembleEquipment([]uint32{1}, nil)
	if err != nil || len(items) != 1 {
		t.Fatalf("valid disassembly failed: %v", err)
	}
	if p.Item.GetItemWeaponInfo(1) != nil || p.Item.GetItemBaseInfo(250).Num != 1 {
		t.Fatal("disassembly did not destroy the instance and grant the configured reward")
	}
	if _, err := p.DisassembleEquipment([]uint32{1}, nil); err == nil {
		t.Fatal("repeated disassembly succeeded")
	}
	if p.Item.GetItemBaseInfo(250).Num != 1 {
		t.Fatal("repeated disassembly granted another reward")
	}
}

func TestDisassembleRejectsLockedInstance(t *testing.T) {
	installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	weapon := addEquipmentTestWeapon(p, 1)
	weapon.IsLock = true
	if _, err := p.DisassembleEquipment([]uint32{1}, nil); err == nil {
		t.Fatal("locked weapon disassembly succeeded")
	}
	if p.Item.GetItemWeaponInfo(1) == nil {
		t.Fatal("locked weapon was destroyed")
	}
}

func TestPosterStarUpConsumesMaterialsAndDetachesEquipment(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	target := addEquipmentTestPoster(p, 1)
	material := addEquipmentTestPoster(p, 2)
	gold := addEquipmentTestBase(t, p, cfg, 101, 10)
	character := &CharacterInfo{CharacterId: 10, EquipmentPresetList: make(map[uint32]*EquipmentPreset)}
	preset := newEquipmentPreset(0)
	preset.Posters[proto.PosterInfo_PosterIndex_First].InstanceId = 2
	character.EquipmentPresetList[0] = preset
	p.Character.CharacterMap[10] = character
	material.SetWearer(10, 0)

	costIDs, characterIDs, err := p.StarUpPoster(1, []uint32{2})
	if err != nil || len(costIDs) != 1 || len(characterIDs) != 1 || characterIDs[0] != 10 {
		t.Fatalf("poster star-up failed: %v", err)
	}
	if target.Star != 2 || gold.Num != 0 || p.Item.GetItemPosterInfo(2) != nil || material.WearerId != 0 {
		t.Fatal("poster star-up did not apply star, cost, and detach atomically")
	}
	if preset.Posters[proto.PosterInfo_PosterIndex_First].InstanceId != 0 {
		t.Fatal("consumed poster remained in an equipment preset")
	}
}

func TestPosterIllustrationRewardIsOneTime(t *testing.T) {
	installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	addEquipmentTestPoster(p, 1)

	entry, err := p.ClaimPosterIllustration(3)
	if err != nil || entry.RewardStatus != proto.RewardStatus_Reward {
		t.Fatalf("illustration reward failed: %v", err)
	}
	_, err = p.ClaimPosterIllustration(3)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_POSTER_ILLUSTRATION_REWARDED)
	illustrations := p.PosterIllustrations()
	if len(illustrations) != 1 || illustrations[0].RewardStatus != proto.RewardStatus_Reward {
		t.Fatal("illustration list did not preserve rewarded status")
	}
}

func TestEquipmentPresetSwitchOnlyAcceptsExistingPreset(t *testing.T) {
	p := newEquipmentTestPlayer()
	character := &CharacterInfo{CharacterId: 10, EquipmentPresetList: make(map[uint32]*EquipmentPreset)}
	character.EquipmentPresetList[0] = newEquipmentPreset(0)
	p.Character.CharacterMap[10] = character

	if err := p.SwitchEquipmentPreset(10, 1); err == nil {
		t.Fatal("switching to a missing preset succeeded")
	}
	if character.InUseEquipmentPresetIndex != 0 {
		t.Fatal("failed preset switch changed the active preset")
	}
	if err := p.SwitchEquipmentPreset(10, 0); err != nil {
		t.Fatalf("switching to an existing preset failed: %v", err)
	}
}

func TestItemUseGiftValidatesChoiceQuantityAndCooldown(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	gift := addEquipmentTestBase(t, p, cfg, 6000010, 2)
	addEquipmentTestBase(t, p, cfg, 900, 0)
	now := time.Unix(1000, 0)

	if _, err := p.UseItem(6000010, 1, 901, 1, now); err == nil {
		t.Fatal("gift use accepted an unconfigured choice")
	}
	if gift.Num != 2 || p.Item.GetItemBaseInfo(900).Num != 0 {
		t.Fatal("invalid gift choice changed inventory")
	}
	result, err := p.UseItem(6000010, 2, 900, 1, now)
	if err != nil {
		t.Fatalf("gift use failed: %v", err)
	}
	if gift.Num != 0 || p.Item.GetItemBaseInfo(900).Num != 4 || len(result.Items) != 2 {
		t.Fatalf("unexpected gift use result: gift=%d reward=%d details=%d", gift.Num, p.Item.GetItemBaseInfo(900).Num, len(result.Items))
	}
	gift.Num = 1
	_, err = p.UseItem(6000010, 1, 900, 1, now.Add(time.Second))
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_REQUESTS_TOO_FREQUENT)
	if gift.Num != 1 || p.Item.GetItemBaseInfo(900).Num != 4 {
		t.Fatal("cooldown failure changed inventory")
	}
}

func TestItemUseFoodAppliesConfiguredBuffAndRejectsActiveReuse(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	food := addEquipmentTestBase(t, p, cfg, 5013000, 2)
	now := time.Unix(1000, 0)

	result, err := p.UseItem(5013000, 1, 0, 1, now)
	if err != nil {
		t.Fatalf("food use failed: %v", err)
	}
	if food.Num != 1 || len(result.PlayerBuffs) != 1 || len(result.PlayerBuffs[0].BuffLst) != 1 {
		t.Fatal("food use did not consume one item and produce one configured buff")
	}
	buff := result.PlayerBuffs[0].BuffLst[0]
	if buff.BuffId != 5013000 || buff.EndTime != now.Add(60*time.Second).UnixMilli() {
		t.Fatalf("unexpected food buff: %+v", buff)
	}
	_, err = p.UseItem(5013000, 1, 0, 1, now.Add(time.Second))
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_REQUESTS_TOO_FREQUENT)
	if food.Num != 1 {
		t.Fatal("active food reuse consumed an item")
	}
}

func TestItemUseRejectsZeroCountAndUnsupportedPurpose(t *testing.T) {
	cfg := installTestEquipmentGrowthConfig(t)
	p := newEquipmentTestPlayer()
	material := addEquipmentTestBase(t, p, cfg, 270, 3)
	now := time.Unix(1000, 0)

	_, err := p.UseItem(270, 0, 0, 1, now)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_BAD_REQ)
	_, err = p.UseItem(270, 1, 0, 1, now)
	requireGrowthStatus(t, err, proto.StatusCode_StatusCode_BAD_REQ)
	if material.Num != 3 {
		t.Fatal("invalid item use consumed material")
	}
}
