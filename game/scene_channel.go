package game

import (
	"gucooing/lolo/gdconf"
	"time"

	"github.com/bytedance/sonic"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type ChannelInfo struct {
	game             *Game
	channelType      int                                   // 房间类型
	SceneInfo        *SceneInfo                            // 所属场景
	ChannelId        uint32                                // 房间号
	allPlayer        map[uint32]*ScenePlayer               // 当前房间的全部玩家
	weatherType      proto.WeatherType                     // 天气
	todSeq           int64                                 // 时间段
	doneChan         chan struct{}                         // done
	sceneSyncDatas   []*proto.SceneSyncData                // 一个tick中待同步的内容
	sceneServerDatas map[uint32]*proto.ServerSceneSyncData // 一个tick中玩家变动内容
	chatChannel      *ChatChannel                          // 当前场景的聊天房间
	sceneGardenData  *model.SceneGardenData                // 花园信息
	chaiInfoMap      map[uint32]*proto.ChairInfo           // 座位信息
	// chan
	addScenePlayerChan   chan *ScenePlayer             // 玩家进入通道
	delScenePlayerChan   chan *ScenePlayer             // 玩家退出通道
	addSceneSyncDataChan chan *proto.SceneSyncData     // 同步器通道
	serverSceneSyncChan  chan *ServerSceneSyncCtx      // 服务端场景同步通道
	actionSyncChan       chan *ActionSyncCtx           // action同步通道
	interActionSyncChan  chan *InterActionCtx          // 玩家交互同步通道
	gardenFurnitureChan  chan *SceneGardenFurnitureCtx // 家具通道
	chairSyncChan        chan *ChairSyncCtx            // 交换同步通道
}

func (s *SceneInfo) newChannelInfo(channelId uint32, channelType int) *ChannelInfo {
	info := &ChannelInfo{
		game:                 s.game,
		SceneInfo:            s,
		ChannelId:            channelId,
		channelType:          channelType,
		allPlayer:            make(map[uint32]*ScenePlayer),
		weatherType:          proto.WeatherType_WeatherType_Sunny,
		chatChannel:          newChatChannel(),
		sceneGardenData:      model.GetSceneGardenData(channelId, s.SceneId),
		chaiInfoMap:          make(map[uint32]*proto.ChairInfo),
		doneChan:             make(chan struct{}),
		addScenePlayerChan:   make(chan *ScenePlayer, 10),
		delScenePlayerChan:   make(chan *ScenePlayer, 10),
		addSceneSyncDataChan: make(chan *proto.SceneSyncData, 100),
		sceneSyncDatas:       make([]*proto.SceneSyncData, 100),
		sceneServerDatas:     make(map[uint32]*proto.ServerSceneSyncData),
		serverSceneSyncChan:  make(chan *ServerSceneSyncCtx, 100),
		actionSyncChan:       make(chan *ActionSyncCtx, 100),
		interActionSyncChan:  make(chan *InterActionCtx, 100),
		gardenFurnitureChan:  make(chan *SceneGardenFurnitureCtx, 100),
		chairSyncChan:        make(chan *ChairSyncCtx, 100),
	}

	info.chatChannel.doneChan = info.doneChan
	info.chatChannel.Type = proto.ChatChannelType_ChatChannelType_ChatChannelDefault

	go info.chatChannel.channelMainLoop()
	go info.channelMainLoop()

	return info
}

func (c *ChannelInfo) Close() {
	close(c.doneChan)
}

func (c *ChannelInfo) getAllPlayer() map[uint32]*ScenePlayer {
	if c.allPlayer == nil {
		c.allPlayer = make(map[uint32]*ScenePlayer)
	}
	return c.allPlayer
}

func (c *ChannelInfo) sendAllPlayer(packetId uint32, payloadMsg pb.Message) {
	for _, player := range c.getAllPlayer() {
		if player.NetFreeze { // 网络冻结排除
			continue
		}
		player.Conn.Send(packetId, payloadMsg)
	}
}

func (c *ChannelInfo) sendPlayer(player *ScenePlayer, packetId uint32, payloadMsg pb.Message) {
	player.Conn.Send(packetId, payloadMsg)
}

// 房间主线程
func (c *ChannelInfo) channelMainLoop() {
	syncTimer := time.NewTimer(model.ChannelTick) // 0.2s 同步一次
	defer func() {
		syncTimer.Stop()
		log.Game.Debugf("场景:%v房间:%v退出", c.SceneInfo.SceneId, c.ChannelId)
	}()
	for {
		select {
		case <-syncTimer.C: // 定时同步
			c.channelTick()
			syncTimer.Reset(model.ChannelTick)
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
		case ctx := <-c.interActionSyncChan: // 交互同步
			c.SceneInterActionPlayStatusNotice(ctx)
		case ctx := <-c.gardenFurnitureChan: // 家具通道
			c.SceneGardenFurnitureUpdate(ctx)
		case ctx := <-c.chairSyncChan: // 交互通道
			c.SceneChairSync(ctx)
		case <-c.doneChan:
			return
		}
	}
}

func (c *ChannelInfo) channelTick() {
	// 天气更新 按小时更新
	if weather := c.game.worldTask.Weather(); weather != c.weatherType {
		c.weatherType = weather
		c.SceneWeatherChangeNotice()
	}
	// 时间更新
	if todSeq := c.game.worldTask.TodSeq(); todSeq != c.todSeq {
		c.todSeq = todSeq
		c.serverSceneSync(&ServerSceneSyncCtx{
			ActionType: proto.SceneActionType_SceneActionType_TodUpdate,
		})
	}

	// 场景自动化更新
	// 场景变化同步
	if len(c.sceneSyncDatas) > 0 {
		notice := &proto.PlayerSceneSyncDataNotice{
			Status: proto.StatusCode_StatusCode_Ok,
			Data:   c.sceneSyncDatas,
		}
		c.sendAllPlayer(0, notice)
		c.sceneSyncDatas = make([]*proto.SceneSyncData, 0)
	}
	// 玩家变化同步
	if len(c.sceneServerDatas) > 0 {
		notice := &proto.ServerSceneSyncDataNotice{
			Status: proto.StatusCode_StatusCode_Ok,
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
	if _, ok := list[scenePlayer.UserId]; !ok {
		scenePlayer.channelInfo = c
		list[scenePlayer.UserId] = scenePlayer
	}
	// 通知包
	c.SceneDataNotice(scenePlayer)
	c.serverSceneSync(&ServerSceneSyncCtx{
		ScenePlayer: scenePlayer,
		ActionType:  proto.SceneActionType_SceneActionType_Enter,
	})

	c.chatChannel.addUserChan <- c.game.getChatInfo().getChannelSceneUser(scenePlayer.Player)
	return true
}

func (c *ChannelInfo) delPlayer(scenePlayer *ScenePlayer) {
	list := c.getAllPlayer()
	if _, ok := list[scenePlayer.UserId]; ok { // 移除玩家
		delete(list, scenePlayer.UserId)
	}
	c.serverSceneSync(&ServerSceneSyncCtx{
		ScenePlayer: scenePlayer,
		ActionType:  proto.SceneActionType_SceneActionType_Leave,
	})

	if scenePlayer.UserId != c.ChannelId { // 移除家具
		c.sceneGardenData.RemoveFurniture(scenePlayer.Player, c.ChannelId, 0, false)
	}
	if _, ok := c.chaiInfoMap[scenePlayer.UserId]; ok { // 移除交互
		delete(c.chaiInfoMap, scenePlayer.UserId)
	}
	c.chatChannel.delUserChan <- scenePlayer.UserId
}

// 通知客户端场景信息
func (c *ChannelInfo) SceneDataNotice(scenePlayer *ScenePlayer) {
	notice := &proto.SceneDataNotice{
		Status: proto.StatusCode_StatusCode_Ok,
		Data:   nil,
	}
	defer c.sendPlayer(scenePlayer, 0, notice)
	data := c.GetPbSceneData(scenePlayer)
	if data == nil {
		str, _ := sonic.MarshalString(c)
		log.Game.Errorf("玩家场景信息异常|场景快照:%s", str)
		notice.Status = proto.StatusCode_StatusCode_CantJoinPlayerCurrentSceneChannel
		return
	}
	notice.Data = data
}

// 服务端场景同步上下文
type ServerSceneSyncCtx struct {
	ScenePlayer *ScenePlayer
	ActionType  proto.SceneActionType
	CharacterId uint32 // 需要更新的角色id
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
		Player:     new(proto.ScenePlayer),
	}
	alg.AddList(&playerData.ServerData, serverData)
	switch ctx.ActionType {
	case proto.SceneActionType_SceneActionType_Enter: // 进入场景
		serverData.Player = c.GetPbScenePlayer(ctx.ScenePlayer)
	case proto.SceneActionType_SceneActionType_Leave: // 退出场景
	case proto.SceneActionType_SceneActionType_UpdateEquip: /*更新装备*/
		sceneCharacter := ctx.NewTeamSceneCharacter(serverData)
		ctx.ScenePlayer.CurScene.GetScenePlayerInfo().UpdateCharacterEquip(sceneCharacter)
	case proto.SceneActionType_SceneActionType_UpdateFashion, /*更新服装*/
		proto.SceneActionType_SceneActionType_UpdateTeam,       /*更新队伍*/
		proto.SceneActionType_SceneActionType_UpdateAppearance: /*更新外观*/
		serverData.Player = &proto.ScenePlayer{
			Team: ctx.ScenePlayer.CurScene.GetScenePlayerInfo().GetPbSceneTeam(),
		}
	case proto.SceneActionType_SceneActionType_UpdateNickname: // 更新昵称
		basic, ok := db.GetGameBasic(ctx.ScenePlayer.UserId)
		if !ok {
			log.Game.Errorf("UserId:%v func serverSceneSync 获取玩家基础数据失败:玩家不存在", ctx.ScenePlayer.UserId)
			return
		}
		serverData.Player = &proto.ScenePlayer{
			PlayerId:   ctx.ScenePlayer.UserId,
			PlayerName: basic.NickName,
		}
	case proto.SceneActionType_SceneActionType_TodUpdate: /* 时间更新*/
		serverData.TodTime = c.game.worldTask.TodTime()
	case proto.SceneActionType_SceneActionType_UpdateMusicalItem: // 乐器更新
		ctx.ScenePlayer.UpdateMusicalItem(serverData.Player)
	case proto.SceneActionType_SceneActionType_UpdateCharacterLv, // 角色升级
		proto.SceneActionType_SceneActionType_UpdateCharacterBreakLv, // 角色突破升阶
		proto.SceneActionType_SceneActionType_UpdateCharacterStar:    // 角色升星
		sceneCharacter := ctx.NewTeamSceneCharacter(serverData)
		ctx.ScenePlayer.CurScene.GetScenePlayerInfo().UpdateCharacterBasic(sceneCharacter)
	}
}

func (ctx *ServerSceneSyncCtx) NewTeamSceneCharacter(serverDate *proto.SceneServerData) *proto.SceneCharacter {
	teamInfo := ctx.ScenePlayer.GetTeamModel().GetTeamInfo()
	info := &proto.SceneCharacter{
		CharId: ctx.CharacterId,
	}
	if serverDate.Player.Team == nil {
		serverDate.Player.Team = new(proto.SceneTeam)
	}
	switch ctx.CharacterId {
	case teamInfo.Char1:
		serverDate.Player.Team.Char1 = info
	case teamInfo.Char2:
		serverDate.Player.Team.Char2 = info
	case teamInfo.Char3:
		serverDate.Player.Team.Char3 = info
	default:
		return nil
	}
	return info
}

type SceneGardenFurnitureCtx struct {
	Remove         bool                          // 删除家具
	ScenePlayer    *ScenePlayer                  // 操作玩家
	FurnitureInfo  *proto.FurnitureDetailsInfo   // 添加的家具信息
	FurnitureInfos []*proto.FurnitureDetailsInfo // 覆盖式更新家具
	AllUpdate      bool                          // 是否覆盖更新
	FurnitureId    int64                         // 删除的家具信息
	CharacterId    uint32                        // 摆放/移除的角色
}

// 花园家具更新
func (c *ChannelInfo) SceneGardenFurnitureUpdate(ctx *SceneGardenFurnitureCtx) {
	if ctx.Remove { // 移除
		if ctx.FurnitureId != 0 { // 移除家具
			furnitureInfo := c.sceneGardenData.RemoveFurniture(
				ctx.ScenePlayer.Player,
				c.ChannelId, ctx.FurnitureId, true)
			if furnitureInfo != nil { // 移除家具
				notice := &proto.SceneGardenFurnitureRemoveNotice{
					Status:      proto.StatusCode_StatusCode_Ok,
					FurnitureId: furnitureInfo.FurnitureId,
					ItemId:      furnitureInfo.FurnitureItemId,
					UpdateItems: make([]*proto.ItemDetail, 0),
				}
				c.sendAllPlayer(0, notice)
			}
		}
		if ctx.CharacterId != 0 {
			notice := &proto.GardenPlaceCharacterNotice{
				Status:            proto.StatusCode_StatusCode_Ok,
				RemoveCharacterId: ctx.CharacterId,
			}
			c.sendAllPlayer(0, notice)
		}
	} else if ctx.AllUpdate && ctx.ScenePlayer.UserId == c.ChannelId { // 移除全部
		for _, v := range c.sceneGardenData.GardenFurnitureInfoMap {
			c.sceneGardenData.RemoveFurniture(
				ctx.ScenePlayer.Player,
				c.ChannelId, v.FurnitureId, false)
		}
		mapLen := len(ctx.FurnitureInfos)
		i := 1
		for _, v := range ctx.FurnitureInfos {
			c.sceneGardenData.AddFurniture(
				ctx.ScenePlayer.Player,
				c.ChannelId, v, mapLen == i)
			i++
		}
		c.GardenFurnitureBatchUpdateNotice(ctx.ScenePlayer)
	} else if ctx.FurnitureInfo != nil { // 添加家具
		c.sceneGardenData.AddFurniture(
			ctx.ScenePlayer.Player,
			c.ChannelId,
			ctx.FurnitureInfo,
			true,
		)
		notice := &proto.SceneGardenFurnitureUpdateNotice{
			Status:        proto.StatusCode_StatusCode_Ok,
			FurnitureInfo: ctx.FurnitureInfo,
		}
		c.sendAllPlayer(0, notice)
	} else if ctx.CharacterId != 0 { // 摆放角色
		notice := &proto.GardenPlaceCharacterNotice{
			Status:            proto.StatusCode_StatusCode_Ok,
			Character:         c.sceneGardenData.GetScenePlacedCharacter(ctx.CharacterId),
			RemoveCharacterId: 0,
		}
		c.sendAllPlayer(0, notice)
	}
}

func (c *ChannelInfo) GardenFurnitureBatchUpdateNotice(player *ScenePlayer) {
	notice := &proto.GardenFurnitureBatchUpdateNotice{
		Status:            proto.StatusCode_StatusCode_Ok,
		PlayerId:          player.UserId,
		FurniturePointNum: 0,
		NewFurnitureList:  c.sceneGardenData.NewFurnitureList(),
	}
	c.sendAllPlayer(0, notice)
}

// 交互上下文
type ChairSyncCtx struct {
	SyncMsg  pb.Message
	PlayerId uint32
	ChairId  int64
	SeatId   int32
	IsSit    bool
}

func (c *ChannelInfo) SceneChairSync(ctx *ChairSyncCtx) {
	defer c.sendAllPlayer(0, ctx.SyncMsg)
	if ctx.IsSit { // 坐？
		c.chaiInfoMap[ctx.PlayerId] = &proto.ChairInfo{
			ChairId:  ctx.ChairId,
			SeatId:   ctx.SeatId,
			PlayerId: ctx.PlayerId,
		}
	} else {
		delete(c.chaiInfoMap, ctx.PlayerId)
	}
}

// action同步上下文
type ActionSyncCtx struct {
	ScenePlayer *ScenePlayer
	ActionId    uint32
}

func (c *ChannelInfo) SendActionNotice(ctx *ActionSyncCtx) {
	notice := &proto.SendActionNotice{
		Status:            proto.StatusCode_StatusCode_Ok,
		ActionId:          ctx.ActionId,
		FromPlayerId:      ctx.ScenePlayer.UserId,
		FromPlayerName:    ctx.ScenePlayer.NickName,
		IsStudy:           false,
		EndTime:           0,
		MultipleNeedCount: 0,
	}
	c.sendAllPlayer(0, notice)
}

type InterActionCtx struct {
	ScenePlayer  *ScenePlayer
	ActionStatus *proto.ScenePlayerActionStatus
	PushType     proto.InterActionPushType
}

func (c *ChannelInfo) SceneInterActionPlayStatusNotice(ctx *InterActionCtx) {
	notice := &proto.SceneInterActionPlayStatusNotice{
		Status:       proto.StatusCode_StatusCode_Ok,
		ActionStatus: ctx.ActionStatus,
		PushType:     ctx.PushType,
		PlayerId:     ctx.ScenePlayer.UserId,
	}
	c.sendAllPlayer(0, notice)
}

func (c *ChannelInfo) GetPbSceneData(scenePlayer *ScenePlayer) (info *proto.SceneData) {
	info = &proto.SceneData{
		SceneId:              c.SceneInfo.SceneId,           // ok
		GatherLimits:         make([]*proto.GatherLimit, 0), // ok
		DropItems:            make([]*proto.DropItem, 0),
		Areas:                make([]*proto.AreaData, 0),       // ok
		Collections:          make([]*proto.CollectionData, 0), // ok
		Challenges:           make([]*proto.ChallengeData, 0),
		TreasureBoxes:        make([]*proto.TreasureBoxData, 0), // ok
		Riddles:              make([]*proto.RiddleData, 0),      // TODO 解谜
		Monsters:             make([]*proto.MonsterData, 0),
		EncounterData:        make([]*proto.BattleEncounterData, 0),
		Flags:                make([]*proto.FlagBattleData, 0),
		RegionVoices:         make([]uint32, 0),
		BonFires:             make([]*proto.Bonfire, 0),
		SoccerPosition:       new(proto.SoccerPosition),
		ChairInfoList:        make([]*proto.ChairInfo, 0),         // ok
		Dungeons:             make([]*proto.DungeonData, 0),       // TODO 地牢
		FlagIds:              make([]uint32, 0),                   // ok
		SceneGardenData:      c.sceneGardenData.SceneGardenData(), // ok
		CurrentGatherGroupId: 0,
		Players:              make([]*proto.ScenePlayer, 0), // ok
		ChannelId:            c.ChannelId,                   // ok
		TodTime:              c.game.worldTask.TodTime(),    // ok
		CampFires:            make([]*proto.CampFire, 0),
		WeatherType:          c.weatherType, // ok
		ChannelLabel:         c.ChannelId,   // ok
		FireworksInfo: &proto.FireworksInfo{
			FireworksId:           10005,
			FireworksDurationTime: 60 * 60 * 24,
			FireworksStartTime:    time.Now().Unix(),
		},
		MpBeacons:        make([]*proto.MPBeacon, 0),
		NetworkEvent:     make([]*proto.NetworkEventData, 0),
		PlacedCharacters: c.sceneGardenData.PlacedCharacters(), // ok
		MoonSpots:        make([]*proto.MoonSpotData, 0),
		RoomDecorList:    make([]*proto.RoomDecorData, 0),
	}
	// 添加资源点
	for _, gather := range scenePlayer.GetSceneModel().GetSceneInfo(c.SceneInfo.SceneId).GetGatherLimits() {
		alg.AddList(&info.GatherLimits, gather.GatherLimit())
	}
	// 添加锚点
	for _, area := range scenePlayer.GetSceneModel().GetSceneInfo(c.SceneInfo.SceneId).GetAreaDatas() {
		alg.AddList(&info.Areas, area.AreaData())
	}
	// 添加收集情况
	for _, collectInfo := range scenePlayer.GetSceneModel().GetSceneInfo(c.SceneInfo.SceneId).Collections {
		if len(collectInfo.ItemMap) == 0 {
			continue
		}
		alg.AddList(&info.Collections, collectInfo.CollectionData())
	}
	// 添加宝箱
	for _, treasur := range scenePlayer.GetSceneModel().GetSceneInfo(c.SceneInfo.SceneId).GetTreasurBoxs() {
		alg.AddList(&info.TreasureBoxes, treasur.TreasureBoxData())
	}
	// 添加场景中的玩家
	for _, player := range c.getAllPlayer() {
		alg.AddList(&info.Players, c.GetPbScenePlayer(player))
	}
	// 添加座位信息
	for _, chaiInfo := range c.chaiInfoMap {
		alg.AddList(&info.ChairInfoList, chaiInfo)
	}
	// 副本
	for _, dungeonInfo := range gdconf.GetSceneInfo(c.SceneInfo.SceneId).Info.DungeonInfos {
		alg.AddList(&info.Dungeons, scenePlayer.GetDungeonModel().GetDungeonInfo(uint32(dungeonInfo.ID)).DungeonData())
	}
	// 解锁id TODO 目前直接解锁全部
	for _, flagConfigure := range gdconf.GetSceneFlagConfigure(c.SceneInfo.SceneId) {
		alg.AddLists(&info.FlagIds, uint32(flagConfigure.ID))
	}
	return
}

func (c *ChannelInfo) GetPbScenePlayer(scenePlayer *ScenePlayer) (info *proto.ScenePlayer) {
	basic, ok := db.GetGameBasic(scenePlayer.UserId)
	if !ok {
		log.Game.Errorf("UserId:%v func GetPbScenePlayer 获取玩家基础数据失败:玩家不存在", scenePlayer.UserId)
		return
	}
	info = &proto.ScenePlayer{
		PlayerId:              scenePlayer.UserId,
		PlayerName:            scenePlayer.NickName,
		Team:                  scenePlayer.CurScene.GetScenePlayerInfo().GetPbSceneTeam(),
		Status:                new(proto.ScenePlayerActionStatus),
		FoodBuffIds:           make([]uint32, 0),
		GlobalBuffIds:         make([]uint32, 0),
		IsBirthday:            false,             // 是生日？
		AvatarFrame:           basic.AvatarFrame, // 头像框
		MusicalItemId:         0,                 // ok
		MusicalItemSource:     0,                 // ok
		MusicalItemInstanceId: 0,                 // ok
		AbyssRank:             0,
		PlayingMusicNote:      nil, // ok
		PhoneCase:             0,
		VehicleItemId:         0,
	}
	scenePlayer.UpdateMusicalItem(info) // 赋值音乐物品
	return
}

func (s *ScenePlayer) UpdateMusicalItem(info *proto.ScenePlayer) {
	if info == nil {
		return
	}
	info.MusicalItemId = s.MusicalItemId                 // 音乐物品id
	info.MusicalItemSource = s.MusicalItemSource         // 音乐来源
	info.MusicalItemInstanceId = s.MusicalItemInstanceId // 音乐实例id
	info.PlayingMusicNote = s.PlayingMusicNote           // 演奏音符
}

func (c *ChannelInfo) SceneWeatherChangeNotice() {
	notice := &proto.SceneWeatherChangeNotice{
		Status:      proto.StatusCode_StatusCode_Ok,
		WeatherType: c.weatherType,
	}
	c.sendAllPlayer(0, notice)
}

func (g *Game) SceneActionCharacterUpdate(s *model.Player, t proto.SceneActionType, characterId ...uint32) {
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.channelInfo == nil {
		return
	}
	if len(characterId) == 0 {
		scenePlayer.channelInfo.serverSceneSyncChan <- &ServerSceneSyncCtx{
			ScenePlayer: scenePlayer,
			ActionType:  t,
		}
	} else {
		curTeam := s.GetTeamModel().GetTeamInfo()
		for _, id := range characterId {
			if id == curTeam.Char1 || id == curTeam.Char2 || id == curTeam.Char3 {
				scenePlayer.channelInfo.serverSceneSyncChan <- &ServerSceneSyncCtx{
					ScenePlayer: scenePlayer,
					ActionType:  t,
					CharacterId: id,
				}
			}
		}
	}
}
