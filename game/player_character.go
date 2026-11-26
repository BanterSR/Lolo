package game

import (
	"strconv"

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
	items := s.GetItemModel()
	for _, character := range s.GetCharacterModel().GetCharacterMap() {
		if character == nil {
			continue
		}
		preset := character.GetEquipmentPreset(character.InUseEquipmentPresetIndex)
		if preset == nil {
			continue
		}
		if weapon := items.GetItemWeaponInfo(preset.WeaponInstanceId); weapon != nil {
			alg.AddList(&rsp.Items, weapon.ItemDetail())
		}
		for _, armor := range preset.Armors {
			if armor == nil {
				continue
			}
			if info := items.GetItemArmorInfo(armor.InstanceId); info != nil {
				alg.AddList(&rsp.Items, info.ItemDetail())
			}
		}
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) CharacterLevelUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterLevelUpReq)
	rsp := &proto.CharacterLevelUpRsp{
		Status: proto.StatusCode_StatusCode_OK,
		CharId: req.CharId,
		Level:  0,
		Exp:    0,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UPDATE_CHARACTER_LV, req.CharId)
	}()
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
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UPDATE_CHARACTER_BREAK_LV, req.CharId)
	}()
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
		Preset: nil,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UPDATE_FASHION, req.CharId)
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

func (g *Game) OutfitPresetSwitch(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitPresetSwitchReq)
	rsp := &proto.OutfitPresetSwitchRsp{
		Status:         proto.StatusCode_StatusCode_OK,
		CharId:         req.CharId,
		UsePresetIndex: req.UsePresetIndex,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UPDATE_FASHION, req.CharId)
	}()
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色预设装扮失败,角色%v不存在", req.CharId)
		return
	}
	characterInfo.InUseOutfitPresetIndex = req.UsePresetIndex
}

func (g *Game) OutfitColorantSelect(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.OutfitColorantSelectReq)
	rsp := &proto.OutfitColorantSelectRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Param: &proto.OutfitDyeParam{
			Params: make([]float64, 0),
		},
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil || req.OutfitId == 0 || req.ColorantItemId == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if s.GetItemModel().GetItemFashionInfo(req.OutfitId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	s.GetArchive().SetArchiveMap(
		"outfitColorant:"+strconv.Itoa(int(req.OutfitId)),
		strconv.Itoa(int(req.ColorantItemId)),
	)
	rsp.Param.PictureId = req.ColorantItemId
	rsp.Param.IsDye = true
}

func (g *Game) OutfitDye(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitDyeReq)
	rsp := &proto.OutfitDyeRsp{
		Status:               proto.StatusCode_StatusCode_OK,
		OutfitId:             req.OutfitId,
		TotalGuaranteeDyeNum: 5,
	}
	defer g.send(s, msg.PacketId, rsp)

	fashion := s.GetItemModel().GetItemFashionInfo(req.OutfitId)
	if fashion == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	scheme := fashion.GetDyeScheme(req.SchemeIndex)
	if scheme == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}

	color := req.SpecifyColor
	if color == nil {
		// 没有指定颜色时用请求参数生成稳定颜色，保证同一操作重放结果一致。
		seed := uint32(req.OutfitId*37 + req.Pos*17 + req.SchemeIndex*7 + req.AddRateColorIndex)
		color = &proto.PosColor{
			Pos:   req.Pos,
			Red:   (seed * 73) % 256,
			Green: (seed * 151) % 256,
			Blue:  (seed * 199) % 256,
		}
	}
	color.Pos = req.Pos
	scheme.Colors = upsertOutfitDyeColor(scheme.Colors, color)
	rsp.PosColor = &proto.PosColor{Pos: color.Pos, Red: color.Red, Green: color.Green, Blue: color.Blue}
}

