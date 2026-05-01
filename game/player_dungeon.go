package game

import (
	"sort"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) DungeonView(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonViewReq)
	rsp := &proto.DungeonViewRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		DungeonData: nil,
		TaskDataLst: make([]*proto.Achieve, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	if req.GetDungeonId() == 0 || gdconf.GetDungeonConfigure(req.GetDungeonId()) == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}

	runtime := g.ensureDungeonRuntime(scenePlayer, req.GetDungeonId(), 0, 0, 0, nil, nil)
	if runtime == nil || runtime.Data == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	rsp.DungeonData = copyDungeonData(runtime.Data)
	rsp.TaskDataLst = copyTaskData(runtime.TaskDataLst)
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
	dungeonId := req.GetDungeonId()
	conf := gdconf.GetDungeonConfigure(dungeonId)
	if dungeonId == 0 || conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}

	// 副本门票扣除采用“尽力而为”，避免测试环境因为道具不足而无法跑通流程。
	g.tryConsumeDungeonCost(s, uint32(conf.GetItemID()), uint32(conf.GetItemNum()))

	if req.GetPos() != nil {
		scenePlayer.Pos = copyVector3(req.GetPos())
	}
	if req.GetRot() != nil {
		scenePlayer.Rot = copyVector3(req.GetRot())
	}

	runtime := g.ensureDungeonRuntime(
		scenePlayer,
		dungeonId,
		req.GetChar1(),
		req.GetChar2(),
		req.GetChar3(),
		req.GetPos(),
		req.GetRot(),
	)
	if runtime == nil || runtime.Data == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}

	runtime.InDungeon = true
	runtime.StartTimeMs = time.Now().UnixMilli()
	runtime.ConsumeTime = 0
	runtime.FinalBoxOpened = false
	runtime.PickedBoxes = make(map[uint32]bool)

	runtime.Data.EnterTimes++
	runtime.Data.LastEnterTime = time.Now().Unix()
	runtime.Data.AllTaskFinished = true
	runtime.Data.TaskFinishReward = proto.RewardStatus_RewardStatus_Reward
	runtime.Data.Pos = copyVector3(scenePlayer.Pos)
	runtime.Data.Rot = copyVector3(scenePlayer.Rot)

	rsp.Team = g.buildDungeonSceneTeam(
		scenePlayer,
		runtime.Data.GetChar1(),
		runtime.Data.GetChar2(),
		runtime.Data.GetChar3(),
	)
	rsp.DungeonData = copyDungeonData(runtime.Data)
	rsp.TreasureBoxes = g.dungeonTreasureBoxList(runtime.TreasureBoxes)
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

func (g *Game) ensureDungeonRuntime(
	scenePlayer *ScenePlayer,
	dungeonId, char1, char2, char3 uint32,
	pos, rot *proto.Vector3,
) *PlayerDungeon {
	if scenePlayer == nil || dungeonId == 0 {
		return nil
	}
	if scenePlayer.Dungeon == nil || scenePlayer.Dungeon.DungeonId != dungeonId {
		scenePlayer.Dungeon = g.newPlayerDungeon(scenePlayer, dungeonId, char1, char2, char3, pos, rot)
		return scenePlayer.Dungeon
	}

	runtime := scenePlayer.Dungeon
	if runtime.Data == nil {
		tmp := g.newPlayerDungeon(scenePlayer, dungeonId, char1, char2, char3, pos, rot)
		runtime.Data = tmp.Data
	}
	if runtime.TaskDataLst == nil {
		runtime.TaskDataLst = g.buildDungeonTaskData(dungeonId)
	}
	if runtime.TreasureBoxes == nil {
		runtime.TreasureBoxes = g.newDungeonTreasureBoxes(dungeonId)
	}
	if runtime.PickedBoxes == nil {
		runtime.PickedBoxes = make(map[uint32]bool)
	}

	teamInfo := scenePlayer.GetTeamModel().GetTeamInfo()
	if char1 == 0 {
		char1 = teamInfo.Char1
	}
	if char2 == 0 {
		char2 = teamInfo.Char2
	}
	if char3 == 0 {
		char3 = teamInfo.Char3
	}

	runtime.Data.DungeonId = dungeonId
	runtime.Data.Char1 = char1
	runtime.Data.Char2 = char2
	runtime.Data.Char3 = char3
	if len(runtime.Data.Coins) == 0 {
		runtime.Data.Coins = g.defaultDungeonCoins()
	}
	if len(runtime.Data.Monsters) == 0 {
		runtime.Data.Monsters = g.defaultDungeonMonsters()
	}
	if pos != nil {
		runtime.Data.Pos = copyVector3(pos)
	}
	if rot != nil {
		runtime.Data.Rot = copyVector3(rot)
	}

	return runtime
}

