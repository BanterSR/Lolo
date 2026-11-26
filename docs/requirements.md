# 待实现需求

目标客户端 1.1.12（程序版本 `2026-10-21-14-05-50`，资源版本 `2026-11-07-22-39-35`）。全部模块都待实现。

- **以重写为主。** Lolo 现有业务代码基于旧版本，设计可能很不合理，不视为已实现。进行到一个模块时直接重写整条链路（请求处理、状态、规则、持久化、推送、登录数据），不调用旧实现。
- **依赖未重写即排期错误。** 本模块依赖的链路如果还没有重写，不得调用其旧实现：调整排期先做依赖模块，或在本次一并完整重写依赖模块。
- **服务端完整实现。** 模块的全部请求、推送、登录数据与配置定义的功能一次做完；不得以“流程还没到这里”“客户端不走这里”“其他模块缺少”为由少做或留空。
- 入口取自 `cmdid.csv` 与 `client/index/REQ.md`，请求的回包为同名 Rsp。推送的客户端消费方尚未建索引，实现时按协议与客户端代码核对发送时机与字段；用 REQ.md 的发送、消费、调用方 RVA 查全规则。
- 开发阶段没有旧数据，不写兼容与迁移。GmCodeReq 不实现（GM 由 Momoko 管理平台负责）。
- 完成一个模块即从本文件删除并写入 `README.md`。

## 实现顺序

按依赖排序：被依赖的模块先完成，后面的模块只调用已重写的模块。

1. 基础：连接与登录、通用消息、玩家信息、功能解锁、背包与奖励、邮件、任务与剧情、成就。其余模块的登录、奖励、消耗与开放条件都依赖它们。
2. 养成：角色、队伍、装备、服装与外观、能力与徽章、宠物、体力与精力。
3. 大世界：场景与房间，以及场景上的世界玩法。
4. 战斗与副本。
5. 生活与玩法。
6. 家园。
7. 社交与多人。
8. 商业与运营。

