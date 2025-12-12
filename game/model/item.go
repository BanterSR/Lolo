package model

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type ItemModel struct {
	transactionLock    sync.Mutex                      `json:"-"`                        // 事务锁
	InstanceIndex      uint32                          `json:"instanceIndex,omitempty"`  // 物品索引生成器
	ItemBaseInfo       map[uint32]*ItemBaseInfo        `json:"itemBaseInfo,omitempty"`   // 基础物品 徽章 伞
	ItemWeaponMap      map[uint32]*ItemWeaponInfo      `json:"itemWeaponMap,omitempty"`  // 武器
	ItemFashionMap     map[uint32]*ItemFashionInfo     `json:"itemFashionMap,omitempty"` // 服装
	ItemArmorMap       map[uint32]*ItemArmorInfo       `json:"itemArmorMap,omitempty"`   // 盔甲
	ItemPosterMap      map[uint32]*ItemPosterInfo      `json:"itemPosterMap,omitempty"`  // 海报
	ItemInscriptionMap map[uint32]*ItemInscriptionInfo `json:"itemInscriptionMap,omitempty"`
}

func DefaultItemModel() *ItemModel {
	return &ItemModel{}
}

func (s *Player) GetItemModel() *ItemModel {
	if s == nil {
		return nil
	}
	if s.Item == nil {
		s.Item = DefaultItemModel()
	}
	return s.Item
}

func (i *ItemModel) InitItem() {
	i.AddItemBase(gdconf.GetConstant().DefaultBadge, 1)
	i.AddItemBase(gdconf.GetConstant().DefaultUmbrellaId, 1)
}

func (i *ItemModel) NextInstanceIndex() uint32 {
	if i == nil {
		return 0
	}
	i.InstanceIndex++
	return i.InstanceIndex
}

func (i *ItemModel) AllItemModel() {
	for tag, confList := range gdconf.GetItemByNewBagItemTagAll() {
		switch tag {
		case proto.EBagItemTag_EBagItemTag_Gift,
			proto.EBagItemTag_EBagItemTag_Fragment,
			proto.EBagItemTag_EBagItemTag_Collection,
			proto.EBagItemTag_EBagItemTag_Card,
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
			proto.EBagItemTag_EBagItemTag_Head,
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
			proto.EBagItemTag_EBagItemTag_Expression,
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
			proto.EBagItemTag_EBagItemTag_SeasonalMiniGamesItem:
			for _, conf := range confList {
				i.AddItemBase(uint32(conf.ID), 9999)
			}
		case proto.EBagItemTag_EBagItemTag_Currency:
			for _, conf := range confList {
				i.AddItemBase(uint32(conf.ID), 99999999)
			}
		case proto.EBagItemTag_EBagItemTag_UnlockItem:
			for _, conf := range confList {
				if gdconf.GetPlayerUnlockConfigure(conf.ID) == nil {
					continue
				}
				i.AddItemBase(uint32(conf.ID), 1)
			}
		case proto.EBagItemTag_EBagItemTag_AbilityItem:
			// for _, conf := range confList {
			// 	if gdconf.GetPlayerAbilityConfigure(conf.ID) == nil {
			// 		continue
			// 	}
			// 	i.AddItemBase(uint32(conf.ID), 1)
			// }
		case proto.EBagItemTag_EBagItemTag_Weapon:
			for _, conf := range confList {
				i.AddItemWeapon(uint32(conf.ID))
			}
		case proto.EBagItemTag_EBagItemTag_Fashion:
			for _, conf := range confList {
				i.AddItemFashion(uint32(conf.ID))
			}
		case proto.EBagItemTag_EBagItemTag_Armor:
			for _, conf := range confList {
				i.AddItemArmor(uint32(conf.ID))
			}
		case proto.EBagItemTag_EBagItemTag_Poster:
			for _, conf := range confList {
				i.AddItemPoster(uint32(conf.ID))
			}
		case proto.EBagItemTag_EBagItemTag_Inscription:
			for _, conf := range confList {
				i.AddItemInscription(uint32(conf.ID))
			}
		default:
			log.Game.Warnf("未知的物品类型TypeID:%v Type:%s", tag, tag.String())
		}
	}
}

type EBagItemTag interface {
	ItemDetail() *proto.ItemDetail
}

