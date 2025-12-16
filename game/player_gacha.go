package game

import (
	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GachaList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GachaListRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Gachas: make([]*proto.GachaInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	list := s.GetGachaModel().GetGachaMap()
	for _, v := range gdconf.GetOpenGachas() {
		info, ok := list[uint32(v.Conf.ID)]
		if !ok {
			alg.AddList(&rsp.Gachas,
				model.DefaultGachaInfo(uint32(v.Conf.ID)).GachaInfo())
		} else {
			alg.AddList(&rsp.Gachas, info.GachaInfo())
		}
	}
}

func (g *Game) GachaRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GachaRecordReq)
	rsp := &proto.GachaRecordRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		GachaId:   req.GachaId,
		Page:      req.Page, // 当前页
		TotalPage: 0,        // 总页
		Records:   nil,
	}
	defer g.send(s, msg.PacketId, rsp)
	list, totalPage, err := db.GetGachaRecords(s.UserId, req.GachaId, req.Page)
	if err != nil {
		log.Game.Errorf("UserId:%v func db.GetGachaRecords err:%v", s.UserId, err)
		return
	}
	rsp.TotalPage = totalPage
	rsp.Records = make([]*proto.PlayerGachaRecord, len(list))
	for index, info := range list {
		rsp.Records[index] = info.PlayerGachaRecord()
	}
}

func (g *Game) Gacha(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GachaReq)
	rsp := &proto.GachaRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  nil,
		Info:   nil,
	}
	defer func() {
		rsp.Info = s.GetGachaModel().GetGachaInfo(req.GachaId).GachaInfo()
		g.send(s, msg.PacketId, rsp)
	}()
	ctx, err := s.NewGachaCtx(req)
	if err != nil {
		log.Game.Errorf("NewGachaCtx err:%v", err)
		return
	}
	ctx.Run()
	rsp.Items = ctx.ItemDetails
}
