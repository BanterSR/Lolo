package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) PrivateChatOfflineNotice(s *model.Player) {
	notice := &proto.PrivateChatOfflineNotice{
		Status:     proto.StatusCode_StatusCode_OK,
		OfflineMsg: make([]*proto.PrivateChatOffline, 0),
	}
	defer g.send(s, 0, notice)
}

func (g *Game) ChatMsgRecordInitNotice(s *model.Player) {
	notice := &proto.ChatMsgRecordInitNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Type:   0,
		Msg:    make([]*proto.ChatMsgData, 0),
	}
	defer g.send(s, 0, notice)
}
