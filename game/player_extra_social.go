package game

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

// 多人、照片墙、好友心愿与限时探索在单机环境下也要能完整应答。
// 状态保存在玩家 Archive 中；不依赖官服匹配/审核服务。

func (g *Game) mpTeamName(s *model.Player) string {
	name := s.GetArchive().GetArchiveValue("mp:name")
	if name == "" {
		name = s.NickName + "的小队"
	}
	return name
}

func (g *Game) mpTeamDetail(s *model.Player) *proto.MPTeamDetail {
	modeID, _ := strconv.ParseUint(s.GetArchive().GetArchiveValue("mp:mode"), 10, 32)
	subModeID, _ := strconv.ParseUint(s.GetArchive().GetArchiveValue("mp:submode"), 10, 32)
	members := map[uint32]*proto.MPPlayer{
		s.UserId: {
			PlayerId:   s.UserId,
			PlayerInfo: &proto.MPPlayerInfo{},
		},
	}
	characters := s.GetCharacterModel().GetAllPbCharacter()
	if len(characters) > 0 {
		members[s.UserId].PlayerInfo.Character = characters[0]
	}
	return &proto.MPTeamDetail{
		Base: &proto.MPTeamBase{
			ModeInfo: &proto.MPGameModeInfo{
				ModeId:    uint32(modeID),
				SubModeId: uint32(subModeID),
			},
			TeamId:         s.UserId,
			TeamName:       g.mpTeamName(s),
			CurrentMembers: 1,
			MaxMembers:     4,
		},
		Members: &proto.MPTeamMembers{
			Leader:  s.UserId,
			Members: members,
		},
	}
}

func (g *Game) MPTeamCreate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPTeamCreateReq)
	archive := s.GetArchive()
	name := strings.TrimSpace(req.TeamName)
	if name == "" {
		name = s.NickName + "的小队"
	}
	archive.SetArchiveMap("mp:name", name)
	archive.SetArchiveMap("mp:mode", strconv.Itoa(int(req.ModeId)))
	archive.SetArchiveMap("mp:submode", strconv.Itoa(int(req.SubModeId)))
	g.send(s, msg.PacketId, &proto.MPTeamCreateRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		TeamDetail: g.mpTeamDetail(s),
	})
}

func (g *Game) MPTeamInfo(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.MPTeamInfoRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		TeamDetail: g.mpTeamDetail(s),
	})
}

func (g *Game) MPTeamJoin(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPTeamJoinReq)
	rsp := &proto.MPTeamJoinRsp{Status: proto.StatusCode_StatusCode_OK}
	if req.TeamId != 0 && req.TeamId != s.UserId {
		rsp.Status = proto.StatusCode_StatusCode_MP_TEAM_NOT_EXIST
	}
	rsp.TeamDetail = g.mpTeamDetail(s)
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) MPTeamRename(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPTeamRenameReq)
	name := strings.TrimSpace(req.NewName)
	status := proto.StatusCode_StatusCode_OK
	if name == "" {
		status = proto.StatusCode_StatusCode_BAD_REQ
	} else {
		s.GetArchive().SetArchiveMap("mp:name", name)
	}
	g.send(s, msg.PacketId, &proto.MPTeamRenameRsp{
		Status:  status,
		NewName: name,
		TeamId:  s.UserId,
	})
}

func (g *Game) MPTeamGameModeChange(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPTeamGameModeChangeReq)
	archive := s.GetArchive()
	archive.SetArchiveMap("mp:mode", strconv.Itoa(int(req.ModeId)))
	archive.SetArchiveMap("mp:submode", strconv.Itoa(int(req.SubModeId)))
	g.send(s, msg.PacketId, &proto.MPTeamGameModeChangeRsp{
		Status: proto.StatusCode_StatusCode_OK,
		TeamId: s.UserId,
		GameMode: &proto.MPGameModeInfo{
			ModeId:    req.ModeId,
			SubModeId: req.SubModeId,
		},
	})
}

