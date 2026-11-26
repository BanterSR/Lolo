package game

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) PlayerLogin(conn ofnet.Conn, userId uint32, uuid string, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerLoginReq)
	rsp := &proto.PlayerLoginRsp{
		Status:           proto.StatusCode_StatusCode_Ok,
		IsReconnect:      req.IsReconnect,
		ReconnectSuccess: req.IsReconnect, // 重新连接是否成功
	}
	var s *model.Player
	defer func() {
		if s != nil && s.Conn != nil {
			g.send(s, msg.PacketId, rsp)
		} else {
			conn.Send(msg.PacketId, rsp)
		}
		if rsp.Status == proto.StatusCode_StatusCode_Ok && rsp.IsReconnect && s != nil {
			g.loginGame(s)
		}
	}()
	// 重复登录检查
	s = g.GetUser(userId)
	if req.IsReconnect { // 是否重新连接 无玩家缓存时拒绝重新连接
		rsp.IsReconnect = s != nil
		rsp.ReconnectSuccess = s != nil

		if !rsp.IsReconnect {
			conn.Send(msg.PacketId, rsp)
			return
		}
	}
	if s != nil {
		if s.LoginUUID == uuid { // uuid相同代表同一连接，直接跳出初始化
			goto login
		}
		g.offlinePlayer(s,
			proto.PlayerOfflineReason_PlayerOfflineReason_AnotherLogin) // 下线老玩家
	} else {
		// 新登录 判断是否满人了

		// 拉取数据
		dbUser, err := db.GetOFGameByUserId(userId)
		if err != nil {
			rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
			log.Game.Warnf("数据库拉取玩家:%v数据失败:%s", userId, err.Error())
			return
		}
		basic, err := db.GetGameBasicOrCreate(userId)
		if err != nil {
			log.Game.Warnf("UserId:%v 登录失败,获取玩家基础数据失败:%s", userId, err.Error())
			rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
			return
		}
		s = &model.Player{
			UserId:   userId,
			NickName: basic.NickName,
			Created:  basic.CreatedAt,
		}
		if len(dbUser.BinData) != 0 {
			if err := sonic.Unmarshal(dbUser.BinData, s); err != nil {
				rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
				log.Game.Warnf("玩家:%v数据序列化失败:%s", userId, err.Error())
				return
			}
		} else {
			// newPlayer
			characterInfo := s.AddCharacter(101001)
			if characterInfo == nil {
				log.Game.Errorf("初始化默认角色:%v失败", 101001)
			}
			for _, characterId := range gdconf.GetConstant().DefaultCharacter {
				s.AddCharacter(characterId)
			}
			s.GetItemModel().InitItem()
			s.AcceptFirstMainQuest()
			if err := s.SavePlayerNow(); err != nil {
				log.Game.Errorf("UserId:%v 新号初始化落库失败:%s", userId, err.Error())
			}
		}
		g.userMap[userId] = s
	}
login:
	s.Init(conn, uuid)

	basic, err := db.GetGameBasicOrCreate(userId)
	if err != nil {
		log.Game.Warnf("UserId:%v 登录失败,获取玩家基础数据失败:%s", s.UserId, err.Error())
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	// pack
	{
		rsp.ClientSeqId = msg.SeqId
		rsp.ServerSeqId = s.GetSeqId()
	}
	// 基础信息
	{
		rsp.PlayerName = basic.NickName
		rsp.RegisterTime = uint32(s.Created.Unix())
		rsp.AnalysisAccountId = strconv.Itoa(int(s.UserId))
	}
	// 加入房间
	scenePlayer := g.getWordInfo().addScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_SceneChannelNotExist
		return
	}
	// 场景
	{
		rsp.SceneId = scenePlayer.CurScene.GetSceneId()
		rsp.ChannelId = scenePlayer.ChannelId
	}
	// 其他信息
	{
		rsp.ServerTimeMs = time.Now().UnixMilli()
		rsp.RegionName = "cn_prod_main"
		rsp.PlayerAgeRange = 0
		rsp.Tags = 0
		rsp.ServerTimeZone = 28800
		rsp.ClientLogServerToken = "114514"
	}
	log.Game.Infof("UserId:%v Name:%s IP:%s 设备:%s 系统:%s 登录成功!",
		s.UserId, rsp.PlayerName, req.Ip, req.DeviceModel, req.OsVer)
}

