package game

import (
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type ChannelInfo struct {
	SceneId     uint32                  // 场景号
	ChannelId   uint32                  // 房间号
	allPlayer   map[uint32]*ScenePlayer // 当前房间的全部玩家
	weatherType proto.WeatherType       // 天气
	todTime     uint32                  // 时间
}

func (s *SceneInfo) newChannelInfo(channelId uint32) *ChannelInfo {
	info := &ChannelInfo{
		SceneId:     s.SceneId,
		ChannelId:   channelId,
		allPlayer:   make(map[uint32]*ScenePlayer),
		weatherType: proto.WeatherType_WeatherType_RAINY,
	}

	return info
}

func (c *ChannelInfo) getAllPlayer() map[uint32]*ScenePlayer {
	if c.allPlayer == nil {
		c.allPlayer = make(map[uint32]*ScenePlayer)
	}
	return c.allPlayer
}

func (c *ChannelInfo) addPlayer(scenePlayer *ScenePlayer) bool {
	list := c.getAllPlayer()
	if _, ok := list[scenePlayer.UserId]; ok {
		return false
	}
	list[scenePlayer.UserId] = scenePlayer
	return true
}

func (c *ChannelInfo) sendAllPlayer(cmdId uint32, packetId uint32, payloadMsg pb.Message) {
	for _, player := range c.getAllPlayer() {
		player.Conn.Send(cmdId, packetId, payloadMsg)
	}
}

func (c *ChannelInfo) GetPbSceneData() (info *proto.SceneData) {
	info = &proto.SceneData{
		SceneId:        c.SceneId,
		GatherLimits:   make([]*proto.GatherLimit, 0),
		DropItems:      make([]*proto.DropItem, 0),
		Areas:          make([]*proto.AreaData, 0),
		Collections:    make([]*proto.CollectionData, 0),
		Challenges:     make([]*proto.ChallengeData, 0),
		TreasureBoxes:  make([]*proto.TreasureBoxData, 0),
		Riddles:        make([]*proto.RiddleData, 0),
		Monsters:       make([]*proto.MonsterData, 0),
		EncounterData:  make([]*proto.BattleEncounterData, 0),
		Flags:          make([]*proto.FlagBattleData, 0),
		RegionVoices:   make([]uint32, 0),
		BonFires:       make([]*proto.Bonfire, 0),
		SoccerPosition: new(proto.SoccerPosition),
		ChairInfoList:  make([]*proto.ChairInfo, 0),
		Dungeons:       make([]*proto.DungeonData, 0),
		FlagIds:        make([]uint32, 0),
		SceneGardenData: &proto.SceneGardenData{
			GardenFurnitureInfoMap:      make(map[int64]*proto.FurnitureDetailsInfo),
			LikesNum:                    0,
			AccessPlayerNum:             0,
			LeftLikeNum:                 0,
			GardenName:                  "",
			FurniturePlayerMap:          make(map[int64]uint32),
			OtherPlayerFurnitureInfoMap: make(map[int64]*proto.SceneGardenOtherPlayerData),
			FurnitureCurrentPointNum:    0,
		},
		CurrentGatherGroupId: 0,
		Players:              make([]*proto.ScenePlayer, 0),
		ChannelId:            c.ChannelId,
		TodTime:              c.todTime,
		CampFires:            make([]*proto.CampFire, 0),
		WeatherType:          c.weatherType,
		ChannelLabel:         c.ChannelId,
		FireworksInfo:        new(proto.FireworksInfo),
		MpBeacons:            make([]*proto.MPBeacon, 0),
		NetworkEvent:         make([]*proto.NetworkEventData, 0),
		PlacedCharacters:     make([]*proto.ScenePlacedCharacter, 0),
		MoonSpots:            make([]*proto.MoonSpotData, 0),
		RoomDecorList:        make([]*proto.RoomDecorData, 0),
	}
	// 添加场景中的玩家
	for _, scenePlayer := range c.getAllPlayer() {
		alg.AddList(&info.Players, c.GetPbScenePlayer(scenePlayer))
	}
	return
}

func (c *ChannelInfo) GetPbScenePlayer(scenePlayer *ScenePlayer) (info *proto.ScenePlayer) {
	info = &proto.ScenePlayer{
		PlayerId:              scenePlayer.UserId,
		PlayerName:            scenePlayer.GetBasicModel().PlayerName,
		Team:                  c.GetPbSceneTeam(scenePlayer),
		Status:                new(proto.ScenePlayerActionStatus),
		FoodBuffIds:           make([]uint32, 0),
		GlobalBuffIds:         make([]uint32, 0),
		IsBirthday:            false, // 是生日？
		AvatarFrame:           0,     // 头像框
		MusicalItemId:         0,
		MusicalItemSource:     0,
		MusicalItemInstanceId: 0,
		AbyssRank:             0,
		PlayingMusicNote:      new(proto.PlayingMusicNote),
	}
	return
}

func (c *ChannelInfo) GetPbSceneTeam(scenePlayer *ScenePlayer) (info *proto.SceneTeam) {
	teamInfo := scenePlayer.GetTeamInfo()
	info = &proto.SceneTeam{
		Char1: scenePlayer.GetPbSceneCharacter(teamInfo.Char1),
		Char2: scenePlayer.GetPbSceneCharacter(teamInfo.Char2),
		Char3: scenePlayer.GetPbSceneCharacter(teamInfo.Char3),
	}
	return
}

func (s *ScenePlayer) GetPbSceneCharacter(characterId uint32) (info *proto.SceneCharacter) {
	characterInfo := s.GetCharacterInfo(characterId)
	if characterInfo == nil {
		log.Game.Warnf("玩家:%v队伍角色:%v不存在", s.UserId, characterId)
		return
	}
	info = &proto.SceneCharacter{
		Pos: s.Pos,
		Rot: s.Rot,

		CharId:              characterInfo.CharacterId,
		CharLv:              characterInfo.Level,
		CharStar:            characterInfo.Star,
		CharacterAppearance: s.GetPbCharacterAppearance(characterInfo),
		OutfitPreset:        s.GetPbSceneCharacterOutfitPreset(characterInfo),
		WeaponId:            0,
		WeaponStar:          0,

		GatherWeapon:  0,
		IsDead:        false,
		CharBreakLv:   0,
		Armors:        make([]*proto.BaseArmor, 0),
		InscriptionId: 0,
		InscriptionLv: 0,
		Posters:       make([]*proto.BasePoster, 0),
		MpGameWeapon:  0,
	}
	// 装备
	{
		equipmentPreset := s.GetEquipmentPreset(characterInfo, characterInfo.InUseEquipmentPresetIndex)
		if equipmentPreset == nil {
			log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
				s.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
		} else {
			weaponInfo := s.GetItemModel().GetItemWeaponInfo(equipmentPreset.Weapon)
			if weaponInfo == nil {
				log.Game.Warnf("玩家:%v角色:%v装备-武器:%v缺少",
					s.UserId, characterInfo.CharacterId, equipmentPreset.Weapon)
			} else {
				info.WeaponStar = weaponInfo.Star
				info.WeaponId = weaponInfo.WeaponId
			}
		}
	}

	return
}

func (s *ScenePlayer) GetPbSceneCharacterOutfitPreset(characterInfo *model.CharacterInfo) *proto.SceneCharacterOutfitPreset {
	outfitPresetInfo := s.GetOutfitPreset(characterInfo, characterInfo.InUseOutfitPresetIndex)
	if outfitPresetInfo == nil {
		log.Game.Warnf("玩家:%v角色:%v外貌序号:%v缺少",
			s.UserId, characterInfo.CharacterId, characterInfo.InUseOutfitPresetIndex)
		return nil
	}
	getOutfitDyeScheme := func(id uint32) *proto.OutfitDyeScheme {
		return &proto.OutfitDyeScheme{
			SchemeIndex: 0,
			Colors:      make([]*proto.PosColor, 0),
			IsUnLock:    id != 0,
		}
	}
	info := &proto.SceneCharacterOutfitPreset{
		Hat:                    outfitPresetInfo.Hat,
		Hair:                   outfitPresetInfo.Hair,
		Clothes:                outfitPresetInfo.Clothes,
		Ornament:               outfitPresetInfo.Ornament,
		HatDyeScheme:           getOutfitDyeScheme(outfitPresetInfo.Hat),
		HairDyeScheme:          getOutfitDyeScheme(outfitPresetInfo.Hair),
		ClothDyeScheme:         getOutfitDyeScheme(outfitPresetInfo.Clothes),
		OrnDyeScheme:           getOutfitDyeScheme(0),
		HideInfo:               outfitPresetInfo.OutfitHideInfo.OutfitHideInfo(),
		PendTop:                0,
		PendChest:              0,
		PendPelvis:             0,
		PendUpFace:             0,
		PendDownFace:           0,
		PendLeftHand:           0,
		PendRightHand:          0,
		PendLeftFoot:           0,
		PendRightFoot:          0,
		PendTopDyeScheme:       getOutfitDyeScheme(0),
		PendChestDyeScheme:     getOutfitDyeScheme(0),
		PendPelvisDyeScheme:    getOutfitDyeScheme(0),
		PendUpFaceDyeScheme:    getOutfitDyeScheme(0),
		PendDownFaceDyeScheme:  getOutfitDyeScheme(0),
		PendLeftHandDyeScheme:  getOutfitDyeScheme(0),
		PendRightHandDyeScheme: getOutfitDyeScheme(0),
		PendLeftFootDyeScheme:  getOutfitDyeScheme(0),
		PendRightFootDyeScheme: getOutfitDyeScheme(0),
	}

	return info
}