func (g *Game) MPTeamDismiss(s *model.Player, msg *alg.GameMsg) {
	archive := s.GetArchive()
	for _, key := range []string{"mp:name", "mp:mode", "mp:submode", "mp:room", "mp:playing", "mp:ready"} {
		archive.SetArchiveMap(key, "")
	}
	g.send(s, msg.PacketId, &proto.MPTeamDismissRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) MPTeamQuit(s *model.Player, msg *alg.GameMsg) {
	s.GetArchive().SetArchiveMap("mp:name", "")
	g.send(s, msg.PacketId, &proto.MPTeamQuitRsp{
		Status: proto.StatusCode_StatusCode_OK,
		TeamId: s.UserId,
	})
}

func (g *Game) MPTeamInvite(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPTeamInviteReq)
	status := proto.StatusCode_StatusCode_OK
	if req.TargetId == 0 || req.TargetId == s.UserId || s.GetArchive().GetArchiveValue("mp:name") == "" {
		status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	g.send(s, msg.PacketId, &proto.MPTeamInviteRsp{Status: status})
}

func (g *Game) MPTeamKickMember(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPTeamKickMemberReq)
	g.send(s, msg.PacketId, &proto.MPTeamKickMemberRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		TargetId: req.TargetId,
	})
}

func (g *Game) MPGameStart(s *model.Player, msg *alg.GameMsg) {
	archive := s.GetArchive()
	status := proto.StatusCode_StatusCode_OK
	if archive.GetArchiveValue("mp:name") == "" {
		status = proto.StatusCode_StatusCode_MP_TEAM_NOT_EXIST
	} else {
		archive.SetArchiveMap("mp:playing", strconv.FormatInt(time.Now().Unix(), 10))
	}
	g.send(s, msg.PacketId, &proto.MPGameStartRsp{
		Status: status,
		TeamId: s.UserId,
	})
}

func (g *Game) MPPlayRoomReady(s *model.Player, msg *alg.GameMsg) {
	archive := s.GetArchive()
	status := proto.StatusCode_StatusCode_OK
	if archive.GetArchiveValue("mp:name") == "" {
		status = proto.StatusCode_StatusCode_MP_TEAM_NOT_EXIST
	} else {
		archive.SetArchiveMap("mp:ready", "1")
	}
	g.send(s, msg.PacketId, &proto.MPPlayRoomReadyRsp{Status: status})
}

func (g *Game) MPPlayRoomEvent(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.MPPlayRoomEventRsp{Status: proto.StatusCode_StatusCode_OK})
}

func (g *Game) MPPlayRoomExit(s *model.Player, msg *alg.GameMsg) {
	archive := s.GetArchive()
	archive.SetArchiveMap("mp:room", "")
	archive.SetArchiveMap("mp:playing", "")
	archive.SetArchiveMap("mp:ready", "")
	g.send(s, msg.PacketId, &proto.MPPlayRoomExitRsp{
		Status: proto.StatusCode_StatusCode_OK,
		TeamId: s.UserId,
	})
}

func (g *Game) MPPlayRoomRewardInfo(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.MPPlayRoomRewardInfoRsp{
		Status: proto.StatusCode_StatusCode_OK,
	})
}

func (g *Game) MPSwapCharacter(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPSwapCharacterReq)
	info := &proto.MPPlayerInfo{}
	if infoModel := s.GetCharacterModel().GetCharacterInfo(req.CharacterId); infoModel != nil {
		selected := infoModel.Character()
		info.Character = selected
	}
	g.send(s, msg.PacketId, &proto.MPSwapCharacterRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		NewInfo: info,
	})
}

func (g *Game) MPBeaconAction(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MPBeaconActionReq)
	beacon := &proto.MPBeacon{
		OwnerId:  s.UserId,
		Name:     s.NickName,
		Position: req.Position,
		Rotation: req.Rotation,
	}
	if scenePlayer := g.getWordInfo().getScenePlayer(s); scenePlayer != nil {
		beacon.SceneId = scenePlayer.CurScene.GetSceneId()
		beacon.ChannelLabel = scenePlayer.ChannelId
	}
	g.send(s, msg.PacketId, &proto.MPBeaconActionRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Beacon: beacon,
		TeamId: s.UserId,
	})
}