func (g *Game) PlayerMainData(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PlayerMainDataRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.loginGame(s)
	}()
	// 基础信息
	{
		basic, ok := db.GetGameBasic(s.UserId)
		if !ok {
			log.Game.Warnf("UserId:%v 登录失败,获取玩家基础数据失败 玩家不存在", s.UserId)
			rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
			return
		}
		rsp.PlayerId = s.UserId
		rsp.PlayerLabel = s.UserId // 玩家标签
		rsp.PlayerName = basic.NickName
		rsp.Level = basic.Level
		rsp.Sign = basic.Sign
		rsp.Exp = basic.Exp
		rsp.Head = basic.Head
		rsp.Sex = basic.Sex
		rsp.CreateTime = uint32(s.Created.Unix())
		rsp.Birthday = basic.Birthday
		rsp.IsHideBirthday = basic.IsHideBirthday
		rsp.PhoneBackground = basic.PhoneBackground
		if profile := model.GetPlayerProfile(s); profile != nil {
			rsp.PhoneCase = profile.PhoneCase
			rsp.RewardLevels = profile.RewardedPlayerLevels()
		}
		rsp.Appearance = model.GetPlayerAppearance(s.UserId)
		rsp.WorldLevel = s.GetSceneModel().GetWorldLevel()
		rsp.MonthCardOverDueTime = 0
		rsp.MonthCardRewardDays = 0
		rsp.PetInstanceId = s.GetSceneModel().CurPetInstanceId
	}
	// 已获得的角色
	{
		rsp.Characters = s.GetCharacterModel().GetAllPbCharacter()
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil {
		rsp.Status = proto.StatusCode_StatusCode_SceneChannelIsFull
		return
	}
	// 场景
	{
		rsp.ChannelId = scenePlayer.ChannelId
		rsp.ChannelLabel = scenePlayer.ChannelId // 房间标签
		rsp.SceneId = scenePlayer.CurScene.GetSceneId()
	}
	// 队伍
	{
		rsp.Team = s.GetTeamModel().GetTeamInfo().Team()
	}
	// buff
	{
		rsp.PlayerBuffs = make([]*proto.PlayerBuff, 0)
	}
	// 其他
	{
		rsp.PlayerDropRateInfo = s.GetPbPlayerDropRateInfo()
		rsp.QuestDetail = s.GetQuestDetail()
		rsp.QuestionnaireInfo = s.GetPlayerQuestionnaireInfo()
		rsp.UnlockFunctions = s.GetUnlockFunctions()
		rsp.PlacedCharacters = s.GetCharacterModel().GetPlacedCharacters() // 摆放的角色列表
		rsp.FurnitureItemInfo = s.GetItemModel().FurnitureItemInfo()       // 已摆放的家具
		rsp.DailyTask = &proto.PlayerDailyTask{
			Tasks:             make(map[uint32]uint32),
			TodayConverted:    0,
			ExchangeTimesLeft: 0,
		}
	}
}

func (g *Game) loginGame(s *model.Player) {
	// 资源点采集
	g.GatherSceneLimitRecoveryNotice(s)
	// 物品
	g.AllPackNotice(s)
	// 祝福树
	g.BlessTreeNotice(s)
	// 进入房间
	g.getWordInfo().joinSceneChannel(s)
	// 初始化聊天
	g.chatInit(s)
	g.send(s, 0, &proto.GmNotice{
		Status: proto.StatusCode_StatusCode_Ok,
		Notice: alg.GmNotice,
	})
}

func (g *Game) PlayerPing(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerPingReq)
	rsp := &proto.PlayerPingRsp{
		Status:       proto.StatusCode_StatusCode_Ok,
		ClientTimeMs: req.ClientTimeMs,
		ServerTimeMs: time.Now().UnixMilli(),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ChangeNickName(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeNickNameReq)
	rsp := &proto.ChangeNickNameRsp{
		Status:          proto.StatusCode_StatusCode_Ok,
		NickName:        "",
		Items:           make([]*proto.ItemDetail, 0),
		RenameAllowTime: 0,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdateNickname)
	}()
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.NickName = req.NickName
		basic.Birthday = req.Birthday
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v 修改基础信息失败:%s", s.UserId, err.Error())
		return
	}
	s.NickName = req.NickName
	rsp.NickName = s.NickName
}

