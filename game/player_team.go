package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) UpdateTeam(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UpdateTeamReq)
	rsp := &proto.UpdateTeamRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(
			s, proto.SceneActionType_SceneActionType_UpdateTeam, req.Char1)
	}()

	// 更新队伍
	upChar := func(target *uint32, char uint32) bool {
		*target = char
		return true
	}
	teamInfo := s.GetTeamModel().GetTeamInfo()
	upChar(&teamInfo.Char1, req.Char1)
	upChar(&teamInfo.Char2, req.Char2)
	upChar(&teamInfo.Char3, req.Char3)
}
