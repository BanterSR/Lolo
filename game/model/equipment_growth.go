package model

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/config"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/constant"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

// GrowthError keeps protocol status selection in the model layer, so every
// equipment handler rejects invalid state in the same way.
type GrowthError struct {
	Status proto.StatusCode
	Err    error
}

func (e *GrowthError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *GrowthError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func growthError(status proto.StatusCode, format string, args ...any) error {
	return &GrowthError{Status: status, Err: fmt.Errorf(format, args...)}
}

func EquipmentErrorStatus(err error) proto.StatusCode {
	var e *GrowthError
	if errors.As(err, &e) && e != nil {
		return e.Status
	}
	return proto.StatusCode_StatusCode_FAIL
}

type equipmentGrowthConfig struct {
	weapons             map[uint32]*excel.WeaponConfigure
	weaponProperties    map[int32]*excel.WeaponPropertyConfigure
	weaponStrengthen    map[int32]*excel.WeaponStrengthenConfigure
	weaponDisassemble   map[int32]*excel.WeaponDisassembleConfigure
	armors              map[uint32]*excel.ArmorConfigure
	armorProperties     map[int32]*excel.ArmorPropertyConfigure
	armorStrengthen     map[int32]*excel.ArmorStrengthenConfigure
	armorDisassemble    map[int32]*excel.ArmorDisassembleConfigure
	posters             map[uint32]*excel.PosterConfigure
	posterProperties    map[int32]*excel.PropertyConfigure
	posterIllustrations map[uint32]*excel.PosterIllustrationConfigure
	inscriptions        map[uint32]*excel.InscriptionConfigure
	items               map[uint32]*excel.ItemConfigure
	cookAffects         map[int32]*excel.CookAffectConfigure
	buffs               map[int32]*excel.BuffConfigure
	giftBags            map[int32]*excel.GiftBagConfigure
	rewardPools         map[int32]*excel.RewardPoolConfigure
	rewardItemPools     map[int32]*excel.RewardItemPoolConfigure
	globalConfig        map[string]*excel.GlobalConfigConfigure
}

var loadedEquipmentGrowthConfig struct {
	sync.Once
	cfg *equipmentGrowthConfig
	err error
}

// Tests install an in-memory configuration to avoid coupling gameplay rules to
// a particular resource directory. Production always uses the lazy loader.
var equipmentGrowthConfigOverride *equipmentGrowthConfig

func getEquipmentGrowthConfig() (*equipmentGrowthConfig, error) {
	if equipmentGrowthConfigOverride != nil {
		return equipmentGrowthConfigOverride, nil
	}
	loadedEquipmentGrowthConfig.Do(func() {
		loadedEquipmentGrowthConfig.cfg, loadedEquipmentGrowthConfig.err = loadEquipmentGrowthConfig()
	})
	return loadedEquipmentGrowthConfig.cfg, loadedEquipmentGrowthConfig.err
}

func loadEquipmentGrowthConfig() (*equipmentGrowthConfig, error) {
	root := config.GetResources().GetResourcePath()
	candidates := []string{
		filepath.Join(root, "Excel"),
		filepath.Join(root, "data"),
		root,
	}
	var dir string
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "Weapon.json")); err == nil && !info.IsDir() {
			dir = candidate
			break
		}
	}
	if dir == "" {
		return nil, fmt.Errorf("equipment growth configuration directory not found under %q", root)
	}

	weapon, err := readEquipmentGrowthJSON[excel.AllWeaponDatas](dir, "Weapon.json")
	if err != nil {
		return nil, err
	}
	armor, err := readEquipmentGrowthJSON[excel.AllArmorDatas](dir, "Armor.json")
	if err != nil {
		return nil, err
	}
	poster, err := readEquipmentGrowthJSON[excel.AllPosterDatas](dir, "Poster.json")
	if err != nil {
		return nil, err
	}
	inscription, err := readEquipmentGrowthJSON[excel.AllInscriptionDatas](dir, "Inscription.json")
	if err != nil {
		return nil, err
	}
	item, err := readEquipmentGrowthJSON[excel.AllItemDatas](dir, "Item.json")
	if err != nil {
		return nil, err
	}
	cooking, err := readEquipmentGrowthJSON[excel.AllCookingDatas](dir, "Cooking.json")
	if err != nil {
		return nil, err
	}
	buff, err := readEquipmentGrowthJSON[excel.AllBuffDatas](dir, "Buff.json")
	if err != nil {
		return nil, err
	}
	giftBag, err := readEquipmentGrowthJSON[excel.AllGiftBagDatas](dir, "GiftBag.json")
	if err != nil {
		return nil, err
	}
	reward, err := readEquipmentGrowthJSON[excel.AllRewardDatas](dir, "Reward.json")
	if err != nil {
		return nil, err
	}
	global, err := readEquipmentGrowthJSON[excel.AllGlobalConfigDatas](dir, "GlobalConfig.json")
	if err != nil {
		return nil, err
	}

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
	for _, v := range weapon.GetWeapon().GetDatas() {
		if v != nil {
			cfg.weapons[uint32(v.ID)] = v
		}
	}
	for _, v := range weapon.GetWeaponProperty().GetDatas() {
		if v != nil {
			cfg.weaponProperties[v.ID] = v
		}
	}
	for _, v := range weapon.GetWeaponStrengthen().GetDatas() {
		if v != nil {
			cfg.weaponStrengthen[v.ID] = v
		}
	}
	for _, v := range weapon.GetWeaponDisassemble().GetDatas() {
		if v != nil {
			cfg.weaponDisassemble[v.ID] = v
		}
	}
	for _, v := range armor.GetArmor().GetDatas() {
		if v != nil {
			cfg.armors[uint32(v.ID)] = v
		}
	}
	for _, v := range armor.GetArmorProperty().GetDatas() {
		if v != nil {
			cfg.armorProperties[v.ID] = v
		}
	}
	for _, v := range armor.GetArmorStrengthen().GetDatas() {
		if v != nil {
			cfg.armorStrengthen[v.ID] = v
		}
	}
	for _, v := range armor.GetArmorDisassemble().GetDatas() {
		if v != nil {
			cfg.armorDisassemble[v.ID] = v
		}
	}
	for _, v := range poster.GetPoster().GetDatas() {
		if v != nil {
			cfg.posters[uint32(v.ID)] = v
		}
	}
	for _, v := range poster.GetPosterProperty().GetDatas() {
		if v != nil {
			cfg.posterProperties[v.ID] = v
		}
	}
	for _, v := range poster.GetPosterIllustration().GetDatas() {
		if v != nil {
			cfg.posterIllustrations[uint32(v.ID)] = v
		}
	}
	for _, v := range inscription.GetInscription().GetDatas() {
		if v != nil {
			cfg.inscriptions[uint32(v.ID)] = v
		}
	}
	for _, v := range item.GetItem().GetDatas() {
		if v != nil {
			cfg.items[uint32(v.ID)] = v
		}
	}
	for _, v := range cooking.GetCookAffect().GetDatas() {
		if v != nil {
			cfg.cookAffects[v.ID] = v
		}
	}
	for _, v := range buff.GetBuff().GetDatas() {
		if v != nil {
			cfg.buffs[v.ID] = v
		}
	}
	for _, v := range giftBag.GetGiftBag().GetDatas() {
		if v != nil {
			cfg.giftBags[v.ID] = v
		}
	}
	for _, v := range reward.GetRewardPool().GetDatas() {
		if v != nil {
			cfg.rewardPools[v.ID] = v
		}
	}
	for _, v := range reward.GetRewardItemPool().GetDatas() {
		if v != nil {
			cfg.rewardItemPools[v.ID] = v
		}
	}
	for _, v := range global.GetConfig().GetDatas() {
		if v != nil {
			cfg.globalConfig[v.Key] = v
		}
	}
	return cfg, nil
}