func (g *Game) UnlockHeadList(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.UnlockHeadListReq)
	rsp := &proto.UnlockHeadListRsp{
		Status: 0,
		Heads:  s.GetItemModel().GetHeads(),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) UpdatePlayerAppearance(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UpdatePlayerAppearanceReq)
	rsp := &proto.UpdatePlayerAppearanceRsp{
		Status:     proto.StatusCode_StatusCode_Ok,
		Appearance: nil,
	}
	defer func() {
		rsp.Appearance = model.GetPlayerAppearance(s.UserId)
		g.send(s, msg.PacketId, rsp)
	}()
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.Pendant = req.Appearance.Pendant
		basic.AvatarFrame = req.Appearance.AvatarFrame
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v func db.UpGameBasic err:%v", s.UserId, err)
	}
}

func (g *Game) ChangePhoneBackground(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangePhoneBackgroundReq)
	rsp := &proto.ChangePhoneBackgroundRsp{
		Status:          proto.StatusCode_StatusCode_Ok,
		PhoneBackground: req.PhoneBackground,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.PhoneBackground = req.PhoneBackground
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v func db.UpGameBasic err:%v", s.UserId, err)
	}
}

func (g *Game) ChangeIsHideBirthday(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.ChangeIsHideBirthdayReq)
	rsp := &proto.ChangeIsHideBirthdayRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.IsHideBirthday = !basic.IsHideBirthday
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v func db.UpGameBasic err:%v", s.UserId, err)
	}
}

func (g *Game) ChangeHideType(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeHideTypeReq)
	rsp := &proto.ChangeHideTypeRsp{
		Status:    proto.StatusCode_StatusCode_Ok,
		HideValue: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		if basic.HideValue&uint32(req.HideType) != 0 {
			basic.HideValue &^= uint32(req.HideType)
		} else {
			basic.HideValue |= uint32(req.HideType)
		}
		rsp.HideValue = basic.HideValue
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v func db.UpGameBasic err:%v", s.UserId, err)
	}
}

func (g *Game) ChangeSign(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeSignReq)
	rsp := &proto.ChangeSignRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Sign:   req.Sign,
	}
	defer g.send(s, msg.PacketId, rsp)
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.Sign = req.Sign
		return true
	})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v func db.UpGameBasic err:%v", s.UserId, err)
	}
}

func (g *Game) GamePlayReward(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GamePlayRewardReq)
	rsp := &proto.GamePlayRewardRsp{
		Status:                 proto.StatusCode_StatusCode_Ok,
		DynamicTreasureBoxInfo: nil,
		Items:                  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetAchieveOneGroup(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetAchieveOneGroupReq)
	rsp := &proto.GetAchieveOneGroupRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		CurrentGroupAchieveInfo: &proto.OneGroupAchieveInfo{
			GroupId:              req.GroupId,
			RewardedAchieveIdLst: make([]uint32, 0),
			AchieveLst:           make([]*proto.Achieve, 0),
			FinishAchieveLst:     make([]*proto.FinishAchieveInfo, 0),
		},
	}
	defer g.send(s, msg.PacketId, rsp)

	group := gdconf.GetAchieveQuestGroup(req.GroupId)
	if group == nil {
		for _, candidate := range gdconf.GetAchieveQuestGroups() {
			for _, item := range candidate.GetAchieveQuestGroupInfo() {
				if item != nil && uint32(item.GetGroupID()) == req.GroupId {
					group = candidate
					break
				}
			}
			if group != nil {
				break
			}
		}
	}
	if group == nil || len(group.GetAchieveQuestGroupInfo()) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	archive := s.GetArchive()
	for _, item := range group.GetAchieveQuestGroupInfo() {
		if item == nil {
			continue
		}
		achieveID := uint32(item.GetAchieveConditionID())
		count := uint64(0)
		if profile.CompletedAchievements[achieveID] {
			count = 1
			rsp.CurrentGroupAchieveInfo.FinishAchieveLst = append(
				rsp.CurrentGroupAchieveInfo.FinishAchieveLst,
				&proto.FinishAchieveInfo{AchieveId: achieveID},
			)
		}
		rsp.CurrentGroupAchieveInfo.AchieveLst = append(
			rsp.CurrentGroupAchieveInfo.AchieveLst,
			&proto.Achieve{AchieveId: achieveID, Count: count},
		)
		if archive.GetArchiveValue("achieveQuestReward:"+strconv.Itoa(int(achieveID))) != "" {
			rsp.CurrentGroupAchieveInfo.RewardedAchieveIdLst = append(
				rsp.CurrentGroupAchieveInfo.RewardedAchieveIdLst, achieveID,
			)
		}
	}
}

