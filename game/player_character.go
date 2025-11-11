package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetAllCharacterEquip(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GetAllCharacterEquipRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, cmd.GetAllCharacterEquipRsp, msg.PacketId, rsp)
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
	defer g.send(s, cmd.GetCharacterAchievementListRsp, msg.PacketId, rsp)
}

func (g *Game) OutfitPresetUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitPresetUpdateReq)
	rsp := &proto.OutfitPresetUpdateRsp{
		Status: proto.StatusCode_StatusCode_OK,
		CharId: req.CharId,
		Preset: req.Preset,
	}
	defer func() {
		g.send(s, cmd.OutfitPresetUpdateRsp, msg.PacketId, rsp)
		teamInfo := s.GetTeamInfo()
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
	characterInfo := s.GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色预设装扮失败,角色%v不存在", req.CharId)
		return
	}
	outfitPreset := s.GetOutfitPreset(characterInfo, req.Preset.PresetIndex)

	outfitPreset.Hair = req.Preset.Hair
	outfitPreset.Hair = req.Preset.Hair
	outfitPreset.Clothes = req.Preset.Clothes
	outfitPreset.Ornament = req.Preset.Ornament
	outfitPreset.HatDyeSchemeIndex = req.Preset.HatDyeSchemeIndex
	outfitPreset.HairDyeSchemeIndex = req.Preset.HairDyeSchemeIndex
	outfitPreset.ClothesDyeSchemeIndex = req.Preset.ClothesDyeSchemeIndex
	outfitPreset.OrnamentDyeSchemeIndex = req.Preset.OrnamentDyeSchemeIndex
	outfitPreset.OutfitHideInfo = &model.OutfitHideInfo{
		HideOrn:   req.Preset.OutfitHideInfo.HideOrn,
		HideBraid: req.Preset.OutfitHideInfo.HideBraid,
	}
}

func (g *Game) CharacterEquipUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CharacterEquipUpdateReq)
	rsp := &proto.CharacterEquipUpdateRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		Character: make([]*proto.Character, 0),
		Items:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, cmd.CharacterEquipUpdateRsp, msg.PacketId, rsp)
	characterInfo := s.GetCharacterInfo(req.CharId)
	if characterInfo == nil {
		log.Game.Warnf("保存角色装备失败,角色%v不存在", req.CharId)
		return
	}
	defer alg.AddList(&rsp.Character, s.GetPbCharacter(characterInfo))

	equipmentPreset := s.GetEquipmentPreset(characterInfo, req.EquipmentPreset.PresetIndex)
	// 更新武器
	if req.EquipmentPreset.Weapon != equipmentPreset.Weapon {
		oldEquipmentInfo := s.GetItemModel().GetItemWeaponInfo(equipmentPreset.Weapon)
		newEquipmentInfo := s.GetItemModel().GetItemWeaponInfo(req.EquipmentPreset.Weapon)
		if newEquipmentInfo != nil &&
			oldEquipmentInfo != nil {
			oldEquipmentInfo.WearerId = 0
			alg.AddList(&rsp.Items, oldEquipmentInfo.GetPbItemDetail())

			if oldCharacterInfo := s.GetCharacterInfo(newEquipmentInfo.WearerId); oldCharacterInfo != nil {
				// 移除装备上的角色
			}
			newEquipmentInfo.WearerId = req.CharId
			equipmentPreset.Weapon = newEquipmentInfo.InstanceId
			alg.AddList(&rsp.Items, newEquipmentInfo.GetPbItemDetail())
		}
	}
	// 更新盔甲
	// 更新海报
}
