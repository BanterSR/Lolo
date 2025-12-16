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
		cmd.PlayerMainDataReq: g.PlayerMainData, // 获取玩家信息
		cmd.ChangeNickNameReq: g.ChangeNickName, // 修改玩家昵称和生日
		// 场景
		cmd.PlayerSceneRecordReq:          g.PlayerSceneRecord,          // 玩家场景同步器
		cmd.SendActionReq:                 g.SendAction,                 // 场景自动化同步器
		cmd.ChangeSceneChannelReq:         g.ChangeSceneChannel,         // 切换场景/房间
		cmd.GenericSceneBReq:              g.GenericSceneB,              // 获取房间中的天气
		cmd.SceneInterActionPlayStatusReq: g.SceneInterActionPlayStatus, // 同步玩家交互请求
		// 队伍
		cmd.UpdateTeamReq: g.UpdateTeam, // 更新队伍
		// 物品
		cmd.GetWeaponReq:              g.GetWeapon,              // 获取武器列表-根据type
		cmd.GetArmorReq:               g.GetArmor,               // 获取盔甲列表-根据type
		cmd.GetPosterReq:              g.GetPoster,              // 获取海报列表
		cmd.PosterIllustrationListReq: g.PosterIllustrationList, // 海报插头列表
		// 卡池
		cmd.GachaListReq:   g.GachaList,   // 获取卡池信息
		cmd.GachaRecordReq: g.GachaRecord, // 获取抽卡记录
		cmd.GachaReq:       g.Gacha,       // 抽卡，启动！
		// 角色
		cmd.GetCharacterAchievementListReq: g.GetCharacterAchievementList, // 获取角色成就情况
		cmd.CharacterLevelUpReq:            g.CharacterLevelUp,            // 角色升级
		cmd.CharacterLevelBreakReq:         g.CharacterLevelBreak,         // 角色突破
		cmd.OutfitPresetUpdateReq:          g.OutfitPresetUpdate,          // 保存预设装扮
		cmd.CharacterEquipUpdateReq:        g.CharacterEquipUpdate,        // 角色更新装备
		cmd.UpdateCharacterAppearanceReq:   g.UpdateCharacterAppearance,   // 更新角色外观
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
		cmd.ShopInfoReq:              g.ShopInfo,
		cmd.WishListByFriendIdReq:    g.WishListByFriendId,
		cmd.GetLifeInfoReq:           g.GetLifeInfo,
		cmd.GetMailsReq:              g.GetMails,
		cmd.GetAchieveOneGroupReq:    g.GetAchieveOneGroup,
		cmd.GetAchieveGroupListReq:   g.GetAchieveGroupList,
		cmd.GenericGameAReq:          g.GenericGameA,
		cmd.GenericGameBReq:          g.GenericGameB,
		cmd.GetCollectItemIdsReq:     g.GetCollectItemIds,
		cmd.ManualListReq:            g.ManualList,
		cmd.ChangeMusicalItemReq:     g.ChangeMusicalItem,
		cmd.PlayerAbilityListReq:     g.PlayerAbilityList,     //
		cmd.WorldLevelAchieveListReq: g.WorldLevelAchieveList, //
		cmd.SupplyBoxInfoReq:         g.SupplyBoxInfo,         //
		cmd.GetAllCharacterEquipReq:  g.GetAllCharacterEquip,  //
		cmd.GamePlayRewardReq:        g.GamePlayReward,        //
		cmd.AcceptQuestReq:           g.AcceptQuest,           //
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
