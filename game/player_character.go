package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/constant"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetAllCharacterEquip(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GetAllCharacterEquipRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetCharacterAchievementList(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCharacterAchievementListReq)
	rsp := &proto.GetCharacterAchievementListRsp{
		Status:                  proto.StatusCode_StatusCode_Ok,
		CharacterAchievementLst: make([]*proto.Achieve, 0),
		HasRewardedIds:          make([]uint32, 0),
		IsUnlockedPayment:       false,
		CharacterId:             req.CharacterId,
		RewardedIdLst:           make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) CharacterLevelUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterLevelUpReq)
	rsp := &proto.CharacterLevelUpRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		CharId: req.CharId,
		Level:  0,
		Exp:    0,
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CharacterPlaced
		log.Game.Warnf("保存角色升级失败,角色%v不存在", req.CharId)
		return
	}
	// 申请事务
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_ItemNotEnough
		log.Game.Errorf("玩家:%v申请背包事务失败:%s", s.UserId, err.Error())
		return
	}
	// 物品消耗
	itemList := gdconf.GetGlobalConfigConfigure(constant.CharacterLevelUpNeedItem)
	itemExp := uint32(0)
	for index, num := range req.Nums {
		itemId := alg.S2U32(itemList.Values[index])
		if tx.DelBaseItem(itemId, num).Error != nil {
			tx.Rollback()
			rsp.Status = proto.StatusCode_StatusCode_ExploreNumLimit
			log.Game.Errorf("玩家:%v扣除背包物品失败:%s", s.UserId, tx.Error.Error())
			return
		}
		itemExp += constant.ItemAddExp[itemId] * uint32(num)
	}
	level, exp := gdconf.AddCharacterExp(
		gdconf.GetCharacterAll(characterInfo.CharacterId).CharacterInfo.GetGrowthLevelID(),
		int32(characterInfo.Exp+itemExp),
		characterInfo.Level,
		characterInfo.MaxLevel,
	)
	if tx.DelBaseItem(constant.CurrencyGold, int64(itemExp/50)).Error != nil {
		tx.Rollback()
		rsp.Status = proto.StatusCode_StatusCode_ExploreNumLimit
		log.Game.Errorf("玩家:%v扣除金币失败:%s", s.UserId, tx.Error.Error())
		return
	}
	// 给予经验
	characterInfo.Level = level
	characterInfo.Exp = exp

	// 提交事务
	tx.Commit()
	g.send(s, 0, tx.PackNotice)

	rsp.Level = characterInfo.Level
	rsp.Exp = characterInfo.Exp
}

func (g *Game) CharacterLevelBreak(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterLevelBreakReq)
	rsp := &proto.CharacterLevelBreakRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		CharId:   req.CharId,
		Level:    0,
		Exp:      0,
		MaxLevel: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CharacterPlaced
		log.Game.Warnf("保存角色升级失败,角色%v不存在", req.CharId)
		return
	}
	conf := gdconf.GetCharacterAll(characterInfo.CharacterId)
	if len(conf.LevelRules) < int(characterInfo.BreakLevel) {
		rsp.Status = proto.StatusCode_StatusCode_CharacterPlaced
		return
	}
	// 申请事务
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_ItemNotEnough
		log.Game.Errorf("玩家:%v申请背包事务失败:%s", s.UserId, err.Error())
		return
	}
	for _, itemInfo := range conf.LevelRules[int(characterInfo.BreakLevel)].RuleNeedItem {
		if tx.DelBaseItem(uint32(itemInfo.NeedItemID), int64(itemInfo.NeedItemCount)).Error != nil {
			tx.Rollback()
			rsp.Status = proto.StatusCode_StatusCode_ExploreNumLimit
			log.Game.Errorf("玩家:%v扣除背包物品失败:%s", s.UserId, tx.Error.Error())
			return
		}
	}
	tx.Commit()
	g.send(s, 0, tx.PackNotice)

	characterInfo.BreakLevel++
	characterInfo.UpMaxLevel()
	rsp.MaxLevel = characterInfo.MaxLevel
	rsp.Level = characterInfo.Level
	rsp.Exp = characterInfo.Exp
}