func (g *Game) MPBeaconPlayerReady(s *model.Player, msg *alg.GameMsg) {
	g.send(s, msg.PacketId, &proto.MPBeaconPlayerReadyRsp{Status: proto.StatusCode_StatusCode_OK})
}

// 照片墙：本地保存标题/标签，重进后可读取；图片本体不写入玩家 blob。
func (g *Game) photoArchivePrefix() string { return "photo:" }

func (g *Game) UploadPhotoShare(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UploadPhotoShareReq)
	if len(req.File) == 0 {
		g.send(s, msg.PacketId, &proto.UploadPhotoShareRsp{Status: proto.StatusCode_StatusCode_BAD_REQ})
		return
	}
	archive := s.GetArchive()
	maxUpload, _ := strconv.ParseUint(archive.GetArchiveValue("photo:uploadMax"), 10, 32)
	if maxUpload == 0 {
		maxUpload = 100
	}
	if uint64(len(sortedPhotoCodes(archive.GetArchiveMap()))) >= maxUpload {
		g.send(s, msg.PacketId, &proto.UploadPhotoShareRsp{Status: proto.StatusCode_StatusCode_PS_UPLOAD_LIMIT})
		return
	}
	code := time.Now().UnixNano()
	prefix := g.photoArchivePrefix() + strconv.FormatInt(code, 10) + ":"
	archive.SetArchiveMap(prefix+"title", req.Title)
	archive.SetArchiveMap(prefix+"tag1", req.Tag1)
	archive.SetArchiveMap(prefix+"tag2", req.Tag2)
	archive.SetArchiveMap(prefix+"tag3", req.Tag3)
	archive.SetArchiveMap(prefix+"time", strconv.FormatInt(time.Now().Unix(), 10))
	if req.IsActivity {
		archive.SetArchiveMap(prefix+"activity", "1")
	}
	info := &proto.PhotoDetailInfo{
		Code:        code,
		PlayerId:    s.UserId,
		PlayerName:  s.NickName,
		UploadTime:  time.Now().Unix(),
		Title:       req.Title,
		Tag1:        req.Tag1,
		Tag2:        req.Tag2,
		Tag3:        req.Tag3,
		CameraParam: req.CameraParam,
		ActivityId:  0,
	}
	if req.IsActivity {
		info.ActivityId = 1
	}
	g.send(s, msg.PacketId, &proto.UploadPhotoShareRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Info:   info,
	})
}

func (g *Game) photoInfo(s *model.Player, code int64) *proto.PhotoDetailInfo {
	prefix := g.photoArchivePrefix() + strconv.FormatInt(code, 10) + ":"
	archive := s.GetArchive()
	title := archive.GetArchiveValue(prefix + "title")
	if title == "" && archive.GetArchiveValue(prefix+"tag1") == "" {
		return nil
	}
	uploadTime, _ := strconv.ParseInt(archive.GetArchiveValue(prefix+"time"), 10, 64)
	activityID := uint32(0)
	if archive.GetArchiveValue(prefix+"activity") == "1" {
		activityID = 1
	}
	return &proto.PhotoDetailInfo{
		Code:       code,
		PlayerId:   s.UserId,
		PlayerName: s.NickName,
		UploadTime: uploadTime,
		Title:      title,
		Tag1:       archive.GetArchiveValue(prefix + "tag1"),
		Tag2:       archive.GetArchiveValue(prefix + "tag2"),
		Tag3:       archive.GetArchiveValue(prefix + "tag3"),
		ActivityId: activityID,
	}
}

func (g *Game) PhotoShareDetail(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PhotoShareDetailReq)
	info := g.photoInfo(s, req.Code)
	if info == nil {
		g.send(s, msg.PacketId, &proto.PhotoShareDetailRsp{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND})
		return
	}
	g.send(s, msg.PacketId, &proto.PhotoShareDetailRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Info:   info,
	})
}