func readEquipmentGrowthJSON[T any](dir, name string) (*T, error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	// Resource exports use snake_case (i_d, item_i_d), while the generated Go
	// structs expose protobuf field names (ID, ItemID). Normalize keys before
	// decoding so both exported and protobuf-style JSON are accepted.
	var raw any
	if err := sonic.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	normalized, err := sonic.Marshal(normalizeEquipmentGrowthJSONKeys(raw))
	if err != nil {
		return nil, fmt.Errorf("normalize %s: %w", name, err)
	}
	value := new(T)
	if err := sonic.Unmarshal(normalized, value); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	return value, nil
}

func normalizeEquipmentGrowthJSONKeys(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			normalizedKey := strings.ToLower(strings.ReplaceAll(key, "_", ""))
			result[normalizedKey] = normalizeEquipmentGrowthJSONKeys(child)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for i, child := range typed {
			result[i] = normalizeEquipmentGrowthJSONKeys(child)
		}
		return result
	default:
		return value
	}
}

func (c *equipmentGrowthConfig) globalValues(key string) []string {
	if c == nil || c.globalConfig[key] == nil {
		return nil
	}
	return c.globalConfig[key].Values
}

func (c *equipmentGrowthConfig) globalInt(key string) (int64, bool, error) {
	values := c.globalValues(key)
	if len(values) == 0 {
		return 0, false, nil
	}
	value, err := strconv.ParseInt(strings.TrimSpace(values[0]), 10, 64)
	if err != nil {
		return 0, true, fmt.Errorf("invalid global configuration %s: %w", key, err)
	}
	return value, true, nil
}

func parseConfiguredItemCost(c *equipmentGrowthConfig, key string) (uint32, int64, error) {
	values := c.globalValues(key)
	if len(values) != 1 {
		return 0, 0, fmt.Errorf("missing global configuration %s", key)
	}
	parts := strings.Split(strings.TrimSpace(values[0]), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid global configuration %s", key)
	}
	itemID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid item id in %s", key)
	}
	count, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid item count in %s", key)
	}
	if itemID == 0 || count <= 0 {
		return 0, 0, fmt.Errorf("invalid item cost in %s", key)
	}
	return uint32(itemID), count, nil
}

func (s *Player) validGrowthBaseItem(itemID uint32, count int64) (*ItemBaseInfo, *excel.ItemConfigure, error) {
	cfg, err := getEquipmentGrowthConfig()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	conf := cfg.items[itemID]
	if conf == nil {
		return nil, nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "item %d is not configured", itemID)
	}
	info := s.GetItemModel().GetItemBaseInfo(itemID)
	if info == nil || info.ItemId != itemID || info.ItemType != proto.EBagItemTag(conf.NewBagItemTag) ||
		info.PackType != proto.ItemDetail_PackType_Inventory {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "item %d is not an owned inventory item", itemID)
	}
	if count < 0 || info.Num < count {
		return nil, nil, growthError(proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, "item %d is insufficient", itemID)
	}
	return info, conf, nil
}

func (s *Player) validGrowthWeapon(instanceID uint32) (*ItemWeaponInfo, *excel.WeaponConfigure, error) {
	if instanceID == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance id is zero")
	}
	info := s.GetItemModel().GetItemWeaponInfo(instanceID)
	if info == nil || info.ItemBaseInfo == nil || info.InstanceId != instanceID {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance %d is not owned", instanceID)
	}
	cfg, err := getEquipmentGrowthConfig()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	conf := cfg.weapons[info.WeaponId]
	if conf == nil || info.ItemType != proto.EBagItemTag_EBagItemTag_Weapon ||
		info.WeaponSystemType != proto.EWeaponSystemType(conf.NewWeaponSystemType) ||
		info.ItemId != uint32(conf.ItemID) || info.PackType != proto.ItemDetail_PackType_Inventory {
		return nil, nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "weapon instance %d has invalid type or configuration", instanceID)
	}
	return info, conf, nil
}

func (s *Player) validGrowthArmor(instanceID uint32) (*ItemArmorInfo, *excel.ArmorConfigure, error) {
	if instanceID == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "armor instance id is zero")
	}
	info := s.GetItemModel().GetItemArmorInfo(instanceID)
	if info == nil || info.ItemBaseInfo == nil || info.InstanceId != instanceID {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "armor instance %d is not owned", instanceID)
	}
	cfg, err := getEquipmentGrowthConfig()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	conf := cfg.armors[info.ArmorId]
	if conf == nil || info.ItemType != proto.EBagItemTag_EBagItemTag_Armor ||
		info.WeaponSystemType != proto.EWeaponSystemType(conf.NewWeaponSystemType) ||
		info.ItemId != uint32(conf.ItemID) || info.PackType != proto.ItemDetail_PackType_Inventory {
		return nil, nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "armor instance %d has invalid type or configuration", instanceID)
	}
	return info, conf, nil
}

func (s *Player) validGrowthPoster(instanceID uint32) (*ItemPosterInfo, *excel.PosterConfigure, error) {
	if instanceID == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "poster instance id is zero")
	}
	info := s.GetItemModel().GetItemPosterInfo(instanceID)
	if info == nil || info.ItemBaseInfo == nil || info.InstanceId != instanceID {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "poster instance %d is not owned", instanceID)
	}
	cfg, err := getEquipmentGrowthConfig()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	conf := cfg.posters[info.PosterId]
	illustration := cfg.posterIllustrations[uint32(conf.GetIllustrationID())]
	if conf == nil || illustration == nil || !illustration.IsShow ||
		info.ItemType != proto.EBagItemTag_EBagItemTag_Poster ||
		info.ItemId != uint32(conf.ItemID) || info.PackType != proto.ItemDetail_PackType_Inventory {
		return nil, nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "poster instance %d has invalid type or configuration", instanceID)
	}
	return info, conf, nil
}

func (s *Player) validGrowthInscription(inscriptionID uint32) (*ItemInscriptionInfo, *excel.InscriptionConfigure, error) {
	if inscriptionID == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "inscription id is zero")
	}
	info := s.GetItemModel().GetItemInscriptionMap()[inscriptionID]
	if info == nil || info.ItemBaseInfo == nil || info.InscriptionId != inscriptionID || info.Level == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "inscription %d is not owned", inscriptionID)
	}
	cfg, err := getEquipmentGrowthConfig()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	conf := cfg.inscriptions[inscriptionID]
	if conf == nil || info.ItemType != proto.EBagItemTag_EBagItemTag_Inscription ||
		info.ItemId != uint32(conf.ItemID) || info.PackType != proto.ItemDetail_PackType_Inventory {
		return nil, nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "inscription %d has invalid type or configuration", inscriptionID)
	}
	return info, conf, nil
}

