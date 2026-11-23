package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetMails(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetMailsReq)
	rsp := &proto.GetMailsRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Mails:  make([]*proto.MailBriefData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) OperateMails(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OperateMailsReq)
	rsp := &proto.OperateMailsRsp{
		Status:         proto.StatusCode_StatusCode_OK,
		Mails:          make([]*proto.MailBriefData, 0),
		ClaimResult:    make([]*proto.ItemDetail, 0),
		DeletedMailIds: make([]uint32, 0),
		OperateType:    req.OperateType,
		CollectStatus:  false,
	}
	defer g.send(s, msg.PacketId, rsp)
}
