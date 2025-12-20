package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) AllPackNotice(s *model.Player) {
	notice := &proto.PackNotice{
		Status:          proto.StatusCode_StatusCode_Ok,
		Items:           make([]*proto.ItemDetail, 0),
		TempPackMaxSize: 30,
		IsClearTempPack: false,
	}
	defer g.send(s, 0, notice)
	// 基础物品
	for _, v := range s.GetItemModel().GetItemBaseMap() {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	// 服装
	for _, v := range s.GetItemModel().GetItemFashionMap() {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	// 武器
	for _, v := range s.GetItemModel().GetItemWeaponMap() {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	// 盔甲
	for _, v := range s.GetItemModel().GetItemArmorMap() {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	// 海报
	for _, v := range s.GetItemModel().GetItemPosterMap() {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
	//
	for _, v := range s.GetItemModel().GetItemInscriptionMap() {
		notice.Items = append(notice.Items, v.ItemDetail())
	}
}

func (g *Game) GetWeapon(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetWeaponReq)
	rsp := &proto.GetWeaponRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Weapons:  make([]*proto.WeaponInstance, 0),
		TotalNum: uint32(len(s.GetItemModel().GetItemWeaponMap())),
		EndIndex: uint32(len(s.GetItemModel().GetItemWeaponMap())),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemWeaponMap() {
		if req.WeaponSystemType == proto.EWeaponSystemType_EWeaponSystemType_None ||
			req.WeaponSystemType == v.WeaponSystemType {
			rsp.Weapons = append(rsp.Weapons, v.WeaponInstance())
		}
	}
}

func (g *Game) GetArmor(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetArmorReq)
	rsp := &proto.GetArmorRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Armors:   make([]*proto.ArmorInstance, 0),
		TotalNum: uint32(len(s.GetItemModel().GetItemArmorMap())),
		EndIndex: uint32(len(s.GetItemModel().GetItemArmorMap())),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemArmorMap() {
		if req.WeaponSystemType == proto.EWeaponSystemType_EWeaponSystemType_None ||
			req.WeaponSystemType == v.WeaponSystemType {
			rsp.Armors = append(rsp.Armors, v.ArmorInstance())
		}
	}
}

func (g *Game) GetPoster(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetPosterReq)
	rsp := &proto.GetPosterRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Posters:  make([]*proto.PosterInstance, 0),
		TotalNum: uint32(len(s.GetItemModel().GetItemPosterMap())),
		EndIndex: uint32(len(s.GetItemModel().GetItemPosterMap())),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemPosterMap() {
		alg.AddList(&rsp.Posters, v.PosterInstance())
	}
}

func (g *Game) PosterIllustrationList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PosterIllustrationListRsp{
		Status:              proto.StatusCode_StatusCode_Ok,
		PosterIllustrations: make([]*proto.PosterIllustration, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemPosterMap() {
		alg.AddList(&rsp.PosterIllustrations, &proto.PosterIllustration{
			PosterIllustrationId: v.PosterId,
			Status:               proto.RewardStatus_RewardStatus_Reward,
		})
	}
}
