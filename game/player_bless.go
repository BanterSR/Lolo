package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) BlessTreeNotice(s *model.Player) {
	notice := &proto.BlessTreeNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Tress:  make(map[uint32]*proto.PlayerBlessTreeInfo),
	}
	defer g.send(s, 0, notice)
}
