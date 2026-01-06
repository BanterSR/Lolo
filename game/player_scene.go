package game

import (
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) PlayerSceneRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerSceneRecordReq)
	rsp := &proto.PlayerSceneRecordRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	if charRecorderDataLst := req.Data.CharRecorderDataLst; charRecorderDataLst != nil {
		for _, v := range charRecorderDataLst {
			if v.Rot != nil && v.Pos != nil {
				scenePlayer.Rot = v.Rot
				scenePlayer.Pos = v.Pos
			}
		}
	}
	scenePlayer.channelInfo.addSceneSyncDataChan <- &proto.SceneSyncData{
		PlayerId: s.UserId,
		Data:     []*proto.PlayerRecorderData{req.Data},
	}
}

func (g *Game) SceneProcessList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.SceneProcessListRsp{
		Status:           proto.StatusCode_StatusCode_Ok,
		SceneProcessList: make([]*proto.SceneProcess, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	rsp.SceneProcessList = append(rsp.SceneProcessList, &proto.SceneProcess{
		SceneId: 9999,
		Process: 1,
	})
	rsp.SceneProcessList = append(rsp.SceneProcessList, &proto.SceneProcess{
		SceneId: 1,
		Process: 1,
	})
	rsp.SceneProcessList = append(rsp.SceneProcessList, &proto.SceneProcess{
		SceneId: 100,
		Process: 1,
	})
}

func (g *Game) SendAction(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SendActionReq)
	rsp := &proto.SendActionRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	scenePlayer.channelInfo.actionSyncChan <- &ActionSyncCtx{
		ScenePlayer: scenePlayer,
		ActionId:    req.ActionId,
	}
}

func (g *Game) ChangeSceneChannel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeSceneChannelReq)
	rsp := &proto.ChangeSceneChannelRsp{
		Status:            proto.StatusCode_StatusCode_Ok,
		SceneId:           req.SceneId,
		ChannelId:         0,
		ChannelLabel:      0,
		PasswordAllowTime: 0,
		TargetPlayerId:    0,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	oldChannelInfo := scenePlayer.channelInfo

	alg.NoZero(&scenePlayer.SceneId, req.SceneId)
	alg.NoZero(&scenePlayer.ChannelId, req.ChannelLabel)
	alg.NoZero(&scenePlayer.Pos, req.Pos)
	alg.NoZero(&scenePlayer.Rot, req.Rot)

	newChannelInfo, err := g.getWordInfo().getChannel(scenePlayer.SceneId, scenePlayer.ChannelId)
	if err != nil {
		scenePlayer.SceneId = oldChannelInfo.SceneInfo.SceneId
		scenePlayer.ChannelId = oldChannelInfo.ChannelId
		rsp.Status = proto.StatusCode_StatusCode_SceneChannelNotExist
		log.Game.Warnf("场景:%v没有目标房间:%v err:%s", req.SceneId, req.ChannelLabel, err)
		return
	}
	if oldChannelInfo != newChannelInfo {
		log.Game.Debugf("玩家:%v切换场景%v房间%v",
			s.UserId, scenePlayer.SceneId, scenePlayer.ChannelId)
		oldChannelInfo.delScenePlayerChan <- scenePlayer // 退出旧房间
		newChannelInfo.addScenePlayerChan <- scenePlayer // 加入新房间
	}
}

func (g *Game) GenericSceneB(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GenericSceneBReq)
	rsp := &proto.GenericSceneBRsp{
		Status:       proto.StatusCode_StatusCode_Ok,
		GenericMsgId: req.GenericMsgId,
		Params:       make([]*proto.CommonParam, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	h := scenePlayer.channelInfo.getTodTimeH()

	for i := int64(0); i < 12; i++ {
		value := (h + i) % 24
		alg.AddList(&rsp.Params, &proto.CommonParam{
			ParamType: proto.CommonParamType_CommonParamType_None,
			IntValue:  value,
			StringValue: func() string {
				if value/12 == 0 {
					return ""
				}
				return "1"
			}(),
		})
	}
}

func (g *Game) SceneInterActionPlayStatus(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SceneInterActionPlayStatusReq)
	rsp := &proto.SceneInterActionPlayStatusRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	scenePlayer.channelInfo.interActionSyncChan <- &InterActionCtx{
		ScenePlayer:  scenePlayer,
		ActionStatus: req.ActionStatus,
		PushType:     req.PushType,
	}
}

func (g *Game) GetGardenInfo(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetGardenInfoReq)
	rsp := &proto.GetGardenInfoRsp{
		Status:     proto.StatusCode_StatusCode_Ok,
		GardenInfo: nil,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	rsp.GardenInfo = scenePlayer.channelInfo.sceneGardenData.GardenBaseInfo()
}

