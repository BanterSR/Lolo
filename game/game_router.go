package game

import (
	"runtime/debug"

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
		cmd.ChangePhoneBackgroundReq:  g.ChangePhoneBackground,  // 更换手机背景
		cmd.ChangeIsHideBirthdayReq:   g.ChangeIsHideBirthday,   // 更改是否隐藏生日
		cmd.ChangeHideTypeReq:         g.ChangeHideType,         // 更改隐身状态
		cmd.ChangeSignReq:             g.ChangeSign,             // 更改签名
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
		cmd.SceneSitChairReq:              g.SceneSitChair,              // 坐/离 椅子请求
		cmd.SceneSitVehicleReq:            g.SceneSitVehicle,            // 上下车请求
		cmd.ChangeMusicalItemReq:          g.ChangeMusicalItem,          // 切换音乐源请求
		cmd.PlayMusicNoteReq:              g.PlayMusicNote,              // 演奏请求
		cmd.AreaCloseReq:                  g.AreaClose,                  // 区域关闭
		cmd.AreaUnlockReq:                 g.AreaUnlock,                 // 解锁传送锚点
		cmd.AreaAchieveListReq:            g.AreaAchieveList,            // 获取区域成就列表
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
		cmd.ChooseGachaReq:        g.ChooseGacha,        // 设置自选物品
		cmd.GachaFullPickReq:      g.GachaFullPick,      // 领取满奖
		// 角色
		cmd.CharacterLevelUpReq:             g.CharacterLevelUp,             // 角色升级
		cmd.CharacterLevelBreakReq:          g.CharacterLevelBreak,          // 角色突破
		cmd.OutfitPresetUpdateReq:           g.OutfitPresetUpdate,           // 保存预设装扮
		cmd.OutfitPresetSwitchReq:           g.OutfitPresetSwitch,           // 切换装扮方案
		cmd.OutfitColorantSelectReq:         g.OutfitColorantSelect,         // 染色选择请求
		cmd.OutfitDyeReq:                    g.OutfitDye,                    // 染色请求
		cmd.OutFitDyeSaveReq:                g.OutFitDyeSave,                // 保存染色结果
		cmd.CharacterEquipUpdateReq:         g.CharacterEquipUpdate,         // 角色更新装备
		cmd.CharacterEquipPresetSwitchReq:   g.CharacterEquipPresetSwitch,   // 切换装备预设
		cmd.PosterStarUpReq:                 g.PosterStarUp,                 // 海报升星
		cmd.PosterIllustrationRewardReq:     g.PosterIllustrationReward,     // 海报插画奖励
		cmd.LockEquipReq:                    g.LockEquip,                    // 装备锁定
		cmd.WeaponInscriptionInlaidReq:      g.WeaponInscriptionInlaid,      // 武器铭刻
		cmd.WeaponInscriptionRemoveReq:      g.WeaponInscriptionRemove,      // 移除铭刻
		cmd.EquipDisassembleReq:             g.EquipDisassemble,             // 装备分解
		cmd.ItemUseReq:                      g.ItemUse,                      // 物品使用
		cmd.WeaponStrengthReq:               g.WeaponStrength,               // 武器强化
		cmd.ArmorStrengthReq:                g.ArmorStrength,                // 防具强化
		cmd.UpdateCharacterAppearanceReq:    g.UpdateCharacterAppearance,    // 更新角色外观
		cmd.CharacterGatherWeaponUpdateReq:  g.CharacterGatherWeaponUpdate,  // 更新手持工具请求
		cmd.CharacterStarUpReq:              g.CharacterStarUp,              // 角色升星请求
		cmd.CharacterSkillLevelUpReq:        g.CharacterSkillLevelUp,        // 角色技能升级请求
		cmd.GetCharacterAchievementListReq:  g.GetCharacterAchievementList,  // 获取角色成就情况
		cmd.GetCharacterAchievementAwardReq: g.GetCharacterAchievementAward, // 角色领取成就奖励
		// 信息记录
		cmd.GetArchiveInfoReq: g.GetArchiveInfo, // 获取记录的信息
		cmd.SetArchiveInfoReq: g.SetArchiveInfo, // 设置信息
		// 好友
		cmd.FriendReq:           g.Friend,           // 获取好友聚合请求
		cmd.FriendAddReq:        g.FriendAdd,        // 添加好友请求
		cmd.FriendHandleReq:     g.FriendHandle,     // 处理好友申请
		cmd.FriendDelReq:        g.FriendDel,        // 删除好友关系
		cmd.FriendBlackReq:      g.FriendBlack,      // 拉黑玩家
		cmd.OtherPlayerInfoReq:  g.OtherPlayerInfo,  // 获取其他玩家信息
		cmd.FriendSearchReq:     g.FriendSearch,     // 搜索玩家
		cmd.SetFriendExtInfoReq: g.SetFriendExtInfo, // 设置好友扩展信息
		// 聊天
		cmd.PrivateChatMsgRecordReq: g.PrivateChatMsgRecord, // 获取私聊聊天记录
		cmd.SendChatMsgReq:          g.SendChatMsg,          // 发送聊天消息
		cmd.ChangeChatChannelReq:    g.ChangeChatChannel,    // 切换聊天房间
		// 商店
		cmd.ShopInfoReq:       g.ShopInfo,       // 获取商店信息
		cmd.ShopBuyReq:        g.ShopBuy,        // 商店购买请求
		cmd.CreatePayOrderReq: g.CreatePayOrder, // 创建支付订单请求
		// 战斗
		cmd.BattleEncounterInfoReq:        g.BattleEncounterInfo,        // 获取遭遇战斗信息
		cmd.BattleEncounterStateUpdateReq: g.BattleEncounterStateUpdate, // 遭遇战斗更新请求
		cmd.MonsterDeadReq:                g.MonsterDead,                // 怪物死亡
		cmd.FlagBattleStateUpdateReq:      g.FlagBattleStateUpdate,      // 更新flag战斗状态
		cmd.PickupReq:                     g.Pickup,                     // 捡起掉落物请求
		// 深渊
		cmd.AbyssInfoReq: g.AbyssInfo, // 深渊信息请求
		// 邮箱
		cmd.GetMailsReq:     g.GetMails,     // 获取邮件列表
		cmd.OperateMailsReq: g.OperateMails, // 邮件操作请求
		// 收集
		cmd.GetCollectItemIdsReq:  g.GetCollectItemIds,  // 获取收集物品列表
		cmd.CollectingReq:         g.Collecting,         // 收集请求
		cmd.CollectionRewardReq:   g.CollectionReward,   // 领取收集奖励
		cmd.GatherReq:             g.Gather,             // 收集请求
		cmd.TreasureBoxOpenReq:    g.TreasureBoxOpen,    // 打开宝箱
		cmd.TreasureBoxPickupReq:  g.TreasureBoxPickup,  // 拾取宝箱奖励
		cmd.GetCollectMoonInfoReq: g.GetCollectMoonInfo, // 获取星云树信息
		cmd.CollectMoonReq:        g.CollectMoon,        // 收集月亮请求
		// 副本
		cmd.DungeonViewReq:    g.DungeonView,    // 副本信息
		cmd.DungeonEnterReq:   g.DungeonEnter,   // 进入副本
		cmd.DungeonOperateReq: g.DungeonOperate, // 副本开始/结束
		cmd.DungeonFinishReq:  g.DungeonFinish,  // 副本结算
		cmd.DungeonExitReq:    g.DungeonExit,    // 退出副本
		// 宠物
		cmd.GetPetReq:                    g.GetPetReq,                 // 获取宠物列表
		cmd.ChangePetReq:                 g.ChangePet,                 // 宠物携带
		cmd.PetCaptureReq:                g.PetCapture,                // 捕获宠物
		cmd.PetRenameReq:                 g.PetRename,                 // 宠物重命名
		cmd.PetDecomposeReq:              g.PetDecompose,              // 宠物分解
		cmd.RestaurantGameInfoReq:        g.RestaurantGameInfo,        // 餐厅信息
		cmd.RestaurantGameOperateReq:     g.RestaurantGameOperate,     // 餐厅经营
		cmd.RestaurantGameEatReq:         g.RestaurantGameEat,         // 餐厅进食
		cmd.RestaurantGameQuestRewardReq: g.RestaurantGameQuestReward, // 餐厅任务奖励
		// 五子棋与崩溃
		cmd.CollapseFailReq:                 g.CollapseFail,
		cmd.GomokuSinglePlayerRewardReq:     g.GomokuSinglePlayerReward,
		cmd.GomokuOnlinePlayerReadyReq:      g.GomokuOnlinePlayerReady,
		cmd.GomokuOnlinePlayerStartPlayReq:  g.GomokuOnlinePlayerStartPlay,
		cmd.GomokuOnlinePlayerPlaceStoneReq: g.GomokuOnlinePlayerPlaceStone,
		cmd.GomokuOnlinePlayerLeaveReq:      g.GomokuOnlinePlayerLeave,
		cmd.GomokuOnlinePlayerWatchReq:      g.GomokuOnlinePlayerWatch,
		cmd.GetGomokuOnlineChessboardReq:    g.GetGomokuOnlineChessboard,
		cmd.GmCodeReq:                       g.GmCode,

		cmd.PlayerVitalityReq:                   g.PlayerVitality,
		cmd.BossRushInfoReq:                     g.BossRushInfo,
		cmd.FriendIntervalInitReq:               g.FriendIntervalInit,
		cmd.SelfIntervalInitReq:                 g.SelfIntervalInit,
		cmd.ExploreInitReq:                      g.ExploreInit,
		cmd.ExploreReq:                          g.Explore,
		cmd.ExploreQuickFinishReq:               g.ExploreQuickFinish,
		cmd.ExploreRewardReq:                    g.ExploreReward,
		cmd.ExploreCancelReq:                    g.ExploreCancel,
		cmd.ExploreCollectRewardReq:             g.ExploreCollectReward,
		cmd.TempPackItemDropReq:                 g.TempPackItemDrop,
		cmd.TempPackItemStoreReq:                g.TempPackItemStore,
		cmd.TempPackSortReq:                     g.TempPackSort,
		cmd.TempPackWearEquipReq:                g.TempPackWearEquip,
		cmd.NpcTalkReq:                          g.NpcTalk,  // npc对话
		cmd.TutorialReq:                         g.Tutorial, // 开始教程
		cmd.ChallengeFriendRankReq:              g.ChallengeFriendRank,
		cmd.AbilityBadgeListReq:                 g.AbilityBadgeList,
		cmd.SceneProcessListReq:                 g.SceneProcessList,
		cmd.WishListByFriendIdReq:               g.WishListByFriendId,
		cmd.GetLifeInfoReq:                      g.GetLifeInfo,
		cmd.GetLifeSkillReq:                     g.GetLifeSkill,
		cmd.GetLifeAchieveReq:                   g.GetLifeAchieve,
		cmd.LifeSkillLevelUpReq:                 g.LifeSkillLevelUp,
		cmd.GetLifeAchievementRewardReq:         g.GetLifeAchievementReward,
		cmd.LifeLeveUpReq:                       g.LifeLeveUp,
		cmd.FishingReq:                          g.Fishing,
		cmd.CookingFoodReq:                      g.CookingFood,
		cmd.MakePropReq:                         g.MakeProp,
		cmd.HandicraftReq:                       g.Handicraft,
		cmd.GetAchieveOneGroupReq:               g.GetAchieveOneGroup,
		cmd.GetAchieveGroupListReq:              g.GetAchieveGroupList,
		cmd.GenericGameAReq:                     g.GenericGameA,
		cmd.GenericGameBReq:                     g.GenericGameB,
		cmd.ManualListReq:                       g.ManualList,                       // 获取手册列表
		cmd.PlayerAbilityListReq:                g.PlayerAbilityList,                //
		cmd.WorldLevelAchieveListReq:            g.WorldLevelAchieveList,            //
		cmd.SupplyBoxInfoReq:                    g.SupplyBoxInfo,                    //
		cmd.SupplyBoxRewardReq:                  g.SupplyBoxReward,                  //
		cmd.ActivitySignInReq:                   g.ActivitySignIn,                   //
		cmd.ActivityQuestRewardReq:              g.ActivityQuestReward,              //
		cmd.ActivityRegularRewardReq:            g.ActivityRegularReward,            //
		cmd.ActivityGiftRewardReq:               g.ActivityGiftReward,               //
		cmd.ActivityBattlePassBuyExpReq:         g.ActivityBattlePassBuyExp,         //
		cmd.GetActivityBattlePassLevelRewardReq: g.GetActivityBattlePassLevelReward, //
		cmd.GetActivityBattlePassQuestRewardReq: g.GetActivityBattlePassQuestReward, //
		cmd.MonthCardRewardReq:                  g.MonthCardReward,                  //
		cmd.ActivityTurntableDrawReq:            g.ActivityTurntableDraw,            //
		cmd.ActivityTurntableRecordReq:          g.ActivityTurntableRecord,          //
		cmd.AbyssQuestRewardReq:                 g.AbyssQuestReward,                 //
		cmd.GetAchieveGroupAwardReq:             g.GetAchieveGroupAward,             //
		cmd.GetOneAchieveAwardReq:               g.GetOneAchieveAward,               //
		cmd.GetAllCharacterEquipReq:             g.GetAllCharacterEquip,             //
		cmd.GamePlayRewardReq:                   g.GamePlayReward,                   //
		cmd.AcceptQuestReq:                      g.AcceptQuest,                      //
		cmd.ClaimQuestRewardReq:                 g.ClaimQuestReward,                 //
		cmd.TreasureIllustrationInfoReq:         g.TreasureIllustrationInfo,
		// 已实现但此前未注册的玩家资料/动作请求。
		cmd.ChangeHeadReq:        g.ChangeHead,
		cmd.ChangePhoneCaseReq:   g.ChangePhoneCase,
		cmd.ChangePlayerSexReq:   g.ChangePlayerSex,
		cmd.ChangeWorldLevelReq:  g.ChangeWorldLevel,
		cmd.ClapReq:              g.Clap,
		cmd.OutfitHideSwitchReq:  g.OutfitHideSwitch,
		cmd.PlayerEnergyBuyReq:   g.PlayerEnergyBuy,
		cmd.PlayerEnergyInfoReq:  g.PlayerEnergyInfo,
		cmd.PlayerLevelRewardReq: g.PlayerLevelReward,
		cmd.PopEmojiReq:          g.PopEmoji,
		cmd.ThrowDiceReq:         g.ThrowDice,
		cmd.UnlockWorldLevelReq:  g.UnlockWorldLevel,
		// 通用请求与活动/收集奖励确认。
		cmd.PlayerOfflineReq:                        g.PlayerOffline,
		cmd.AccountNameBindingCancelReq:             g.AccountNameBindingCancel,
		cmd.ActionStudyReq:                          g.ActionStudy,
		cmd.ClientLogAuthReq:                        g.ClientLogAuth,
		cmd.ClientLogMessageReq:                     g.ClientLogMessage,
		cmd.ReportReq:                               g.Report,
		cmd.UnfreezePlayerActionReq:                 g.UnfreezePlayerAction,
		cmd.MultipleActionAcceptReq:                 g.MultipleActionAccept,
		cmd.GenericSceneAReq:                        g.GenericSceneA,
		cmd.SPOperateReq:                            g.SPOperate,
		cmd.DailyTaskExchangeReq:                    g.DailyTaskExchange,
		cmd.DoLikesReq:                              g.DoLikes,
		cmd.AreaLevelUpReq:                          g.AreaLevelUp,
		cmd.ChallengeStateUpdateReq:                 g.ChallengeStateUpdate,
		cmd.RiddleStateUpdateReq:                    g.RiddleStateUpdate,
		cmd.SceneTransmitReq:                        g.SceneTransmit,
		cmd.TransmitSceneReq:                        g.TransmitScene,
		cmd.CollectCoinReq:                          g.CollectCoin,
		cmd.CollectVoiceRegionReq:                   g.CollectVoiceRegion,
		cmd.PlaceCampFireReq:                        g.PlaceCampFire,
		cmd.RemoveCampFireReq:                       g.RemoveCampFire,
		cmd.PlaceMoonSpotReq:                        g.PlaceMoonSpot,
		cmd.EmotionMoonInteractReq:                  g.EmotionMoonInteract,
		cmd.SetRoomDecorReq:                         g.SetRoomDecor,
		cmd.CharacterDeadReq:                        g.CharacterDead,
		cmd.CharacterReviveReq:                      g.CharacterRevive,
		cmd.BuyGameCoinReq:                          g.BuyGameCoin,
		cmd.UseRedemptionCodeReq:                    g.UseRedemptionCode,
		cmd.InviteCodeUseReq:                        g.InviteCodeUse,
		cmd.RecallCodeUseReq:                        g.RecallCodeUse,
		cmd.InheritCodeUseReq:                       g.InheritCodeUse,
		cmd.GetExpressAddressReq:                    g.GetExpressAddress,
		cmd.UploadExpressAddressReq:                 g.UploadExpressAddress,
		cmd.UploadFGBytesReq:                        g.UploadFGBytes,
		cmd.PlayerVitalityBuyReq:                    g.PlayerVitalityBuy,
		cmd.UseItemFriendIntimacyReq:                g.UseItemFriendIntimacy,
		cmd.BlessTreeReq:                            g.BlessTree,
		cmd.BlessTreeUnlockReq:                      g.BlessTreeUnlock,
		cmd.ActivityUploadWishTxtReq:                g.ActivityUploadWishTxt,
		cmd.ActivityInviteRewardClaimReq:            g.ActivityInviteRewardClaim,
		cmd.ActivityRecallRewardClaimReq:            g.ActivityRecallRewardClaim,
		cmd.ActivityKoiUseItemReq:                   g.ActivityKoiUseItem,
		cmd.ActivitySandbagHitReq:                   g.ActivitySandbagHit,
		cmd.ActivitySandbagRewardReq:                g.ActivitySandbagReward,
		cmd.ManualFlagUnlockReq:                     g.ManualFlagUnlock,
		cmd.ManualFlagRewardReq:                     g.ManualFlagReward,
		cmd.PlayerAbilityLevelUpReq:                 g.PlayerAbilityLevelUp,
		cmd.AbilityBadgeAchieveRewardReq:            g.AbilityBadgeAchieveReward,
		cmd.AbilityBadgePageBoxActiveReq:            g.AbilityBadgePageBoxActive,
		cmd.AbilityBadgePageRewardReq:               g.AbilityBadgePageReward,
		cmd.GetCharacterAchievementBadgeAwardReq:    g.GetCharacterAchievementBadgeAward,
		cmd.GetCharacterAchievementUnlockPaymentReq: g.GetCharacterAchievementUnlockPayment,
		cmd.GetOneTypeLifePictorialBookCountReq:     g.GetOneTypeLifePictorialBookCount,
		cmd.GetTreasureAchieveRewardReq:             g.GetTreasureAchieveReward,
		cmd.TreasureIllustrationRewardReq:           g.TreasureIllustrationReward,
		cmd.TreasureDigReq:                          g.TreasureDig,
		cmd.SewingReq:                               g.Sewing,
		cmd.ItemSweepReq:                            g.ItemSweep,
		cmd.OutfitRandomColorReq:                    g.OutfitRandomColor,
		cmd.OutfitDyeUnlockIndexReq:                 g.OutfitDyeUnlockIndex,
		// 副本重复进入与奖励账本。
		cmd.DungeonRepeatEnterReq:      g.DungeonRepeatEnter,
		cmd.DungeonStarRewardReq:       g.DungeonStarReward,
		cmd.DungeonSweepReq:            g.DungeonSweep,
		cmd.DungeonTaskFinishRewardReq: g.DungeonTaskFinishReward,
		// BossRush / 深渊赛季流程。
		cmd.BossRushStartChallengeReq:     g.BossRushStartChallenge,
		cmd.BossRushEnterStageReq:         g.BossRushEnterStage,
		cmd.BossRushStartBattleReq:        g.BossRushStartBattle,
		cmd.BossRushLeaveStageReq:         g.BossRushLeaveStage,
		cmd.BossRushTerminateChallengeReq: g.BossRushTerminateChallenge,
		cmd.BossRushFriendDetailReq:       g.BossRushFriendDetail,
		cmd.BossRushFriendRankReq:         g.BossRushFriendRank,
		cmd.BossRushQuestRewardReq:        g.BossRushQuestReward,
		cmd.AbyssAllServerRankReq:         g.AbyssAllServerRank,
		cmd.AbyssFriendRankReq:            g.AbyssFriendRank,
		cmd.AbyssTeamSwitchReq:            g.AbyssTeamSwitch,
		cmd.AbyssTeamUpdateReq:            g.AbyssTeamUpdate,
		// 多人小队与游玩房间（单机环境按本地队伍应答）。
		cmd.MPTeamCreateReq:         g.MPTeamCreate,
		cmd.MPTeamInfoReq:           g.MPTeamInfo,
		cmd.MPTeamJoinReq:           g.MPTeamJoin,
		cmd.MPTeamRenameReq:         g.MPTeamRename,
		cmd.MPTeamGameModeChangeReq: g.MPTeamGameModeChange,
		cmd.MPTeamDismissReq:        g.MPTeamDismiss,
		cmd.MPTeamQuitReq:           g.MPTeamQuit,
		cmd.MPTeamInviteReq:         g.MPTeamInvite,
		cmd.MPTeamKickMemberReq:     g.MPTeamKickMember,
		cmd.MPGameStartReq:          g.MPGameStart,
		cmd.MPPlayRoomReadyReq:      g.MPPlayRoomReady,
		cmd.MPPlayRoomEventReq:      g.MPPlayRoomEvent,
		cmd.MPPlayRoomExitReq:       g.MPPlayRoomExit,
		cmd.MPPlayRoomRewardInfoReq: g.MPPlayRoomRewardInfo,
		cmd.MPSwapCharacterReq:      g.MPSwapCharacter,
		cmd.MPBeaconActionReq:       g.MPBeaconAction,
		cmd.MPBeaconPlayerReadyReq:  g.MPBeaconPlayerReady,
		// 照片墙与好友心愿单。
		cmd.UploadPhotoShareReq:               g.UploadPhotoShare,
		cmd.PhotoShareDetailReq:               g.PhotoShareDetail,
		cmd.ChangePhotoShareTitleReq:          g.ChangePhotoShareTitle,
		cmd.OperatePhotoShareReq:              g.OperatePhotoShare,
		cmd.PhotoShareCopyFashionDyeReq:       g.PhotoShareCopyFashionDye,
		cmd.PhotoSharePaymentAddUploadGridReq: g.PhotoSharePaymentAddUploadGrid,
		cmd.WishListOperateReq:                g.WishListOperate,
		cmd.WishListByShopIdReq:               g.WishListByShopId,
		cmd.WishListBuyReq:                    g.WishListBuy,
		// 限时探索。
		cmd.IntervalStartReq:  g.IntervalStart,
		cmd.IntervalJoinReq:   g.IntervalJoin,
		cmd.IntervalQuickReq:  g.IntervalQuick,
		cmd.IntervalRewardReq: g.IntervalReward,
	}
}

