package game

import (
	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) Friend(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendReq)
	rsp := &proto.FriendRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Info:   make([]*proto.FriendBriefInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	switch req.Type {
	case proto.FriendListType_FriendListType_None:
	case proto.FriendListType_FriendListType_Apply:
		applyList, err := db.GetAllFriendApply(s.UserId)
		if err != nil {
			log.Game.Warnf("UserId:%v func db.GetAllFriendApply:%v", s.UserId, err)
			return
		}
		for _, v := range applyList {
			alg.AddList(&rsp.Info, g.GetFriendBriefInfo(v.RequestUserId, nil))
		}
	case proto.FriendListType_FriendListType_Friend:
		allFriend, err := db.GetAllFiend(s.UserId)
		if err != nil {
			log.Game.Warnf("UserId:%v func db.GetAllFriend:%v", s.UserId, err)
			return
		}
		for _, v := range allFriend {
			alg.AddList(&rsp.Info, g.GetFriendBriefInfo(v.FriendId, v))
		}
	case proto.FriendListType_FriendListType_Black:
		allBlack, err := db.GetAllFriendBlack(s.UserId)
		if err != nil {
			log.Game.Warnf("UserId:%v func db.GetAllFriendBlack:%v", s.UserId, err)
		}
		for _, v := range allBlack {
			alg.AddList(&rsp.Info, g.GetFriendBriefInfo(v.BlackId, nil))
		}
	}
}

func (g *Game) FriendAdd(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendAddReq)
	rsp := &proto.FriendAddRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	// 判断是否存在好友关系
	if conn, err := db.GetIsFiend(s.UserId, req.PlayerId); err != nil {
		log.Game.Warnf("UserId:%v db.GetIsFiend:%v", s.UserId, err)
		return
	} else if conn != 0 {
		rsp.Status = proto.StatusCode_StatusCode_FriendAddFail
		return
	}
	// 判断是否已经申请
	if conn, err := db.GetIsFriendApply(req.PlayerId, s.UserId); err != nil {
		log.Game.Warnf("UserId:%v db.GetIsFriendApply:%v", s.UserId, err)
		return
	} else if conn != 0 {
		// 直接同意好友申请
		err = db.FriendHandleApply(s.UserId, req.PlayerId, true)
		return
	}
	// 都没有就写入申请请求
	err := db.CreateFriendApply(s.UserId, req.PlayerId)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_FriendAddFail
		log.Game.Warnf("UserId:%v db.CreateFriendApply:%v", s.UserId, err)
		return
	}
	// 如果在线通知对方
	if friend := g.GetUser(req.PlayerId); friend != nil {
		g.send(friend, 0, &proto.FriendHandleNotice{
			Status:         proto.StatusCode_StatusCode_Ok,
			Type:           proto.FriendHandleType_FriendHandleType_Apply,
			TargetPlayerId: s.UserId,
		})
	}
}

func (g *Game) FriendHandle(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendHandleReq)
	rsp := &proto.FriendHandleRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.FriendHandleApply(s.UserId, req.PlayerId, req.IsAgree)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_FriendNotApply
		log.Game.Warnf("UserId:%v func db.FriendHandleApply:%v", s.UserId, err)
	}
	// 如果在线通知对方
	if friend := g.GetUser(req.PlayerId); friend != nil && req.IsAgree {
		g.send(friend, 0, &proto.FriendHandleNotice{
			Status:         proto.StatusCode_StatusCode_Ok,
			Type:           proto.FriendHandleType_FriendHandleType_Add,
			TargetPlayerId: s.UserId,
		})
	}
}

func (g *Game) FriendDel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendDelReq)
	rsp := &proto.FriendDelRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	// 直接删除好友关系
	err := db.DelFiend(s.UserId, req.PlayerId)
	if err != nil {
		log.Game.Warnf("UserId:%v db.DelFiend:%v", s.UserId, err)
	}
	// 如果在线通知对方
	if friend := g.GetUser(req.PlayerId); friend != nil {
		g.send(friend, 0, &proto.FriendHandleNotice{
			Status:         proto.StatusCode_StatusCode_Ok,
			Type:           proto.FriendHandleType_FriendHandleType_Del,
			TargetPlayerId: s.UserId,
		})
	}
}

func (g *Game) FriendBlack(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendBlackReq)
	rsp := &proto.FriendBlackRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.CreateFriendBlack(s.UserId, req.PlayerId, req.IsRemove)
	if err != nil {
		log.Game.Warnf("UserId:%v db.CreateFriendBlack:%v", s.UserId, err)
	}
}

func (g *Game) OtherPlayerInfo(s *model.Player, msg *alg.GameMsg) {
	//req := msg.Body.(*proto.OtherPlayerInfoReq)
	//rsp := &proto.OtherPlayerInfoRsp{
	//	Status:           proto.StatusCode_StatusCode_Ok,
	//	OtherInfo:        nil,
	//	FriendStatus:     0,
	//	Alias:            "",
	//	FriendTag:        0,
	//	FriendIntimacy:   0,
	//	FriendBackground: 0,
	//}
	//defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) FriendSearch(s *model.Player, msg *alg.GameMsg) {
	//req := msg.Body.(*proto.FriendSearchReq)
	//rsp := &proto.FriendSearchRsp{
	//	Status:       proto.StatusCode_StatusCode_Ok,
	//	Data:         nil,
	//	FriendStatus: 0,
	//}
	//defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) WishListByFriendId(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.WishListByFriendIdReq)
	rsp := &proto.WishListByFriendIdRsp{
		Status:        proto.StatusCode_StatusCode_Ok,
		PlayerId:      0,
		WishList:      make([]*proto.WishListInfo, 0),
		WeekSendCount: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ChallengeFriendRank(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.ChallengeFriendRankReq)
	rsp := &proto.ChallengeFriendRankRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		RankInfo: make([]*proto.ChallengeFriendRankInfo, 0),
		SelfChallenge: &proto.PlayerChallengeCache{
			PlayerId:       s.UserId,
			ChallengeInfos: make([]*proto.PlayerChallengeInfo, 0),
		},
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) FriendIntervalInit(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.FriendIntervalInitReq)
	rsp := &proto.FriendIntervalInitRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		FriendInfos: make([]*proto.IntervalInfo, 0),
		JoinInfos:   make([]*proto.IntervalInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