func findWeaponPropertyGroup(cfg *equipmentGrowthConfig, weaponConf *excel.WeaponConfigure, level uint32) (*excel.WeaponPropertyGroupInfo, error) {
	if weaponConf == nil || level == 0 {
		return nil, fmt.Errorf("invalid weapon level")
	}
	property := cfg.weaponProperties[weaponConf.WeaponPropertyID]
	if property == nil {
		return nil, fmt.Errorf("weapon property %d is not configured", weaponConf.WeaponPropertyID)
	}
	for _, group := range property.WeaponPropertyGroupInfo {
		if group != nil && uint32(group.MinLevel) <= level && level <= uint32(group.MaxLevel) {
			return group, nil
		}
	}
	return nil, fmt.Errorf("weapon level %d has no property group", level)
}

func findArmorPropertyGroup(cfg *equipmentGrowthConfig, armorConf *excel.ArmorConfigure, level uint32) (*excel.ArmorPropertyGroupInfo, error) {
	if armorConf == nil || level == 0 {
		return nil, fmt.Errorf("invalid armor level")
	}
	property := cfg.armorProperties[armorConf.ArmorPropertyID]
	if property == nil {
		return nil, fmt.Errorf("armor property %d is not configured", armorConf.ArmorPropertyID)
	}
	for _, group := range property.ArmorPropertyGroupInfo {
		if group != nil && uint32(group.MinLevel) <= level && level <= uint32(group.MaxLevel) {
			return group, nil
		}
	}
	return nil, fmt.Errorf("armor level %d has no property group", level)
}

func findWeaponStrengthenGroup(groups []*excel.WeaponStrengthenGroupInfo, level uint32) *excel.WeaponStrengthenGroupInfo {
	for _, group := range groups {
		if group != nil && uint32(group.GetLevel()) == level {
			return group
		}
	}
	return nil
}

func findArmorStrengthenGroup(groups []*excel.ArmorStrengthenGroupInfo, level uint32) *excel.ArmorStrengthenGroupInfo {
	for _, group := range groups {
		if group != nil && uint32(group.GetLevel()) == level {
			return group
		}
	}
	return nil
}

func validateGrowthCost(tx *ItemTransaction, cfg *equipmentGrowthConfig, itemID uint32, count int64) error {
	if count < 0 {
		return growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "negative cost for item %d", itemID)
	}
	if count == 0 {
		return nil
	}
	conf := cfg.items[itemID]
	if conf == nil || !isGrowthBaseItemTag(proto.EBagItemTag(conf.NewBagItemTag)) {
		return growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "cost item %d is not a configured stackable item", itemID)
	}
	info := tx.i.GetItemBaseInfo(itemID)
	if info == nil || info.ItemType != proto.EBagItemTag(conf.NewBagItemTag) ||
		info.PackType != proto.ItemDetail_PackType_Inventory {
		return growthError(proto.StatusCode_StatusCode_BAD_REQ, "cost item %d is not an owned inventory item", itemID)
	}
	if tx.DelBaseItem(itemID, count).Error != nil {
		return &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: tx.Error}
	}
	return nil
}

func (s *Player) addGrowthBaseReward(conf *excel.ItemConfigure, count int64) *ItemBaseInfo {
	if conf == nil || count <= 0 {
		return nil
	}
	list := s.GetItemModel().GetItemBaseMap()
	itemID := uint32(conf.ID)
	info := list[itemID]
	if info == nil {
		info = &ItemBaseInfo{
			ItemId:   itemID,
			ItemType: proto.EBagItemTag(conf.NewBagItemTag),
			PackType: proto.ItemDetail_PackType_Inventory,
		}
		list[itemID] = info
	}
	info.Num += count
	return info
}

func isGrowthBaseItemTag(tag proto.EBagItemTag) bool {
	switch tag {
	case proto.EBagItemTag_EBagItemTag_Gift,
		proto.EBagItemTag_EBagItemTag_Fragment,
		proto.EBagItemTag_EBagItemTag_Collection,
		proto.EBagItemTag_EBagItemTag_Material,
		proto.EBagItemTag_EBagItemTag_Food,
		proto.EBagItemTag_EBagItemTag_SpellCard,
		proto.EBagItemTag_EBagItemTag_Item,
		proto.EBagItemTag_EBagItemTag_Fish,
		proto.EBagItemTag_EBagItemTag_Recipe,
		proto.EBagItemTag_EBagItemTag_Baitbox,
		proto.EBagItemTag_EBagItemTag_Quest,
		proto.EBagItemTag_EBagItemTag_StrengthStone,
		proto.EBagItemTag_EBagItemTag_ExpBook,
		proto.EBagItemTag_EBagItemTag_UnlockAbilityItem,
		proto.EBagItemTag_EBagItemTag_CharacterBadge,
		proto.EBagItemTag_EBagItemTag_DyeStuff,
		proto.EBagItemTag_EBagItemTag_PlayerExp,
		proto.EBagItemTag_EBagItemTag_WorldLevel,
		proto.EBagItemTag_EBagItemTag_Agentia,
		proto.EBagItemTag_EBagItemTag_MoonStone,
		proto.EBagItemTag_EBagItemTag_Umbrella,
		proto.EBagItemTag_EBagItemTag_Vitality,
		proto.EBagItemTag_EBagItemTag_Badge,
		proto.EBagItemTag_EBagItemTag_Furniture,
		proto.EBagItemTag_EBagItemTag_Energy,
		proto.EBagItemTag_EBagItemTag_ShowWeapon,
		proto.EBagItemTag_EBagItemTag_ShowArmor,
		proto.EBagItemTag_EBagItemTag_TeleportKey,
		proto.EBagItemTag_EBagItemTag_WallPaper,
		proto.EBagItemTag_EBagItemTag_MoonCard,
		proto.EBagItemTag_EBagItemTag_PhoneCase,
		proto.EBagItemTag_EBagItemTag_Pendant,
		proto.EBagItemTag_EBagItemTag_AvatarFrame,
		proto.EBagItemTag_EBagItemTag_IntimacyGift,
		proto.EBagItemTag_EBagItemTag_MusicNote,
		proto.EBagItemTag_EBagItemTag_MonthlyCard,
		proto.EBagItemTag_EBagItemTag_BattlePassCard,
		proto.EBagItemTag_EBagItemTag_MonthlyGiftCard,
		proto.EBagItemTag_EBagItemTag_BattlePassGiftCard,
		proto.EBagItemTag_EBagItemTag_SeasonalMiniGamesItem,
		proto.EBagItemTag_EBagItemTag_TreasureMap,
		proto.EBagItemTag_EBagItemTag_TreasureMapFrag,
		proto.EBagItemTag_EBagItemTag_Catcher,
		proto.EBagItemTag_EBagItemTag_PetGrid,
		proto.EBagItemTag_EBagItemTag_PetMmaterials,
		proto.EBagItemTag_EBagItemTag_Currency:
		return true
	default:
		return false
	}
}

func calculateWeaponStrengthAttack(current uint32, oldLevel uint32, oldConf, newConf *excel.WeaponStrengthenGroupInfo) (uint32, error) {
	if newConf == nil {
		return 0, fmt.Errorf("missing weapon strength configuration")
	}
	oldAdd, oldRatio := float64(0), float64(0)
	if oldLevel > 0 {
		if oldConf == nil {
			return 0, fmt.Errorf("previous weapon strength configuration is missing")
		}
		oldAdd = float64(oldConf.AddAttack)
		oldRatio = float64(oldConf.AddAttackRatio)
	}
	base := float64(current) - oldAdd
	if oldRatio != 0 {
		base /= 1 + oldRatio
	}
	value := base*(1+float64(newConf.AddAttackRatio)) + float64(newConf.AddAttack)
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("invalid weapon attack after strengthening")
	}
	return uint32(math.Round(value)), nil
}

