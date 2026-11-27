package game

import (
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func equipmentFailure(rspStatus *proto.StatusCode, err error) bool {
	if err == nil {
		return false
	}
	*rspStatus = model.EquipmentErrorStatus(err)
	log.Game.Warnf("装备操作失败:%s", err.Error())
	return true
}

func (g *Game) CharacterEquipPresetSwitch(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.CharacterEquipPresetSwitchReq)
	rsp := &proto.CharacterEquipPresetSwitchRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		PresetIndex: reqPresetIndex(req),
		Items:       make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.PresetIndex = req.PresetIndex
	if err := s.SwitchEquipmentPreset(req.CharId, req.PresetIndex); err != nil {
		equipmentFailure(&rsp.Status, err)
		return
	}
	g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UPDATE_EQUIP, req.CharId)
}

func reqPresetIndex(req *proto.CharacterEquipPresetSwitchReq) uint32 {
	if req == nil {
		return 0
	}
	return req.PresetIndex
}

func (g *Game) PosterStarUp(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.PosterStarUpReq)
	rsp := &proto.PosterStarUpRsp{
		Status:               proto.StatusCode_StatusCode_OK,
		PosterInstanceId:     posterReqInstance(req),
		CostPosterInstanceId: append([]uint32(nil), posterReqCosts(req)...),
		Character:            make([]*proto.Character, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	costIDs, characterIDs, err := s.StarUpPoster(req.PosterInstanceId, req.CostPosterInstanceId)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.CostPosterInstanceId = costIDs
	poster := s.GetItemModel().GetItemPosterInfo(req.PosterInstanceId)
	if poster != nil {
		rsp.Star = poster.Star
		g.PackNoticeByItems(s, []*proto.ItemDetail{poster.ItemDetail()})
	}
	seen := make(map[uint32]struct{}, len(characterIDs))
	for _, id := range characterIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if character := s.GetCharacterModel().GetCharacterInfo(id); character != nil {
			rsp.Character = append(rsp.Character, character.Character())
		}
	}
}

func posterReqInstance(req *proto.PosterStarUpReq) uint32 {
	if req == nil {
		return 0
	}
	return req.PosterInstanceId
}

func posterReqCosts(req *proto.PosterStarUpReq) []uint32 {
	if req == nil {
		return nil
	}
	return req.CostPosterInstanceId
}

func (g *Game) PosterIllustrationReward(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.PosterIllustrationRewardReq)
	rsp := &proto.PosterIllustrationRewardRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	entry, err := s.ClaimPosterIllustration(req.PosterIllustrationId)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.PosterIllustration = &proto.PosterIllustration{
		PosterIllustrationId: entry.PosterIllustrationId,
		Status:               entry.RewardStatus,
	}
}

func (g *Game) LockEquip(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.LockEquipReq)
	rsp := &proto.LockEquipRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	item, err := s.ToggleEquipmentLock(req.InstanceId, req.EquipType)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	g.PackNoticeByItems(s, []*proto.ItemDetail{item})
}

func (g *Game) WeaponInscriptionInlaid(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.WeaponInscriptionInlaidReq)
	rsp := &proto.WeaponInscriptionInlaidRsp{
		Status:          proto.StatusCode_StatusCode_OK,
		WeaponInstances: make([]*proto.WeaponInstance, 0),
		Inscriptions:    make([]*proto.Inscription, 0),
		Items:           make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	weapons, inscriptions, err := s.InscribeWeapon(req.WeaponInstanceId, req.InscriptionId)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.WeaponInstances = weapons
	rsp.Inscriptions = inscriptions
}

func (g *Game) WeaponInscriptionRemove(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.WeaponInscriptionRemoveReq)
	rsp := &proto.WeaponInscriptionRemoveRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		WeaponInstanceId: inscriptionRemoveReqInstance(req),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	weapon, inscription, err := s.RemoveWeaponInscription(req.WeaponInstanceId)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.WeaponInstanceId = weapon.InstanceId
	items := []*proto.ItemDetail{weaponItemDetail(s, weapon.InstanceId)}
	if inscription != nil {
		if info := s.GetItemModel().GetItemInscriptionMap()[inscription.InscriptionId]; info != nil {
			items = append(items, info.ItemDetail())
		}
	}
	g.PackNoticeByItems(s, items)
}

