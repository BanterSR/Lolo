package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
	"reflect"
)

func (g *Game) BattleEncounterInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BattleEncounterInfoReq)
	rsp := &proto.BattleEncounterInfoRsp{
		Status:     proto.StatusCode_StatusCode_Ok,
		Encounters: make([]*proto.BattleEncounterData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	for _, encounterId := range req.EncounterIds {
		alg.AddList(&rsp.Encounters, &proto.BattleEncounterData{
			BattleId: encounterId,
			State:    proto.BattleState_BattleState_Start,
			BoxId:    300000,
		})
	}
}

func (g *Game) BattleEncounterStateUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BattleEncounterStateUpdateReq)
	rsp := &proto.BattleEncounterStateUpdateRsp{
		Status:                     proto.StatusCode_StatusCode_Ok,
		Encounter:                  nil,
		DynamicTreasureBoxBaseInfo: new(proto.DynamicTreasureBoxBaseData),
	}
	defer g.send(s, msg.PacketId, rsp)
	rsp.Encounter = &proto.BattleEncounterData{
		BattleId: req.EncounterId,
		State:    req.BattleState,
		BoxId:    300000,
	}
}

func (g *Game) FlagBattleStateUpdate(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FlagBattleStateUpdateReq)
	rsp := &proto.FlagBattleStateUpdateRsp{
		Status:                     proto.StatusCode_StatusCode_Ok,
		BattleData:                 nil,
		DynamicTreasureBoxBaseInfo: new(proto.DynamicTreasureBoxBaseData),
	}
	defer g.send(s, msg.PacketId, rsp)

	flag := &proto.FlagBattleData{
		BattleId:    req.BattleId,
		State:       req.BattleState,
		Type:        req.MissionType,
		FinishTimes: 0,
		VoiceId:     0,
	}
	rsp.BattleData = flag
}

func (g *Game) MonsterDead(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MonsterDeadReq)
	rsp := &proto.MonsterDeadRsp{
		Status:       proto.StatusCode_StatusCode_Ok,
		MonsterIndex: req.MonsterIndex,
		DropItem:     nil,
	}
	defer g.send(s, msg.PacketId, rsp)

	cfg := gdconf.GetMonsterCharacterConfigure(req.MonsterId)
	if cfg == nil {
		return
	}

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}
	switch ty := scenePlayer.CurScene.(type) {
	case *model.ScenePlayerInfo: // 普通场景
		rsp.DropItem = scenePlayer.CurScene.GetTempPack().GenMonsterDead(s, proto.EMonsterTag(cfg.NewTag), true).PbDropItem()
	case *model.SceneDungeon: // 副本
		rsp.DropItem = scenePlayer.CurScene.GetTempPack().GenMonsterDead(s, proto.EMonsterTag(cfg.NewTag), false).PbDropItem()
	default:
		log.Game.Errorf("[%s]未处理的场景怪物掉落,请提交反馈修正", reflect.TypeOf(ty).Name())
	}
}

func (g *Game) Pickup(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PickupReq)
	rsp := &proto.PickupRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Items:  nil,
	}
	defer g.send(s, msg.PacketId, rsp)

	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		return
	}

	item := scenePlayer.CurScene.GetTempPack().Pickup(req.DropItemIndex, req.PickIndex)
	if item == nil {
		rsp.Status = proto.StatusCode_StatusCode_GachaAlreadyFullPick
		return
	}
	g.PackNoticeByItems(s, item)
	rsp.Items = item
}