func (g *Game) newPlayerDungeon(
	scenePlayer *ScenePlayer,
	dungeonId, char1, char2, char3 uint32,
	pos, rot *proto.Vector3,
) *PlayerDungeon {
	teamInfo := scenePlayer.GetTeamModel().GetTeamInfo()
	if char1 == 0 {
		char1 = teamInfo.Char1
	}
	if char2 == 0 {
		char2 = teamInfo.Char2
	}
	if char3 == 0 {
		char3 = teamInfo.Char3
	}
	if pos == nil {
		pos = scenePlayer.Pos
	}
	if rot == nil {
		rot = scenePlayer.Rot
	}

	data := &proto.DungeonData{
		DungeonId:        dungeonId,
		AllTaskFinished:  true,
		EnterTimes:       0,
		ExitTimes:        0,
		FinishTimes:      0,
		Coins:            g.defaultDungeonCoins(),
		LastFinishTime:   0,
		TaskFinishReward: proto.RewardStatus_RewardStatus_Reward,
		StarReward:       proto.RewardStatus_RewardStatus_NotReward,
		Monsters:         g.defaultDungeonMonsters(),
		Char1:            char1,
		Char2:            char2,
		Char3:            char3,
		LastEnterTime:    0,
		Pos:              copyVector3(pos),
		Rot:              copyVector3(rot),
		IsOpenSecretBox:  true,
	}

	return &PlayerDungeon{
		DungeonId:      dungeonId,
		InDungeon:      false,
		StartTimeMs:    0,
		ConsumeTime:    0,
		Data:           data,
		TaskDataLst:    g.buildDungeonTaskData(dungeonId),
		TreasureBoxes:  g.newDungeonTreasureBoxes(dungeonId),
		PickedBoxes:    make(map[uint32]bool),
		FinalBoxOpened: false,
	}
}

func (g *Game) buildDungeonTaskData(dungeonId uint32) []*proto.Achieve {
	questId := dungeonId
	if conf := gdconf.GetDungeonConfigure(dungeonId); conf != nil && conf.GetDengeonQuestID() != 0 {
		questId = uint32(conf.GetDengeonQuestID())
	}
	questConf := gdconf.GetDungeonQuestConfigure(questId)
	if questConf == nil {
		return make([]*proto.Achieve, 0)
	}

	taskData := make([]*proto.Achieve, 0, len(questConf.GetDungeonQuestGroupInfo()))
	for _, info := range questConf.GetDungeonQuestGroupInfo() {
		achieveId := uint32(info.GetAchieveConditionID())
		if achieveId == 0 {
			continue
		}
		count := uint64(1)
		if achieveConf := gdconf.GetAchieveConfigure(achieveId); achieveConf != nil && achieveConf.GetCountParam() > 0 {
			count = uint64(achieveConf.GetCountParam())
		}
		taskData = append(taskData, &proto.Achieve{
			AchieveId: achieveId,
			Count:     count,
		})
	}
	return taskData
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

	finalBoxId := uint32(39600000 + dungeonId%1000)
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

func (g *Game) dungeonTreasureBoxList(boxMap map[uint32]*proto.TreasureBoxData) []*proto.TreasureBoxData {
	keys := make([]uint32, 0, len(boxMap))
	for k := range boxMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	list := make([]*proto.TreasureBoxData, 0, len(keys))
	for _, k := range keys {
		list = append(list, copyTreasureBoxData(boxMap[k]))
	}
	return list
}

func (g *Game) buildDungeonSceneTeam(scenePlayer *ScenePlayer, char1, char2, char3 uint32) *proto.SceneTeam {
	if scenePlayer == nil {
		return nil
	}
	return &proto.SceneTeam{
		Char1: scenePlayer.GetPbSceneCharacter(char1),
		Char2: scenePlayer.GetPbSceneCharacter(char2),
		Char3: scenePlayer.GetPbSceneCharacter(char3),
	}
}

func (g *Game) defaultDungeonCoins() []uint32 {
	return []uint32{2, 1, 0, 3}
}

func (g *Game) defaultDungeonMonsters() []uint32 {
	return []uint32{2, 1, 0}
}

func (g *Game) tryConsumeDungeonCost(s *model.Player, itemId, itemNum uint32) {
	if itemId == 0 || itemNum == 0 {
		return
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return
	}
	defer func() {
		if tx.Error != nil {
			tx.Rollback()
		}
	}()

	tx.DelBaseItem(itemId, int64(itemNum))
	if tx.Error != nil {
		log.Game.Debugf("副本门票扣除失败, userId:%v itemId:%v num:%v err:%v",
			s.UserId, itemId, itemNum, tx.Error)
		return
	}
	tx.Commit()
	if tx.PackNotice != nil && len(tx.PackNotice.Items) > 0 {
		g.send(s, 0, tx.PackNotice)
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

func copyTaskData(src []*proto.Achieve) []*proto.Achieve {
	if len(src) == 0 {
		return make([]*proto.Achieve, 0)
	}
	dst := make([]*proto.Achieve, 0, len(src))
	for _, info := range src {
		if info == nil {
			continue
		}
		dst = append(dst, &proto.Achieve{
			AchieveId: info.GetAchieveId(),
			Count:     info.GetCount(),
		})
	}
	return dst
}

func copyTreasureBoxData(src *proto.TreasureBoxData) *proto.TreasureBoxData {
	if src == nil {
		return nil
	}
	return &proto.TreasureBoxData{
		Index:           src.GetIndex(),
		BoxId:           src.GetBoxId(),
		Type:            src.GetType(),
		State:           src.GetState(),
		NextRefreshTime: src.GetNextRefreshTime(),
		Rewards:         append([]*proto.ItemDetail(nil), src.GetRewards()...),
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