func (g *Game) ChangePhotoShareTitle(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangePhotoShareTitleReq)
	prefix := g.photoArchivePrefix() + strconv.FormatInt(req.Code, 10) + ":"
	archive := s.GetArchive()
	if archive.GetArchiveValue(prefix+"title") == "" && archive.GetArchiveValue(prefix+"tag1") == "" {
		g.send(s, msg.PacketId, &proto.ChangePhotoShareTitleRsp{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND})
		return
	}
	archive.SetArchiveMap(prefix+"title", req.Title)
	archive.SetArchiveMap(prefix+"tag1", req.Tag1)
	archive.SetArchiveMap(prefix+"tag2", req.Tag2)
	g.send(s, msg.PacketId, &proto.ChangePhotoShareTitleRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Title:  req.Title,
		Tag1:   req.Tag1,
		Tag2:   req.Tag2,
	})
}

func (g *Game) OperatePhotoShare(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OperatePhotoShareReq)
	status := proto.StatusCode_StatusCode_OK
	if g.photoInfo(s, req.Code) == nil {
		status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	g.send(s, msg.PacketId, &proto.OperatePhotoShareRsp{
		Status: status,
		Code:   req.Code,
	})
}

func (g *Game) PhotoShareCopyFashionDye(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PhotoShareCopyFashionDyeReq)
	status := proto.StatusCode_StatusCode_OK
	if req.Code == 0 || req.Outfit == nil || g.photoInfo(s, req.Code) == nil {
		status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	g.send(s, msg.PacketId, &proto.PhotoShareCopyFashionDyeRsp{Status: status})
}

func (g *Game) PhotoSharePaymentAddUploadGrid(s *model.Player, msg *alg.GameMsg) {
	archive := s.GetArchive()
	maxUpload, _ := strconv.ParseUint(archive.GetArchiveValue("photo:uploadMax"), 10, 32)
	if maxUpload == 0 {
		maxUpload = 100
	}
	const maxGridLimit = 1000
	if maxUpload < maxGridLimit {
		maxUpload += 10
		if maxUpload > maxGridLimit {
			maxUpload = maxGridLimit
		}
	}
	archive.SetArchiveMap("photo:uploadMax", strconv.FormatUint(maxUpload, 10))
	g.send(s, msg.PacketId, &proto.PhotoSharePaymentAddUploadGridRsp{Status: proto.StatusCode_StatusCode_OK})
}

// 好友心愿单：记录玩家对某个商店格子的心愿，返回空好友列表。
func (g *Game) WishListOperate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.WishListOperateReq)
	key := "wish:" + strconv.Itoa(int(req.ShopId)) + ":" + strconv.Itoa(int(req.GridId))
	s.GetArchive().SetArchiveMap(key, strconv.Itoa(int(req.OperateType)))
	g.send(s, msg.PacketId, &proto.WishListOperateRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		OperateType: req.OperateType,
		ShopId:      req.ShopId,
		GridId:      req.GridId,
	})
}

func (g *Game) wishWeekKey() string {
	year, week := time.Now().ISOWeek()
	return "wish:week:" + strconv.Itoa(year) + "-" + strconv.Itoa(week)
}

func (g *Game) WishListByShopId(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.WishListByShopIdReq)
	rsp := &proto.WishListByShopIdRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		Info:     make([]*proto.FriendBriefInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if req.ShopId == 0 || req.GridId == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	key := "wish:" + strconv.Itoa(int(req.ShopId)) + ":" + strconv.Itoa(int(req.GridId))
	if s.GetArchive().GetArchiveValue(key) == strconv.Itoa(int(proto.WishListOperateType_WishListOperateType_Add)) {
		alg.AddList(&rsp.Info, &proto.FriendBriefInfo{Info: g.playerBriefInfo(s)})
	}
	if count, err := strconv.ParseUint(s.GetArchive().GetArchiveValue(g.wishWeekKey()), 10, 32); err == nil {
		rsp.WeekSendCount = uint32(count)
	}
}

