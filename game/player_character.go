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
		Status: proto.StatusCode_StatusCode_OK,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetCharacterAchievementList(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCharacterAchievementListReq)
	rsp := &proto.GetCharacterAchievementListRsp{
		Status:                  proto.StatusCode_StatusCode_OK,
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
		Status: proto.StatusCode_StatusCode_OK,
		CharId: req.CharId,
		Level:  0,
		Exp:    0,
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		log.Game.Warnf("保存角色升级失败,角色%v不存在", req.CharId)
		return
	}
	// 申请事务
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
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
			rsp.Status = proto.StatusCode_StatusCode_EXPLORE_NUM_LIMIT
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
		rsp.Status = proto.StatusCode_StatusCode_EXPLORE_NUM_LIMIT
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
		Status:   proto.StatusCode_StatusCode_OK,
		CharId:   req.CharId,
		Level:    0,
		Exp:      0,
		MaxLevel: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		log.Game.Warnf("保存角色升级失败,角色%v不存在", req.CharId)
		return
	}
	conf := gdconf.GetCharacterAll(characterInfo.CharacterId)
	if len(conf.LevelRules) < int(characterInfo.BreakLevel) {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		return
	}
	// 申请事务
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
		log.Game.Errorf("玩家:%v申请背包事务失败:%s", s.UserId, err.Error())
		return
	}
	for _, itemInfo := range conf.LevelRules[int(characterInfo.BreakLevel)].RuleNeedItem {
		if tx.DelBaseItem(uint32(itemInfo.NeedItemID), int64(itemInfo.NeedItemCount)).Error != nil {
			tx.Rollback()
			rsp.Status = proto.StatusCode_StatusCode_EXPLORE_NUM_LIMIT
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
		Status: proto.StatusCode_StatusCode_OK,
		CharId: req.CharId,
		Preset: req.Preset,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		teamInfo := s.GetTeamModel().GetTeamInfo()
		scenePlayer := g.getWordInfo().getScenePlayer(s)
		if (req.CharId == teamInfo.Char1 ||
			req.CharId == teamInfo.Char2 ||
			req.CharId == teamInfo.Char3) &&
			(scenePlayer != nil &&
				scenePlayer.channelInfo != nil) {
			scenePlayer.channelInfo.serverSceneSyncChan <- &ServerSceneSyncCtx{
				ScenePlayer: scenePlayer,
				ActionType:  proto.SceneActionType_SceneActionType_UPDATE_FASHION,
			}
		}
	}()
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色预设装扮失败,角色%v不存在", req.CharId)
		return
	}
	outfitPreset := characterInfo.GetOutfitPreset(req.Preset.PresetIndex)

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
		Status:    proto.StatusCode_StatusCode_OK,
		Character: make([]*proto.Character, 0),
		Items:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色装备失败,角色%v不存在", req.CharId)
		return
	}
	defer alg.AddList(&rsp.Character, characterInfo.Character())

	equipmentPreset := characterInfo.GetEquipmentPreset(req.EquipmentPreset.PresetIndex)
	// 更新武器
	if req.EquipmentPreset.Weapon != equipmentPreset.WeaponInstanceId {
		oldEquipmentInfo := s.GetItemModel().GetItemWeaponInfo(equipmentPreset.WeaponInstanceId)
		newEquipmentInfo := s.GetItemModel().GetItemWeaponInfo(req.EquipmentPreset.Weapon)

		if newEquipmentInfo != nil {
			oldCharacterInfo := s.GetCharacterModel().GetCharacterInfo(newEquipmentInfo.WearerId)
			if oldCharacterInfo != nil {
				// 移除装备上的角色
				for _, oldCharacterEquipmentPreset := range oldCharacterInfo.GetEquipmentPresetList() {
					if oldCharacterEquipmentPreset.WeaponInstanceId == newEquipmentInfo.InstanceId {
						oldCharacterEquipmentPreset.WeaponInstanceId = 0
						alg.AddList(&rsp.Character, oldCharacterInfo.Character())
						break
					}
				}
			}

			equipmentPreset.WeaponInstanceId = newEquipmentInfo.InstanceId
			newEquipmentInfo.WearerId = characterInfo.CharacterId
			alg.AddList(&rsp.Items, newEquipmentInfo.ItemDetail())
		}

		if oldEquipmentInfo != nil {
			oldEquipmentInfo.WearerId = 0
			alg.AddList(&rsp.Items, oldEquipmentInfo.ItemDetail())
		}
	}
	// 更新盔甲
	// 更新海报
}