func (s *Player) StrengthenWeapon(instanceID uint32) (*proto.PackNotice, uint32, error) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, 0, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	info, weaponConf, err := s.validGrowthWeapon(instanceID)
	if err != nil {
		return nil, 0, err
	}
	cfg, _ := getEquipmentGrowthConfig()
	property, err := findWeaponPropertyGroup(cfg, weaponConf, info.Level)
	if err != nil || property.StrengthenID == 0 {
		return nil, 0, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	strengthenConf := cfg.weaponStrengthen[property.StrengthenID]
	if strengthenConf == nil {
		return nil, 0, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "weapon strength configuration %d is missing", property.StrengthenID)
	}
	if info.StrengthLevel == math.MaxUint32 {
		return nil, 0, growthError(proto.StatusCode_StatusCode_WEAPON_MAX_STRENGTH_LEVEL, "weapon instance %d is at max strength level", instanceID)
	}
	newLevel := info.StrengthLevel + 1
	newGroup := findWeaponStrengthenGroup(strengthenConf.WeaponStrengthenGroupInfo, newLevel)
	if newGroup == nil {
		return nil, 0, growthError(proto.StatusCode_StatusCode_WEAPON_MAX_STRENGTH_LEVEL, "weapon instance %d is at max strength level", instanceID)
	}
	oldGroup := findWeaponStrengthenGroup(strengthenConf.WeaponStrengthenGroupInfo, info.StrengthLevel)
	newAttack, err := calculateWeaponStrengthAttack(info.Attack, info.StrengthLevel, oldGroup, newGroup)
	if err != nil {
		return nil, 0, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	if newGroup.NeedGoldCount < 0 {
		return nil, 0, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "negative weapon strength gold cost")
	}
	if err := validateGrowthCost(tx, cfg, constant.CurrencyGold, int64(newGroup.NeedGoldCount)); err != nil {
		return nil, 0, err
	}
	for _, cost := range newGroup.WeaponStrengthNeedItem {
		if cost == nil || cost.NeedItemCount < 0 {
			return nil, 0, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "invalid weapon strength material cost")
		}
		if err := validateGrowthCost(tx, cfg, uint32(cost.NeedItemID), int64(cost.NeedItemCount)); err != nil {
			return nil, 0, err
		}
	}
	info.StrengthLevel = newLevel
	info.Attack = newAttack
	tx.Commit()
	return tx.PackNotice, info.StrengthLevel, nil
}

func (s *Player) StrengthenArmor(instanceID uint32) (*proto.PackNotice, uint32, error) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, 0, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	info, armorConf, err := s.validGrowthArmor(instanceID)
	if err != nil {
		return nil, 0, err
	}
	cfg, _ := getEquipmentGrowthConfig()
	property, err := findArmorPropertyGroup(cfg, armorConf, info.Level)
	if err != nil || property.StrengthenID == 0 {
		return nil, 0, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	strengthenConf := cfg.armorStrengthen[property.StrengthenID]
	if strengthenConf == nil {
		return nil, 0, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "armor strength configuration %d is missing", property.StrengthenID)
	}
	if info.StrengthLevel == math.MaxUint32 {
		return nil, 0, growthError(proto.StatusCode_StatusCode_ARMOR_MAX_STRENGTH_LEVEL, "armor instance %d is at max strength level", instanceID)
	}
	newLevel := info.StrengthLevel + 1
	newGroup := findArmorStrengthenGroup(strengthenConf.ArmorStrengthenGroupInfo, newLevel)
	if newGroup == nil {
		return nil, 0, growthError(proto.StatusCode_StatusCode_ARMOR_MAX_STRENGTH_LEVEL, "armor instance %d is at max strength level", instanceID)
	}
	if newGroup.NeedGoldCount < 0 {
		return nil, 0, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "negative armor strength gold cost")
	}
	if err := validateGrowthCost(tx, cfg, constant.CurrencyGold, int64(newGroup.NeedGoldCount)); err != nil {
		return nil, 0, err
	}
	for _, cost := range newGroup.ArmorStrengthNeedItem {
		if cost == nil || cost.NeedItemCount < 0 {
			return nil, 0, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "invalid armor strength material cost")
		}
		if err := validateGrowthCost(tx, cfg, uint32(cost.NeedItemID), int64(cost.NeedItemCount)); err != nil {
			return nil, 0, err
		}
	}
	info.StrengthLevel = newLevel
	tx.Commit()
	return tx.PackNotice, info.StrengthLevel, nil
}

func (s *Player) ToggleEquipmentLock(instanceID uint32, equipType proto.LockEquipReq_LockEquip_EquipType) (*proto.ItemDetail, error) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	switch equipType {
	case proto.LockEquipReq_LockEquip_EquipType_Weapon:
		info, _, err := s.validGrowthWeapon(instanceID)
		if err != nil {
			return nil, err
		}
		info.IsLock = !info.IsLock
		tx.Commit()
		return info.ItemDetail(), nil
	case proto.LockEquipReq_LockEquip_EquipType_Armor:
		info, _, err := s.validGrowthArmor(instanceID)
		if err != nil {
			return nil, err
		}
		info.IsLock = !info.IsLock
		tx.Commit()
		return info.ItemDetail(), nil
	default:
		return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "unknown equipment lock type %d", equipType)
	}
}

func (s *Player) SwitchEquipmentPreset(characterID, presetIndex uint32) error {
	character := s.GetCharacterModel().GetCharacterInfo(characterID)
	if character == nil {
		return growthError(proto.StatusCode_StatusCode_CharNotExist, "character %d is not owned", characterID)
	}
	preset, ok := character.EquipmentPresetList[presetIndex]
	if !ok || preset == nil || preset.PresetIndex != presetIndex {
		return growthError(proto.StatusCode_StatusCode_BAD_REQ, "equipment preset %d does not exist", presetIndex)
	}
	character.InUseEquipmentPresetIndex = presetIndex
	return nil
}

func (s *Player) InscribeWeapon(weaponInstanceID, inscriptionID uint32) ([]*proto.WeaponInstance, []*proto.Inscription, error) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	weapon, _, err := s.validGrowthWeapon(weaponInstanceID)
	if err != nil {
		return nil, nil, err
	}
	inscription, inscriptionConf, err := s.validGrowthInscription(inscriptionID)
	if err != nil {
		return nil, nil, err
	}
	if inscriptionConf.NewWeaponSystemType != 0 &&
		proto.EWeaponSystemType(inscriptionConf.NewWeaponSystemType) != weapon.WeaponSystemType {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "inscription %d does not match the weapon type", inscriptionID)
	}
	if weapon.Inscription1 == inscriptionID {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "inscription %d is already installed", inscriptionID)
	}
	if inscription.WeaponInstanceId != 0 {
		oldWeapon, _, err := s.validGrowthWeapon(inscription.WeaponInstanceId)
		if err != nil || oldWeapon.Inscription1 != inscriptionID {
			return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "inscription %d has an invalid owner", inscriptionID)
		}
	}
	var replaced *ItemInscriptionInfo
	if weapon.Inscription1 != 0 {
		replaced, _, err = s.validGrowthInscription(weapon.Inscription1)
		if err != nil || replaced.WeaponInstanceId != weaponInstanceID {
			return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance %d has an invalid inscription", weaponInstanceID)
		}
	}

	weapons := make(map[uint32]*ItemWeaponInfo)
	inscriptions := make([]*proto.Inscription, 0, 2)
	if inscription.WeaponInstanceId != 0 {
		oldWeapon := s.GetItemModel().GetItemWeaponInfo(inscription.WeaponInstanceId)
		oldWeapon.Inscription1 = 0
		weapons[oldWeapon.InstanceId] = oldWeapon
	}
	if replaced != nil {
		replaced.WeaponInstanceId = 0
		inscriptions = append(inscriptions, replaced.GetPbInscription())
	}
	weapon.Inscription1 = inscriptionID
	inscription.WeaponInstanceId = weaponInstanceID
	weapons[weapon.InstanceId] = weapon
	inscriptions = append(inscriptions, inscription.GetPbInscription())
	result := make([]*proto.WeaponInstance, 0, len(weapons))
	for _, info := range weapons {
		result = append(result, info.WeaponInstance())
	}
	tx.Commit()
	return result, inscriptions, nil
}