## 1. 基础

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 连接与登录 | 请求：1001 VerifyLoginTokenReq、1003 PlayerLoginReq、1005 PlayerMainDataReq、1007 PlayerPingReq、1009 PlayerOfflineReq、2201 ClientLogAuthReq、2203 ClientLogMessageReq、2263 AccountNameBindingCancelReq、2521 UploadFGBytesReq、3001 InheritCodeUseReq；推送：1022 NeedLoginNotice、1352 PlayerTimeOffsetNotice、2320 GameRefreshNotice、2520 UploadFGBytesNotice | GlobalConfig | 登录数据：player_time_offset、is_today_first_login、accountType、inheritCode、hasInheritGift。登录顺序 1001 → 1003 → 1005，1005 回 PlayerMainDataRsp 全量（字段分属各模块，本行只列登录自身的字段）；1003 is_reconnect 为断线重连；1009 下线；日志服 2201/2203；2520/2521 为 FairGuardSDK 反作弊字节下发与上报；2320 为刷新推送；GlobalConfig 为全局常量，各模块经 gdconf 读取。客户端未发现静态发送点：3001，按协议与消费方核实后实现。旧代码：game/player_basic.go、gateway/app.go、logserver/login.go（整条链路重写，不调用） |
| 通用消息 | 请求：2301 GenericGameAReq、2303 GenericGameBReq、2305 GenericSceneAReq、2307 GenericSceneBReq | — | generic_msg_id 取值见 player.proto GenericMsgId（1 多人房间时长、2 多人副本、3 信标准备、4 天气预报、5 付费增加染色方案、6 情绪月亮场景、7 重置染色参数、8/9 读写偏好频道），本模块只负责按 id 分派，各 id 的处理归所属模块（如 4 天气预报归场景与房间、5/7 染色归服装与外观、8/9 偏好频道归聊天），在所属模块中完整实现；2305/2307 的 msg_route 为 SCENE。旧代码：game/player_basic.go、game/player_scene.go（整条链路重写，不调用） |
| 玩家信息 | 请求：1211 SetArchiveInfoReq、1213 GetArchiveInfoReq、1511 ChangeNickNameReq、1513 ChangeSignReq、1515 ChangeHeadReq、1517 ChangePhoneBackgroundReq、1521 ChangePlayerSexReq、1529 UnlockHeadListReq、1531 PlayerLevelRewardReq、1539 ChangePhoneCaseReq、1965 OtherPlayerInfoReq、2313 ChangeIsHideBirthdayReq、2317 ChangeHideTypeReq、2631 UpdatePlayerAppearanceReq；推送：1442 PlayerTagUpdateNotice、1520 PlayerLevelExpNotice | Head、Level、Phone | 登录数据：player_id、player_name、level、exp、head、sign、phone_background、create_time、sex、reward_levels、player_label、birthday、isHideBirthday、hideValue、appearance、phone_case。等级经验与等级奖励（1520、1531、reward_levels）；1965 查看他人信息；1211/1213 为客户端键值存档（INetworkSystem.SaveServerData/LoadServerData）。客户端未发现静态发送点：1531，按协议与消费方核实后实现。旧代码：game/player_archive.go、game/player_basic.go、game/player_friend.go（整条链路重写，不调用） |
| 功能解锁与引导 | 请求：1589 TutorialReq；推送：1600 PlayerUnlockFunctionNotice | PlayerUnlock、Guide | 登录数据：unlockFunctions。旧代码：game/player_tutorial.go（整条链路重写，不调用） |
| 背包与奖励 | 请求：1407 TempPackItemDropReq、1409 TempPackItemStoreReq、1411 TempPackSortReq、1413 TempPackWearEquipReq、1959 ItemUseReq；推送：1400 PackNotice、2106 PackNotShowNotice | Item、ItemLevel、ItemSource、QuickObtaining、GiftBag、Reward | 各模块的奖励发放与消耗统一经背包；1407–1413 为临时背包；2106 为不展示的背包变化。客户端未发现静态发送点：1413，按协议与消费方核实后实现。旧代码：无（整条链路重写，不调用） |
| 邮件 | 请求：1121 GetMailsReq、1125 OperateMailsReq；推送：1128 ReceiveMailNotice | Mail | 旧代码：game/player_mail.go（整条链路重写，不调用） |
| 任务与剧情 | 请求：1723 AcceptQuestReq、1725 ClaimQuestRewardReq、1803 NpcTalkReq、2602 DailyTaskExchangeReq；推送：1718 QuestNotice、2601 DailyTaskUpdateNotice | Quest、NpcQuest、Story、Dialog、DailyTask、NPC | 登录数据：quest_detail、daily_task。含每日任务（2601、2602）与 NPC 对话（1803）。旧代码：game/player_basic.go、game/player_tutorial.go（整条链路重写，不调用） |
| 成就 | 请求：1755 GetAchieveGroupAwardReq、1757 GetAchieveGroupListReq、1759 GetOneAchieveAwardReq、1761 GetAchieveOneGroupReq；推送：1720 AchieveNotice、1764 AchieveFinishNotice | Achieve、AchieveQuest | 旧代码：game/player_basic.go（整条链路重写，不调用） |

