package game

import (
	"time"

	"github.com/bytedance/sonic"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

var (
	channelTick   time.Duration = 50 * time.Millisecond // 50 毫秒
	oneSTickCount int
)

type ChannelInfo struct {
	SceneInfo        *SceneInfo                            // 所属场景
	ChannelId        uint32                                // 房间号
	allPlayer        map[uint32]*ScenePlayer               // 当前房间的全部玩家
	weatherType      proto.WeatherType                     // 天气
	tickCount        int                                   // 已经过去的时刻数量
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
		weatherType:          proto.WeatherType_WeatherType_SUNNY,
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

func (c *ChannelInfo) sendAllPlayer(packetId uint32, payloadMsg pb.Message) {
	for _, player := range c.getAllPlayer() {
		player.Conn.Send(packetId, payloadMsg)
	}
}

func (c *ChannelInfo) sendPlayer(player *ScenePlayer, packetId uint32, payloadMsg pb.Message) {
	player.Conn.Send(packetId, payloadMsg)
}

func (c *ChannelInfo) getTodTime() uint32 {
	return uint32(c.tickCount/oneSTickCount*48) % (28 * 61 * 48)
}

func (c *ChannelInfo) getTodTimeH() int64 {
	return int64((c.tickCount / oneSTickCount / 61) % 24)
}

// 房间主线程
func (c *ChannelInfo) channelMainLoop() {
	syncTimer := time.NewTimer(channelTick) // 0.2s 同步一次
	defer func() {
		syncTimer.Stop()
		log.Game.Debugf("场景:%v房间:%v退出", c.SceneInfo.SceneId, c.ChannelId)
	}()
	for {
		select {
		case <-syncTimer.C: // 定时同步
			c.channelTick()
			syncTimer.Reset(channelTick)
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

func (c *ChannelInfo) channelTick() {
	c.tickCount++
	// 时间帧递进
	if c.tickCount != 0 && c.tickCount%(oneSTickCount*61) == 0 {
		log.Game.Debugf("场景%v房间%v时间%vH 天气%s",
			c.SceneInfo.SceneId, c.ChannelId, c.getTodTimeH(), c.weatherType.String())
		c.serverSceneSync(&ServerSceneSyncCtx{
			ActionType: proto.SceneActionType_SceneActionType_TOD_UPDATE,
		})
	}
	// 天气更新
	if proto.WeatherType(c.getTodTimeH()/12) != c.weatherType {
		c.weatherType = proto.WeatherType(c.getTodTimeH() / 12)
		c.SceneWeatherChangeNotice()
	}

	// 场景自动化更新
	// 场景变化同步
	if len(c.sceneSyncDatas) > 0 {
		notice := &proto.PlayerSceneSyncDataNotice{
			Status: proto.StatusCode_StatusCode_OK,
			Data:   c.sceneSyncDatas,
		}
		c.sendAllPlayer(0, notice)
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
		c.sendAllPlayer(0, notice)
		c.sceneServerDatas = make(map[uint32]*proto.ServerSceneSyncData)
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
	defer c.sendPlayer(scenePlayer, 0, notice)
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
	getPlayerData := func(userId uint32) *proto.ServerSceneSyncData {
		pd, ok := c.sceneServerDatas[userId]
		if !ok {
			pd = &proto.ServerSceneSyncData{
				PlayerId:   userId,
				ServerData: make([]*proto.SceneServerData, 0),
			}
			c.sceneServerDatas[userId] = pd
		}
		return pd
	}
	var playerData *proto.ServerSceneSyncData
	if ctx.ScenePlayer == nil {
		playerData = getPlayerData(0)
	} else {
		playerData = getPlayerData(ctx.ScenePlayer.UserId)
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
	case proto.SceneActionType_SceneActionType_TOD_UPDATE: /* 时间更新*/
		serverData.TodTime = c.getTodTime()
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
	c.sendAllPlayer(0, notice)
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
		TodTime:              c.getTodTime(),                // ok
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
	teamInfo := scenePlayer.GetTeamModel().GetTeamInfo()
	info = &proto.SceneTeam{
		Char1: scenePlayer.GetPbSceneCharacter(teamInfo.Char1),
		Char2: scenePlayer.GetPbSceneCharacter(teamInfo.Char2),
		Char3: scenePlayer.GetPbSceneCharacter(teamInfo.Char3),
	}
	return
}

func (s *ScenePlayer) GetPbSceneCharacter(characterId uint32) (info *proto.SceneCharacter) {
	characterInfo := s.GetCharacterModel().GetCharacterInfo(characterId)
	if characterInfo == nil {
		log.Game.Warnf("玩家:%v队伍角色:%v不存在", s.UserId, characterId)
		return new(proto.SceneCharacter)
	}
	info = &proto.SceneCharacter{
		Pos:                 s.Pos,
		Rot:                 s.Rot,
		CharId:              characterInfo.CharacterId,
		CharLv:              characterInfo.Level,
		CharBreakLv:         characterInfo.BreakLevel,
		CharStar:            characterInfo.Star,
		CharacterAppearance: characterInfo.GetPbCharacterAppearance(),
		OutfitPreset:        s.GetPbSceneCharacterOutfitPreset(characterInfo),
		WeaponId:            0,
		WeaponStar:          0,
		Armors:              make([]*proto.BaseArmor, 0),
		Posters:             make([]*proto.BasePoster, 0),

		GatherWeapon:  0,
		IsDead:        false,
		InscriptionId: 0,
		InscriptionLv: 0,
		MpGameWeapon:  0,
	}
	// 装备
	{
		equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
		if equipmentPreset == nil {
			log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
				s.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
		} else {
			// 武器
			weaponInfo := s.GetItemModel().GetItemWeaponInfo(equipmentPreset.WeaponInstanceId)
			if weaponInfo == nil {
				log.Game.Warnf("玩家:%v角色:%v装备-武器:%v缺少",
					s.UserId, characterInfo.CharacterId, equipmentPreset.WeaponInstanceId)
			} else {
				info.WeaponStar = weaponInfo.Star
				info.WeaponId = weaponInfo.WeaponId
			}
			// 盔甲
			for _, armor := range equipmentPreset.Armors {
				item := s.GetItemModel().GetItemArmorInfo(armor.InstanceId)
				alg.AddList(&info.Armors, item.BaseArmor())
			}
			// 海报
			for _, poster := range equipmentPreset.Posters {
				item := s.GetItemModel().GetItemPosterInfo(poster.InstanceId)
				alg.AddList(&info.Posters, item.BasePoster())
			}
		}

	}

	return
}

func (s *ScenePlayer) GetPbSceneCharacterOutfitPreset(characterInfo *model.CharacterInfo) *proto.SceneCharacterOutfitPreset {
	outfit := characterInfo.GetOutfitPreset(characterInfo.InUseOutfitPresetIndex)
	if outfit == nil {
		log.Game.Warnf("玩家:%v角色:%v外貌序号:%v缺少",
			s.UserId, characterInfo.CharacterId, characterInfo.InUseOutfitPresetIndex)
		return nil
	}
	getODS := func(id, index uint32) *proto.OutfitDyeScheme {
		fashionInfo := s.GetItemModel().GetItemFashionInfo(id)
		if fashionInfo == nil ||
			fashionInfo.GetDyeScheme(index) == nil {
			return &proto.OutfitDyeScheme{
				SchemeIndex: 0,
				Colors:      make([]*proto.PosColor, 0),
				IsUnLock:    false,
			}
		}
		return fashionInfo.GetDyeScheme(index).OutfitDyeScheme()
	}
	info := &proto.SceneCharacterOutfitPreset{
		Hat:                    outfit.Hat,
		HatDyeScheme:           getODS(outfit.Hat, outfit.HatDyeSchemeIndex),
		Hair:                   outfit.Hair,
		HairDyeScheme:          getODS(outfit.Hair, outfit.HairDyeSchemeIndex),
		Clothes:                outfit.Clothes,
		ClothDyeScheme:         getODS(outfit.Clothes, outfit.ClothesDyeSchemeIndex),
		Ornament:               outfit.Ornament,
		OrnDyeScheme:           getODS(outfit.Ornament, outfit.OrnamentDyeSchemeIndex),
		HideInfo:               outfit.OutfitHideInfo.OutfitHideInfo(),
		PendTop:                outfit.PendTop,
		PendTopDyeScheme:       getODS(outfit.PendTop, outfit.PendTopDyeSchemeIndex),
		PendChest:              outfit.PendChest,
		PendChestDyeScheme:     getODS(outfit.PendChest, outfit.PendChestDyeSchemeIndex),
		PendPelvis:             outfit.PendPelvis,
		PendPelvisDyeScheme:    getODS(outfit.PendPelvis, outfit.PendPelvisDyeSchemeIndex),
		PendUpFace:             outfit.PendUpFace,
		PendUpFaceDyeScheme:    getODS(outfit.PendUpFace, outfit.PendUpFaceDyeSchemeIndex),
		PendDownFace:           outfit.PendDownFace,
		PendDownFaceDyeScheme:  getODS(outfit.PendDownFace, outfit.PendDownFaceDyeSchemeIndex),
		PendLeftHand:           outfit.PendLeftHand,
		PendLeftHandDyeScheme:  getODS(outfit.PendLeftHand, outfit.PendLeftHandDyeSchemeIndex),
		PendRightHand:          outfit.PendRightHand,
		PendRightHandDyeScheme: getODS(outfit.PendRightHand, outfit.PendRightHandDyeSchemeIndex),
		PendLeftFoot:           outfit.PendLeftFoot,
		PendLeftFootDyeScheme:  getODS(outfit.PendLeftFoot, outfit.PendLeftFootDyeSchemeIndex),
		PendRightFoot:          outfit.PendRightFoot,
		PendRightFootDyeScheme: getODS(outfit.PendRightFoot, outfit.PendRightFootDyeSchemeIndex),
	}

	return info
}

func (c *ChannelInfo) SceneWeatherChangeNotice() {
	notice := &proto.SceneWeatherChangeNotice{
		Status:      proto.StatusCode_StatusCode_OK,
		WeatherType: c.weatherType,
	}
	c.sendAllPlayer(0, notice)
}
