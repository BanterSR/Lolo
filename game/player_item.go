package game

import (
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

	notice := &proto.PackNotice{
		Status:          proto.StatusCode_StatusCode_Ok,
		Items:           make([]*proto.ItemDetail, 0, len(baseMap)+len(fashionMap)+len(weaponMap)+len(armorMap)+len(posterMap)+len(inscriptionMap)),
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
	posterMap := s.GetItemModel().GetItemPosterMap()
	rsp := &proto.PosterIllustrationListRsp{
		Status:              proto.StatusCode_StatusCode_Ok,
		PosterIllustrations: make([]*proto.PosterIllustration, 0, len(posterMap)),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range posterMap {
		alg.AddList(&rsp.PosterIllustrations, &proto.PosterIllustration{
			PosterIllustrationId: v.PosterId,
			Status:               proto.RewardStatus_RewardStatus_Reward,
		})
	}
}