## 2. 养成

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 角色 | 请求：1035 CharacterSkillLevelUpReq、1037 CharacterStarUpReq、1039 CharacterLevelUpReq、1041 CharacterLevelBreakReq、1479 GetCharacterAchievementListReq、1481 GetCharacterAchievementAwardReq、1483 GetCharacterAchievementUnlockPaymentReq、1485 GetCharacterAchievementBadgeAwardReq、1953 CharacterDeadReq、1955 CharacterGatherWeaponUpdateReq、2213 CharacterReviveReq；推送：1044 TeamCharExpUpdateNotice | Character、CharacterPractice、Spell、SpellCard、PassiveSpell | 登录数据：characters。升级、突破、升星、技能；角色成就（1479–1485）；阵亡与复活（1953、2213）；采集武器（1955）。旧代码：game/player_character.go（整条链路重写，不调用） |
| 队伍 | 请求：1641 UpdateTeamReq | — | 登录数据：team。旧代码：game/player_team.go（整条链路重写，不调用） |
| 装备 | 请求：1401 GetWeaponReq、1403 GetArmorReq、1405 GetPosterReq、1415 GetAllCharacterEquipReq、1417 CharacterEquipUpdateReq、1419 CharacterEquipPresetSwitchReq、1421 PosterStarUpReq、1423 PosterIllustrationListReq、1425 PosterIllustrationRewardReq、1427 LockEquipReq、1701 WeaponInscriptionInlaidReq、1703 WeaponInscriptionRemoveReq、1957 EquipDisassembleReq、1961 WeaponStrengthReq、1963 ArmorStrengthReq | Weapon、Armor、Poster、Inscription | 武器、盔甲、海报（含图鉴 1423/1425）、铭文、强化、分解、锁定与装备预设。客户端未发现静态发送点：1417，按协议与消费方核实后实现。旧代码：game/player_character.go、game/player_item.go（整条链路重写，不调用） |
| 服装与外观 | 请求：1645 OutfitDyeUnlockIndexReq、1647 OutfitDyeReq、1651 OutfitColorantSelectReq、1653 OutfitRandomColorReq、1655 OutFitDyeSaveReq、1657 OutfitPresetUpdateReq、1659 OutfitPresetSwitchReq、1661 OutfitHideSwitchReq、1671 UpdateCharacterAppearanceReq；推送：1650 OutfitPresetUpdateNotice、1674 CharacterAppearanceUpdateNotice | Fashion、DyeStuff、Umbrella、Pendant | 登录数据：unSaveOutfitDyeScheme、totalGuaranteeDyeNum。服装预设、染色与染色方案、隐藏、角色外观。客户端未发现静态发送点：1653，按协议与消费方核实后实现。旧代码：game/player_character.go（整条链路重写，不调用） |
| 能力与徽章 | 请求：1611 PlayerAbilityListReq、1613 PlayerAbilityLevelUpReq、1631 AbilityBadgeListReq、1633 AbilityBadgePageBoxActiveReq、1635 AbilityBadgePageRewardReq、1637 AbilityBadgeAchieveRewardReq；推送：1616 PlayerAbilityUnlockNotice | PlayerAbility、Ability、MoonStone | 旧代码：game/player_ability.go（整条链路重写，不调用） |
| 宠物 | 请求：4531 PetCaptureReq、4533 PetRenameReq、4535 PetDecomposeReq、4539 GetPetReq、4541 ChangePetReq；推送：4538 PetWarehouseLimitNotice | Pet | 登录数据：pet_instance_id。旧代码：game/player_pet.go（整条链路重写，不调用） |
| 体力与精力 | 请求：1051 PlayerEnergyInfoReq、1053 PlayerEnergyBuyReq、1621 PlayerVitalityReq、1623 PlayerVitalityBuyReq；推送：2018 PlayerTiredDropNotice | — | 登录数据：playerDropRateInfo。精力（Energy）、活力（Vitality）的恢复与购买；疲劳掉落率（2018、playerDropRateInfo）。旧代码：game/player_basic.go（整条链路重写，不调用） |