func (g *Game) OutfitPresetUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitPresetUpdateReq)
	rsp := &proto.OutfitPresetUpdateRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		CharId: req.CharId,
		Preset: nil,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UpdateFashion, req.CharId)
	}()
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色预设装扮失败,角色%v不存在", req.CharId)
		return
	}
	outfitPreset := characterInfo.GetOutfitPreset(req.Preset.PresetIndex)
	defer func() {
		rsp.Preset = outfitPreset.OutfitPreset()
	}()

	outfitPreset.Hat = req.Preset.Hat
	outfitPreset.HatDyeSchemeIndex = req.Preset.HatDyeSchemeIndex
	outfitPreset.Hair = req.Preset.Hair
	outfitPreset.HairDyeSchemeIndex = req.Preset.HairDyeSchemeIndex
	outfitPreset.Clothes = req.Preset.Clothes
	outfitPreset.ClothesDyeSchemeIndex = req.Preset.ClothesDyeSchemeIndex
	outfitPreset.Ornament = req.Preset.Ornament
	outfitPreset.OrnamentDyeSchemeIndex = req.Preset.OrnamentDyeSchemeIndex
	outfitPreset.OutfitHideInfo = &model.OutfitHideInfo{
		HideOrn:   req.Preset.OutfitHideInfo.HideOrn,
		HideBraid: req.Preset.OutfitHideInfo.HideBraid,
	}
	outfitPreset.PendTop = req.Preset.PendTop
	outfitPreset.PendTopDyeSchemeIndex = req.Preset.PendTopDyeSchemeIndex
	outfitPreset.PendChest = req.Preset.PendChest
	outfitPreset.PendChestDyeSchemeIndex = req.Preset.PendChestDyeSchemeIndex
	outfitPreset.PendPelvis = req.Preset.PendPelvis
	outfitPreset.PendPelvisDyeSchemeIndex = req.Preset.PendPelvisDyeSchemeIndex
	outfitPreset.PendUpFace = req.Preset.PendUpFace
	outfitPreset.PendUpFaceDyeSchemeIndex = req.Preset.PendUpFaceDyeSchemeIndex
	outfitPreset.PendDownFace = req.Preset.PendDownFace
	outfitPreset.PendDownFaceDyeSchemeIndex = req.Preset.PendDownFaceDyeSchemeIndex
	outfitPreset.PendLeftHand = req.Preset.PendLeftHand
	outfitPreset.PendLeftHandDyeSchemeIndex = req.Preset.PendLeftHandDyeSchemeIndex
	outfitPreset.PendRightHand = req.Preset.PendRightHand
	outfitPreset.PendRightHandDyeSchemeIndex = req.Preset.PendRightHandDyeSchemeIndex
	outfitPreset.PendLeftFoot = req.Preset.PendLeftFoot
	outfitPreset.PendLeftFootDyeSchemeIndex = req.Preset.PendLeftFootDyeSchemeIndex
	outfitPreset.PendRightFoot = req.Preset.PendRightFoot
	outfitPreset.PendRightFootDyeSchemeIndex = req.Preset.PendRightFootDyeSchemeIndex
}