type ItemBaseInfo struct {
	ItemId   uint32            `json:"itemId,omitempty"`
	Num      int64             `json:"num,omitempty"`
	ItemType proto.EBagItemTag `json:"itemType,omitempty"`
}

func (i *ItemModel) GetItemBaseMap() map[uint32]*ItemBaseInfo {
	if i == nil {
		return nil
	}
	if i.ItemBaseInfo == nil {
		i.ItemBaseInfo = make(map[uint32]*ItemBaseInfo)
	}
	return i.ItemBaseInfo
}

func (i *ItemModel) GetItemBaseInfo(itemId uint32) *ItemBaseInfo {
	list := i.GetItemBaseMap()
	info, ok := list[itemId]
	if !ok {
		return nil
	}
	return info
}

func (i *ItemModel) AddItemBase(itemId uint32, num int64) {
	conf := gdconf.GetItemConfigure(itemId)
	list := i.GetItemBaseMap()
	if conf == nil || list == nil {
		log.Game.Warnf("添加基础物品失败,数据异常或不存在ItemID:%v", itemId)
		return
	}
	info := list[itemId]
	if info == nil {
		info = &ItemBaseInfo{
			ItemId:   itemId,
			Num:      0,
			ItemType: proto.EBagItemTag(conf.NewBagItemTag),
		}
		list[itemId] = info
	}
	info.Num += num
}

func (i *ItemBaseInfo) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  i.ItemId,
			ItemTag: i.ItemType,
			Item: &proto.ItemInfo_BaseItem{
				BaseItem: &proto.BaseItem{
					ItemId: i.ItemId,
					Num:    i.Num,
				},
			},
		},
		PackType: proto.ItemDetail_PackType_Inventory,
	}
	return info
}

type ItemWeaponInfo struct {
	ItemId           uint32                  `json:"itemId,omitempty"`           // 在背包中的id
	WeaponId         uint32                  `json:"weaponId,omitempty"`         // 武器id
	InstanceId       uint32                  `json:"instanceId,omitempty"`       // 索引id
	WeaponSystemType proto.EWeaponSystemType `json:"weaponSystemType,omitempty"` // 装备类型
	Attack           uint32                  `json:"attack,omitempty"`           // 攻击力
	DamageBalance    uint32                  `json:"damageBalance,omitempty"`    // 伤害平衡
	CriticalRatio    uint32                  `json:"criticalRatio,omitempty"`    // 临界比率
	RandomProperty   RandomPropertys         `json:"randomProperty,omitempty"`   // 随机属性
	WearerId         uint32                  `json:"wearerId,omitempty"`         // 装备者id
	WearerIndex      uint32                  `json:"wearerIndex,omitempty"`      // 装备格索引
	Level            uint32                  `json:"level,omitempty"`            // 等级
	StrengthLevel    uint32                  `json:"strengthLevel,omitempty"`    // 强度等级
	StrengthExp      uint32                  `json:"strengthExp,omitempty"`      // 强度经验
	Star             uint32                  `json:"star,omitempty"`             // 星数
	Inscription1     uint32                  `json:"inscription1,omitempty"`     // 铭文1
	Durability       uint32                  `json:"durability,omitempty"`       // 耐用性
	PropertyIndex    uint32                  `json:"propertyIndex,omitempty"`    // ?指数
	IsLock           bool                    `json:"isLock,omitempty"`           // 是否锁
}

type RandomPropertys []*RandomProperty

func (rs *RandomPropertys) RandomPropertys() []*proto.RandomProperty {
	list := make([]*proto.RandomProperty, 0, len(*rs))
	for _, v := range *rs {
		list = append(list, &proto.RandomProperty{
			PropertyType: v.PropertyType,
			Value:        v.Value,
		})
	}
	return list
}

type RandomProperty struct {
	PropertyType proto.EPropertyType // 类型
	Value        uint32              // 值
}

func (i *ItemModel) GetItemWeaponMap() map[uint32]*ItemWeaponInfo {
	if i == nil {
		return nil
	}
	if i.ItemWeaponMap == nil {
		i.ItemWeaponMap = make(map[uint32]*ItemWeaponInfo)
	}
	return i.ItemWeaponMap
}

func (i *ItemModel) GetItemWeaponInfo(instanceId uint32) *ItemWeaponInfo {
	list := i.GetItemWeaponMap()
	return list[instanceId]
}