func (g *Game) WishListBuy(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.WishListBuyReq)
	rsp := &proto.WishListBuyRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		PlayerId: req.PlayerId,
		Info:     req.Info,
	}
	defer g.send(s, msg.PacketId, rsp)
	if req.PlayerId == 0 || req.Info == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	key := g.wishWeekKey()
	count, _ := strconv.ParseUint(s.GetArchive().GetArchiveValue(key), 10, 32)
	const weekLimit = 10
	if count >= weekLimit {
		rsp.Status = proto.StatusCode_StatusCode_WISH_LIST_SHOP_ERR
		return
	}
	count++
	s.GetArchive().SetArchiveMap(key, strconv.FormatUint(count, 10))
	rsp.WeekSendCount = uint32(count)
}

// 限时探索：当前玩家维护一个开始时间，好友加入/领取只回应同一条记录。
func (g *Game) intervalStartTime(s *model.Player) (uint32, int64) {
	archive := s.GetArchive()
	exploreID, _ := strconv.ParseUint(archive.GetArchiveValue("interval:exploreId"), 10, 32)
	finish, _ := strconv.ParseInt(archive.GetArchiveValue("interval:finishTime"), 10, 64)
	return uint32(exploreID), finish
}

func (g *Game) IntervalStart(s *model.Player, msg *alg.GameMsg) {
	exploreID := uint32(1)
	finish := time.Now().Add(8 * time.Hour).Unix()
	archive := s.GetArchive()
	archive.SetArchiveMap("interval:exploreId", strconv.Itoa(int(exploreID)))
	archive.SetArchiveMap("interval:finishTime", strconv.FormatInt(finish, 10))
	g.send(s, msg.PacketId, &proto.IntervalStartRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		ExploreId:  exploreID,
		FinishTime: finish,
	})
}

func (g *Game) IntervalJoin(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.IntervalJoinReq)
	exploreID, finish := g.intervalStartTime(s)
	status := proto.StatusCode_StatusCode_OK
	if finish == 0 {
		status = proto.StatusCode_StatusCode_INTERVAL_NOT_EXISTS
	}
	g.send(s, msg.PacketId, &proto.IntervalJoinRsp{
		Status:     status,
		FriendId:   strconv.Itoa(int(req.FriendId)),
		ExploreId:  exploreID,
		FinishTime: finish,
	})
}

func (g *Game) IntervalQuick(s *model.Player, msg *alg.GameMsg) {
	exploreID, finish := g.intervalStartTime(s)
	status := proto.StatusCode_StatusCode_OK
	if finish == 0 {
		status = proto.StatusCode_StatusCode_INTERVAL_NOT_EXISTS
	}
	g.send(s, msg.PacketId, &proto.IntervalQuickRsp{
		Status:     status,
		ExploreId:  exploreID,
		FinishTime: finish,
	})
}

func (g *Game) IntervalReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.IntervalRewardReq)
	exploreID, finish := g.intervalStartTime(s)
	status := proto.StatusCode_StatusCode_OK
	if finish == 0 {
		status = proto.StatusCode_StatusCode_INTERVAL_NOT_EXISTS
	} else if time.Now().Unix() < finish {
		status = proto.StatusCode_StatusCode_INTERVAL_NOT_FINISH
	}
	g.send(s, msg.PacketId, &proto.IntervalRewardRsp{
		Status:    status,
		FriendId:  req.FriendId,
		ExploreId: exploreID,
		Rewards:   make([]*proto.ItemDetail, 0),
	})
}

// BossRush：保存赛季/阶段/战斗开始时间，客户端可在单机模式完成整套流程。
func (g *Game) bossRushKey(s *model.Player, seasonID uint32, part string) string {
	return "bossRush:" + strconv.Itoa(int(seasonID)) + ":" + part
}

