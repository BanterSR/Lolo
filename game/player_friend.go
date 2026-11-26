package game

import (
	"errors"
	"strconv"
	"strings"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) Friend(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendReq)
	rsp := &proto.FriendRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Info:   make([]*proto.FriendBriefInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	var friendStatus proto.FriendStatus
	switch req.Type {
	case proto.FriendListType_FriendListType_NONE:
		friendStatus = proto.FriendStatus_FriendStatus_NONE
	case proto.FriendListType_FriendListType_APPLY:
		friendStatus = proto.FriendStatus_FriendStatus_APPLY
	case proto.FriendListType_FriendListType_FRIEND:
		friendStatus = proto.FriendStatus_FriendStatus_FRIEND
	case proto.FriendListType_FriendListType_BLACK:
		friendStatus = proto.FriendStatus_FriendStatus_BLACK
	}

	var err error
	if friendStatus == proto.FriendStatus_FriendStatus_BLACK {
		var blacks []*db.OFFriendBlack
		blacks, err = db.GetAllFriendBlack(s.UserId)
		if err == nil {
			for _, black := range blacks {
				alg.AddList(&rsp.Info, g.GetFriendBriefInfo(black.BlackId, &db.OFFriend{
					UserId:   s.UserId,
					FriendId: black.BlackId,
					Status:   proto.FriendStatus_FriendStatus_BLACK,
				}))
			}
		}
	} else {
		var friendList []*db.OFFriend
		friendList, err = db.GetAllFriendByStatus(s.UserId, friendStatus)
		if err == nil {
			for _, v := range friendList {
				alg.AddList(&rsp.Info, g.GetFriendBriefInfo(v.FriendId, v))
			}
		}
	}
	if err != nil {
		log.Game.Warnf("UserId:%v func friend list:%v", s.UserId, err)
	}
}

func (g *Game) FriendAdd(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendAddReq)
	rsp := &proto.FriendAddRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	defer g.send(s, msg.PacketId, rsp)
	if req.PlayerId == 0 || req.PlayerId == s.UserId {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		return
	}
	// 任意一方拉黑都会阻止申请和自动接受。
	if ok, err := db.IsFriendBlocked(s.UserId, req.PlayerId); err != nil {
		log.Game.Warnf("UserId:%v db.IsFriendBlocked:%v", s.UserId, err)
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		return
	} else if ok {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_BLACK
		return
	}
	// 判断是否存在一致的双向好友关系。
	if conn, err := db.GetIsFiend(s.UserId, req.PlayerId); err != nil {
		log.Game.Warnf("UserId:%v db.GetIsFiend:%v", s.UserId, err)
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		return
	} else if conn != 0 {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		return
	}
	// 重复申请不写重复行，也不重复发送通知。
	if conn, err := db.GetIsFriendApply(s.UserId, req.PlayerId); err != nil {
		log.Game.Warnf("UserId:%v db.GetIsFriendApply:%v", s.UserId, err)
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		return
	} else if conn != 0 {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_APPLY_EXIST
		return
	}
	// 对方已经申请自己时直接接受，并由数据库再次检查黑名单。
	if conn, err := db.GetIsFriendApply(req.PlayerId, s.UserId); err != nil {
		log.Game.Warnf("UserId:%v db.GetIsFriendApply:%v", s.UserId, err)
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		return
	} else if conn != 0 {
		if err = db.FriendHandleApply(s.UserId, req.PlayerId, true); err != nil {
			rsp.Status = friendErrorStatus(err)
			log.Game.Warnf("UserId:%v db.FriendHandleApply:%v", s.UserId, err)
			return
		}
		if friend := g.GetUser(req.PlayerId); friend != nil {
			g.send(friend, 0, &proto.FriendHandleNotice{
				Status:         proto.StatusCode_StatusCode_OK,
				Type:           proto.FriendHandleType_FriendHandleType_ADD,
				TargetPlayerId: s.UserId,
			})
		}
		return
	}
	if err := db.CreateFriendApply(s.UserId, req.PlayerId); err != nil {
		rsp.Status = friendErrorStatus(err)
		log.Game.Warnf("UserId:%v db.CreateFriendApply:%v", s.UserId, err)
		return
	}
	// 如果在线通知对方
	if friend := g.GetUser(req.PlayerId); friend != nil {
		g.send(friend, 0, &proto.FriendHandleNotice{
			Status:         proto.StatusCode_StatusCode_OK,
			Type:           proto.FriendHandleType_FriendHandleType_APPLY,
			TargetPlayerId: s.UserId,
		})
	}
}

