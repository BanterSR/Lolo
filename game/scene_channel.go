package game

import (
	"time"

	"github.com/bytedance/sonic"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type ChannelInfo struct {
	SceneInfo        *SceneInfo                            // 所属场景
	ChannelId        uint32                                // 房间号
	allPlayer        map[uint32]*ScenePlayer               // 当前房间的全部玩家
	weatherType      proto.WeatherType                     // 天气
	todTime          uint32                                // 时间
	tick             time.Duration                         // 时间刻 ms
	doneChan         chan struct{}                         // done
	sceneSyncDatas   []*proto.SceneSyncData                // 一个tick中待同步的内容
	sceneServerDatas map[uint32]*proto.ServerSceneSyncData // 一个tick中玩家变动内容
	// chan
	addScenePlayerChan   chan *ScenePlayer         // 玩家进入通道
	delScenePlayerChan   chan *ScenePlayer         // 玩家退出通道
	addSceneSyncDataChan chan *proto.SceneSyncData // 同步器通道
	serverSceneSyncChan  chan *ServerSceneSyncCtx  // 服务端场景同步通道
	actionSyncChan       chan *ActionSyncCtx       // action同步通道
}

func (s *SceneInfo) newChannelInfo(channelId uint32) *ChannelInfo {
	info := &ChannelInfo{
		SceneInfo:            s,
		ChannelId:            channelId,
		allPlayer:            make(map[uint32]*ScenePlayer),
		weatherType:          proto.WeatherType_WeatherType_RAINY,
		tick:                 time.Duration(alg.MaxInt(50, gdconf.GetConstant().ChannelTick)) * time.Millisecond,
		doneChan:             make(chan struct{}),
		addScenePlayerChan:   make(chan *ScenePlayer, 10),
		delScenePlayerChan:   make(chan *ScenePlayer, 10),
		addSceneSyncDataChan: make(chan *proto.SceneSyncData, 100),
		sceneSyncDatas:       make([]*proto.SceneSyncData, 100),
		sceneServerDatas:     make(map[uint32]*proto.ServerSceneSyncData),
		serverSceneSyncChan:  make(chan *ServerSceneSyncCtx, 100),
		actionSyncChan:       make(chan *ActionSyncCtx, 100),
	}

	go info.channelMainLoop()

	return info
}

func (c *ChannelInfo) getAllPlayer() map[uint32]*ScenePlayer {
	if c.allPlayer == nil {
		c.allPlayer = make(map[uint32]*ScenePlayer)
	}
	return c.allPlayer
}

func (c *ChannelInfo) sendAllPlayer(cmdId uint32, packetId uint32, payloadMsg pb.Message) {
	for _, player := range c.getAllPlayer() {
		player.Conn.Send(cmdId, packetId, payloadMsg)
	}
}

func (c *ChannelInfo) sendPlayer(player *ScenePlayer, cmdId uint32, packetId uint32, payloadMsg pb.Message) {
	player.Conn.Send(cmdId, packetId, payloadMsg)
}

// 房间主线程
func (c *ChannelInfo) channelMainLoop() {
	syncTimer := time.NewTimer(c.tick) // 0.2s 同步一次
	defer func() {
		syncTimer.Stop()
		log.Game.Debugf("场景:%v房间:%v退出", c.SceneInfo.SceneId, c.ChannelId)
	}()
	for {
		select {
		case <-syncTimer.C: // 定时同步
			c.channelTick()
			syncTimer.Reset(c.tick)
		case scenePlayer := <-c.addScenePlayerChan: // 玩家进入
			c.addPlayer(scenePlayer)
		case scenePlayer := <-c.delScenePlayerChan: // 玩家退出
			c.delPlayer(scenePlayer)
		case syncData := <-c.addSceneSyncDataChan: // 添加同步内容
			alg.AddList(&c.sceneSyncDatas, syncData)
		case ctx := <-c.serverSceneSyncChan: // 服务场景同步
			c.serverSceneSync(ctx)
		case ctx := <-c.actionSyncChan: // action同步
			c.SendActionNotice(ctx)
		case <-c.doneChan:
			return
		}
	}
}

func (c *ChannelInfo) addPlayer(scenePlayer *ScenePlayer) bool {
	list := c.getAllPlayer()
	if _, ok := list[scenePlayer.UserId]; ok {
		c.SceneDataNotice(scenePlayer) // 已在场景中，说明是重连，直接发场景通知就行了
		return false
	}
	scenePlayer.channelInfo = c
	list[scenePlayer.UserId] = scenePlayer
	// 通知包
	c.SceneDataNotice(scenePlayer)
	c.serverSceneSync(&ServerSceneSyncCtx{
		ScenePlayer: scenePlayer,
		ActionType:  proto.SceneActionType_SceneActionType_ENTER,
	})
	return true
}

func (c *ChannelInfo) delPlayer(scenePlayer *ScenePlayer) {
	list := c.getAllPlayer()
	if _, ok := list[scenePlayer.UserId]; ok {
		delete(list, scenePlayer.UserId)
	}
	c.serverSceneSync(&ServerSceneSyncCtx{
		ScenePlayer: scenePlayer,
		ActionType:  proto.SceneActionType_SceneActionType_LEAVE,
	})
}

// 通知客户端场景信息
func (c *ChannelInfo) SceneDataNotice(scenePlayer *ScenePlayer) {
	notice := &proto.SceneDataNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Data:   nil,
	}
	defer c.sendPlayer(scenePlayer, cmd.SceneDataNotice, 0, notice)
	data := c.GetPbSceneData()
	if data == nil {
		str, _ := sonic.MarshalString(c)
		log.Game.Errorf("玩家场景信息异常:%s", str)
		notice.Status = proto.StatusCode_StatusCode_CANT_JOIN_PLAYER_CURRENT_SCENE_CHANNEL
		return
	}
	notice.Data = data
}