func (s *Player) RemoveWeaponInscription(weaponInstanceID uint32) (*proto.WeaponInstance, *proto.Inscription, error) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	weapon, _, err := s.validGrowthWeapon(weaponInstanceID)
	if err != nil {
		return nil, nil, err
	}
	if weapon.Inscription1 == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance %d has no inscription", weaponInstanceID)
	}
	inscription, _, err := s.validGrowthInscription(weapon.Inscription1)
	if err != nil || inscription.WeaponInstanceId != weaponInstanceID {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance %d has an invalid inscription owner", weaponInstanceID)
	}
	weapon.Inscription1 = 0
	inscription.WeaponInstanceId = 0
	tx.Commit()
	return weapon.WeaponInstance(), inscription.GetPbInscription(), nil
}

func (s *Player) weaponHasPresetReference(instanceID uint32) bool {
	if s.Character == nil {
		return false
	}
	for _, character := range s.Character.CharacterMap {
		if character == nil {
			continue
		}
		for _, preset := range character.EquipmentPresetList {
			if preset != nil && preset.WeaponInstanceId == instanceID {
				return true
			}
		}
	}
	return false
}

func (s *Player) armorHasPresetReference(instanceID uint32) bool {
	if s.Character == nil {
		return false
	}
	for _, character := range s.Character.CharacterMap {
		if character == nil {
			continue
		}
		for _, preset := range character.EquipmentPresetList {
			if preset == nil {
				continue
			}
			for _, armor := range preset.Armors {
				if armor != nil && armor.InstanceId == instanceID {
					return true
				}
			}
		}
	}
	return false
}

func (s *Player) detachPosterInstance(instanceID uint32) []uint32 {
	poster := s.GetItemModel().GetItemPosterInfo(instanceID)
	affected := make(map[uint32]struct{})
	if s.Character != nil {
		for _, character := range s.Character.CharacterMap {
			if character == nil {
				continue
			}
			for _, preset := range character.EquipmentPresetList {
				if preset == nil {
					continue
				}
				for _, entry := range preset.Posters {
					if entry != nil && entry.InstanceId == instanceID {
						entry.InstanceId = 0
						affected[character.CharacterId] = struct{}{}
					}
				}
			}
		}
	}
	if poster != nil {
		poster.SetWearer(0, 0)
	}
	result := make([]uint32, 0, len(affected))
	for id := range affected {
		result = append(result, id)
	}
	return result
}

func (s *Player) posterMaxStar(conf *excel.PosterConfigure, cfg *equipmentGrowthConfig) (uint32, error) {
	property := cfg.posterProperties[conf.PropertyID]
	if property == nil || len(property.PosterPropertyGroupInfo) == 0 {
		return 0, fmt.Errorf("poster property %d is not configured", conf.PropertyID)
	}
	var maxStar uint32
	for _, group := range property.PosterPropertyGroupInfo {
		if group != nil && group.Star > 0 && uint32(group.Star) > maxStar {
			maxStar = uint32(group.Star)
		}
	}
	if maxStar == 0 {
		return 0, fmt.Errorf("poster property %d has no star levels", conf.PropertyID)
	}
	return maxStar, nil
}

func (s *Player) StarUpPoster(posterInstanceID uint32, costPosterInstanceIDs []uint32) ([]uint32, []uint32, error) {
	if len(costPosterInstanceIDs) == 0 {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "poster star-up materials are empty")
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	target, targetConf, err := s.validGrowthPoster(posterInstanceID)
	if err != nil {
		return nil, nil, err
	}
	cfg, _ := getEquipmentGrowthConfig()
	maxStar, err := s.posterMaxStar(targetConf, cfg)
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	if target.Star == 0 || target.Star > maxStar {
		return nil, nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "poster instance %d has an invalid star", posterInstanceID)
	}
	if target.Star == maxStar {
		return nil, nil, growthError(proto.StatusCode_StatusCode_POSTER_MAX_STAR, "poster instance %d is at max star", posterInstanceID)
	}
	if uint32(len(costPosterInstanceIDs)) > maxStar-target.Star {
		return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "poster star-up materials exceed max star")
	}
	costItemID, costCount, err := parseConfiguredItemCost(cfg, "UpgradeStarNeedItem")
	if err != nil {
		return nil, nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	if err := validateGrowthCost(tx, cfg, costItemID, costCount*int64(len(costPosterInstanceIDs))); err != nil {
		return nil, nil, err
	}
	seen := make(map[uint32]struct{}, len(costPosterInstanceIDs))
	for _, id := range costPosterInstanceIDs {
		if id == posterInstanceID {
			return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "poster cannot consume itself")
		}
		if _, ok := seen[id]; ok {
			return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "duplicate poster material %d", id)
		}
		seen[id] = struct{}{}
		_, conf, err := s.validGrowthPoster(id)
		if err != nil {
			return nil, nil, err
		}
		if conf.ID != targetConf.ID {
			return nil, nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "poster material %d has a different type", id)
		}
	}
	affected := make(map[uint32]struct{})
	for _, id := range costPosterInstanceIDs {
		for _, characterID := range s.detachPosterInstance(id) {
			affected[characterID] = struct{}{}
		}
		delete(s.GetItemModel().ItemPosterMap, id)
	}
	target.Star += uint32(len(costPosterInstanceIDs))
	characterIDs := make([]uint32, 0, len(affected))
	for id := range affected {
		characterIDs = append(characterIDs, id)
	}
	tx.Commit()
	return append([]uint32(nil), costPosterInstanceIDs...), characterIDs, nil
}

type disassembleReward struct {
	itemID uint32
	count  int64
}

func configuredDisassembleCount(item *excel.WeaponDisassembleItem) (int64, bool) {
	if item == nil || item.ItemID == 0 || item.ItemMinCount < 0 || item.ItemMaxCount < item.ItemMinCount || item.ItemRate <= 0 {
		return 0, false
	}
	if item.ItemRate < 1 && rand.Float64() >= float64(item.ItemRate) {
		return 0, false
	}
	minCount, maxCount := int64(item.ItemMinCount), int64(item.ItemMaxCount)
	if maxCount == minCount {
		return minCount, true
	}
	return minCount + int64(rand.IntN(int(maxCount-minCount)+1)), true
}

