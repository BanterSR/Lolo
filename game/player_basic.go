package game

import (
	"strconv"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/config"
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
		IsReconnect:      req.IsReconnect, // 是否重新连接
		ReconnectSuccess: req.IsReconnect, // 重新连接是否成功
	}
	// 重复登录检查
	s := g.GetUser(userId)
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
		basic, err := db.GetGameBasic(userId)
		if err != nil {
			log.Game.Warnf("UserId:%v 登录失败,获取玩家基础数据失败:%s", userId, err.Error())
			rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
			return
		}
		s = &model.Player{
			UserId:    userId,
			LoginUUID: uuid,
			NickName:  basic.NickName,
			Created:   basic.CreatedAt,
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
			if config.GetMode() == config.ModeDev {
				s.AllItemModel()
			}
		}
		g.userMap[userId] = s
	}
login:
	s.Init(conn)

	basic, err := db.GetGameBasic(userId)
	if err != nil {
		log.Game.Warnf("UserId:%v 登录失败,获取玩家基础数据失败:%s", s.UserId, err.Error())
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		if req.IsReconnect {
			g.loginGame(s)
		}
	}()
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
		rsp.SceneId = scenePlayer.SceneId
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
		basic, err := db.GetGameBasic(s.UserId)
		if err != nil {
			log.Game.Warnf("UserId:%v 登录失败,获取玩家基础数据失败:%s", s.UserId, err.Error())
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
		rsp.CreateTime = uint32(s.Created.Unix())
		rsp.Birthday = basic.Birthday
		rsp.IsHideBirthday = basic.IsHideBirthday
		rsp.PhoneBackground = basic.PhoneBackground
		rsp.Appearance = model.GetPlayerAppearance(s.UserId)
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
		rsp.SceneId = scenePlayer.SceneId
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

func (g *Game) AcceptQuest(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AcceptQuestReq)
	rsp := &proto.AcceptQuestRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Quest: &proto.Quest{
			QuestId:       req.QuestId,
			Conditions:    make([]*proto.Condition, 0),
			Status:        proto.QuestStatus_QuestStatus_InProgress,
			CompleteCount: 0,
			BonusTimes:    0,
			ActivityId:    0,
		},
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetAchieveOneGroup(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetAchieveOneGroupReq)
	rsp := &proto.GetAchieveOneGroupRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		CurrentGroupAchieveInfo: &proto.OneGroupAchieveInfo{
			GroupId:              0,
			RewardedAchieveIdLst: make([]uint32, 0),
			AchieveLst:           make([]*proto.Achieve, 0),
			FinishAchieveLst:     make([]*proto.FinishAchieveInfo, 0),
		},
		IsReward: false,
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetAchieveGroupList(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetAchieveGroupListReq)
	rsp := &proto.GetAchieveGroupListRsp{
		Status:             proto.StatusCode_StatusCode_Ok,
		RewardedGroupIdLst: make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
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
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) SelfIntervalInit(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.SelfIntervalInitReq)
	rsp := &proto.SelfIntervalInitRsp{
		Status:     proto.StatusCode_StatusCode_Ok,
		IntervalId: 0,
		EndTime:    0,
		IsStart:    false,
		Interval: &proto.IntervalInfo{
			IntervalId: 0,
			FinishTime: 0,
			PlayerId:   s.UserId,
			CreateTime: 0,
			Member:     make([]*proto.FriendIntervalInfo, 0),
		},
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) BossRushInfo(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.BossRushInfoReq)
	rsp := &proto.BossRushInfoRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Info: &proto.BossRushInfo{
			SeasonId:          1002,
			BestTotalScore:    0,
			TotalRankRatio:    0,
			CurrentStageIndex: 0,
			StageInfos:        make([]*proto.BossRushStageInfo, 0),
			StartTime:         1762600320,
			EndTime:           1762620320,
			ShowRankTime:      0,
			ChallengeEndTime:  0,
			UsedCharacters:    make([]uint32, 0),
		},
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) PlayerVitality(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.PlayerVitalityReq)
	rsp := &proto.PlayerVitalityRsp{
		Status:         proto.StatusCode_StatusCode_Ok,
		VitalityBuyNum: 0,
		Items:          make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GemDuelInfo(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GemDuelInfoReq)
	rsp := &proto.GemDuelInfoRsp{
		Status:                 proto.StatusCode_StatusCode_Ok,
		GameData:               new(proto.GemDuelGameData),
		Characters:             make([]*proto.GemDuelCharacterData, 0),
		Items:                  make([]*proto.GemDuelItem, 0),
		BuyStaminaCount:        0,
		Teams:                  make([]*proto.GemDuelTeamData, 0),
		PassedMainDungeons:     make([]uint32, 0),
		Arena:                  new(proto.GemDuelArenaData),
		Quest:                  new(proto.GemDuelQuestDetail),
		MonthCardOverDueTime:   0,
		MonthCardRewardDays:    0,
		MonthCardIsRewardToday: false,
		GrowthFund:             make([]*proto.GemDuelGrowthFund, 0),
		PlayerLevel:            0,
		PlayerExp:              0,
	}
	defer g.send(s, msg.PacketId, rsp)
}

func (g *Game) GemDuelGachaList(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GemDuelGachaListReq)
	rsp := &proto.GemDuelGachaListRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Gachas: make([]*proto.GemDuelGachaInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
}