func (i *ItemModel) AddItemWeapon(weaponId uint32) *ItemWeaponInfo {
	conf := gdconf.GetWeaponAllInfo(weaponId)
	list := i.GetItemWeaponMap()
	if conf == nil || list == nil {
		log.Game.Warnf("添加Weapon失败,数据异常或不存在WeaponId:%v", weaponId)
		return nil
	}
	instanceId := i.NextInstanceIndex()
	info := &ItemWeaponInfo{
		ItemId:           uint32(conf.WeaponInfo.GetItemID()),
		WeaponId:         conf.WeaponId,
		InstanceId:       instanceId,
		WeaponSystemType: proto.EWeaponSystemType(conf.WeaponInfo.NewWeaponSystemType),
		Attack:           1, // 攻击力
		DamageBalance:    1, // 伤害平衡
		CriticalRatio:    1, // 临界比率
		RandomProperty:   make([]*RandomProperty, 0),
		WearerId:         0,
		Level:            1,
		StrengthLevel:    0, // 强度等级
		StrengthExp:      0, // 强度经验
		Star:             1, // 星
		Inscription1:     0, //
		Durability:       0, // 磨损度
		PropertyIndex:    1, //
		IsLock:           false,
	}
	list[instanceId] = info

	return info
}

func (i *ItemWeaponInfo) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  i.ItemId,
			ItemTag: proto.EBagItemTag_EBagItemTag_Weapon,
			Item: &proto.ItemInfo_Weapon{
				Weapon: i.WeaponInstance(),
			},
		},
		PackType: proto.ItemDetail_PackType_Inventory,
	}
	return info
}

func (i *ItemWeaponInfo) SetWearerId(id, index uint32) {
	i.WearerId = id
	i.WearerIndex = index
}

func (i *ItemWeaponInfo) WeaponInstance() *proto.WeaponInstance {
	info := &proto.WeaponInstance{
		WeaponId:       i.WeaponId,
		InstanceId:     i.InstanceId,
		Attack:         i.Attack,
		DamageBalance:  i.DamageBalance,
		CriticalRatio:  i.CriticalRatio,
		RandomProperty: i.RandomProperty.RandomPropertys(),
		WearerId:       i.WearerId,
		Level:          i.Level,
		StrengthLevel:  i.StrengthLevel,
		StrengthExp:    i.StrengthExp,
		Star:           i.Star,
		Inscription1:   i.Inscription1,
		Durability:     i.Durability,
		PropertyIndex:  i.PropertyIndex,
		IsLock:         i.IsLock,
	}

	return info
}

type ItemFashionInfo struct {
	ItemId     uint32                      `json:"itemId,omitempty"`
	OutfitId   uint32                      `json:"outfitId,omitempty"`
	DyeSchemes map[uint32]*OutfitDyeScheme `json:"dyeSchemes,omitempty"`
}

type OutfitDyeScheme struct {
	SchemeIndex uint32 `json:"schemeIndex,omitempty"`
	IsUnLock    bool   `json:"isUnLock,omitempty"`
	Colors      Colors `json:"colors,omitempty"`
}

func (o *OutfitDyeScheme) OutfitDyeScheme() *proto.OutfitDyeScheme {
	info := &proto.OutfitDyeScheme{
		SchemeIndex: o.SchemeIndex,
		Colors:      o.Colors.PosColor(),
		IsUnLock:    o.IsUnLock,
	}
	return info
}

type Colors []*PosColor

type PosColor struct {
	Pos   uint32 `json:"pos,omitempty"`
	Red   uint32 `json:"red,omitempty"`
	Green uint32 `json:"green,omitempty"`
	Blue  uint32 `json:"blue,omitempty"`
}

func (is Colors) PosColor() []*proto.PosColor {
	list := make([]*proto.PosColor, 0)
	for _, i := range is {
		alg.AddList(&list, &proto.PosColor{
			Pos:   i.Pos,
			Red:   i.Red,
			Green: i.Green,
			Blue:  i.Blue,
		})
	}
	return list
}

func (i *ItemModel) GetItemFashionMap() map[uint32]*ItemFashionInfo {
	if i == nil {
		return nil
	}
	if i.ItemFashionMap == nil {
		i.ItemFashionMap = make(map[uint32]*ItemFashionInfo)
	}
	return i.ItemFashionMap
}

