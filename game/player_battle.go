package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) BattleEncounterInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BattleEncounterInfoReq)
	rsp := &proto.BattleEncounterInfoRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		Encounters: make([]*proto.BattleEncounterData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	if sd, ok := scenePlayer.CurScene.(*model.SceneDungeon); ok {
		for _, encounterID := range req.EncounterIds {
			state, status := authoritativeDungeons.encounterState(s, sd, encounterID)
			if status != proto.StatusCode_StatusCode_OK {
				rsp.Status = status
				return
			}
			alg.AddList(&rsp.Encounters, &proto.BattleEncounterData{
				BattleId: encounterID,
				State:    state,
				BoxId:    0,
			})
		}
		return
	}
	// 普通场景没有本轮副本权威状态，只能返回未开始，不能回显客户端的完成状态。
	for _, encounterID := range req.EncounterIds {
		alg.AddList(&rsp.Encounters, &proto.BattleEncounterData{
			BattleId: encounterID,
			State:    proto.BattleState_Start,
			BoxId:    0,
		})
	}
}

func (g *Game) BattleEncounterStateUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BattleEncounterStateUpdateReq)
	rsp := &proto.BattleEncounterStateUpdateRsp{
		Status:                     proto.StatusCode_StatusCode_OK,
		Encounter:                  nil,
		DynamicTreasureBoxBaseInfo: new(proto.DynamicTreasureBoxBaseData),
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	sd, ok := scenePlayer.CurScene.(*model.SceneDungeon)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInDungeon
		return
	}
	if status := authoritativeDungeons.updateEncounter(s, sd, req.EncounterId, req.BattleState); status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	state, status := authoritativeDungeons.encounterState(s, sd, req.EncounterId)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	rsp.Encounter = &proto.BattleEncounterData{
		BattleId: req.EncounterId,
		State:    state,
		BoxId:    0,
	}
}

func (g *Game) FlagBattleStateUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FlagBattleStateUpdateReq)
	rsp := &proto.FlagBattleStateUpdateRsp{
		Status:                     proto.StatusCode_StatusCode_OK,
		BattleData:                 nil,
		DynamicTreasureBoxBaseInfo: new(proto.DynamicTreasureBoxBaseData),
	}
	defer g.send(s, msg.PacketId, rsp)

	if req.BattleState != proto.BattleState_Start {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.BattleData = &proto.FlagBattleData{
		BattleId:    req.BattleId,
		State:       proto.BattleState_Start,
		Type:        req.MissionType,
		FinishTimes: 0,
		VoiceId:     0,
	}
}

func (g *Game) MonsterDead(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MonsterDeadReq)
	rsp := &proto.MonsterDeadRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		MonsterIndex: req.MonsterIndex,
		DropItem:     nil,
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	switch scene := scenePlayer.CurScene.(type) {
	case *model.ScenePlayerInfo:
		if scene == nil {
			rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
			return
		}
		cfg := gdconf.GetMonsterCharacterConfigure(req.MonsterId)
		if cfg == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
		if drop := scene.GetTempPack().GenMonsterDead(s, proto.EMonsterTag(cfg.NewTag), true); drop != nil {
			rsp.DropItem = drop.PbDropItem()
		}
	case *model.SceneDungeon:
		drop, status := authoritativeDungeons.monsterDead(s, scene, req.MonsterId, req.MonsterIndex)
		rsp.Status = status
		if drop != nil {
			rsp.DropItem = drop.PbDropItem()
		}
	default:
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
	}
}

func (g *Game) Pickup(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PickupReq)
	rsp := &proto.PickupRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Items:  nil,
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	if scenePlayer.CurScene == nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
		return
	}
	if sd, ok := scenePlayer.CurScene.(*model.SceneDungeon); ok {
		items, status := authoritativeDungeons.pickup(s, sd, req.DropItemIndex, req.PickIndex)
		rsp.Status = status
		rsp.Items = items
		if status == proto.StatusCode_StatusCode_OK {
			g.PackNoticeByItems(s, items)
		}
		return
	}
	status, items := g.pickupTempPack(s, req)
	rsp.Status = status
	rsp.Items = items
}