func friendErrorStatus(err error) proto.StatusCode {
	switch {
	case errors.Is(err, db.ErrFriendBlack):
		return proto.StatusCode_StatusCode_FRIEND_BLACK
	case errors.Is(err, db.ErrFriendNotApply):
		return proto.StatusCode_StatusCode_FRIEND_NOT_APPLY
	default:
		return proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
	}
}

func (g *Game) FriendHandle(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendHandleReq)
	rsp := &proto.FriendHandleRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.FriendHandleApply(req.PlayerId, s.UserId, req.IsAgree)
	if err != nil {
		rsp.Status = friendErrorStatus(err)
		log.Game.Warnf("UserId:%v func db.FriendHandleApply:%v", s.UserId, err)
		return
	}
	// 只有接受成功后才通知对方。
	if friend := g.GetUser(req.PlayerId); friend != nil && req.IsAgree {
		g.send(friend, 0, &proto.FriendHandleNotice{
			Status:         proto.StatusCode_StatusCode_OK,
			Type:           proto.FriendHandleType_FriendHandleType_ADD,
			TargetPlayerId: s.UserId,
		})
	}
}

func (g *Game) FriendDel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendDelReq)
	rsp := &proto.FriendDelRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	defer g.send(s, msg.PacketId, rsp)
	// 直接删除好友关系
	err := db.DelFiend(s.UserId, req.PlayerId)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_NOT_EXIST
		log.Game.Warnf("UserId:%v db.DelFiend:%v", s.UserId, err)
		return
	}
	// 如果在线通知对方
	if friend := g.GetUser(req.PlayerId); friend != nil {
		g.send(friend, 0, &proto.FriendHandleNotice{
			Status:         proto.StatusCode_StatusCode_OK,
			Type:           proto.FriendHandleType_FriendHandleType_DEL,
			TargetPlayerId: s.UserId,
		})
	}
}

func (g *Game) FriendBlack(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendBlackReq)
	rsp := &proto.FriendBlackRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.CreateFriendBlack(s.UserId, req.PlayerId, req.IsRemove)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_ADD_FAIL
		log.Game.Warnf("UserId:%v db.CreateFriendBlack:%v", s.UserId, err)
	}
}

func (g *Game) OtherPlayerInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OtherPlayerInfoReq)
	rsp := &proto.OtherPlayerInfoRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		OtherInfo:        nil,
		FriendStatus:     0,
		Alias:            "",
		FriendTag:        0,
		FriendIntimacy:   0,
		FriendBackground: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	friend, err := db.GetFiend(s.UserId, req.PlayerId)
	if err != nil {
		log.Game.Warnf("UserId:%v db.GetFiend:%v", s.UserId, err)
		return
	}
	basic, ok := db.GetGameBasic(req.PlayerId)
	if !ok {
		log.Game.Warnf("GetGameBasic:%v func OtherPlayerInfo 玩家不存在", req.PlayerId)
		return
	}
	rsp.OtherInfo = g.PlayerBriefInfo(basic)
	if friend != nil {
		rsp.FriendStatus = friend.Status
		rsp.Alias = friend.Alias
		rsp.FriendTag = friend.FriendTag
		rsp.FriendIntimacy = friend.FriendIntimacy
		rsp.FriendBackground = friend.FriendBackground
	}
}

