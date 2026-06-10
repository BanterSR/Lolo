package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
	"time"
)

func (g *Game) DungeonView(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonViewReq)
	rsp := &proto.DungeonViewRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		DungeonData: nil,
		TaskDataLst: make([]*proto.Achieve, 0), // TODO
	}
	defer g.send(s, msg.PacketId, rsp)

	info := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)
	rsp.DungeonData = info.DungeonData()
}

func (g *Game) DungeonEnter(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonEnterReq)
	rsp := &proto.DungeonEnterRsp{
		Status:               proto.StatusCode_StatusCode_Ok,
		Team:                 nil, // OK
		DungeonData:          nil, // OK
		UpdateItem:           make([]*proto.ItemDetail, 0),
		TreasureBoxes:        make([]*proto.TreasureBoxData, 0),
		CurrentGatherGroupId: 0,
		GatherLimits:         make([]*proto.GatherLimit, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}

	scene := gdconf.GetDungeonSceneInfo(req.DungeonId)
	if scene == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	dungeonInfo := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)

	// 写入坐标
	dungeonInfo.Rot = req.Rot
	dungeonInfo.Pos = req.Pos
	// 写入角色
	dungeonInfo.TeamInfo = model.TeamInfo{
		Char1: req.Char1,
		Char2: req.Char2,
		Char3: req.Char3,
	}

	poss, rots := gdconf.GetSceneInfoRandomBorn(scene.Info.GetBorn())
	pos := gdconf.ConfigVector3ToProtoVector3(poss)
	rot := gdconf.ConfigVector4ToProtoVector3(rots)

	newCurScene := model.NewScenePlayerInfo(s, new(uint32), &dungeonInfo.TeamInfo, pos, rot)
	g.toScene(scenePlayer, scenePlayer.ChannelId, &model.SceneDungeon{
		ScenePlayerInfo: newCurScene,
		Info:            dungeonInfo,
	})

	rsp.Team = newCurScene.GetPbSceneTeam()
	rsp.DungeonData = dungeonInfo.DungeonData()
}

func (g *Game) DungeonOperate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonOperateReq)
	rsp := &proto.DungeonOperateRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		ConsumeTime: 0,
		Star:        0,
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	// 获取当前玩家用户状态
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	curTime := time.Now()
	switch req.OperateType {
	case proto.DungeonOperateType_DungeonOperateType_Start:
		sd.Info.LastEnterTime = curTime.Unix()
	case proto.DungeonOperateType_DungeonOperateType_End:
		sd.Info.LastFinishTime = curTime.Unix() - sd.Info.LastEnterTime
	}

	rsp.ConsumeTime = uint64(curTime.Unix() - sd.Info.LastEnterTime)
}

func (g *Game) DungeonFinish(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.DungeonFinishReq)
	rsp := &proto.DungeonFinishRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		SceneId:     0,   // ok
		DungeonData: nil, // ok
		UpdateItems: make([]*proto.ItemDetail, 0),
		Rewards:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	// 获取当前玩家用户状态
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}

	rsp.DungeonData = sd.Info.DungeonData()
	rsp.SceneId = scenePlayer.LastScene.GetSceneId()
}

func (g *Game) DungeonExit(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonExitReq)
	rsp := &proto.DungeonExitRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		SceneId: 0,
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	// 获取当前玩家用户状态
	_, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	if req.IsForceExit {
		rsp.SceneId = scenePlayer.LastScene.GetSceneId()
		g.toScene(scenePlayer, scenePlayer.ChannelId, scenePlayer.LastScene) // 回到原来的场景中
	}
}
