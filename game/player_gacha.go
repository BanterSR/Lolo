package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GachaList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GachaListRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Gachas: make([]*proto.GachaInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, v := range gdconf.GetAllGacha().GetInfo().GetDatas() {
		info := &proto.GachaInfo{
			GachaId:        uint32(v.ID),
			GachaTimes:     1,
			HasFullPick:    false,
			IsFree:         false,
			OptionalUpItem: 0,
			OptionalValue:  0,
			Guarantee:      0,
		}
		alg.AddList(&rsp.Gachas, info)
	}
}

func (g *Game) GachaRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GachaRecordReq)
	rsp := &proto.GachaRecordRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		GachaId:   req.GachaId,
		Page:      req.Page,
		TotalPage: 0,
		Records:   make([]*proto.PlayerGachaRecord, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