func (g *Game) OutFitDyeSave(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutFitDyeSaveReq)
	rsp := &proto.OutFitDyeSaveRsp{
		Status:          proto.StatusCode_StatusCode_OK,
		SchemeIndex:     req.SchemeIndex,
		Items:           make([]*proto.ItemDetail, 0),
		IsSaveDyeResult: req.IsSaveDyeResult,
	}
	defer g.send(s, msg.PacketId, rsp)

	fashion := s.GetItemModel().GetItemFashionInfo(req.OutfitId)
	if fashion == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	scheme := fashion.GetDyeScheme(req.SchemeIndex)
	if scheme == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if req.IsSaveDyeResult {
		scheme.IsUnLock = true
	}
	color := findOutfitDyeColor(scheme.Colors, req.Pos)
	if color == nil {
		color = &model.PosColor{Pos: req.Pos}
	}
	rsp.PosColor = &proto.PosColor{Pos: color.Pos, Red: color.Red, Green: color.Green, Blue: color.Blue}
}

func upsertOutfitDyeColor(colors model.Colors, color *proto.PosColor) model.Colors {
	if color == nil {
		return colors
	}
	for _, exist := range colors {
		if exist.Pos == color.Pos {
			exist.Red = color.Red
			exist.Green = color.Green
			exist.Blue = color.Blue
			return colors
		}
	}
	return append(colors, &model.PosColor{
		Pos:   color.Pos,
		Red:   color.Red,
		Green: color.Green,
		Blue:  color.Blue,
	})
}

func findOutfitDyeColor(colors model.Colors, pos uint32) *model.PosColor {
	for _, color := range colors {
		if color.Pos == pos {
			return color
		}
	}
	return nil
}

func (g *Game) CharacterEquipUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterEquipUpdateReq)
	rsp := &proto.CharacterEquipUpdateRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		Character: make([]*proto.Character, 0),
		Items:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	changedCharacterIds := applyCharacterEquipment(s, req, rsp)
	if rsp.Status == proto.StatusCode_StatusCode_OK {
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UPDATE_EQUIP, changedCharacterIds...)
	}
}

func (g *Game) UpdateCharacterAppearance(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UpdateCharacterAppearanceReq)
	rsp := &proto.UpdateCharacterAppearanceRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		CharId:     req.GetCharId(),
		Appearance: nil,
	}
	updated := false
	defer func() {
		g.send(s, msg.PacketId, rsp)
		if updated {
			g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UPDATE_APPEARANCE, req.GetCharId())
		}
	}()
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	defer tx.Rollback()
	appearance, ok := validateCharacterAppearance(s, req)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		log.Game.Warnf("保存角色外观失败,角色%v或外观实例不合法", req.GetCharId())
		return
	}
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.GetCharId())
	characterInfo.CharacterAppearance = appearance
	rsp.Appearance = characterInfo.GetPbCharacterAppearance()
	tx.Commit()
	updated = true

	// 更新基础信息
	if team := s.GetTeamModel().GetTeamInfo(); team != nil && req.GetCharId() == team.Char1 {
		if err := s.UpBasicByTeam(); err != nil {
			log.Game.Errorf("UserId:%v UpBasicByTeam err:%s", s.UserId, err.Error())
		}
	}
}

func (g *Game) CharacterGatherWeaponUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterGatherWeaponUpdateReq)
	rsp := &proto.CharacterGatherWeaponUpdateRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	updated := false
	defer func() {
		g.send(s, msg.PacketId, rsp)
		if updated {
			g.SceneActionCharacterUpdate(
				s, proto.SceneActionType_SceneActionType_UPDATE_EQUIP, req.GetCharacterId())
		}
	}()
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	defer tx.Rollback()
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.GetCharacterId())
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		log.Game.Warnf("玩家:%v 角色:%v 不存在", s.UserId, req.GetCharacterId())
		return
	}
	itemInfo, ok := validGatherWeapon(s, req.GetWeaponId())
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if itemInfo == nil {
		characterInfo.GatherWeapon = 0
	} else {
		characterInfo.GatherWeapon = itemInfo.WeaponId
	}
	tx.Commit()
	updated = true
}

