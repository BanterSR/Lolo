package game

import (
	"maps"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) SwitchGardenStatus(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SwitchGardenStatusReq)
	rsp := &proto.SwitchGardenStatusRsp{
		Status:             proto.StatusCode_StatusCode_Ok,
		IsOpen:             false,
		Password:           "",
		PasswordExpireTime: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	garden := model.GetSceneGardenData(s.UserId, 9999)
	garden.IsOpen = req.IsOpen
	garden.Password = req.Password
	garden.Save()
	rsp.IsOpen = garden.IsOpen
	rsp.Password = garden.Password
	rsp.PasswordExpireTime = garden.PasswordExpireTime
}

func (g *Game) GardenLikeRecord(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GardenLikeRecordReq)
	rsp := &proto.GardenLikeRecordRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Record: make([]*proto.GardenLikeRecordInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GardenFurnitureScheme(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GardenFurnitureSchemeReq)
	rsp := &proto.GardenFurnitureSchemeRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		Schemes: s.GetGardenModel().Schemes(),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GardenSchemeFurnitureList(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GardenSchemeFurnitureListReq)
	rsp := &proto.GardenSchemeFurnitureListRsp{
		Status:          proto.StatusCode_StatusCode_Ok,
		SchemeId:        req.SchemeId,
		FurnitureItemId: s.GetGardenModel().GetGardenSchemeInfo(req.SchemeId).FurnitureItemId(),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GardenFurnitureSave(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GardenFurnitureSaveReq)
	rsp := &proto.GardenFurnitureSaveRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		SchemeId: req.SchemeId,
	}
	defer g.send(s, msg.PacketId, rsp)
	curGarden := model.GetSceneGardenData(s.UserId, 9999)
	schema := s.GetGardenModel().GetGardenSchemeInfo(req.SchemeId)
	schema.FurnitureInfoMap = make(map[int64]*model.FurnitureDetailsInfo)

	maps.Copy(schema.FurnitureInfoMap, curGarden.GardenFurnitureInfoMap)
}

func (g *Game) GardenFurnitureRemoveAll(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GardenFurnitureRemoveAllReq)
	rsp := &proto.GardenFurnitureRemoveAllRsp{
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
	scenePlayer.channelInfo.gardenFurnitureChan <- &SceneGardenFurnitureCtx{
		FurnitureInfos: make([]*proto.FurnitureDetailsInfo, 0),
		AllUpdate:      true,
		ScenePlayer:    scenePlayer,
	}
}

func (g *Game) GardenFurnitureSchemeSetName(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GardenFurnitureSchemeSetNameReq)
	rsp := &proto.GardenFurnitureSchemeSetNameRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		SchemeId: req.SchemeId,
		Name:     "",
	}
	defer g.send(s, msg.PacketId, rsp)
	schema := s.GetGardenModel().GetGardenSchemeInfo(req.SchemeId)
	schema.SchemeName = req.Name
	rsp.Name = schema.SchemeName
}

func (g *Game) GardenFurnitureApplyScheme(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GardenFurnitureApplySchemeReq)
	rsp := &proto.GardenFurnitureApplySchemeRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		SchemeId: req.SchemeId,
	}
	defer g.send(s, msg.PacketId, rsp)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil ||
		scenePlayer.channelInfo == nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotInChannel
		log.Game.Warnf("玩家:%v没有加入房间", s.UserId)
		return
	}
	schema := s.GetGardenModel().GetGardenSchemeInfo(req.SchemeId)
	scenePlayer.channelInfo.gardenFurnitureChan <- &SceneGardenFurnitureCtx{
		FurnitureInfos: schema.FurnitureDetailsInfos(),
		AllUpdate:      true,
		ScenePlayer:    scenePlayer,
	}
}

func (g *Game) GardenPlaceCharacter(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GardenPlaceCharacterReq)
	rsp := &proto.GardenPlaceCharacterRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		CharacterId: req.CharacterId,
		FurnitureId: req.FurnitureId,
		SeatId:      req.SeatId,
		IsRemove:    req.IsRemove,
	}
	defer g.send(s, msg.PacketId, rsp)
	characterInfo := s.GetCharacterModel().GetCharacterInfo(req.CharacterId)
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if characterInfo == nil || scenePlayer == nil ||
		scenePlayer.channelInfo == nil ||
		scenePlayer.channelInfo.ChannelId != s.UserId {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	garden := model.GetSceneGardenData(s.UserId, 9999)
	if req.IsRemove {
		if !garden.RemovePlacedCharacter(req.CharacterId) {
			rsp.Status = proto.StatusCode_StatusCode_BadReq
			return
		}
	} else {
		if !garden.AddPlacedCharacter(&proto.ScenePlacedCharacter{
			CharacterId:  req.CharacterId,
			OutfitPreset: scenePlayer.GetPbSceneCharacterOutfitPreset(characterInfo),
			FurnitureId:  req.FurnitureId,
			SeatId:       req.SeatId,
		}) {
			rsp.Status = proto.StatusCode_StatusCode_BadReq
			return
		}
	}
	scenePlayer.channelInfo.gardenFurnitureChan <- &SceneGardenFurnitureCtx{
		Remove:      req.IsRemove,
		CharacterId: req.CharacterId,
	}
}
