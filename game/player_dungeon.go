package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) DungeonView(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonViewReq)
	rsp := &proto.DungeonViewRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		DungeonData: nil,
		TaskDataLst: make([]*proto.Achieve, 0), // TODO
	}
	defer g.send(s, msg.PacketId, rsp)

	if _, status := authoritativeDungeons.config(req.DungeonId); status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	info := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)
	rsp.DungeonData = info.DungeonData()
}

func (g *Game) DungeonEnter(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonEnterReq)
	rsp := &proto.DungeonEnterRsp{
		Status:               proto.StatusCode_StatusCode_Ok,
		Team:                 nil,
		DungeonData:          nil,
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
	var active *model.SceneDungeon
	if current, ok := scenePlayer.CurScene.(*model.SceneDungeon); ok {
		active = current
	}
	team := model.TeamInfo{Char1: req.Char1, Char2: req.Char2, Char3: req.Char3}
	runtime, status := authoritativeDungeons.enterDungeon(s, active, req.DungeonId, team)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	dungeonInfo := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)
	dungeonInfo.Pos = req.Pos
	dungeonInfo.Rot = req.Rot

	scene := gdconf.GetDungeonSceneInfo(req.DungeonId)
	poss, rots := gdconf.GetSceneInfoRandomBorn(scene.Info.GetBorn())
	pos := gdconf.ConfigVector3ToProtoVector3(poss)
	rot := gdconf.ConfigVector4ToProtoVector3(rots)
	newCurScene := model.NewScenePlayerInfo(s, new(uint32), &dungeonInfo.TeamInfo, pos, rot)
	if newCurScene == nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	g.toScene(scenePlayer, scenePlayer.ChannelId, &model.SceneDungeon{
		ScenePlayerInfo: newCurScene,
		Info:            dungeonInfo,
		Runtime:         runtime,
	})

	rsp.Team = newCurScene.GetPbSceneTeam()
	rsp.DungeonData = dungeonInfo.DungeonData()
	for _, tb := range dungeonInfo.GetTreasureBoxes() {
		alg.AddLists(&rsp.TreasureBoxes, tb.TreasureBoxData())
	}
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
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	switch req.OperateType {
	case proto.DungeonOperateType_DungeonOperateType_Start:
		rsp.Status = authoritativeDungeons.start(s, sd)
	case proto.DungeonOperateType_DungeonOperateType_End:
		rsp.Status = authoritativeDungeons.end(s, sd)
		if sd.Runtime != nil {
			rsp.Star = sd.Runtime.Stars
		}
	default:
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if rsp.Status != proto.StatusCode_StatusCode_Ok || sd.Runtime == nil {
		return
	}
	end := sd.Runtime.StartedAt
	if sd.Runtime.EndedAt > 0 {
		end = sd.Runtime.EndedAt
	} else {
		end = authoritativeDungeons.now().Unix()
	}
	if end >= sd.Runtime.StartedAt {
		rsp.ConsumeTime = uint64(end - sd.Runtime.StartedAt)
	}
}

func (g *Game) DungeonFinish(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.DungeonFinishRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		SceneId:     0,
		DungeonData: nil,
		UpdateItems: make([]*proto.ItemDetail, 0),
		Rewards:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok || sd.Info == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	rsp.DungeonData = sd.Info.DungeonData()
	if scenePlayer.LastScene != nil {
		rsp.SceneId = scenePlayer.LastScene.GetSceneId()
	}
	items, status := authoritativeDungeons.finish(s, sd)
	rsp.Status = status
	rsp.DungeonData = sd.Info.DungeonData()
	if status != proto.StatusCode_StatusCode_Ok {
		return
	}
	rsp.Rewards = items
	rsp.UpdateItems = items
	if len(items) > 0 {
		g.PackNoticeByItems(s, items)
	}
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
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	if scenePlayer.LastScene == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if status := authoritativeDungeons.exit(s, sd); status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	// 是否强制退出只影响客户端确认流程；服务端一旦收到 Exit 就清理本轮权威状态。
	_ = req.IsForceExit
	rsp.SceneId = scenePlayer.LastScene.GetSceneId()
	g.toScene(scenePlayer, scenePlayer.ChannelId, scenePlayer.LastScene)
}