// 服务端场景同步上下文
type ServerSceneSyncCtx struct {
	ScenePlayer *ScenePlayer
	ActionType  proto.SceneActionType
}

func (c *ChannelInfo) serverSceneSync(ctx *ServerSceneSyncCtx) {
	playerData, ok := c.sceneServerDatas[ctx.ScenePlayer.UserId]
	if !ok {
		playerData = &proto.ServerSceneSyncData{
			PlayerId:   ctx.ScenePlayer.UserId,
			ServerData: make([]*proto.SceneServerData, 0),
		}
		c.sceneServerDatas[ctx.ScenePlayer.UserId] = playerData
	}
	serverData := &proto.SceneServerData{
		ActionType: ctx.ActionType,
	}
	alg.AddList(&playerData.ServerData, serverData)
	switch ctx.ActionType {
	case proto.SceneActionType_SceneActionType_ENTER: // 进入场景
		serverData.Player = c.GetPbScenePlayer(ctx.ScenePlayer)
	case proto.SceneActionType_SceneActionType_LEAVE: // 退出场景
	case proto.SceneActionType_SceneActionType_UPDATE_TEAM, /* 更新队伍*/
		proto.SceneActionType_SceneActionType_UPDATE_FASHION: /* 更新服装*/
		serverData.Player = &proto.ScenePlayer{
			Team: c.GetPbSceneTeam(ctx.ScenePlayer),
		}
	}
}

func (c *ChannelInfo) channelTick() {
	// 场景变化同步
	if len(c.sceneSyncDatas) > 0 {
		notice := &proto.PlayerSceneSyncDataNotice{
			Status: proto.StatusCode_StatusCode_OK,
			Data:   c.sceneSyncDatas,
		}
		c.sendAllPlayer(cmd.PlayerSceneSyncDataNotice, 0, notice)
		c.sceneSyncDatas = make([]*proto.SceneSyncData, 0)
	}
	// 玩家变化同步
	if len(c.sceneServerDatas) > 0 {
		notice := &proto.ServerSceneSyncDataNotice{
			Status: proto.StatusCode_StatusCode_OK,
			Data:   make([]*proto.ServerSceneSyncData, 0),
		}
		for _, data := range c.sceneServerDatas {
			alg.AddList(&notice.Data, data)
		}
		c.sendAllPlayer(cmd.ServerSceneSyncDataNotice, 0, notice)
		c.sceneServerDatas = make(map[uint32]*proto.ServerSceneSyncData)
	}

}

// action同步上下文
type ActionSyncCtx struct {
	ScenePlayer *ScenePlayer
	ActionId    uint32
}

func (c *ChannelInfo) SendActionNotice(ctx *ActionSyncCtx) {
	notice := &proto.SendActionNotice{
		Status:            proto.StatusCode_StatusCode_OK,
		ActionId:          ctx.ActionId,
		FromPlayerId:      ctx.ScenePlayer.UserId,
		FromPlayerName:    ctx.ScenePlayer.GetBasicModel().PlayerName,
		IsStudy:           false,
		EndTime:           0,
		MultipleNeedCount: 0,
	}
	c.sendAllPlayer(cmd.SendActionNotice, 0, notice)
}

func (c *ChannelInfo) GetPbSceneData() (info *proto.SceneData) {
	info = &proto.SceneData{
		SceneId:        c.SceneInfo.SceneId, // ok
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
		Players:              make([]*proto.ScenePlayer, 0), // ok
		ChannelId:            c.ChannelId,                   // ok
		TodTime:              c.todTime,                     // ok
		CampFires:            make([]*proto.CampFire, 0),
		WeatherType:          c.weatherType, // ok
		ChannelLabel:         c.ChannelId,   // ok
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