func inscriptionRemoveReqInstance(req *proto.WeaponInscriptionRemoveReq) uint32 {
	if req == nil {
		return 0
	}
	return req.WeaponInstanceId
}

func weaponItemDetail(s *model.Player, instanceID uint32) *proto.ItemDetail {
	if weapon := s.GetItemModel().GetItemWeaponInfo(instanceID); weapon != nil {
		return weapon.ItemDetail()
	}
	return nil
}

func (g *Game) EquipDisassemble(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.EquipDisassembleReq)
	rsp := &proto.EquipDisassembleRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		Items:     make([]*proto.ItemDetail, 0),
		Character: make([]*proto.Character, 0),
		WeaponIds: append([]uint32(nil), disassembleReqWeapons(req)...),
		ArmorIds:  append([]uint32(nil), disassembleReqArmors(req)...),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	items, err := s.DisassembleEquipment(req.WeaponIds, req.ArmorIds)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.Items = items
}

func disassembleReqWeapons(req *proto.EquipDisassembleReq) []uint32 {
	if req == nil {
		return nil
	}
	return req.WeaponIds
}

func disassembleReqArmors(req *proto.EquipDisassembleReq) []uint32 {
	if req == nil {
		return nil
	}
	return req.ArmorIds
}

func (g *Game) ItemUse(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.ItemUseReq)
	rsp := &proto.ItemUseRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	basic, ok := db.GetGameBasic(s.UserId)
	if !ok || basic == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
		return
	}
	result, err := s.UseItem(req.ItemId, req.Num, req.ChooseItemId, basic.Level, time.Now())
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.Items = result.Items
	rsp.DungeonId = result.DungeonID
	if result.PackNotice != nil {
		g.send(s, 0, result.PackNotice)
	}
	if len(result.PlayerBuffs) > 0 {
		g.send(s, 0, &proto.PlayerBuffNotice{
			Status:      proto.StatusCode_StatusCode_OK,
			PlayerBuffs: result.PlayerBuffs,
		})
	}
}

func (g *Game) WeaponStrength(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.WeaponStrengthReq)
	rsp := &proto.WeaponStrengthRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		WeaponInstanceId: weaponStrengthReqInstance(req),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	notice, level, err := s.StrengthenWeapon(req.WeaponInstanceId)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.WeaponInstanceId = req.WeaponInstanceId
	rsp.StrengthLevel = level
	if notice != nil {
		g.send(s, 0, notice)
	}
	if weapon := s.GetItemModel().GetItemWeaponInfo(req.WeaponInstanceId); weapon != nil {
		g.PackNoticeByItems(s, []*proto.ItemDetail{weapon.ItemDetail()})
	}
}

func weaponStrengthReqInstance(req *proto.WeaponStrengthReq) uint32 {
	if req == nil {
		return 0
	}
	return req.WeaponInstanceId
}

func (g *Game) ArmorStrength(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.ArmorStrengthReq)
	rsp := &proto.ArmorStrengthRsp{
		Status:          proto.StatusCode_StatusCode_OK,
		ArmorInstanceId: armorStrengthReqInstance(req),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	notice, level, err := s.StrengthenArmor(req.ArmorInstanceId)
	if equipmentFailure(&rsp.Status, err) {
		return
	}
	rsp.ArmorInstanceId = req.ArmorInstanceId
	rsp.StrengthLevel = level
	if notice != nil {
		g.send(s, 0, notice)
	}
	if armor := s.GetItemModel().GetItemArmorInfo(req.ArmorInstanceId); armor != nil {
		g.PackNoticeByItems(s, []*proto.ItemDetail{armor.ItemDetail()})
	}
}

func armorStrengthReqInstance(req *proto.ArmorStrengthReq) uint32 {
	if req == nil {
		return 0
	}
	return req.ArmorInstanceId
}
