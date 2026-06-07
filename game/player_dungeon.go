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
	// 更新场景状态 TODO 这里暂时冻结场景发包
	scenePlayer.NetFreeze = true

	// 写入坐标
	dungeonInfo.Rot = req.Rot
	dungeonInfo.Pos = req.Pos
	// 写入角色
	dungeonInfo.TeamInfo = model.TeamInfo{
		Char1: req.Char1,
		Char2: req.Char2,
		Char3: req.Char3,
	}

	pos, rot := gdconf.GetSceneInfoRandomBorn(scene.Info.GetBorn())
	sceneTeam := &model.SceneTeamInfo{
		Player: s,
		Pos:    gdconf.ConfigVector3ToProtoVector3(pos),
		Rot:    gdconf.ConfigVector4ToProtoVector3(rot),
		Team:   &dungeonInfo.TeamInfo,
	}

	rsp.Team = sceneTeam.GetPbSceneTeam()
	rsp.DungeonData = dungeonInfo.DungeonData()
}

func (g *Game) DungeonOperate(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.DungeonOperateReq)
	rsp := &proto.DungeonOperateRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		ConsumeTime: 0,
		Star:        0,
	}
	defer g.send(s, msg.PacketId, rsp)

	// 获取当前玩家用户状态
}

func (g *Game) DungeonFinish(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.DungeonFinishReq)
	rsp := &proto.DungeonFinishRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		SceneId:     0,
		DungeonData: nil,
		UpdateItems: make([]*proto.ItemDetail, 0),
		Rewards:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) DungeonExit(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.DungeonExitReq)
	rsp := &proto.DungeonExitRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		SceneId: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
}