func (g *Game) BossRushStartChallenge(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushStartChallengeReq)
	rsp := &proto.BossRushStartChallengeRsp{SeasonId: req.SeasonId}
	if req.SeasonId == 0 || req.StageIndex < 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		g.send(s, msg.PacketId, rsp)
		return
	}
	archive := s.GetArchive()
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "stage"), strconv.Itoa(int(req.StageIndex)))
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "start"), strconv.FormatInt(time.Now().Unix(), 10))
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "battle"), "")
	rsp.Status = proto.StatusCode_StatusCode_OK
	rsp.StageInfo = &proto.BossRushStageInfo{StageIndex: req.StageIndex}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) BossRushEnterStage(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushEnterStageReq)
	rsp := &proto.BossRushEnterStageRsp{SeasonId: req.SeasonId}
	stageText := s.GetArchive().GetArchiveValue(g.bossRushKey(s, req.SeasonId, "stage"))
	if stageText == "" {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		g.send(s, msg.PacketId, rsp)
		return
	}
	stageIndex, _ := strconv.ParseInt(stageText, 10, 32)
	rsp.Status = proto.StatusCode_StatusCode_OK
	rsp.BattleInfo = &proto.BossRushStageBattleInfo{StageIndex: int32(stageIndex)}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) BossRushStartBattle(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushStartBattleReq)
	rsp := &proto.BossRushStartBattleRsp{SeasonId: req.SeasonId}
	stageText := s.GetArchive().GetArchiveValue(g.bossRushKey(s, req.SeasonId, "stage"))
	if stageText == "" {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		g.send(s, msg.PacketId, rsp)
		return
	}
	stageIndex, _ := strconv.ParseInt(stageText, 10, 32)
	archive := s.GetArchive()
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "battle"), strconv.FormatInt(time.Now().Unix(), 10))
	rsp.Status = proto.StatusCode_StatusCode_OK
	rsp.StageIndex = int32(stageIndex)
	rsp.BattleStartTime = time.Now().Unix()
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) BossRushLeaveStage(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushLeaveStageReq)
	rsp := &proto.BossRushLeaveStageRsp{}
	archive := s.GetArchive()
	battle := archive.GetArchiveValue(g.bossRushKey(s, req.SeasonId, "battle"))
	if battle == "" {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		g.send(s, msg.PacketId, rsp)
		return
	}
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "battle"), "")
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "stage"), "")
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "start"), "")
	best, _ := strconv.ParseUint(archive.GetArchiveValue(g.bossRushKey(s, req.SeasonId, "best")), 10, 64)
	if uint64(req.TotalDamageHp) > best {
		best = uint64(req.TotalDamageHp)
		archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "best"), strconv.FormatUint(best, 10))
	}
	rsp.Status = proto.StatusCode_StatusCode_OK
	rsp.StageBestScore = req.TotalDamageHp
	rsp.BestTotalScore = best
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) BossRushTerminateChallenge(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushTerminateChallengeReq)
	archive := s.GetArchive()
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "battle"), "")
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "stage"), "")
	archive.SetArchiveMap(g.bossRushKey(s, req.SeasonId, "start"), "")
	g.send(s, msg.PacketId, &proto.BossRushTerminateChallengeRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		SeasonId: req.SeasonId,
	})
}

func (g *Game) BossRushFriendDetail(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushFriendDetailReq)
	g.send(s, msg.PacketId, &proto.BossRushFriendDetailRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		PlayerId: req.PlayerId,
		Details:  make([]*proto.BossRushSeasonDetail, 0),
	})
}

func (g *Game) BossRushFriendRank(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushFriendRankReq)
	g.send(s, msg.PacketId, &proto.BossRushFriendRankRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		SeasonId: req.SeasonId,
		RankInfo: make([]*proto.BossRushFriendRankInfo, 0),
	})
}

func (g *Game) BossRushQuestReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BossRushQuestRewardReq)
	key := "bossRushQuest:" + strconv.Itoa(int(req.SeasonId)) + ":" + strconv.Itoa(int(req.AchieveId))
	status := proto.StatusCode_StatusCode_OK
	if s.GetArchive().GetArchiveValue(key) != "" {
		status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	} else {
		s.GetArchive().SetArchiveMap(key, "1")
	}
	g.send(s, msg.PacketId, &proto.BossRushQuestRewardRsp{
		Status:    status,
		SeasonId:  req.SeasonId,
		AchieveId: req.AchieveId,
		Rewards:   make([]*proto.ItemDetail, 0),
	})
}