func configuredArmorDisassembleCount(item *excel.ArmorDisassembleItem) (int64, bool) {
	if item == nil || item.ItemID == 0 || item.ItemMinCount < 0 || item.ItemMaxCount < item.ItemMinCount || item.ItemRate <= 0 {
		return 0, false
	}
	if item.ItemRate < 1 && rand.Float64() >= float64(item.ItemRate) {
		return 0, false
	}
	minCount, maxCount := int64(item.ItemMinCount), int64(item.ItemMaxCount)
	if maxCount == minCount {
		return minCount, true
	}
	return minCount + int64(rand.IntN(int(maxCount-minCount)+1)), true
}

func (s *Player) DisassembleEquipment(weaponIDs, armorIDs []uint32) ([]*proto.ItemDetail, error) {
	if len(weaponIDs)+len(armorIDs) == 0 {
		return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "disassemble list is empty")
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	cfg, _ := getEquipmentGrowthConfig()
	maxCount, configured, err := cfg.globalInt("Equip_Disassemble_MaxCount")
	if err != nil || !configured || maxCount <= 0 {
		return nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: fmt.Errorf("invalid Equip_Disassemble_MaxCount: %v", err)}
	}
	if int64(len(weaponIDs)+len(armorIDs)) > maxCount {
		return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "disassemble count exceeds limit")
	}
	type equipmentKey struct {
		armor bool
		id    uint32
	}
	seen := make(map[equipmentKey]struct{}, len(weaponIDs)+len(armorIDs))
	weaponRewards := make(map[int32][]*excel.WeaponDisassembleItem)
	armorRewards := make(map[int32][]*excel.ArmorDisassembleItem)
	for _, id := range weaponIDs {
		key := equipmentKey{id: id}
		if _, ok := seen[key]; ok {
			return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "duplicate weapon instance %d", id)
		}
		seen[key] = struct{}{}
		info, conf, err := s.validGrowthWeapon(id)
		if err != nil {
			return nil, err
		}
		if info.IsLock || info.WearerId != 0 || info.WearerIndex != 0 || s.weaponHasPresetReference(id) {
			return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance %d is locked or equipped", id)
		}
		property, err := findWeaponPropertyGroup(cfg, conf, info.Level)
		if err != nil || property.DisassembleID == 0 {
			return nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
		}
		rewardConf := cfg.weaponDisassemble[property.DisassembleID]
		if rewardConf == nil || len(rewardConf.WeaponDisassembleItem) == 0 {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "weapon disassemble configuration %d is missing", property.DisassembleID)
		}
		weaponRewards[property.DisassembleID] = rewardConf.WeaponDisassembleItem
	}
	for _, id := range armorIDs {
		key := equipmentKey{armor: true, id: id}
		if _, ok := seen[key]; ok {
			return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "duplicate armor instance %d", id)
		}
		seen[key] = struct{}{}
		info, conf, err := s.validGrowthArmor(id)
		if err != nil {
			return nil, err
		}
		if info.IsLock || info.WearerId != 0 || info.WearerIndex != 0 || s.armorHasPresetReference(id) {
			return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "armor instance %d is locked or equipped", id)
		}
		property, err := findArmorPropertyGroup(cfg, conf, info.Level)
		if err != nil || property.DisassembleID == 0 {
			return nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
		}
		rewardConf := cfg.armorDisassemble[property.DisassembleID]
		if rewardConf == nil || len(rewardConf.ArmorDisassembleItem) == 0 {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "armor disassemble configuration %d is missing", property.DisassembleID)
		}
		armorRewards[property.DisassembleID] = rewardConf.ArmorDisassembleItem
	}
	for _, rewards := range weaponRewards {
		for _, reward := range rewards {
			if reward == nil || reward.ItemMinCount < 0 || reward.ItemMaxCount < reward.ItemMinCount {
				return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "invalid weapon disassemble reward")
			}
			conf := cfg.items[uint32(reward.ItemID)]
			if conf == nil || !isGrowthBaseItemTag(proto.EBagItemTag(conf.NewBagItemTag)) {
				return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "weapon disassemble reward item %d is invalid", reward.ItemID)
			}
		}
	}
	for _, rewards := range armorRewards {
		for _, reward := range rewards {
			if reward == nil || reward.ItemMinCount < 0 || reward.ItemMaxCount < reward.ItemMinCount {
				return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "invalid armor disassemble reward")
			}
			conf := cfg.items[uint32(reward.ItemID)]
			if conf == nil || !isGrowthBaseItemTag(proto.EBagItemTag(conf.NewBagItemTag)) {
				return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "armor disassemble reward item %d is invalid", reward.ItemID)
			}
		}
	}

	totals := make(map[uint32]int64)
	// Random rewards are rolled only after the whole batch has passed validation.
	for _, id := range weaponIDs {
		info, conf, _ := s.validGrowthWeapon(id)
		property, _ := findWeaponPropertyGroup(cfg, conf, info.Level)
		for _, reward := range weaponRewards[property.DisassembleID] {
			if count, ok := configuredDisassembleCount(reward); ok && count > 0 {
				totals[uint32(reward.ItemID)] += count
			}
		}
		if info.Inscription1 != 0 {
			inscription, _, err := s.validGrowthInscription(info.Inscription1)
			if err != nil || inscription.WeaponInstanceId != id {
				return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "weapon instance %d has an invalid inscription", id)
			}
		}
	}
	for _, id := range armorIDs {
		info, conf, _ := s.validGrowthArmor(id)
		property, _ := findArmorPropertyGroup(cfg, conf, info.Level)
		for _, reward := range armorRewards[property.DisassembleID] {
			if count, ok := configuredArmorDisassembleCount(reward); ok && count > 0 {
				totals[uint32(reward.ItemID)] += count
			}
		}
	}
	items := make([]*proto.ItemDetail, 0, len(totals)+len(weaponIDs))
	for _, id := range weaponIDs {
		info := s.GetItemModel().GetItemWeaponInfo(id)
		if info.Inscription1 != 0 {
			inscription := s.GetItemModel().GetItemInscriptionMap()[info.Inscription1]
			inscription.WeaponInstanceId = 0
			items = append(items, inscription.ItemDetail())
		}
		delete(s.GetItemModel().ItemWeaponMap, id)
	}
	for _, id := range armorIDs {
		delete(s.GetItemModel().ItemArmorMap, id)
	}
	for itemID, count := range totals {
		info := s.addGrowthBaseReward(cfg.items[itemID], count)
		if info == nil {
			return nil, growthError(proto.StatusCode_StatusCode_Item_Get_Fail, "failed to add disassemble reward item %d", itemID)
		}
		items = append(items, info.ItemDetail())
	}
	tx.Commit()
	return items, nil
}

func (s *Player) PosterIllustrations() []*PosterIllustration {
	cfg, err := getEquipmentGrowthConfig()
	if err != nil {
		return nil
	}
	archive := s.GetArchive()
	if archive.PosterIllustrationMap == nil {
		archive.PosterIllustrationMap = make(map[uint32]*PosterIllustration)
	}
	result := make([]*PosterIllustration, 0)
	seen := make(map[uint32]struct{})
	for _, poster := range s.GetItemModel().GetItemPosterMap() {
		if poster == nil {
			continue
		}
		conf := cfg.posters[poster.PosterId]
		if conf == nil {
			continue
		}
		illustrationID := uint32(conf.IllustrationID)
		illustrationConf := cfg.posterIllustrations[illustrationID]
		if illustrationConf == nil || !illustrationConf.IsShow {
			continue
		}
		if _, ok := seen[illustrationID]; ok {
			continue
		}
		seen[illustrationID] = struct{}{}
		status := proto.RewardStatus_NotReward
		if entry := archive.PosterIllustrationMap[illustrationID]; entry != nil &&
			entry.RewardStatus == proto.RewardStatus_Reward {
			status = proto.RewardStatus_Reward
		}
		result = append(result, &PosterIllustration{
			PosterIllustrationId: illustrationID,
			RewardStatus:         status,
		})
	}
	return result
}