## 3. 大世界

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 场景与房间 | 请求：1201 ChangeSceneChannelReq、1203 PlayerSceneRecordReq、1801 SceneSitChairReq、1919 SceneInterActionPlayStatusReq、2511 SceneTransmitReq、2535 UnfreezePlayerActionReq、2765 SceneSitVehicleReq；推送：1016 SceneDataNotice、1206 PlayerSceneSyncDataNotice、1208 ServerSceneSyncDataNotice、1880 PlayerBuffNotice、1918 SceneWeatherChangeNotice、1922 SceneInterActionPlayStatusNotice、1988 PlayerNotInSceneChannelNotice、2534 FreezePlayerActionNotice | Scene、Weather、TeleportKey、Buff、GlobalBuff | 登录数据：scene_id、channel_id、channel_label、player_buffs。进入场景与房间（1016）、多人同步（1203 SyncFrameData、1206、1208）、传送、座椅与载具、天气、互动状态、冻结玩家操作（2534/2535）、玩家 buff（1880）。客户端未发现静态发送点：2511，按协议与消费方核实后实现。旧代码：game/player_scene.go（整条链路重写，不调用） |
| 区域与世界等级 | 请求：1523 WorldLevelAchieveListReq、1525 UnlockWorldLevelReq、1527 ChangeWorldLevelReq、1871 SceneProcessListReq、1901 AreaCloseReq、1903 AreaUnlockReq、1905 AreaLevelUpReq、1907 AreaAchieveListReq | Area、Exploration | 登录数据：world_level。1871 为场景完成度（ScenesManagerSystem.OnUpdateSceneCompleteRate）。旧代码：game/player_scene.go、game/player_world.go（整条链路重写，不调用） |
| 旗帜与据点 | 请求：1325 FlagBattleStateUpdateReq；推送：1580 FlagUnlockAddNotice、1582 FlagUnLockRemoveNotice | Flag | 旧代码：game/player_battle.go（整条链路重写，不调用） |
| 动作与表情 | 请求：1911 PopEmojiReq、1967 SendActionReq、1977 MultipleActionAcceptReq、1999 ClapReq、2209 ActionStudyReq；推送：1910 PopEmojiNotice、1970 SendActionNotice、1972 SendActionStudyNotice、1974 SendActionAddNotice、1976 SendMultipleActionNotice、1980 SendMultipleActionCompleteNotice、2002 ClapResultNotice、2010 AchieveActionUnLockNotice、2212 ActionStudyNotice | Action、Expression | 单人与多人动作、动作学习、冒泡表情、鼓掌。旧代码：game/player_scene.go（整条链路重写，不调用） |
| 篝火 | 请求：1913 PlaceCampFireReq、1923 RemoveCampFireReq；推送：1916 PlaceCampFireNotice、1926 RemoveCampFireNotice | — | CampsiteSystem。旧代码：无（整条链路重写，不调用） |
| 采集与收集 | 请求：1741 CollectingReq、1743 CollectionRewardReq、1745 TreasureBoxOpenReq、1747 TreasureBoxPickupReq、1749 PickupReq、1751 GatherReq、1951 CollectVoiceRegionReq、1997 GetCollectItemIdsReq、2575 CollectMoonReq、2577 GetCollectMoonInfoReq、2581 EmotionMoonInteractReq；推送：1754 GatherSceneLimitRecoveryNotice、2580 CollectMoonInfoUpdateNotice | CollectionItem、Gather、GatherLevel、TreasureBox、VoiceRegion、Book、Utensil | 收藏品、资源点采集、宝箱、拾取、月亮收集、语音区域。旧代码：game/player_battle.go、game/player_collect.go（整条链路重写，不调用） |
| 图鉴 | 请求：1861 ManualListReq、1863 ManualFlagUnlockReq、1865 ManualFlagRewardReq | Manual | 旧代码：game/player_basic.go（整条链路重写，不调用） |
| 寻宝 | 请求：4521 TreasureDigReq、4523 TreasureIllustrationInfoReq、4525 TreasureIllustrationRewardReq、4527 GetTreasureAchieveRewardReq | Treasure、Shovel、ArchaeologyAchieve | 旧代码：game/player_basic.go（整条链路重写，不调用） |
| 场景玩法与事件 | 请求：1451 GamePlayRewardReq；推送：2742 MoonQuestGamePlayRewardNotice | Mechanism、InteractionActivity、RandomEvent、Scenic | 1451 为玩法奖励（GamePlayRewardManager）。旧代码：game/player_basic.go（整条链路重写，不调用） |
| 补给箱 | 请求：1891 SupplyBoxInfoReq、1893 SupplyBoxRewardReq | — | 旧代码：game/player_supply.go（整条链路重写，不调用） |
| 演奏 | 请求：2671 ChangeMusicalItemReq、2673 PlayMusicNoteReq | MusicNote | 旧代码：game/player_scene.go（整条链路重写，不调用） |

## 4. 战斗与副本

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 战斗与挑战 | 请求：1301 ChallengeFriendRankReq、1321 ChallengeStateUpdateReq、1323 RiddleStateUpdateReq、1327 BattleEncounterStateUpdateReq、1329 BattleEncounterInfoReq、1851 MonsterDeadReq | BattleEncounter、BattleNormal、BattleDefender、BattleGather、BattleGuide、BattleWeapon、Monster、Challenge、Riddle | 遭遇战、怪物死亡、挑战与谜题、好友挑战排行（1301）。旧代码：game/player_battle.go、game/player_friend.go（整条链路重写，不调用） |
| 副本 | 请求：1331 DungeonRepeatEnterReq、1333 DungeonViewReq、1335 DungeonEnterReq、1337 DungeonExitReq、1339 DungeonFinishReq、1341 CollectCoinReq、1343 DungeonTaskFinishRewardReq、1345 DungeonStarRewardReq、1347 DungeonOperateReq、1353 DungeonSweepReq、1355 ItemSweepReq；推送：2760 DungeonOpenFinalBoxNotice | Dungeon | 含扫荡（1353）与道具扫荡（1355，UICharacterSystem 获取途径）。旧代码：game/player_dungeon.go（整条链路重写，不调用） |
| 深渊 | 请求：2111 AbyssInfoReq、2113 AbyssQuestRewardReq、2115 AbyssTeamUpdateReq、2117 AbyssTeamSwitchReq、2119 AbyssFriendRankReq、2661 AbyssAllServerRankReq；推送：2110 AbyssSeasonNotice、2648 AbyssServerRankNotice | Abyss | 旧代码：game/player_abyss.go（整条链路重写，不调用） |
| BossRush | 请求：2701 BossRushInfoReq、2703 BossRushStartChallengeReq、2705 BossRushEnterStageReq、2707 BossRushStartBattleReq、2709 BossRushLeaveStageReq、2711 BossRushFriendRankReq、2713 BossRushFriendDetailReq、2715 BossRushTerminateChallengeReq、2717 BossRushQuestRewardReq | BossRush | 旧代码：game/player_basic.go（整条链路重写，不调用） |