func (g *Game) CharacterEquipUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterEquipUpdateReq)
	rsp := &proto.CharacterEquipUpdateRsp{
		Status:    proto.StatusCode_StatusCode_Ok,
		Character: make([]*proto.Character, 0),
		Items:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色装备失败,角色%v不存在", req.CharId)
		return
	}
	defer func() {
		alg.AddList(&rsp.Character, characterInfo.Character())
	}()

	upCharacterList := make(map[uint32]struct{}, 2)
	defer func() {
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UpdateEquip, alg.OrNum(upCharacterList)...)
		upCharacterList[req.CharId] = struct{}{}
	}()

	equipmentPreset := characterInfo.GetEquipmentPreset(req.EquipmentPreset.PresetIndex)
	// 更新武器
	if req.EquipmentPreset.Weapon != equipmentPreset.WeaponInstanceId {
		oldEquipmentInfo := s.GetItemModel().GetItemWeaponInfo(equipmentPreset.WeaponInstanceId)
		newEquipmentInfo := s.GetItemModel().GetItemWeaponInfo(req.EquipmentPreset.Weapon)

		if newEquipmentInfo != nil {
			// 移除新装备上的角色
			oldCharacterInfo := s.GetCharacterModel().GetCharacterInfo(newEquipmentInfo.WearerId)
			if oldCharacterInfo != nil {
				oldEquipmentPreset := oldCharacterInfo.GetEquipmentPreset(newEquipmentInfo.WearerIndex)
				oldEquipmentPreset.WeaponInstanceId = 0
				alg.AddList(&rsp.Character, oldCharacterInfo.Character())
				upCharacterList[oldCharacterInfo.CharacterId] = struct{}{}
			}
			// 将新装备赋到目标角色上
			equipmentPreset.WeaponInstanceId = newEquipmentInfo.InstanceId
			newEquipmentInfo.SetWearerId(characterInfo.CharacterId, equipmentPreset.PresetIndex)
			alg.AddList(&rsp.Items, newEquipmentInfo.ItemDetail())
		}

		// 将老装备取消装备
		if oldEquipmentInfo != nil {
			oldEquipmentInfo.SetWearerId(0, 0)
			alg.AddList(&rsp.Items, oldEquipmentInfo.ItemDetail())
		}
	}
	// 更新盔甲
	for _, armor := range req.EquipmentPreset.Armors {
		curArmor := equipmentPreset.GetArmor(armor.EquipType)
		if curArmor.InstanceId != armor.ArmorId {
			oldArmorInfo := s.GetItemModel().GetItemArmorInfo(curArmor.InstanceId)
			newArmorInfo := s.GetItemModel().GetItemArmorInfo(armor.ArmorId)

			if newArmorInfo != nil {
				oldCharacterInfo := s.GetCharacterModel().GetCharacterInfo(newArmorInfo.WearerId)
				if oldCharacterInfo != nil {
					oldEquipmentPreset := oldCharacterInfo.GetEquipmentPreset(newArmorInfo.WearerIndex)
					oldArmor := oldEquipmentPreset.GetArmor(armor.EquipType)
					oldArmor.InstanceId = 0
					upCharacterList[oldCharacterInfo.CharacterId] = struct{}{}
				}
				newArmorInfo.SetWearer(req.CharId, equipmentPreset.PresetIndex)
				alg.AddList(&rsp.Items, newArmorInfo.ItemDetail())
			}

			if oldArmorInfo != nil {
				oldArmorInfo.SetWearer(0, 0)
				alg.AddList(&rsp.Items, oldArmorInfo.ItemDetail())
			}

			curArmor.InstanceId = armor.ArmorId
		}
	}

	// 更新海报
	for _, poster := range req.EquipmentPreset.Posters {
		curPoster := equipmentPreset.GetPoster(poster.PosterIndex)
		if curPoster.InstanceId != poster.PosterId {
			oldPosterInfo := s.GetItemModel().GetItemArmorInfo(curPoster.InstanceId)
			newPosterInfo := s.GetItemModel().GetItemPosterInfo(poster.PosterId)

			if newPosterInfo != nil {
				// 检查新海报上是否有角色
				oldCharacterInfo := s.GetCharacterModel().GetCharacterInfo(newPosterInfo.WearerId)
				if oldCharacterInfo != nil {
					oldEquipmentPreset := oldCharacterInfo.GetEquipmentPreset(newPosterInfo.WearerIndex)
					oldPoster := oldEquipmentPreset.GetPoster(poster.PosterIndex)
					oldPoster.InstanceId = 0
					upCharacterList[oldCharacterInfo.CharacterId] = struct{}{}
				}
				newPosterInfo.SetWearer(req.CharId, equipmentPreset.PresetIndex)
				alg.AddList(&rsp.Items, newPosterInfo.ItemDetail())
			}

			// 检查目标位置是否有海报
			if oldPosterInfo != nil {
				oldPosterInfo.SetWearer(0, 0)
				alg.AddList(&rsp.Items, oldPosterInfo.ItemDetail())
			}
			curPoster.InstanceId = poster.PosterId
		}
	}
}

func (g *Game) UpdateCharacterAppearance(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UpdateCharacterAppearanceReq)
	rsp := &proto.UpdateCharacterAppearanceRsp{
		Status:     proto.StatusCode_StatusCode_Ok,
		CharId:     req.CharId,
		Appearance: nil,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdateAppearance, req.CharId)
	}()
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色外观失败,角色%v不存在", req.CharId)
		return
	}
	characterInfo.CharacterAppearance = &model.CharacterAppearance{
		Badge:                      req.Appearance.Badge,
		UmbrellaId:                 req.Appearance.UmbrellaId,
		InsectNetInstanceId:        req.Appearance.InsectNetInstanceId,
		LoggingAxeInstanceId:       req.Appearance.LoggingAxeInstanceId,
		WaterBottleInstanceId:      req.Appearance.WaterBottleInstanceId,
		MiningHammerInstanceId:     req.Appearance.MiningHammerInstanceId,
		CollectionGlovesInstanceId: req.Appearance.CollectionGlovesInstanceId,
		FishingRodInstanceId:       req.Appearance.FishingRodInstanceId,
		VehicleInstanceId:          req.Appearance.VehicleInstanceId,
	}
	rsp.Appearance = characterInfo.GetPbCharacterAppearance()
}

func (g *Game) CharacterGatherWeaponUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterGatherWeaponUpdateReq)
	rsp := &proto.CharacterGatherWeaponUpdateRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UpdateEquip, req.CharacterId)
	}()
	itemInfo := s.GetItemModel().GetItemWeaponInfo(req.WeaponId)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharacterId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v 角色:%v 不存在", s.UserId, req.CharacterId)
		return
	}
	if itemInfo != nil {
		characterInfo.GatherWeapon = itemInfo.WeaponId
	} else {
		characterInfo.GatherWeapon = 0
	}

}