func (s *Player) ClaimPosterIllustration(illustrationID uint32) (*PosterIllustration, error) {
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	cfg, _ := getEquipmentGrowthConfig()
	illustrationConf := cfg.posterIllustrations[illustrationID]
	if illustrationConf == nil || !illustrationConf.IsShow {
		return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "poster illustration %d is not configured", illustrationID)
	}
	owned := false
	for _, poster := range s.GetItemModel().GetItemPosterMap() {
		conf := cfg.posters[poster.PosterId]
		if conf != nil && uint32(conf.IllustrationID) == illustrationID {
			owned = true
			break
		}
	}
	if !owned {
		return nil, growthError(proto.StatusCode_StatusCode_POSTER_ILLUSTRATION_NOT_ACTIVE, "poster illustration %d is not active", illustrationID)
	}
	archive := s.GetArchive()
	if archive.PosterIllustrationMap == nil {
		archive.PosterIllustrationMap = make(map[uint32]*PosterIllustration)
	}
	if entry := archive.PosterIllustrationMap[illustrationID]; entry != nil &&
		entry.RewardStatus == proto.RewardStatus_Reward {
		return nil, growthError(proto.StatusCode_StatusCode_POSTER_ILLUSTRATION_REWARDED, "poster illustration %d was already rewarded", illustrationID)
	}
	entry := &PosterIllustration{
		PosterIllustrationId: illustrationID,
		RewardStatus:         proto.RewardStatus_Reward,
	}
	archive.PosterIllustrationMap[illustrationID] = entry
	tx.Commit()
	return entry, nil
}

const (
	foodBuffsArchiveKey      = "equipment.foodBuffs"
	itemUseCooldownKeyPrefix = "equipment.itemUseCooldown."
)

func (s *Player) foodBuffs(now time.Time) (map[string]int64, error) {
	value := s.GetArchive().GetArchiveValue(foodBuffsArchiveKey)
	buffs := make(map[string]int64)
	if value != "" {
		if err := sonic.UnmarshalString(value, &buffs); err != nil {
			return nil, fmt.Errorf("decode food buffs: %w", err)
		}
	}
	nowMS := now.UnixMilli()
	changed := false
	for id, endTime := range buffs {
		if endTime <= nowMS {
			delete(buffs, id)
			changed = true
		}
	}
	if changed {
		data, err := sonic.MarshalString(buffs)
		if err != nil {
			return nil, err
		}
		s.GetArchive().SetArchiveMap(foodBuffsArchiveKey, data)
	}
	return buffs, nil
}

func (s *Player) FoodBuffs(now time.Time) []*proto.Buff {
	buffs, err := s.foodBuffs(now)
	if err != nil {
		return nil
	}
	result := make([]*proto.Buff, 0, len(buffs))
	for id, endTime := range buffs {
		buffID, err := strconv.ParseUint(id, 10, 32)
		if err == nil && buffID != 0 {
			result = append(result, &proto.Buff{BuffId: uint32(buffID), EndTime: endTime})
		}
	}
	return result
}

func (s *Player) PlayerBuffs(now time.Time) []*proto.PlayerBuff {
	buffs := s.FoodBuffs(now)
	if len(buffs) == 0 {
		return nil
	}
	return []*proto.PlayerBuff{{
		SystemType: proto.PlayerBuff_BuffSystemType_GLOBAL,
		BuffLst:    buffs,
	}}
}

type ItemUseResult struct {
	Items       []*proto.ItemDetail
	PlayerBuffs []*proto.PlayerBuff
	DungeonID   uint32
	PackNotice  *proto.PackNotice
}

func (s *Player) itemUseCooldownSeconds(cfg *equipmentGrowthConfig, itemID uint32) (int64, error) {
	key := fmt.Sprintf("ItemUseCooldown.%d", itemID)
	value, configured, err := cfg.globalInt(key)
	if err != nil {
		return 0, err
	}
	if configured {
		if value < 0 {
			return 0, fmt.Errorf("negative item use cooldown")
		}
		return value, nil
	}
	value, configured, err = cfg.globalInt("ItemUseCooldown")
	if err != nil {
		return 0, err
	}
	if configured && value < 0 {
		return 0, fmt.Errorf("negative item use cooldown")
	}
	return value, nil
}