// 深渊：单机没有跨服排行数据，返回当前玩家的空排行与队伍信息。
func (g *Game) AbyssAllServerRank(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AbyssAllServerRankReq)
	rsp := &proto.AbyssAllServerRankRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		RankList: make([]*proto.AbyssAllServerRankInfo, 0),
	}
	seasonID := req.SeasonId
	if seasonID == 0 {
		seasonID = defaultActivityService().abyssInfo(s).GetInProgressSeasonId()
	}
	totalTime, _ := strconv.ParseUint(
		s.GetArchive().GetArchiveValue("abyss:"+strconv.Itoa(int(seasonID))+":totalTime"), 10, 32)
	if seasonID != 0 && totalTime > 0 {
		alg.AddList(&rsp.RankList, &proto.AbyssAllServerRankInfo{
			SeasonId:   seasonID,
			TotalTime:  uint32(totalTime),
			PlayerInfo: g.playerBriefInfo(s),
		})
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) AbyssFriendRank(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AbyssFriendRankReq)
	rsp := &proto.AbyssFriendRankRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		RankList: make([]*proto.AbyssFriendRankInfo, 0),
	}
	info := defaultActivityService().abyssInfo(s)
	seasonID := req.SeasonId
	if seasonID == 0 {
		seasonID = info.GetInProgressSeasonId()
	}
	totalTime, _ := strconv.ParseUint(
		s.GetArchive().GetArchiveValue("abyss:"+strconv.Itoa(int(seasonID))+":totalTime"), 10, 32)
	if seasonID != 0 && totalTime > 0 {
		alg.AddList(&rsp.RankList, &proto.AbyssFriendRankInfo{
			PlayerInfo: g.playerBriefInfo(s),
			AbyssInfo:  info.GetAbyssInfo().GetSeasonInfo()[seasonID],
		})
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) playerBriefInfo(s *model.Player) *proto.PlayerBriefInfo {
	head, _ := strconv.ParseUint(s.GetArchive().GetArchiveValue("head"), 10, 32)
	return &proto.PlayerBriefInfo{
		PlayerId: s.UserId,
		NickName: s.NickName,
		Level:    safePlayerLevel(s),
		Head:     uint32(head),
		IsOnline: s.Online,
	}
}

// safePlayerLevel 只在展示简要信息时使用，数据库未初始化或查询失败时
// 退回 0 级，避免排行/好友列表因基础信息读取失败而 panic。
func safePlayerLevel(s *model.Player) (level uint32) {
	defer func() {
		if recover() != nil {
			level = 0
		}
	}()
	return defaultActivityService().playerLevel(s)
}

func (g *Game) AbyssTeamSwitch(s *model.Player, msg *alg.GameMsg) {
	team := s.GetTeamModel().GetTeamInfo()
	g.send(s, msg.PacketId, &proto.AbyssTeamSwitchRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Team: &proto.SceneTeam{
			Char1: &proto.SceneCharacter{CharId: team.Char1},
			Char2: &proto.SceneCharacter{CharId: team.Char2},
			Char3: &proto.SceneCharacter{CharId: team.Char3},
		},
	})
}

func (g *Game) AbyssTeamUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AbyssTeamUpdateReq)
	key := "abyssTeam:" + strconv.Itoa(int(req.DungeonId)) + ":" + strconv.Itoa(int(req.TeamIndex))
	value := strconv.Itoa(int(req.AbyssTeam.Char1)) + "," + strconv.Itoa(int(req.AbyssTeam.Char2)) + "," + strconv.Itoa(int(req.AbyssTeam.Char3))
	s.GetArchive().SetArchiveMap(key, value)
	g.send(s, msg.PacketId, &proto.AbyssTeamUpdateRsp{Status: proto.StatusCode_StatusCode_OK})
}

// 兼容旧照片列表排序工具（photo codes 升序）。
func sortedPhotoCodes(values map[string]string) []int64 {
	codes := make([]int64, 0)
	for key := range values {
		if !strings.HasPrefix(key, "photo:") || !strings.HasSuffix(key, ":title") {
			continue
		}
		raw := strings.TrimSuffix(strings.TrimPrefix(key, "photo:"), ":title")
		if code, err := strconv.ParseInt(raw, 10, 64); err == nil {
			codes = append(codes, code)
		}
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}
