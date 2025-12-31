package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
)

type HandlerFunc func(s *model.Player, msg *alg.GameMsg)

func (g *Game) newRouter() {
	g.handlerFuncRouteMap = map[uint32]HandlerFunc{
		// cmd.PlayerLoginReq: g.PlayerLogin 登录第二个包 仅提示用
		// 基础包
		cmd.PlayerPingReq: g.PlayerPing, // ping
		// 玩家基础信息
		cmd.PlayerMainDataReq:         g.PlayerMainData,         // 获取玩家信息
		cmd.ChangeNickNameReq:         g.ChangeNickName,         // 修改玩家昵称和生日
		cmd.UnlockHeadListReq:         g.UnlockHeadList,         // 解锁头像列表
		cmd.UpdatePlayerAppearanceReq: g.UpdatePlayerAppearance, // 更新玩家外观
		// 场景
		cmd.PlayerSceneRecordReq:          g.PlayerSceneRecord,          // 玩家场景同步器
		cmd.SendActionReq:                 g.SendAction,                 // 场景自动化同步器
		cmd.ChangeSceneChannelReq:         g.ChangeSceneChannel,         // 切换场景/房间
		cmd.GenericSceneBReq:              g.GenericSceneB,              // 获取房间中的天气
		cmd.SceneInterActionPlayStatusReq: g.SceneInterActionPlayStatus, // 同步玩家交互请求
		cmd.GetGardenInfoReq:              g.GetGardenInfo,              // 获取花园信息请求
		cmd.HandingFurnitureReq:           g.HandingFurniture,           // 搬起家具请求
		cmd.PlaceFurnitureReq:             g.PlaceFurniture,             // 摆放家具请求
		cmd.TakeOutHandingFurnitureReq:    g.TakeOutHandingFurniture,    // 回收家具请求
		cmd.TakeOutFurnitureReq:           g.TakeOutFurniture,           // 拿起家具请求
		cmd.SceneSitVehicleReq:            g.SceneSitVehicle,            // 上下车请求
		cmd.ChangeMusicalItemReq:          g.ChangeMusicalItem,          // 切换音乐源请求
		cmd.PlayMusicNoteReq:              g.PlayMusicNote,              // 演奏请求
		// 花园
		cmd.SwitchGardenStatusReq:           g.SwitchGardenStatus,           // 更新花园设置
		cmd.GardenLikeRecordReq:             g.GardenLikeRecord,             // 花园点赞记录
		cmd.GardenFurnitureSchemeReq:        g.GardenFurnitureScheme,        // 获取预设花园请求
		cmd.GardenSchemeFurnitureListReq:    g.GardenSchemeFurnitureList,    // 获取预设花园家具列表
		cmd.GardenFurnitureSaveReq:          g.GardenFurnitureSave,          // 将花园保存到预设
		cmd.GardenFurnitureRemoveAllReq:     g.GardenFurnitureRemoveAll,     // 清空全部家具
		cmd.GardenFurnitureSchemeSetNameReq: g.GardenFurnitureSchemeSetName, // 预设重命名
		cmd.GardenFurnitureApplySchemeReq:   g.GardenFurnitureApplyScheme,   // 应用预设
		cmd.GardenPlaceCharacterReq:         g.GardenPlaceCharacter,         // 摆放角色
		// 照片墙
		cmd.PhotoShareSearchReq: g.PhotoShareSearch, // 获取照片墙大厅请求
		// 队伍
		cmd.UpdateTeamReq: g.UpdateTeam, // 更新队伍
		// 物品
		cmd.GetWeaponReq:              g.GetWeapon,              // 获取武器列表-根据type
		cmd.GetArmorReq:               g.GetArmor,               // 获取盔甲列表-根据type
		cmd.GetPosterReq:              g.GetPoster,              // 获取海报列表
		cmd.PosterIllustrationListReq: g.PosterIllustrationList, // 海报插头列表
		// 卡池
		cmd.GachaListReq:          g.GachaList,          // 获取卡池信息
		cmd.GachaRecordReq:        g.GachaRecord,        // 获取抽卡记录
		cmd.GachaReq:              g.Gacha,              // 抽卡，启动！
		cmd.OptionalUpPoolItemReq: g.OptionalUpPoolItem, // 设置保底物品
		// 角色
		cmd.GetCharacterAchievementListReq: g.GetCharacterAchievementList, // 获取角色成就情况
		cmd.CharacterLevelUpReq:            g.CharacterLevelUp,            // 角色升级
		cmd.CharacterLevelBreakReq:         g.CharacterLevelBreak,         // 角色突破
		cmd.OutfitPresetUpdateReq:          g.OutfitPresetUpdate,          // 保存预设装扮
		cmd.CharacterEquipUpdateReq:        g.CharacterEquipUpdate,        // 角色更新装备
		cmd.UpdateCharacterAppearanceReq:   g.UpdateCharacterAppearance,   // 更新角色外观
		cmd.CharacterGatherWeaponUpdateReq: g.CharacterGatherWeaponUpdate, // 更新手持工具请求
		// 信息记录
		cmd.GetArchiveInfoReq: g.GetArchiveInfo, // 获取记录的信息
		cmd.SetArchiveInfoReq: g.SetArchiveInfo, // 设置信息
		// 好友
		cmd.FriendReq:       g.Friend,       // 获取好友聚合请求
		cmd.FriendAddReq:    g.FriendAdd,    // 添加好友请求
		cmd.FriendHandleReq: g.FriendHandle, // 处理好友申请
		cmd.FriendDelReq:    g.FriendDel,    // 删除好友关系
		cmd.FriendBlackReq:  g.FriendBlack,  // 拉黑玩家
		// 聊天
		cmd.PrivateChatMsgRecordReq: g.PrivateChatMsgRecord, // 获取私聊聊天记录
		cmd.SendChatMsgReq:          g.SendChatMsg,          // 发送聊天消息
		cmd.ChangeChatChannelReq:    g.ChangeChatChannel,    // 切换聊天房间
		// 星云树
		cmd.GetCollectMoonInfoReq: g.GetCollectMoonInfo, // 获取星云树信息
		cmd.CollectMoonReq:        g.CollectMoon,        // 收集月亮请求
		// 商店
		cmd.ShopInfoReq:       g.ShopInfo,       // 获取商店信息
		cmd.ShopBuyReq:        g.ShopBuy,        // 商店购买请求
		cmd.CreatePayOrderReq: g.CreatePayOrder, // 创建支付订单请求
		// 战斗
		cmd.BattleEncounterInfoReq: g.BattleEncounterInfo, // 获取战斗遭遇信息

		cmd.PlayerVitalityReq:        g.PlayerVitality,
		cmd.BossRushInfoReq:          g.BossRushInfo,
		cmd.FriendIntervalInitReq:    g.FriendIntervalInit,
		cmd.SelfIntervalInitReq:      g.SelfIntervalInit,
		cmd.ExploreInitReq:           g.ExploreInit,
		cmd.NpcTalkReq:               g.NpcTalk,  // npc对话
		cmd.TutorialReq:              g.Tutorial, // 开始教程
		cmd.ChallengeFriendRankReq:   g.ChallengeFriendRank,
		cmd.AbilityBadgeListReq:      g.AbilityBadgeList,
		cmd.SceneProcessListReq:      g.SceneProcessList,
		cmd.WishListByFriendIdReq:    g.WishListByFriendId,
		cmd.GetLifeInfoReq:           g.GetLifeInfo,
		cmd.GetMailsReq:              g.GetMails,
		cmd.GetAchieveOneGroupReq:    g.GetAchieveOneGroup,
		cmd.GetAchieveGroupListReq:   g.GetAchieveGroupList,
		cmd.GenericGameAReq:          g.GenericGameA,
		cmd.GenericGameBReq:          g.GenericGameB,
		cmd.GetCollectItemIdsReq:     g.GetCollectItemIds,
		cmd.ManualListReq:            g.ManualList,
		cmd.PlayerAbilityListReq:     g.PlayerAbilityList,     //
		cmd.WorldLevelAchieveListReq: g.WorldLevelAchieveList, //
		cmd.SupplyBoxInfoReq:         g.SupplyBoxInfo,         //
		cmd.GetAllCharacterEquipReq:  g.GetAllCharacterEquip,  //
		cmd.GamePlayRewardReq:        g.GamePlayReward,        //
		cmd.AcceptQuestReq:           g.AcceptQuest,           //
		cmd.GemDuelInfoReq:           g.GemDuelInfo,
	}
}

func (g *Game) RouteHandle(conn ofnet.Conn, userId uint32, msg *alg.GameMsg) {
	if msg.MsgId == cmd.PlayerLoginReq {
		g.PlayerLogin(conn, userId, msg)
		return
	}
	handlerFunc, ok := g.handlerFuncRouteMap[msg.MsgId]
	if !ok {
		log.Game.Errorf("no route for msg, cmdId: %v name:%s", msg.MsgId, cmd.Get().GetCmdNameByCmdId(msg.MsgId))
		return
	}
	player := g.GetUser(userId)
	if player == nil {
		log.Game.Errorf("player is nil, userId: %v", userId)
		return
	}
	if !player.Online {
		log.Game.Errorf("player not online, userId: %v", userId)
		return
	}
	if player.NetFreeze {
		return
	}
	handlerFunc(player, msg)
	player.SetActiveTime()
}
