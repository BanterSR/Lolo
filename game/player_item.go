package game

import (
	"fmt"
	"math"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) AllPackNotice(s *model.Player) {
	itemModel := s.GetItemModel()
	baseMap := itemModel.GetItemBaseMap()
	fashionMap := itemModel.GetItemFashionMap()
	weaponMap := itemModel.GetItemWeaponMap()
	armorMap := itemModel.GetItemArmorMap()
	posterMap := itemModel.GetItemPosterMap()
	inscriptionMap := itemModel.GetItemInscriptionMap()
	petMap := itemModel.GetItemPetInfoMap()

	notice := &proto.PackNotice{
		Status:          proto.StatusCode_StatusCode_Ok,
		Items:           make([]*proto.ItemDetail, 0, len(baseMap)+len(fashionMap)+len(weaponMap)+len(armorMap)+len(posterMap)+len(inscriptionMap)+len(petMap)),
		TempPackMaxSize: 30,
		IsClearTempPack: false,
	}
	defer g.send(s, 0, notice)

	for _, v := range baseMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	for _, v := range fashionMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	for _, v := range weaponMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	for _, v := range armorMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	for _, v := range posterMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	for _, v := range inscriptionMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	for _, v := range petMap {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
}

func (g *Game) PackNoticeByItems(s *model.Player, items []*proto.ItemDetail) {
	g.send(s, 0, &proto.PackNotice{
		Status:          proto.StatusCode_StatusCode_Ok,
		Items:           items,
		TempPackMaxSize: 0,
		IsClearTempPack: false,
	})
}

func (g *Game) GetWeapon(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetWeaponReq)
	weaponMap := s.GetItemModel().GetItemWeaponMap()
	rsp := &proto.GetWeaponRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Weapons:  make([]*proto.WeaponInstance, 0, len(weaponMap)),
		TotalNum: uint32(len(weaponMap)),
		EndIndex: uint32(len(weaponMap)),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range weaponMap {
		if req.WeaponSystemType == proto.EWeaponSystemType_EWeaponSystemType_None ||
			req.WeaponSystemType == v.WeaponSystemType {
			rsp.Weapons = append(rsp.Weapons, v.WeaponInstance())
		}
	}
}

func (g *Game) GetArmor(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetArmorReq)
	armorMap := s.GetItemModel().GetItemArmorMap()
	rsp := &proto.GetArmorRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Armors:   make([]*proto.ArmorInstance, 0, len(armorMap)),
		TotalNum: uint32(len(armorMap)),
		EndIndex: uint32(len(armorMap)),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range armorMap {
		if req.WeaponSystemType == proto.EWeaponSystemType_EWeaponSystemType_None ||
			req.WeaponSystemType == v.WeaponSystemType {
			rsp.Armors = append(rsp.Armors, v.ArmorInstance())
		}
	}
}

func (g *Game) GetPoster(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetPosterReq)
	posterMap := s.GetItemModel().GetItemPosterMap()
	rsp := &proto.GetPosterRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Posters:  make([]*proto.PosterInstance, 0, len(posterMap)),
		TotalNum: uint32(len(posterMap)),
		EndIndex: uint32(len(posterMap)),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range posterMap {
		alg.AddList(&rsp.Posters, v.PosterInstance())
	}
}

func (g *Game) PosterIllustrationList(s *model.Player, msg *alg.GameMsg) {
	illustrations := s.PosterIllustrations()
	rsp := &proto.PosterIllustrationListRsp{
		Status:              proto.StatusCode_StatusCode_Ok,
		PosterIllustrations: make([]*proto.PosterIllustration, 0, len(illustrations)),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range illustrations {
		alg.AddList(&rsp.PosterIllustrations, &proto.PosterIllustration{
			PosterIllustrationId: v.PosterIllustrationId,
			Status:               v.RewardStatus,
		})
	}
}

// 基础物品奖励统一在 ItemModel 事务锁内发放。先完整校验配置、类型和溢出，
// 再写入背包，避免奖励列表中途失败造成部分发放。
func grantProfileBaseRewards(s *model.Player, rewards []profileItemReward) ([]*proto.ItemDetail, error) {
	if s == nil {
		return nil, fmt.Errorf("player is nil")
	}
	totals := make(map[uint32]int64, len(rewards))
	for _, reward := range rewards {
		if reward.itemID == 0 || reward.count <= 0 {
			return nil, fmt.Errorf("invalid profile reward %d:%d", reward.itemID, reward.count)
		}
		conf := profileItemConfigure(reward.itemID)
		if conf == nil || !profileBaseRewardTag(proto.EBagItemTag(conf.GetNewBagItemTag())) {
			return nil, fmt.Errorf("profile reward %d is not a configured base item", reward.itemID)
		}
		total := totals[reward.itemID]
		if total > math.MaxInt64-reward.count {
			return nil, fmt.Errorf("profile reward %d count overflow", reward.itemID)
		}
		totals[reward.itemID] = total + reward.count
	}
	items := s.GetItemModel()
	for id, total := range totals {
		info := items.GetItemBaseInfo(id)
		if info != nil && (info.Num < 0 || info.Num > math.MaxInt64-total) {
			return nil, fmt.Errorf("profile reward %d balance overflow", id)
		}
	}
	result := make([]*proto.ItemDetail, 0, len(rewards))
	for _, reward := range rewards {
		conf := profileItemConfigure(reward.itemID)
		info := items.GetItemBaseInfo(reward.itemID)
		if info == nil {
			info = &model.ItemBaseInfo{
				ItemId:   reward.itemID,
				ItemType: proto.EBagItemTag(conf.GetNewBagItemTag()),
				PackType: proto.PackType_PackType_Inventory,
			}
			items.GetItemBaseMap()[reward.itemID] = info
		}
		info.Num += reward.count
		result = append(result, info.ItemDetail())
	}
	return result, nil
}

func profileBaseRewardTag(tag proto.EBagItemTag) bool {
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