func (g *Game) FriendSearch(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FriendSearchReq)
	rsp := &proto.FriendSearchRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		Data:         nil,
		FriendStatus: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	userId := alg.S2U32(req.SearchArgs)
	friend, err := db.GetFiend(s.UserId, userId)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_NOT_EXIST
		log.Game.Warnf("UserId:%v db.GetFiend:%v", s.UserId, err)
		return
	}
	basic, ok := db.GetGameBasic(userId)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_NOT_EXIST
		log.Game.Warnf("GetGameBasic:%v func FriendSearch 玩家不存在", userId)
		return
	}
	rsp.Data = g.PlayerBriefInfo(basic)
	if friend != nil {
		rsp.FriendStatus = friend.Status
	}
}

func (g *Game) SetFriendExtInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SetFriendExtInfoReq)
	rsp := &proto.SetFriendExtInfoRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		PlayerId: req.PlayerId,
		Data:     make([]*proto.SetFriendExtInfoData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.UpOFFriend(s.UserId, req.PlayerId, func(off *db.OFFriend) bool {
		for _, set := range req.Data {
			switch set.Type {
			case 1: // 修改备注
				off.Alias = set.Value
			default:
				log.Game.Warnf("请将本行日志提交给开发人员 SetFriendExtInfo Type:%v", set.Type)
				continue
			}
			alg.AddList(&rsp.Data, set)
		}
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_FRIEND_DAYS_NOT_ENOUGH
		log.Game.Warnf("UserId:%v db.UpOFFriend:%v", s.UserId, err)
		return
	}

}

func (g *Game) WishListByFriendId(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.WishListByFriendIdReq)
	rsp := &proto.WishListByFriendIdRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		PlayerId: s.UserId,
		WishList: make([]*proto.WishListInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.PlayerId = req.PlayerId
	// 单机环境没有其他玩家的实时心愿数据；查询自己时返回本地记录。
	if req.PlayerId != 0 && req.PlayerId != s.UserId {
		return
	}
	for key, value := range s.GetArchive().GetArchiveMap() {
		if !strings.HasPrefix(key, "wish:") {
			continue
		}
		parts := strings.Split(key, ":")
		if len(parts) != 3 || value != strconv.Itoa(int(proto.WishListOperateType_WishListOperateType_Add)) {
			continue
		}
		shopID, err1 := strconv.ParseUint(parts[1], 10, 32)
		gridID, err2 := strconv.ParseUint(parts[2], 10, 32)
		if err1 != nil || err2 != nil {
			continue
		}
		alg.AddList(&rsp.WishList, &proto.WishListInfo{
			ShopId: uint32(shopID),
			GridId: uint32(gridID),
		})
	}
}

func (g *Game) ChallengeFriendRank(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.ChallengeFriendRankRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		RankInfo: make([]*proto.ChallengeFriendRankInfo, 0),
		SelfChallenge: &proto.PlayerChallengeCache{
			PlayerId:       s.UserId,
			ChallengeInfos: make([]*proto.PlayerChallengeInfo, 0),
		},
	}
	defer g.send(s, msg.PacketId, rsp)
	for key, value := range s.GetArchive().GetArchiveMap() {
		if !strings.HasPrefix(key, "challenge:") {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(key, "challenge:"), 10, 32)
		if err != nil {
			continue
		}
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			continue
		}
		useTime, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			continue
		}
		rsp.SelfChallenge.ChallengeInfos = append(rsp.SelfChallenge.ChallengeInfos, &proto.PlayerChallengeInfo{
			ChallengeId: uint32(id),
			UseTime:     uint32(useTime),
		})
	}
	if len(rsp.SelfChallenge.ChallengeInfos) > 0 {
		alg.AddList(&rsp.RankInfo, &proto.ChallengeFriendRankInfo{
			PlayerInfo: g.playerBriefInfo(s),
			Challenge:  rsp.SelfChallenge,
		})
	}
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
