package game

import (
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

// 副本重复进入与奖励领取。重复进入复用当前副本权威状态；
// 星级/任务奖励只做一次性账本，奖励本体在结算阶段已发放。

func (g *Game) currentDungeon(s *model.Player) *model.SceneDungeon {
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		return nil
	}
	current, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok || current.Info == nil {
		return nil
	}
	return current
}

func (g *Game) DungeonRepeatEnter(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonRepeatEnterReq)
	rsp := &proto.DungeonRepeatEnterRsp{
		Status:               proto.StatusCode_StatusCode_OK,
		UpdateItem:           make([]*proto.ItemDetail, 0),
		TreasureBoxes:        make([]*proto.TreasureBoxData, 0),
		CurrentGatherGroupId: 0,
		GatherLimits:         make([]*proto.GatherLimit, 0),
	}
	dungeon := g.currentDungeon(s)
	if dungeon == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		g.send(s, msg.PacketId, rsp)
		return
	}
	dungeon.Info.TeamInfo = model.TeamInfo{Char1: req.Char1, Char2: req.Char2, Char3: req.Char3}
	if dungeon.ScenePlayerInfo != nil {
		dungeon.ScenePlayerInfo.SetTeam(dungeon.Info.TeamInfo)
		rsp.Team = dungeon.ScenePlayerInfo.GetPbSceneTeam()
	}
	rsp.DungeonData = dungeon.Info.DungeonData()
	for _, box := range dungeon.Info.GetTreasureBoxes() {
		alg.AddLists(&rsp.TreasureBoxes, box.TreasureBoxData())
	}
	g.send(s, msg.PacketId, rsp)
}

func dungeonStarMask(stars uint32) uint32 {
	if stars >= 3 {
		return 0b111
	}
	return (1 << stars) - 1
}

func (g *Game) DungeonStarReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonStarRewardReq)
	rsp := &proto.DungeonStarRewardRsp{Status: proto.StatusCode_StatusCode_OK, Rewards: make([]*proto.ItemDetail, 0)}
	defer g.send(s, msg.PacketId, rsp)
	cfg, status := g.dungeonRuntime().config(req.DungeonId)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)
	if info.Stars == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	mask := dungeonStarMask(info.Stars)
	if info.StarRewardClaimed&mask == mask {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	rewards, status := g.dungeonRuntime().rewards(cfg.StarRewardID)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	items, status := g.dungeonRuntime().grantRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info.StarRewardClaimed |= mask
	rsp.Rewards = items
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

func (g *Game) DungeonTaskFinishReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonTaskFinishRewardReq)
	rsp := &proto.DungeonTaskFinishRewardRsp{Status: proto.StatusCode_StatusCode_OK, Rewards: make([]*proto.ItemDetail, 0)}
	defer g.send(s, msg.PacketId, rsp)
	cfg, status := g.dungeonRuntime().config(req.DungeonId)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)
	if info.FinishTimes == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if info.FirstClearRewardClaimed {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	rewards, status := g.dungeonRuntime().rewards(cfg.RewardID)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	items, status := g.dungeonRuntime().grantRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info.FirstClearRewardClaimed = true
	rsp.Rewards = items
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

func (g *Game) DungeonSweep(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.DungeonSweepReq)
	rsp := &proto.DungeonSweepRsp{Status: proto.StatusCode_StatusCode_OK, Rewards: make([]*proto.ItemDetail, 0)}
	defer g.send(s, msg.PacketId, rsp)
	cfg, status := g.dungeonRuntime().config(req.DungeonId)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info := s.GetDungeonModel().GetDungeonInfo(req.DungeonId)
	if info.FinishTimes == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	today := time.Now().Unix() / 86400
	if info.LastSweepDay != today {
		info.LastSweepDay = today
		info.DailySweepCount = 0
	}
	if cfg.DailyLimit > 0 && info.DailySweepCount >= cfg.DailyLimit {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	rewards, status := g.dungeonRuntime().rewards(cfg.RewardID)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	items, status := g.dungeonRuntime().grantRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	info.DailySweepCount++
	rsp.Rewards = items
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}