func (g *Game) HandingFurniture(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.HandingFurnitureReq)
	rsp := &proto.HandingFurnitureRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		FurnitureId: model.NextFurnitureId(),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) PlaceFurniture(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlaceFurnitureReq)
	rsp := &proto.PlaceFurnitureRsp{
		Status:               proto.StatusCode_StatusCode_Ok,
		FurnitureDetailsInfo: nil,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	if scenePlayer.channelInfo.ChannelId == s.UserId {
		// 如果是自己的房间
		ok := s.GetItemModel().CheckFurnitureItem(req.FurnitureItemId)
		if !ok {
			rsp.Status = proto.StatusCode_StatusCode_FurnitureNumLimit
			return
		}
	}
	info := &proto.FurnitureDetailsInfo{
		FurnitureId:     req.FurnitureId,
		FurnitureItemId: req.FurnitureItemId,
		Pos:             req.Pos,
		Rotation:        req.Rot,
		LayerNum:        req.LayerNum,
	}
	scenePlayer.channelInfo.gardenFurnitureChan <- &SceneGardenFurnitureCtx{
		Remove:        false,
		ScenePlayer:   scenePlayer,
		FurnitureInfo: info,
		FurnitureId:   req.FurnitureId,
	}

	rsp.FurnitureDetailsInfo = info
}

func (g *Game) TakeOutHandingFurniture(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TakeOutHandingFurnitureReq)
	rsp := &proto.TakeOutHandingFurnitureRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		FurnitureId: req.FurnitureId,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	scenePlayer.channelInfo.gardenFurnitureChan <- &SceneGardenFurnitureCtx{
		Remove:      true,
		ScenePlayer: scenePlayer,
		FurnitureId: req.FurnitureId,
	}
}

func (g *Game) TakeOutFurniture(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TakeOutFurnitureReq)
	rsp := &proto.TakeOutFurnitureRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		FurnitureId: req.FurnitureId,
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) SceneSitChair(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SceneSitChairReq)
	rsp := &proto.SceneSitChairRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		PlayerId: s.UserId,
		ChairId:  req.ChairId,
		IsSit:    req.IsSit,
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		g.send(s, msg.PacketId, rsp)
		return
	}
	scenePlayer.channelInfo.chairSyncChan <- &ChairSyncCtx{
		SyncMsg:  rsp,
		PlayerId: s.UserId,
		ChairId:  req.ChairId,
		SeatId:   req.SeatId,
		IsSit:    req.IsSit,
	}
}

func (g *Game) SceneSitVehicle(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SceneSitVehicleReq)
	rsp := &proto.SceneSitVehicleRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		PlayerId: s.UserId,
		ChairId:  req.ChairId,
		SeatId:   req.SeatId,
		IsSit:    req.IsSit,
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		g.send(s, msg.PacketId, rsp)
		return
	}
	scenePlayer.channelInfo.chairSyncChan <- &ChairSyncCtx{
		SyncMsg:  rsp,
		PlayerId: s.UserId,
		ChairId:  int64(req.ChairId),
		SeatId:   req.SeatId,
		IsSit:    req.IsSit,
	}
}

func (g *Game) ChangeMusicalItem(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeMusicalItemReq)
	rsp := &proto.ChangeMusicalItemRsp{
		Status:                proto.StatusCode_StatusCode_Ok,
		Source:                req.Source,
		MusicalItemInstanceId: req.MusicalItemInstanceId,
		MusicalItemId:         uint32(req.MusicalItemInstanceId),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	scenePlayer.MusicalItemId = uint32(req.MusicalItemInstanceId)
	scenePlayer.MusicalItemInstanceId = req.MusicalItemInstanceId
	scenePlayer.MusicalItemSource = req.Source
	scenePlayer.channelInfo.serverSceneSyncChan <- &ServerSceneSyncCtx{
		ScenePlayer: scenePlayer,
		ActionType:  proto.SceneActionType_SceneActionType_UpdateMusicalItem,
	}
}

func (g *Game) PlayMusicNote(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayMusicNoteReq)
	rsp := &proto.PlayMusicNoteRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		PlayerId:    s.UserId,
		MusicNoteId: req.MusicNoteId,
		StartTime:   time.Now().UnixMilli(),
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	info := &proto.PlayingMusicNote{
		MusicNoteId: rsp.MusicNoteId,
		StartTime:   rsp.StartTime,
	}
	if req.MusicNoteId != 0 {
		info.MusicNoteId = req.MusicNoteId
		info.StartTime = rsp.StartTime
	}
	scenePlayer.PlayingMusicNote = info
	scenePlayer.channelInfo.serverSceneSyncChan <- &ServerSceneSyncCtx{
		ScenePlayer: scenePlayer,
		ActionType:  proto.SceneActionType_SceneActionType_UpdateMusicalItem,
	}
}