func (g *Game) routeHandle(conn ofnet.Conn, userId uint32, uuid string, msg *alg.GameMsg) {
	if msg == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Game.Errorf("玩家:%v 请求:%v 触发 panic:%v\n%s", userId, msg.MsgId, recovered, debug.Stack())
			if player := g.GetUser(userId); player != nil {
				player.Online = false
				player.NetFreeze = true
				delete(g.userMap, userId)
			}
			if conn != nil {
				conn.Close()
			}
		}
	}()
	switch msg.MsgId {
	case cmd.PlayerLoginReq:
		g.PlayerLogin(conn, userId, uuid, msg)
		return
	case cmd.VerifyLoginTokenReq:
		g.VerifyLoginToken(conn, msg)
		return
	}
	handlerFunc, ok := g.handlerFuncRouteMap[msg.MsgId]
	if !ok {
		log.Game.Errorf("no route for msg, cmdId: %v name:%s", msg.MsgId, cmd.Get().GetCmdNameByCmdId(msg.MsgId))
		return
	}
	player := g.GetUser(userId)
	if player == nil {
		log.Game.Debugf("player is nil, userId: %v", userId)
		return
	}
	if !player.Online {
		log.Game.Debugf("player not online, userId: %v", userId)
		return
	}
	if player.NetFreeze || player.LoginUUID != uuid { // 玩家冻结或脏数据过滤
		return
	}
	handlerFunc(player, msg)
	player.SetActiveTime()
}