func (i *ItemModel) GetItemFashionInfo(id uint32) *ItemFashionInfo {
	list := i.GetItemFashionMap()
	info, ok := list[id]
	if !ok {
		return nil
	}
	return info
}

func (i *ItemModel) AddItemFashion(fashionId uint32) *ItemFashionInfo {
	conf := gdconf.GetFashionAllInfo(fashionId)
	list := i.GetItemFashionMap()
	if conf == nil || list == nil {
		log.Game.Warnf("添加Fashion失败,数据异常或不存在FashionID:%v", fashionId)
		return nil
	}
	info, ok := list[conf.FashionId]
	if !ok {
		info = newItemFashionInfo(conf)
		list[conf.FashionId] = info
	}
	return info
}

func newItemFashionInfo(conf *gdconf.FashionAllInfo) *ItemFashionInfo {
	info := &ItemFashionInfo{
		ItemId:     uint32(conf.FashionInfo.GetItemID()),
		OutfitId:   conf.FashionId,
		DyeSchemes: make(map[uint32]*OutfitDyeScheme),
	}
	info.DyeSchemes[0] = &OutfitDyeScheme{
		SchemeIndex: 0,
		IsUnLock:    true,
		Colors:      make(Colors, 0),
	}
	return info
}

func (f *ItemFashionInfo) GetDyeScheme(index uint32) *OutfitDyeScheme {
	info, ok := f.DyeSchemes[index]
	if !ok {
		return nil
	}
	return info
}

func (f *ItemFashionInfo) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  f.ItemId,
			ItemTag: proto.EBagItemTag_EBagItemTag_Fashion,
			Item: &proto.ItemInfo_Outfit{
				Outfit: &proto.Outfit{
					OutfitId: f.OutfitId,
					DyeSchemes: func() []*proto.OutfitDyeScheme {
						list := make([]*proto.OutfitDyeScheme, 0)
						for _, d := range f.DyeSchemes {
							alg.AddList(&list, d.OutfitDyeScheme())
						}
						return list
					}(),
				},
			},
		},
		PackType: proto.ItemDetail_PackType_Inventory,
	}
	return info
}

type ItemArmorInfo struct {
	WeaponSystemType proto.EWeaponSystemType `json:"weaponSystemType,omitempty"` //  装备类型
	ItemId           uint32                  `json:"itemId,omitempty"`
	ArmorId          uint32                  `json:"armorId,omitempty"`
	Star             uint32                  `json:"star,omitempty"`
	InstanceId       uint32                  `json:"instanceId,omitempty"`
	MainPropertyType proto.EPropertyType     `json:"mainPropertyType,omitempty"`
	MainPropertyVal  uint32                  `json:"mainPropertyVal,omitempty"`
	RandomProperty   RandomPropertys         `json:"randomProperty,omitempty"`
	WearerId         uint32                  `json:"wearerId,omitempty"`
	WearerIndex      uint32                  `json:"wearerIndex,omitempty"`
	Level            uint32                  `json:"level,omitempty"`
	StrengthLevel    uint32                  `json:"strengthLevel,omitempty"`
	StrengthExp      uint32                  `json:"strengthExp,omitempty"`
	PropertyIndex    uint32                  `json:"propertyIndex,omitempty"`
	IsLock           bool                    `json:"isLock,omitempty"`
}

func (i *ItemModel) GetItemArmorMap() map[uint32]*ItemArmorInfo {
	if i == nil {
		return nil
	}
	if i.ItemArmorMap == nil {
		i.ItemArmorMap = make(map[uint32]*ItemArmorInfo)
	}
	return i.ItemArmorMap
}

func (i *ItemModel) GetItemArmorInfo(instanceId uint32) *ItemArmorInfo {
	list := i.GetItemArmorMap()
	info, ok := list[instanceId]
	if !ok {
		return nil
	}
	return info
}