func (g *Game) GetAchieveGroupList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GetAchieveGroupListRsp{
		Status:             proto.StatusCode_StatusCode_Ok,
		RewardedGroupIdLst: make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	const prefix = "achieveGroupReward:"
	for key := range s.GetArchive().GetArchiveMap() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(key, prefix), 10, 32)
		if err == nil {
			rsp.RewardedGroupIdLst = append(rsp.RewardedGroupIdLst, uint32(id))
		}
	}
	sort.Slice(rsp.RewardedGroupIdLst, func(i, j int) bool {
		return rsp.RewardedGroupIdLst[i] < rsp.RewardedGroupIdLst[j]
	})
}

func (g *Game) GenericGameA(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GenericGameAReq)
	rsp := &proto.GenericGameARsp{
		Status:       proto.StatusCode_StatusCode_Ok,
		GenericMsgId: req.GenericMsgId,
		Params:       make([]*proto.CommonParam, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GenericGameB(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GenericGameBReq)
	rsp := &proto.GenericGameBRsp{
		Status:       proto.StatusCode_StatusCode_Ok,
		GenericMsgId: req.GenericMsgId,
		Params:       make([]*proto.CommonParam, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) ManualList(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.ManualListReq)
	rsp := &proto.ManualListRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Flags:  make([]*proto.ManualFlag, 0),
	}
	add := func(flagID uint32) {
		if flagID != 0 {
			alg.AddLists(&rsp.Flags, &proto.ManualFlag{
				FlagId:               flagID,
				Status:               proto.RewardStatus_RewardStatus_Reward,
				DungeonTaskFinishNum: 1,
			})
		}
	}
	for _, item := range gdconf.GetManualItemMap() {
		add(uint32(item.FlagID))
	}
	for _, item := range gdconf.GetManualDungeonItemMap() {
		add(uint32(item.FlagID))
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) SelfIntervalInit(s *model.Player, msg *alg.GameMsg) {
	exploreID, finish := g.intervalStartTime(s)
	rsp := &proto.SelfIntervalInitRsp{
		Status:     proto.StatusCode_StatusCode_Ok,
		IntervalId: exploreID,
		EndTime:    finish,
		IsStart:    finish > 0,
		Interval: &proto.IntervalInfo{
			IntervalId: exploreID,
			FinishTime: finish,
			PlayerId:   s.UserId,
			CreateTime: 0,
			Member:     make([]*proto.FriendIntervalInfo, 0),
		},
	}
	if finish > 0 {
		rsp.Interval.CreateTime = finish - int64(8*time.Hour/time.Second)
		rsp.Interval.Member = append(rsp.Interval.Member, &proto.FriendIntervalInfo{
			PlayerId: s.UserId,
			NickName: s.NickName,
			JoinTime: rsp.Interval.CreateTime,
			IsReward: false,
		})
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) BossRushInfo(s *model.Player, msg *alg.GameMsg) {
	const seasonID uint32 = 1002
	archive := s.GetArchive()
	stageIndex := int32(0)
	if value := archive.GetArchiveValue(g.bossRushKey(s, seasonID, "stage")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			stageIndex = int32(parsed)
		}
	}
	bestScore, _ := strconv.ParseUint(archive.GetArchiveValue(g.bossRushKey(s, seasonID, "best")), 10, 64)
	startTime, _ := strconv.ParseInt(archive.GetArchiveValue(g.bossRushKey(s, seasonID, "start")), 10, 64)

	info := &proto.BossRushInfo{
		SeasonId:          seasonID,
		BestTotalScore:    bestScore,
		CurrentStageIndex: stageIndex,
		StageInfos:        make([]*proto.BossRushStageInfo, 0),
		StartTime:         startTime,
		UsedCharacters:    make([]uint32, 0),
	}
	if startTime > 0 {
		info.EndTime = startTime + int64(24*time.Hour/time.Second)
	}
	if value := archive.GetArchiveValue(g.bossRushKey(s, seasonID, "stage")); value != "" {
		info.StageInfos = append(info.StageInfos, &proto.BossRushStageInfo{
			StageIndex: stageIndex,
			BestScore:  bestScore,
			IsPassed:   bestScore > 0,
		})
	}
	if team := s.GetTeamModel().GetTeamInfo(); team != nil {
		for _, id := range []uint32{team.Char1, team.Char2, team.Char3} {
			if id != 0 {
				info.UsedCharacters = append(info.UsedCharacters, id)
			}
		}
	}
	g.send(s, msg.PacketId, &proto.BossRushInfoRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Info:   info,
	})
}

func (g *Game) PlayerVitality(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PlayerVitalityRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Items:  make([]*proto.ItemDetail, 0),
	}
	if value, err := strconv.ParseUint(s.GetArchive().GetArchiveValue("vitality:buy"), 10, 32); err == nil {
		rsp.VitalityBuyNum = uint32(value)
	}
	for _, info := range s.GetItemModel().GetItemBaseMap() {
		if info == nil {
			continue
		}
		switch info.ItemType {
		case proto.EBagItemTag_EBagItemTag_Vitality, proto.EBagItemTag_EBagItemTag_Energy:
			alg.AddList(&rsp.Items, info.ItemDetail())
		}
	}
	g.send(s, msg.PacketId, rsp)
}