## 5. 生活与玩法

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 生活技能 | 请求：1365 GetOneTypeLifePictorialBookCountReq、1369 GetLifeInfoReq、1379 CookingFoodReq、1381 MakePropReq、1383 HandicraftReq、1385 SewingReq、1387 GetLifeAchievementRewardReq、1389 LifeLeveUpReq、1391 GetLifeAchieveReq、1393 LifeSkillLevelUpReq、1395 GetLifeSkillReq、1397 FishingReq；推送：1368 LifeProficiencyNotice、1374 FishingResultNotice、1376 LifeSkillLevelUpNotice、1378 LifeAchieveNotice | Cooking、Make、Handicraft、Spin、Fishing、FishPond、LifeGather | 烹饪、制作、手工、缝纫、钓鱼、生活等级与技能、生活成就。客户端未发现静态发送点：1391，按协议与消费方核实后实现。旧代码：game/player_life.go（整条链路重写，不调用） |
| 探索派遣 | 请求：1819 ExploreInitReq、1821 ExploreReq、1823 ExploreQuickFinishReq、1825 ExploreRewardReq、1827 ExploreCancelReq、1829 ExploreCollectRewardReq、1833 IntervalStartReq、1835 IntervalJoinReq、1837 IntervalQuickReq、1839 IntervalRewardReq、1841 FriendIntervalInitReq、1843 SelfIntervalInitReq；推送：1832 IntervalChangeNotice | Explore | 旧代码：game/player_basic.go、game/player_explore.go、game/player_friend.go（整条链路重写，不调用） |
| 小游戏 | 请求：2003 ThrowDiceReq、2731 RestaurantGameInfoReq、2733 RestaurantGameOperateReq、2735 RestaurantGameEatReq、2739 RestaurantGameQuestRewardReq、4600 GomokuSinglePlayerRewardReq、4602 GomokuOnlinePlayerReadyReq、4604 GomokuOnlinePlayerWatchReq、4609 GetGomokuOnlineChessboardReq、4611 GomokuOnlinePlayerPlaceStoneReq、4614 GomokuOnlinePlayerLeaveReq、4617 GomokuOnlinePlayerStartPlayReq；推送：2006 ThrowDiceNotice、2008 ThrowDiceResultNotice、2160 FireworksStartNotice、2738 RestaurantGameDayNotice、4606 GomokuOnlinePlayerNotice、4607 GomokuOnlinePlayerWatchNotice、4608 GomokuOnlinePlayerRemoveNotice、4613 GomokuOnlinePlayerPlaceStoneNotice、4616 GomokuOnlinePlayerLeaveNotice、4619 GomokuOnlinePlayerStartPlayNotice、4620 GomokuOnlineSettlementNotice | MiniGame、RestaurantGame、SeasonalMiniGames、FireworksParty | 掷骰（2003）、五子棋单人与联机（4600–4620）、餐厅（2731–2739）、烟花（2160）。旧代码：无（整条链路重写，不调用） |