func (i *ItemModel) AddItemArmor(armorId uint32) *ItemArmorInfo {
	conf := gdconf.GetArmorAllInfo(armorId)
	list := i.GetItemArmorMap()
	if conf == nil || list == nil {
		log.Game.Warnf("添加Armor失败,数据异常或不存在ArmorId:%v", armorId)
		return nil
	}
	instanceId := i.NextInstanceIndex()
	info := &ItemArmorInfo{
		WeaponSystemType: proto.EWeaponSystemType(conf.ArmorInfo.NewWeaponSystemType),
		ItemId:           uint32(conf.ArmorInfo.GetItemID()),
		ArmorId:          conf.ArmorId,
		Star:             0,
		InstanceId:       instanceId,
		MainPropertyType: 0,
		MainPropertyVal:  0,
		RandomProperty:   make([]*RandomProperty, 0),
		WearerId:         0,
		Level:            1,
		StrengthLevel:    0,
		StrengthExp:      0,
		PropertyIndex:    1,
		IsLock:           false,
	}
	list[instanceId] = info
	return info
}

func (a *ItemArmorInfo) SetWearer(wearerId, wearerIndex uint32) {
	a.WearerId = wearerId
	a.WearerIndex = wearerIndex
}

func (a *ItemArmorInfo) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  a.InstanceId,
			ItemTag: proto.EBagItemTag_EBagItemTag_Armor,
			Item: &proto.ItemInfo_Armor{
				Armor: a.ArmorInstance(),
			},
		},
		PackType: proto.ItemDetail_PackType_Inventory,
	}
	return info
}

func (a *ItemArmorInfo) ArmorInstance() *proto.ArmorInstance {
	info := &proto.ArmorInstance{
		ArmorId:          a.ArmorId,
		InstanceId:       a.InstanceId,
		MainPropertyType: a.MainPropertyType,
		MainPropertyVal:  a.MainPropertyVal,
		RandomProperty:   a.RandomProperty.RandomPropertys(),
		WearerId:         a.WearerId,
		Level:            a.Level,
		StrengthLevel:    a.StrengthLevel,
		StrengthExp:      a.StrengthExp,
		PropertyIndex:    a.PropertyIndex,
		IsLock:           a.IsLock,
	}

	return info
}

func (a *ItemArmorInfo) BaseArmor() *proto.BaseArmor {
	if a == nil {
		return nil
	}
	return &proto.BaseArmor{
		ArmorId:   a.ArmorId,
		ArmorStar: a.Star,
	}
}

type ItemPosterInfo struct {
	PosterId    uint32 `json:"posterId,omitempty"`
	ItemId      uint32 `json:"itemId,omitempty"`
	InstanceId  uint32 `json:"instanceId,omitempty"`
	WearerId    uint32 `json:"wearerId,omitempty"`
	WearerIndex uint32 `json:"wearerIndex,omitempty"`
	Star        uint32 `json:"star,omitempty"`
}

func (i *ItemModel) GetItemPosterMap() map[uint32]*ItemPosterInfo {
	if i == nil {
		return nil
	}
	if i.ItemPosterMap == nil {
		i.ItemPosterMap = make(map[uint32]*ItemPosterInfo)
	}
	return i.ItemPosterMap
}

func (i *ItemModel) GetItemPosterInfo(instanceId uint32) *ItemPosterInfo {
	list := i.GetItemPosterMap()
	info, ok := list[instanceId]
	if !ok {
		return nil
	}
	return info
}

func (i *ItemModel) AddItemPoster(posterId uint32) *ItemPosterInfo {
	conf := gdconf.GetPosterAllInfo(posterId)
	list := i.GetItemPosterMap()
	if conf == nil || list == nil ||
		!conf.PosterIllustration.IsShow {
		log.Game.Warnf("添加Poster失败,数据异常或不存在PosterId:%v", posterId)
		return nil
	}
	instanceId := i.NextInstanceIndex()
	info := newItemPosterInfo(conf, instanceId)
	list[instanceId] = info

	return info
}

func newItemPosterInfo(conf *gdconf.PosterAllInfo, instanceId uint32) *ItemPosterInfo {
	return &ItemPosterInfo{
		PosterId:   conf.PosterId,
		ItemId:     uint32(conf.PosterInfo.GetItemID()),
		InstanceId: instanceId,
		WearerId:   0,
		Star:       1,
	}
}

func (p *ItemPosterInfo) SetWearer(wearerId, wearerIndex uint32) {
	p.WearerId = wearerId
	p.WearerIndex = wearerIndex
}

func (p *ItemPosterInfo) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  p.InstanceId,
			ItemTag: proto.EBagItemTag_EBagItemTag_Poster,
			Item: &proto.ItemInfo_Poster{
				Poster: p.PosterInstance(),
			},
		},
		PackType: proto.ItemDetail_PackType_Inventory,
	}
	return info
}