func (g *Game) CharacterStarUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterStarUpReq)
	rsp := &proto.CharacterStarUpRsp{
		Status: proto.StatusCode_StatusCode_OK,
		CharId: req.CharId,
		Star:   0,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UPDATE_CHARACTER_STAR, req.CharId)
	}()
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		log.Game.Warnf("保存角色升星失败,角色%v不存在", req.CharId)
		return
	}
	conf := gdconf.GetCharacterStar(characterInfo.CharacterId, characterInfo.Star+1)
	if conf == nil {
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
	tx.DelBaseItem(uint32(conf.ItemID), int64(conf.ItemNum))
	if tx.Error != nil {
		tx.Rollback()
		rsp.Status = proto.StatusCode_StatusCode_EXPLORE_NUM_LIMIT
		log.Game.Errorf("玩家:%v扣除背包物品失败:%s", s.UserId, tx.Error.Error())
		return
	}
	tx.Commit()
	g.send(s, 0, tx.PackNotice)

	characterInfo.Star++
	rsp.Star = characterInfo.Star
}

func (g *Game) CharacterSkillLevelUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterSkillLevelUpReq)
	rsp := &proto.CharacterSkillLevelUpRsp{
		Status: proto.StatusCode_StatusCode_OK,
		CharId: req.CharId,
		Items:  make([]*proto.ItemDetail, 0),
		Skill:  nil,
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		log.Game.Warnf("保存角色技能升级失败,角色%v不存在", req.CharId)
		return
	}
	skillInfo, ok := characterInfo.CharacterSkillList[req.SkillId]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		return
	}
	conf := gdconf.GetSpellLevelUpInfoBySkillId(req.SkillId, skillInfo.SkillLevel)
	if conf == nil {
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

	for _, item := range conf.SpellLevelUpInfoItem {
		for index, itemId := range item.ItemID {
			tx.DelBaseItem(uint32(itemId), int64(item.Num[index]))
			if tx.Error != nil {
				tx.Rollback()
				rsp.Status = proto.StatusCode_StatusCode_EXPLORE_NUM_LIMIT
				log.Game.Errorf("玩家:%v扣除背包物品失败:%s", s.UserId, tx.Error.Error())
				return
			}
		}
	}

	tx.Commit()
	g.send(s, 0, tx.PackNotice)

	skillInfo.SkillLevel++
	rsp.Skill = skillInfo.CharacterSkill()
}

func (g *Game) GetCharacterAchievementList(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCharacterAchievementListReq)
	rsp := &proto.GetCharacterAchievementListRsp{
		Status:                  proto.StatusCode_StatusCode_OK,
		CharacterAchievementLst: make([]*proto.Achieve, 0),
		HasRewardedIds:          make([]uint32, 0), // ok
		IsUnlockedPayment:       false,             // ok
		CharacterId:             req.CharacterId,   // ok
		RewardedIdLst:           make([]uint32, 0), // ok
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharacterId)
	if characterInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		log.Game.Warnf("获取角色成就列表失败,角色%v不存在", req.CharacterId)
		return
	}
	rsp.IsUnlockedPayment = characterInfo.IsUnlockPayment

	achievement := characterInfo.GetCharacterAchievement()
	rsp.HasRewardedIds = achievement.HasRewardedIds
	rsp.RewardedIdLst = achievement.RewardedIndexLst
}

func (g *Game) GetCharacterAchievementAward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCharacterAchievementAwardReq)
	rsp := &proto.GetCharacterAchievementAwardRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		Items:            make([]*proto.ItemDetail, 0),
		RewardedIndexLst: make([]uint32, 0),
		CharacterId:      req.CharacterId,
	}
	defer g.send(s, msg.PacketId, rsp)
	conf := gdconf.GetAchieveRewardInfo(req.CharacterId, req.RewardIndex)
	achievement := s.GetCharacterModel().GetCharacterInfo(req.CharacterId).GetCharacterAchievement()
	if conf == nil || achievement == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		log.Game.Warnf("获取角色成就列表失败,角色%v不存在", req.CharacterId)
		return
	}
	if int64(conf.UnlockItemCount) > s.GetItemModel().GetItemBaseInfo(uint32(conf.UnlockItemID)).Num {
		rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
		return
	}
	// 领取奖励
	item := s.AddAllTypeItem(uint32(conf.RewardItemID), int64(conf.RewardItemCount))
	if item != nil {
		alg.AddList(&rsp.Items, item.AddItemDetail())
		alg.AddSlice(&achievement.RewardedIndexLst, req.RewardIndex)
	}
	rsp.RewardedIndexLst = achievement.RewardedIndexLst
}
