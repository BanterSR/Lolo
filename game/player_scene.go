package game

import (
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

func (g *Game) HandingFurniture(s *model.Player, msg *alg.GameMsg) {}
