package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetMails(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetMailsReq)
	rsp := &proto.GetMailsRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Mails:  make([]*proto.MailBriefData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
