package game

import (
	"gucooing/lolo/gdconf"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
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
	sceneTeam := &SceneTeamInfo{
		Player: s,
		Pos:    gdconf.ConfigVector3ToProtoVector3(pos),
		Rot:    gdconf.ConfigVector4ToProtoVector3(rot),
		Team:   &dungeonInfo.TeamInfo,
	}

	rsp.Team = sceneTeam.GetPbSceneTeam()
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
	runtime := g.getDungeonRuntime(scenePlayer)
	if runtime == nil || !runtime.InDungeon {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}

	switch req.GetOperateType() {
	case proto.DungeonOperateType_DungeonOperateType_End:
		runtime.ConsumeTime = calcDungeonConsumeTime(runtime.StartTimeMs)
		rsp.ConsumeTime = runtime.ConsumeTime
	default:
		runtime.StartTimeMs = time.Now().UnixMilli()
		runtime.ConsumeTime = 0
	}
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

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	runtime := g.getDungeonRuntime(scenePlayer)
	if runtime == nil || !runtime.InDungeon || runtime.Data == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}

	if runtime.ConsumeTime == 0 {
		runtime.ConsumeTime = calcDungeonConsumeTime(runtime.StartTimeMs)
	}
	runtime.Data.FinishTimes++
	runtime.Data.LastFinishTime = int64(runtime.ConsumeTime)
	runtime.Data.AllTaskFinished = true
	runtime.Data.TaskFinishReward = proto.RewardStatus_RewardStatus_Reward

	rsp.SceneId = scenePlayer.SceneId
	rsp.DungeonData = copyDungeonData(runtime.Data)
}

func (g *Game) DungeonExit(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.DungeonExitReq)
	rsp := &proto.DungeonExitRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		SceneId: 0,
	}

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		g.send(s, msg.PacketId, rsp)
		return
	}
	runtime := g.getDungeonRuntime(scenePlayer)
	if runtime == nil || !runtime.InDungeon || runtime.Data == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		g.send(s, msg.PacketId, rsp)
		return
	}

	runtime.InDungeon = false
	runtime.StartTimeMs = 0
	runtime.Data.ExitTimes++
	rsp.SceneId = scenePlayer.SceneId

	g.send(s, msg.PacketId, rsp)
	// 对齐抓包：退出副本后主动补发 ChangeSceneChannelRsp 和 SceneDataNotice。
	g.send(s, 0, &proto.ChangeSceneChannelRsp{
		Status:    proto.StatusCode_StatusCode_Ok,
		SceneId:   scenePlayer.SceneId,
		ChannelId: scenePlayer.ChannelId,
	})
	scenePlayer.channelInfo.SceneDataNotice(scenePlayer)
}

func (g *Game) getDungeonRuntime(scenePlayer *ScenePlayer) *PlayerDungeon {
	if scenePlayer == nil {
		return nil
	}
	return scenePlayer.Dungeon
}

func (g *Game) newDungeonTreasureBoxes(dungeonId uint32) map[uint32]*proto.TreasureBoxData {
	newBox := func(index, boxId uint32, state proto.TreasureBoxState) *proto.TreasureBoxData {
		return &proto.TreasureBoxData{
			Index:           index,
			BoxId:           boxId,
			Type:            proto.ETreasureBoxType_ETreasureBoxType_Normal,
			State:           state,
			NextRefreshTime: 0,
			Rewards:         make([]*proto.ItemDetail, 0),
		}
	}

	finalBoxId := 39600000 + dungeonId%1000
	if finalBoxId == 0 {
		finalBoxId = dungeonId*100 + 2
	}

	return map[uint32]*proto.TreasureBoxData{
		1: newBox(1, dungeonId*100+2, proto.TreasureBoxState_TreasureBoxState_Open),
		2: newBox(2, finalBoxId, proto.TreasureBoxState_TreasureBoxState_Close),
		3: newBox(3, dungeonId*100+3, proto.TreasureBoxState_TreasureBoxState_Open),
		0: newBox(0, dungeonId*100+1, proto.TreasureBoxState_TreasureBoxState_Open),
	}
}

func copyVector3(v *proto.Vector3) *proto.Vector3 {
	if v == nil {
		return nil
	}
	return &proto.Vector3{
		X:             v.X,
		Y:             v.Y,
		Z:             v.Z,
		DecimalPlaces: v.DecimalPlaces,
	}
}

func copyDungeonData(src *proto.DungeonData) *proto.DungeonData {
	if src == nil {
		return nil
	}
	return &proto.DungeonData{
		DungeonId:        src.GetDungeonId(),
		AllTaskFinished:  src.GetAllTaskFinished(),
		EnterTimes:       src.GetEnterTimes(),
		ExitTimes:        src.GetExitTimes(),
		FinishTimes:      src.GetFinishTimes(),
		Coins:            append([]uint32(nil), src.GetCoins()...),
		LastFinishTime:   src.GetLastFinishTime(),
		TaskFinishReward: src.GetTaskFinishReward(),
		StarReward:       src.GetStarReward(),
		Monsters:         append([]uint32(nil), src.GetMonsters()...),
		Char1:            src.GetChar1(),
		Char2:            src.GetChar2(),
		Char3:            src.GetChar3(),
		LastEnterTime:    src.GetLastEnterTime(),
		Pos:              copyVector3(src.GetPos()),
		Rot:              copyVector3(src.GetRot()),
		IsOpenSecretBox:  src.GetIsOpenSecretBox(),
	}
}

func calcDungeonConsumeTime(startTimeMs int64) uint64 {
	if startTimeMs <= 0 {
		return 69919
	}
	consume := time.Now().UnixMilli() - startTimeMs
	if consume <= 0 {
		return 69919
	}
	return uint64(consume)
}

func scaleDungeonRewardNum(conf *excel.RewardItemPoolGroupInfo) int64 {
	if conf == nil {
		return 0
	}
	num := int64(conf.GetItemMinCount())
	max := int64(conf.GetItemMaxCount())
	if max > num {
		num = max
	}
	if num <= 0 {
		num = 1
	}
	// 抓包里 20 -> 4，按 1/5 缩放更接近客户端体验。
	if num > 5 {
		num /= 5
	}
	if num <= 0 {
		num = 1
	}
	return num
}
