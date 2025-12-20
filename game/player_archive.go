package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetArchiveInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetArchiveInfoReq)
	rsp := &proto.GetArchiveInfoRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Key:    req.Key,
		Value:  s.GetArchive().GetArchiveValue(req.Key),
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) SetArchiveInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SetArchiveInfoReq)
	rsp := &proto.SetArchiveInfoRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Key:    req.Key,
		Value:  req.Value,
	}
	defer g.send(s, msg.PacketId, rsp)
	s.GetArchive().SetArchiveMap(req.Key, req.Value)
}
