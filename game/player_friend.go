package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) Friend(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.FriendReq)
	rsp := &proto.FriendRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Info:   make([]*proto.FriendBriefInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) WishListByFriendId(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.FriendReq)
	rsp := &proto.WishListByFriendIdRsp{
		Status:        proto.StatusCode_StatusCode_OK,
		PlayerId:      0,
		WishList:      make([]*proto.WishListInfo, 0),
		WeekSendCount: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ChallengeFriendRank(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.ChallengeFriendRankReq)
	rsp := &proto.ChallengeFriendRankRsp{
		Status:   proto.StatusCode_StatusCode_OK,
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
		Status:      proto.StatusCode_StatusCode_OK,
		FriendInfos: make([]*proto.IntervalInfo, 0),
		JoinInfos:   make([]*proto.IntervalInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