## 6. 家园

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 花园 | 请求：1681 PlaceFurnitureReq、1683 TakeOutFurnitureReq、1685 GetGardenInfoReq、1687 SwitchGardenStatusReq、1693 DoLikesReq、2205 GardenLikeRecordReq、2507 HandingFurnitureReq、2509 TakeOutHandingFurnitureReq、2611 GardenFurnitureRemoveAllReq、2615 GardenFurnitureSaveReq、2617 GardenFurnitureApplySchemeReq、2619 GardenSchemeFurnitureListReq、2621 GardenFurnitureSchemeSetNameReq、2623 GardenFurnitureSchemeReq、2625 GardenPlaceCharacterReq、2743 PlaceMoonSpotReq、2747 SetRoomDecorReq；推送：1696 SceneGardenFurnitureUpdateNotice、1698 SceneGardenFurnitureRemoveNotice、2312 FurnitureItemChangeNotice、2614 GardenFurnitureBatchUpdateNotice、2628 GardenPlaceCharacterNotice、2746 PlaceMoonSpotNotice、2750 RoomDecorNotice | Furniture、HomeItem、MoonSpot | 登录数据：gardenLikesNum、furnitureItemInfo、placed_characters。家具摆放与方案、点赞、手持家具、摆放角色、月亮摆放、房间装饰。旧代码：game/player_garden.go、game/player_scene.go（整条链路重写，不调用） |

## 7. 社交与多人

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 好友 | 请求：1727 FriendSearchReq、1729 FriendAddReq、1731 FriendDelReq、1733 FriendBlackReq、1735 FriendHandleReq、1739 FriendReq、1781 SetFriendExtInfoReq、2315 TransmitSceneReq、2651 UseItemFriendIntimacyReq；推送：1738 FriendHandleNotice、2658 IntimacyGiftDayCountNotice | Intimacy | 含传送到好友身边（2315）与亲密度赠礼（2651、2658）。旧代码：game/player_friend.go（整条链路重写，不调用） |
| 聊天 | 请求：1930 ChangeChatChannelReq、1933 SendChatMsgReq、2013 PrivateChatMsgRecordReq；推送：1538 SpeechRecyclingNotice、1936 ChatMsgNotice、1938 ChatMsgRecordInitNotice、1940 ChatUnLockExpressionNotice、2012 PrivateChatOfflineNotice、2232 ChangeChatChannelNotice、2630 GMRecommendChannelNotice | Chat | 房间、聊天室、私聊与离线消息、表情解锁、撤回（1538）、推荐频道（2630）。旧代码：game/player_chat.go（整条链路重写，不调用） |
| 多人组队与玩法 | 请求：2406 MPTeamCreateReq、2412 MPSwapCharacterReq、2414 MPBeaconActionReq、2419 MPTeamGameModeChangeReq、2422 MPTeamInfoReq、2431 MPGameStartReq、2441 MPTeamRenameReq、2443 MPTeamInviteReq、2445 MPTeamJoinReq、2447 MPTeamQuitReq、2449 MPTeamKickMemberReq、2451 MPTeamDismissReq、2469 MPBeaconPlayerReadyReq、2501 MPPlayRoomEventReq、2505 MPPlayRoomReadyReq、2515 MPPlayRoomRewardInfoReq、2517 MPPlayRoomExitReq、2531 BuyGameCoinReq、4700 CollapseFailReq；推送：2418 MPTeamBeaconNotice、2421 MPTeamGameModeChangeNotice、2424 SceneMPBeaconNotice、2430 MPBeaconCanStartNotice、2453 MPTeamDismissNotice、2454 MPTeamInviteNotice、2455 MPTeamPlayerKickedNotice、2456 PlayerNotInMPTeamNotice、2458 MPTeamMemberNotice、2460 MPTeamRenameNotice、2462 PlayerMPGameNotice、2465 MPRoomAssignHostNotice、2467 MPRoomDismissNotice、2468 MPRoomEnterNotice、2504 MPPlayRoomEventNotice、2514 MPPlayRoomSettlementNotice、2519 MPPlayRoomExitNotice、2604 MPGameWeaponUpdateNotice、4702 CollapseFailNotice | TeamGameplay | MPTeam 组队、信标、房间、多人玩法结算、游戏币（2531）、坍塌失败（4700）。客户端未发现静态发送点：2469、2515，按协议与消费方核实后实现。旧代码：无（整条链路重写，不调用） |
| 照片分享 | 请求：2551 UploadPhotoShareReq、2553 ChangePhotoShareTitleReq、2555 PhotoShareSearchReq、2557 PhotoShareDetailReq、2559 OperatePhotoShareReq、2561 PhotoShareCopyFashionDyeReq、2563 PhotoSharePaymentAddUploadGridReq | — | 旧代码：game/player_photo.go（整条链路重写，不调用） |

