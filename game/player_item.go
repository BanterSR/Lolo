package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) PackNotice(s *model.Player) {
	notice := &proto.PackNotice{
		Status:          proto.StatusCode_StatusCode_OK,
		Items:           make([]*proto.ItemDetail, 0),
		TempPackMaxSize: 30,
		IsClearTempPack: false,
	}
	defer g.send(s, cmd.PackNotice, 0, notice)
	// 基础物品
	for _, v := range s.GetItemModel().GetItemBaseMap() {
		notice.Items = append(notice.Items, v.GetPbItemDetail())
	}
	// 服装
	for _, v := range s.GetItemModel().GetItemFashionMap() {
		notice.Items = append(notice.Items, v.GetPbItemDetail())
	}
	// 武器
	for _, v := range s.GetItemModel().GetItemWeaponMap() {
		notice.Items = append(notice.Items, v.GetPbItemDetail())
	}
	// 盔甲
	for _, v := range s.GetItemModel().GetItemArmorMap() {
		notice.Items = append(notice.Items, v.GetPbItemDetail())
	}
	// 海报
	for _, v := range s.GetItemModel().GetItemPosterMap() {
		notice.Items = append(notice.Items, v.GetPbItemDetail())
	}
	//
	for _, v := range s.GetItemModel().GetItemInscriptionMap() {
		notice.Items = append(notice.Items, v.GetPbItemDetail())
	}
}

func (g *Game) GetWeapon(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetWeaponReq)
	rsp := &proto.GetWeaponRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		Weapons:  make([]*proto.WeaponInstance, 0),
		TotalNum: uint32(len(s.GetItemModel().GetItemWeaponMap())),
		EndIndex: uint32(len(s.GetItemModel().GetItemWeaponMap())),
	}
	defer g.send(s, cmd.GetWeaponRsp, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemWeaponMap() {
		if req.WeaponSystemType == proto.EWeaponSystemType_EWeaponSystemType_None ||
			req.WeaponSystemType == v.WeaponSystemType {
			rsp.Weapons = append(rsp.Weapons, v.GetPbWeaponInstance())
		}
	}
}

func (g *Game) GetArmor(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetArmorReq)
	rsp := &proto.GetArmorRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		Armors:   make([]*proto.ArmorInstance, 0),
		TotalNum: uint32(len(s.GetItemModel().GetItemArmorMap())),
		EndIndex: uint32(len(s.GetItemModel().GetItemArmorMap())),
	}
	defer g.send(s, cmd.GetArmorRsp, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemArmorMap() {
		if req.WeaponSystemType == proto.EWeaponSystemType_EWeaponSystemType_None ||
			req.WeaponSystemType == v.WeaponSystemType {
			rsp.Armors = append(rsp.Armors, v.GetPbArmorInstance())
		}
	}
}

func (g *Game) GetPoster(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetPosterReq)
	rsp := &proto.GetPosterRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		Posters:  make([]*proto.PosterInstance, 0),
		TotalNum: uint32(len(s.GetItemModel().GetItemPosterMap())),
		EndIndex: uint32(len(s.GetItemModel().GetItemPosterMap())),
	}
	defer g.send(s, cmd.GetPosterRsp, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemPosterMap() {
		alg.AddList(&rsp.Posters, v.GetPbPosterInstance())
	}
}

func (g *Game) PosterIllustrationList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PosterIllustrationListRsp{
		Status:              proto.StatusCode_StatusCode_OK,
		PosterIllustrations: make([]*proto.PosterIllustration, 0),
	}
	defer g.send(s, cmd.PosterIllustrationListRsp, msg.PacketId, rsp)
	for _, v := range s.GetItemModel().GetItemPosterMap() {
		alg.AddList(&rsp.PosterIllustrations, &proto.PosterIllustration{
			PosterIllustrationId: v.PosterId,
			Status:               proto.RewardStatus_Reward,
		})
	}
}