type treasureIllustrationReward struct {
	Type proto.EArchaeologyType
	ID   uint32
}

// treasureIllustrationKey 解析 TreasureIllustrationReward 写入的
// "treasureIllustration:<type>:<id>" 键。
func treasureIllustrationKey(key string) (treasureIllustrationReward, bool) {
	const prefix = "treasureIllustration:"
	if !strings.HasPrefix(key, prefix) {
		return treasureIllustrationReward{}, false
	}
	parts := strings.Split(strings.TrimPrefix(key, prefix), ":")
	if len(parts) != 2 {
		return treasureIllustrationReward{}, false
	}
	typ, err1 := strconv.ParseUint(parts[0], 10, 32)
	id, err2 := strconv.ParseUint(parts[1], 10, 32)
	if err1 != nil || err2 != nil || id == 0 {
		return treasureIllustrationReward{}, false
	}
	return treasureIllustrationReward{Type: proto.EArchaeologyType(typ), ID: uint32(id)}, true
}

func (g *Game) TreasureIllustrationInfo(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.TreasureIllustrationInfoRsp{
		Status:         proto.StatusCode_StatusCode_Ok,
		UtensilReward:  make([]uint32, 0),
		BookReward:     make([]uint32, 0),
		Achieves:       make([]*proto.TreasureAchieve, 0),
		TreasureReward: make([]uint32, 0),
		ShovelReward:   make([]uint32, 0),
	}
	achieved := make(map[proto.EArchaeologyType][]uint32)
	for key := range s.GetArchive().GetArchiveMap() {
		rewarded, ok := treasureIllustrationKey(key)
		if !ok {
			continue
		}
		switch rewarded.Type {
		case proto.EArchaeologyType_EArchaeologyType_Book:
			rsp.BookReward = append(rsp.BookReward, rewarded.ID)
		case proto.EArchaeologyType_EArchaeologyType_TreasureMap:
			rsp.TreasureReward = append(rsp.TreasureReward, rewarded.ID)
		case proto.EArchaeologyType_EArchaeologyType_Utensil:
			rsp.UtensilReward = append(rsp.UtensilReward, rewarded.ID)
		case proto.EArchaeologyType_EArchaeologyType_Shovel:
			rsp.ShovelReward = append(rsp.ShovelReward, rewarded.ID)
		default:
			continue
		}
		achieved[rewarded.Type] = append(achieved[rewarded.Type], rewarded.ID)
	}
	sort.Slice(rsp.BookReward, func(i, j int) bool { return rsp.BookReward[i] < rsp.BookReward[j] })
	sort.Slice(rsp.TreasureReward, func(i, j int) bool { return rsp.TreasureReward[i] < rsp.TreasureReward[j] })
	sort.Slice(rsp.UtensilReward, func(i, j int) bool { return rsp.UtensilReward[i] < rsp.UtensilReward[j] })
	sort.Slice(rsp.ShovelReward, func(i, j int) bool { return rsp.ShovelReward[i] < rsp.ShovelReward[j] })
	for typ, ids := range achieved {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		achieves := make([]*proto.Achieve, 0, len(ids))
		for _, id := range ids {
			achieves = append(achieves, &proto.Achieve{AchieveId: id, Count: 1})
		}
		rsp.Achieves = append(rsp.Achieves, &proto.TreasureAchieve{
			AchieveType:                  typ,
			TreasureAchieve_:             achieves,
			RewardedTreasureAchieveIdLst: ids,
		})
	}
	g.send(s, msg.PacketId, rsp)
}