## 8. 商业与运营

| 模块 | 入口 | 资源 | 要求 |
| --- | --- | --- | --- |
| 卡池 | 请求：1443 GachaListReq、1445 GachaReq、1447 GachaFullPickReq、1449 GachaRecordReq、2207 OptionalUpPoolItemReq、4543 ChooseGachaReq | Gacha | 含自选 UP（2207）与自选卡池（4543）。旧代码：game/player_gacha.go（整条链路重写，不调用） |
| 商店与支付 | 请求：1675 ShopInfoReq、1677 ShopBuyReq、2101 CreatePayOrderReq、2309 MonthCardRewardReq、2681 WishListOperateReq、2683 WishListByShopIdReq、2685 WishListByFriendIdReq、2689 WishListBuyReq；推送：1680 ShopRefreshNotice、1706 ShopInitNotice、2104 PaySendGoodsNotice、2126 MonthCardNotice、2688 WishListBuyNotice | Shop | 登录数据：monthCardOverDueTime、monthCardRewardDays。商店、支付下单与发货、月卡、心愿单。旧代码：game/player_friend.go、game/player_shop.go（整条链路重写，不调用） |
| 活动 | 请求：1985 ActivitySignInReq、1991 ActivityQuestRewardReq、1995 ActivityRegularRewardReq、2121 ActivityGiftRewardReq、2131 ActivityBattlePassBuyExpReq、2133 GetActivityBattlePassLevelRewardReq、2135 GetActivityBattlePassQuestRewardReq、2252 ActivityInviteRewardClaimReq、2254 InviteCodeUseReq、2635 ActivityTurntableDrawReq、2637 ActivityTurntableRecordReq、4501 ActivitySandbagRewardReq、4503 ActivitySandbagHitReq、4511 ActivityRecallRewardClaimReq、4513 RecallCodeUseReq、4515 ActivityUploadWishTxtReq、4519 ActivityKoiUseItemReq；推送：1982 ActivityChangeNotice、1984 ActivitySignInDataNotice、1990 ActivityQuestDataNotice、1994 ActivityRegularDataNotice、2124 ActivityGiftDataNotice、2128 CharacterBpBuyNotice、2130 ActivityBattlePassBuyNotice、2138 ActivityBattlePassInfoNotice、2140 ActivityGiftBuyNotice、2251 ActivityInviteNotice、2256 ActivityInviteCountUpdateNotice、2634 ActivityTurntableDataNotice、4500 ActivitySandbagNotice、4508 ActivitySandbagDropNotice、4510 ActivityRecallNotice、4518 ActivityKoiNotice | Activity、ActivityFashionShow、ActivityGift、ActivityInvite、ActivityJinli、ActivityQuest、ActivityRegular、ActivitySign、ActivityTurntable、BattlePass | 登录数据：recall_end_time。签到、任务、常驻、礼包、通行证、邀请、转盘、沙袋、回流、锦鲤。旧代码：无（整条链路重写，不调用） |
| 祝福树 | 请求：2585 BlessTreeReq、2587 BlessTreeUnlockReq；推送：2584 BlessTreeNotice | BlessingTree | 旧代码：无（整条链路重写，不调用） |
| 运营与系统 | 请求：2151 UseRedemptionCodeReq、2170 ReportReq、2571 GetExpressAddressReq、2573 UploadExpressAddressReq、2695 SPOperateReq；推送：1454 ModuleCloseNotice、1534 ForbiddenInfoNotice、1536 ClientIFIxGMNotice、1722 GmNotice、2016 SystemNotice、2020 NewQuestionnaireNotice、2231 QuestionnaireRemoveNotice | — | 登录数据：forbidden_infos、questionnaire_info、sp_open_status、sp_check_status。兑换码、举报、收货地址、SP 开关（2695 没有回包消费者）、模块关闭、封禁信息、客户端热修推送（1536）、公告（1722 GmNotice）、系统消息、问卷；封禁与公告的管理入口由 Momoko 管理平台提供。旧代码：无（整条链路重写，不调用） |

## 未归入模块的表

BuffImage、GpuQuality、MiniMap、MsgCodeTips、String、Text、UI、UIStaticText、Video、Voice：客户端显示与本地化用表，作为证据查阅；模块规则用到时由该模块在 gdconf 接入。