func (s *Player) UseItem(itemID uint32, count uint32, chooseItemID uint32, playerLevel uint32, now time.Time) (*ItemUseResult, error) {
	if count == 0 {
		return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "item use count is zero")
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, &GrowthError{Status: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH, Err: err}
	}
	defer tx.Rollback()
	item, itemConf, err := s.validGrowthBaseItem(itemID, int64(count))
	if err != nil {
		return nil, err
	}
	cfg, _ := getEquipmentGrowthConfig()
	cooldown, err := s.itemUseCooldownSeconds(cfg, itemID)
	if err != nil {
		return nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
	}
	cooldownKey := fmt.Sprintf("%s%d", itemUseCooldownKeyPrefix, itemID)
	if cooldown > 0 {
		lastUse, _ := strconv.ParseInt(s.GetArchive().GetArchiveValue(cooldownKey), 10, 64)
		if now.Unix() < lastUse+cooldown {
			return nil, growthError(proto.StatusCode_StatusCode_REQUESTS_TOO_FREQUENT, "item %d is on cooldown", itemID)
		}
	}

	tag := proto.EBagItemTag(itemConf.NewBagItemTag)
	var rewards []disassembleReward
	var foodBuffID uint32
	var foodEndTime int64
	switch tag {
	case proto.EBagItemTag_EBagItemTag_Gift:
		gift := cfg.giftBags[int32(itemConf.RelateID)]
		if gift == nil || len(gift.GiftBagGroup) == 0 {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "gift bag %d is not configured", itemConf.RelateID)
		}
		if playerLevel < uint32(gift.PlayerLv) {
			return nil, growthError(proto.StatusCode_StatusCode_PLAYER_LEVEL_LIMIT, "player level is too low for gift bag %d", gift.ID)
		}
		if gift.UseItemID < 0 || gift.UseItemCount < 0 || (gift.UseItemID == 0) != (gift.UseItemCount == 0) {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "gift bag %d has invalid use cost", gift.ID)
		}
		for i := uint32(0); i < count; i++ {
			selected, err := selectGiftRewards(cfg, gift, chooseItemID, playerLevel)
			if err != nil {
				return nil, &GrowthError{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND, Err: err}
			}
			rewards = append(rewards, selected...)
		}
	case proto.EBagItemTag_EBagItemTag_Food:
		if count != 1 || chooseItemID != 0 {
			return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "food must be used one at a time without a choice")
		}
		affect := cfg.cookAffects[itemConf.RelateID]
		if affect == nil || affect.NewCookFoodEffectType != 1 || affect.FoodEffectTypeParam <= 0 || affect.BuffID == 0 {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "food %d has no configured duration buff", itemID)
		}
		buff := cfg.buffs[affect.BuffID]
		if buff == nil || buff.ScriptID == 0 {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "food buff %d is not configured", affect.BuffID)
		}
		active, err := s.foodBuffs(now)
		if err != nil {
			return nil, &GrowthError{Status: proto.StatusCode_StatusCode_SYS_ERR, Err: err}
		}
		if _, ok := active[strconv.FormatUint(uint64(affect.BuffID), 10)]; ok {
			return nil, growthError(proto.StatusCode_StatusCode_REQUESTS_TOO_FREQUENT, "food buff %d is already active", affect.BuffID)
		}
		foodBuffID = uint32(affect.BuffID)
		foodEndTime = now.UnixMilli() + int64(affect.FoodEffectTypeParam)*1000
	default:
		return nil, growthError(proto.StatusCode_StatusCode_BAD_REQ, "item %d has no usable effect", itemID)
	}

	for _, reward := range rewards {
		conf := cfg.items[reward.itemID]
		if conf == nil || !canReceiveGrowthReward(cfg, conf) || reward.count <= 0 {
			return nil, growthError(proto.StatusCode_StatusCode_CFG_NOT_FOUND, "gift reward item %d is invalid", reward.itemID)
		}
	}
	if err := validateGrowthCost(tx, cfg, itemID, int64(count)); err != nil {
		return nil, err
	}
	if tag == proto.EBagItemTag_EBagItemTag_Gift {
		gift := cfg.giftBags[int32(itemConf.RelateID)]
		if gift.UseItemID != 0 {
			if err := validateGrowthCost(tx, cfg, uint32(gift.UseItemID), int64(gift.UseItemCount)*int64(count)); err != nil {
				return nil, err
			}
		}
	}
	tx.Commit()

	result := &ItemUseResult{
		Items:      []*proto.ItemDetail{item.ItemDetail()},
		PackNotice: tx.PackNotice,
	}
	totals := make(map[uint32]int64)
	for _, reward := range rewards {
		totals[reward.itemID] += reward.count
	}
	for rewardItemID, rewardCount := range totals {
		conf := cfg.items[rewardItemID]
		if isGrowthBaseItemTag(proto.EBagItemTag(conf.NewBagItemTag)) {
			info := s.addGrowthBaseReward(conf, rewardCount)
			if info == nil {
				return nil, growthError(proto.StatusCode_StatusCode_Item_Get_Fail, "failed to add gift reward item %d", rewardItemID)
			}
			result.Items = append(result.Items, info.ItemDetail())
			continue
		}
		ctx := s.AddAllTypeItem(rewardItemID, rewardCount)
		if ctx == nil {
			return nil, growthError(proto.StatusCode_StatusCode_Item_Get_Fail, "failed to add gift reward item %d", rewardItemID)
		}
		if detail := ctx.AddItemDetail(); detail != nil {
			result.Items = append(result.Items, detail)
		}
	}
	if cooldown > 0 {
		s.GetArchive().SetArchiveMap(cooldownKey, strconv.FormatInt(now.Unix(), 10))
	}
	if foodBuffID != 0 {
		active, err := s.foodBuffs(now)
		if err != nil {
			return nil, &GrowthError{Status: proto.StatusCode_StatusCode_SYS_ERR, Err: err}
		}
		active[strconv.FormatUint(uint64(foodBuffID), 10)] = foodEndTime
		data, err := sonic.MarshalString(active)
		if err != nil {
			return nil, &GrowthError{Status: proto.StatusCode_StatusCode_SYS_ERR, Err: err}
		}
		s.GetArchive().SetArchiveMap(foodBuffsArchiveKey, data)
		result.PlayerBuffs = s.PlayerBuffs(now)
	}
	return result, nil
}

func canReceiveGrowthReward(cfg *equipmentGrowthConfig, conf *excel.ItemConfigure) bool {
	if conf == nil {
		return false
	}
	tag := proto.EBagItemTag(conf.NewBagItemTag)
	if isGrowthBaseItemTag(tag) {
		return true
	}
	switch tag {
	case proto.EBagItemTag_EBagItemTag_Weapon:
		_, ok := cfg.weapons[uint32(conf.ID)]
		return ok
	case proto.EBagItemTag_EBagItemTag_Armor:
		_, ok := cfg.armors[uint32(conf.ID)]
		return ok
	case proto.EBagItemTag_EBagItemTag_Poster:
		poster, ok := cfg.posters[uint32(conf.ID)]
		if !ok {
			return false
		}
		illustration := cfg.posterIllustrations[uint32(poster.GetIllustrationID())]
		return illustration != nil && illustration.IsShow
	case proto.EBagItemTag_EBagItemTag_Inscription:
		_, ok := cfg.inscriptions[uint32(conf.ID)]
		return ok
	case proto.EBagItemTag_EBagItemTag_Card:
		return gdconf.GetCharacterAll(uint32(conf.ID)) != nil
	case proto.EBagItemTag_EBagItemTag_Fashion:
		return gdconf.GetFashionAllInfo(uint32(conf.ID)) != nil
	case proto.EBagItemTag_EBagItemTag_Head,
		proto.EBagItemTag_EBagItemTag_Expression,
		proto.EBagItemTag_EBagItemTag_Pet:
		return true
	default:
		return false
	}
}

func selectGiftRewards(cfg *equipmentGrowthConfig, gift *excel.GiftBagConfigure, chooseItemID uint32, playerLevel uint32) ([]disassembleReward, error) {
	type candidate struct {
		item  *excel.RewardItemPoolGroupInfo
		pool  *excel.RewardItemPoolConfigure
		group *excel.RewardPoolConfigure
	}
	candidates := make([]candidate, 0)
	for _, giftGroup := range gift.GiftBagGroup {
		if giftGroup == nil {
			continue
		}
		pool := cfg.rewardPools[giftGroup.RewardPoolGroupID]
		if pool == nil || len(pool.RewardPoolGroup) == 0 {
			return nil, fmt.Errorf("gift reward pool %d is missing", giftGroup.RewardPoolGroupID)
		}
		for _, poolGroup := range pool.RewardPoolGroup {
			if poolGroup == nil {
				continue
			}
			itemPool := cfg.rewardItemPools[poolGroup.RewardPoolGroupID]
			if itemPool == nil || len(itemPool.RewardItemPoolGroup) == 0 {
				return nil, fmt.Errorf("gift item pool %d is missing", poolGroup.RewardPoolGroupID)
			}
			for _, item := range itemPool.RewardItemPoolGroup {
				if item == nil || playerLevel < uint32(item.MinLevelLimit) || playerLevel > uint32(item.MaxLevelLimit) {
					continue
				}
				if chooseItemID != 0 && uint32(item.ItemID) != chooseItemID {
					continue
				}
				candidates = append(candidates, candidate{item: item, pool: itemPool, group: pool})
			}
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("gift bag %d has no eligible reward", gift.ID)
	}
	selected := candidates[rand.IntN(len(candidates))]
	if selected.item.ItemMinCount < 0 || selected.item.ItemMaxCount < selected.item.ItemMinCount {
		return nil, fmt.Errorf("gift bag %d has invalid reward count", gift.ID)
	}
	minCount, maxCount := int64(selected.item.ItemMinCount), int64(selected.item.ItemMaxCount)
	count := minCount
	if maxCount > minCount {
		count += int64(rand.IntN(int(maxCount-minCount) + 1))
	}
	if count <= 0 {
		return nil, fmt.Errorf("gift bag %d has an empty reward", gift.ID)
	}
	return []disassembleReward{{itemID: uint32(selected.item.ItemID), count: count}}, nil
}
