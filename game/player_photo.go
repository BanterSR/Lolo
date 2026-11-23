package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) PhotoShareSearch(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.PhotoShareSearchReq)
	rsp := &proto.PhotoShareSearchRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		Photos:           make([]*proto.PhotoPreviewInfo, 0),
		TotalNum:         0,
		EndIndex:         0,
		TodayUploadLimit: 100, // 今日上传限制
		ActivityPhotos:   make([]*proto.PhotoPreviewInfo, 0),
		UploadMaxNum:     100, // 总上传限制
	}
	defer g.send(s, msg.PacketId, rsp)
}
