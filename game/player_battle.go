package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/config"
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
			state, status := g.dungeonRuntime().encounterState(s, sd, encounterID)
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
	if status := g.dungeonRuntime().updateEncounter(s, sd, req.EncounterId, req.BattleState); status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	state, status := g.dungeonRuntime().encounterState(s, sd, req.EncounterId)
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
	tempPackMu.Lock()
	defer tempPackMu.Unlock()
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
		drop, status := ordinaryMonsterDead(s, scene, scene.GetSceneId(), req.MonsterId, req.MonsterIndex)
		rsp.Status = status
		if drop != nil {
			rsp.DropItem = drop.PbDropItem()
		}
	case *model.SceneDungeon:
		drop, status := g.dungeonRuntime().monsterDead(s, scene, req.MonsterId, req.MonsterIndex)
		rsp.Status = status
		if drop != nil {
			rsp.DropItem = drop.PbDropItem()
		}
	default:
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
	}
}

var ordinaryMonsterPlayerLevel = dungeonPlayerLevel

// ordinaryMonsterDead accepts a death notification only when the scene export
// contains that exact index/unit pair. No scene configuration means no drop.
func ordinaryMonsterDead(s *model.Player, scene *model.ScenePlayerInfo, sceneID, monsterID, monsterIndex uint32) (*model.DropItem, proto.StatusCode) {
	if s == nil || scene == nil || scene.Player != s || scene.GetTempPack() == nil {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_IN_CHANNEL
	}
	sceneConf := gdconf.GetSceneInfo(sceneID)
	if sceneConf == nil || sceneConf.Info == nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	var configured *config.DungeonMonsterInfo
	for _, monster := range sceneConf.Info.GetMonsterInfos() {
		if monster != nil && monster.Index >= 0 && uint32(monster.Index) == monsterIndex {
			configured = monster
			break
		}
	}
	if configured == nil || configured.UnitID <= 0 || uint32(configured.UnitID) != monsterID {
		return nil, proto.StatusCode_StatusCode_DungeonCheat
	}
	if _, duplicate := scene.DeadMonsters[monsterIndex]; duplicate {
		return nil, proto.StatusCode_StatusCode_DungeonCheat
	}
	level, ok := ordinaryMonsterPlayerLevel(s)
	if !ok {
		return nil, proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
	drop, err := configuredMonsterDrop(s, scene.GetTempPack(), monsterID, level)
	if err != nil {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	if err := s.SaveTempPackArchive(scene.GetTempPack()); err != nil {
		if drop != nil {
			delete(scene.GetTempPack().TempAreaPack, drop.Index)
		}
		return nil, proto.StatusCode_StatusCode_SYS_ERR
	}
	if scene.DeadMonsters == nil {
		scene.DeadMonsters = make(map[uint32]uint32)
	}
	scene.DeadMonsters[monsterIndex] = monsterID
	return drop, proto.StatusCode_StatusCode_OK
}

func (g *Game) Pickup(s *model.Player, msg *alg.GameMsg) {
	tempPackMu.Lock()
	defer tempPackMu.Unlock()
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
		inventory := model.SnapshotItemModel(s.GetItemModel())
		drop := sd.GetTempPack().GetTempAreaPack(req.DropItemIndex)
		var before []model.EBagItemTag
		if drop != nil {
			before = append([]model.EBagItemTag(nil), drop.Items...)
		}
		items, status := g.dungeonRuntime().pickup(s, sd, req.DropItemIndex, req.PickIndex)
		if status == proto.StatusCode_StatusCode_OK && s.Online {
			if err := tempPackSavePlayer(s); err != nil {
				inventory.Restore()
				if drop != nil {
					drop.Items = before
				}
				status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
				items = nil
			}
		}
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