func (p *ItemPosterInfo) PosterInstance() *proto.PosterInstance {
	info := &proto.PosterInstance{
		PosterId:   p.PosterId,
		InstanceId: p.InstanceId,
		WearerId:   p.WearerId,
		Star:       p.Star,
	}
	return info
}

func (p *ItemPosterInfo) BasePoster() *proto.BasePoster {
	if p == nil {
		return nil
	}
	return &proto.BasePoster{
		PosterId:   p.PosterId,
		PosterStar: p.Star,
	}
}

type ItemInscriptionInfo struct {
	ItemId           uint32 `json:"itemId,omitempty"`
	InscriptionId    uint32 `json:"inscriptionId,omitempty"`
	Level            uint32 `json:"level,omitempty"`
	WeaponInstanceId uint32 `json:"weaponInstanceId,omitempty"`
}

func (i *ItemModel) GetItemInscriptionMap() map[uint32]*ItemInscriptionInfo {
	if i == nil {
		return nil
	}
	if i.ItemInscriptionMap == nil {
		i.ItemInscriptionMap = make(map[uint32]*ItemInscriptionInfo)
	}
	return i.ItemInscriptionMap
}

func (i *ItemModel) AddItemInscription(inscriptionId uint32) *ItemInscriptionInfo {
	conf := gdconf.GetInscriptionAllInfo(inscriptionId)
	list := i.GetItemInscriptionMap()
	if conf == nil || list == nil {
		log.Game.Warnf("添加Inscription失败,数据异常或不存在InscriptionId:%v", inscriptionId)
		return nil
	}
	if _, ok := list[conf.InscriptionId]; ok {
		return nil
	}
	info := &ItemInscriptionInfo{
		ItemId:           uint32(conf.InscriptionInfo.GetItemID()),
		InscriptionId:    conf.InscriptionId,
		Level:            1,
		WeaponInstanceId: 0,
	}
	list[conf.InscriptionId] = info

	return info
}

func (i *ItemInscriptionInfo) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  i.ItemId,
			ItemTag: proto.EBagItemTag_EBagItemTag_Inscription,
			Item: &proto.ItemInfo_Inscription{
				Inscription: i.GetPbInscription(),
			},
		},
		PackType: proto.ItemDetail_PackType_Inventory,
	}
	return info
}

func (i *ItemInscriptionInfo) GetPbInscription() *proto.Inscription {
	info := &proto.Inscription{
		InscriptionId:    i.InscriptionId,
		Level:            i.Level,
		WeaponInstanceId: i.WeaponInstanceId,
	}
	return info
}

// 背包事务
type ItemTransaction struct {
	Error      error
	close      bool
	i          *ItemModel
	baseItem   map[uint32]int64
	PackNotice *proto.PackNotice
}

// 创建新的事务
func (i *ItemModel) Begin() (*ItemTransaction, error) {
	timer := time.NewTimer(time.Second * 1)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil, errors.New("事务锁获取超时")
	default:
		i.transactionLock.Lock()
		t := &ItemTransaction{
			i:        i,
			baseItem: make(map[uint32]int64),
		}
		return t, nil
	}
}

// 提交事务
func (t *ItemTransaction) Commit() (tx *ItemTransaction) {
	defer func() {
		if !t.close {
			t.close = true
			t.i.transactionLock.Unlock()
		}
	}()
	t.PackNotice = &proto.PackNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  make([]*proto.ItemDetail, 0),
	}
	for id, num := range t.baseItem {
		info := t.i.GetItemBaseInfo(id)
		info.Num -= num
		alg.AddList(&t.PackNotice.Items, info.ItemDetail())
	}
	return t
}

// 撤销事务
func (t *ItemTransaction) Rollback() {
	defer func() {
		if !t.close {
			t.close = true
			t.i.transactionLock.Unlock()
		}
	}()
}

func (t *ItemTransaction) DelBaseItem(id uint32, num int64) (tx *ItemTransaction) {
	t.baseItem[id] += num
	info := t.i.GetItemBaseInfo(id)
	if info == nil || info.Num < t.baseItem[id] {
		t.Error = fmt.Errorf("扣除物品:%v数量:%v 失败,原因:物品数量不足", id, num)
		return t
	}
	return t
}
