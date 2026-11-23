# 待实现需求

- 目标客户端 1.1.12（程序版本 `2026-10-21-14-05-50`，资源版本 `2026-11-07-22-39-35`）。开发阶段，没有旧数据，不写兼容、迁移与旧档案补全。
- 全部模块都要按新版本客户端重新实现。Lolo 现有业务代码基于旧版本，不视为已实现：只作为定位入口，与客户端证据不符或不完整的直接重写或删除。
- 以客户端系统为单位完整完成：该系统的全部请求、推送、登录全量数据（`PlayerMainDataRsp` 等）与持久化一并实现，完成后从本表移除并写入 `README.md`。
- GmCodeReq 不在服务端实现（GM 由 Momoko 管理平台负责）。

## 请求

客户端 303 个请求（`client/index/REQ.md`），除 GmCodeReq 外 302 个按发送/消费方中出现最多的客户端类分组，组按最小包号排序。带 * 的 127 个有旧处理入口（`game/game_router.go`、`logserver`），同样待重新实现。

| 客户端系统 | 请求（包号 名称） |
| --- | --- |
| AccountSystem | 1001 VerifyLoginTokenReq*、1003 PlayerLoginReq*、1005 PlayerMainDataReq* |
| PlayerSystem | 1035 CharacterSkillLevelUpReq*、1039 CharacterLevelUpReq*、1051 PlayerEnergyInfoReq、1201 ChangeSceneChannelReq*、1483 GetCharacterAchievementUnlockPaymentReq、1485 GetCharacterAchievementBadgeAwardReq、1511 ChangeNickNameReq*、1621 PlayerVitalityReq*、1661 OutfitHideSwitchReq、1891 SupplyBoxInfoReq*、1965 OtherPlayerInfoReq*、2307 GenericSceneBReq*、2535 UnfreezePlayerActionReq、4541 ChangePetReq* |
| CharacterModel | 1037 CharacterStarUpReq*、1041 CharacterLevelBreakReq*、1419 CharacterEquipPresetSwitchReq、1955 CharacterGatherWeaponUpdateReq* |
| UIVitalitySystem | 1053 PlayerEnergyBuyReq、1623 PlayerVitalityBuyReq |
| MailSystem | 1121 GetMailsReq* |
| UIMailSystem | 1125 OperateMailsReq* |
| GMModel | 1213 GetArchiveInfoReq* |
| FriendSystem | 1301 ChallengeFriendRankReq*、1727 FriendSearchReq*、1729 FriendAddReq*、1731 FriendDelReq*、1733 FriendBlackReq*、1735 FriendHandleReq*、1739 FriendReq*、1781 SetFriendExtInfoReq*、2315 TransmitSceneReq |
| ChallengeSystem | 1321 ChallengeStateUpdateReq |
| RiddleSystem | 1323 RiddleStateUpdateReq |
| MissionEntity | 1325 FlagBattleStateUpdateReq* |
| BattleEncounterSystem | 1327 BattleEncounterStateUpdateReq*、1329 BattleEncounterInfoReq* |
| DungeonSystem | 1331 DungeonRepeatEnterReq、1333 DungeonViewReq*、1335 DungeonEnterReq*、1339 DungeonFinishReq*、1343 DungeonTaskFinishRewardReq、1345 DungeonStarRewardReq、1347 DungeonOperateReq*、2117 AbyssTeamSwitchReq、2707 BossRushStartBattleReq |
| ScenesManagerSystem | 1337 DungeonExitReq* |
| DungeonEntity | 1341 CollectCoinReq |
| UIMissionCardSystem | 1353 DungeonSweepReq |
| UICharacterSystem | 1355 ItemSweepReq、1479 GetCharacterAchievementListReq*、1481 GetCharacterAchievementAwardReq* |
| UILifeInfoSystem | 1365 GetOneTypeLifePictorialBookCountReq、4525 TreasureIllustrationRewardReq、4527 GetTreasureAchieveRewardReq |
| ProductionListSystem | 1369 GetLifeInfoReq*、1379 CookingFoodReq、1381 MakePropReq、1383 HandicraftReq、1385 SewingReq、1387 GetLifeAchievementRewardReq、1389 LifeLeveUpReq、1393 LifeSkillLevelUpReq、1395 GetLifeSkillReq |
| FishingSystem | 1397 FishingReq |
| BagsSystem | 1401 GetWeaponReq*、1403 GetArmorReq*、1405 GetPosterReq*、1407 TempPackItemDropReq、1409 TempPackItemStoreReq、1411 TempPackSortReq、1415 GetAllCharacterEquipReq*、1417 CharacterEquipUpdateReq*、1421 PosterStarUpReq、1423 PosterIllustrationListReq*、1957 EquipDisassembleReq、4535 PetDecomposeReq、4539 GetPetReq* |
| UIPosterListSystem | 1425 PosterIllustrationRewardReq |
| GachaSystem | 1443 GachaListReq* |
| UIGachaSystem | 1445 GachaReq*、1447 GachaFullPickReq、1449 GachaRecordReq*、2207 OptionalUpPoolItemReq*、4543 ChooseGachaReq* |
| GamePlayRewardManager | 1451 GamePlayRewardReq* |
| UIPlayerSystem | 1513 ChangeSignReq*、1515 ChangeHeadReq、1517 ChangePhoneBackgroundReq*、1521 ChangePlayerSexReq、1529 UnlockHeadListReq*、1539 ChangePhoneCaseReq、2313 ChangeIsHideBirthdayReq*、2317 ChangeHideTypeReq* |
| ConditionSystem | 1523 WorldLevelAchieveListReq*、1723 AcceptQuestReq* |
| UIWorldLevelSystem | 1525 UnlockWorldLevelReq、1527 ChangeWorldLevelReq |
| StatisticSystem | 1589 TutorialReq* |
| AbilitySystem | 1611 PlayerAbilityListReq*、1613 PlayerAbilityLevelUpReq |
| MoonStoneSystem | 1631 AbilityBadgeListReq*、1637 AbilityBadgeAchieveRewardReq |
| UIMenuStoneSystem | 1633 AbilityBadgePageBoxActiveReq、1635 AbilityBadgePageRewardReq |
| TeamModel | 1641 UpdateTeamReq* |
| UIAvatarTipsSystem | 1645 OutfitDyeUnlockIndexReq、2561 PhotoShareCopyFashionDyeReq |
| FashionSystem | 1647 OutfitDyeReq*、1651 OutfitColorantSelectReq*、1655 OutFitDyeSaveReq*、1659 OutfitPresetSwitchReq* |
| UIFashionSystem | 1657 OutfitPresetUpdateReq*、2301 GenericGameAReq* |
| PlayerEntity | 1671 UpdateCharacterAppearanceReq*、2631 UpdatePlayerAppearanceReq* |
| ShopSystem | 1675 ShopInfoReq*、2101 CreatePayOrderReq*、2309 MonthCardRewardReq |
| UIMainSystem | 1677 ShopBuyReq*、1871 SceneProcessListReq*、1893 SupplyBoxRewardReq、1911 PopEmojiReq、2209 ActionStudyReq |
| HomeSystem | 1681 PlaceFurnitureReq*、1683 TakeOutFurnitureReq*、1685 GetGardenInfoReq*、1687 SwitchGardenStatusReq*、1693 DoLikesReq、2205 GardenLikeRecordReq*、2507 HandingFurnitureReq*、2509 TakeOutHandingFurnitureReq*、2625 GardenPlaceCharacterReq*、2743 PlaceMoonSpotReq、2747 SetRoomDecorReq |
| UIWeaponStrengSystem | 1701 WeaponInscriptionInlaidReq、1703 WeaponInscriptionRemoveReq、1961 WeaponStrengthReq |
| StorySystem | 1725 ClaimQuestRewardReq |
| CollectionEntity | 1741 CollectingReq* |
| UICollectionInfoSystem | 1743 CollectionRewardReq* |
| TreasureEntity | 1745 TreasureBoxOpenReq*、1747 TreasureBoxPickupReq* |
| DropItemSystem | 1749 PickupReq* |
| GatheringSystem | 1751 GatherReq* |
| UIAchieveQuestSystem | 1755 GetAchieveGroupAwardReq、1759 GetOneAchieveAwardReq |
| AchieveQuestSystem | 1757 GetAchieveGroupListReq*、1761 GetAchieveOneGroupReq* |
| ChairSystem | 1801 SceneSitChairReq* |
| ExploreSystem | 1819 ExploreInitReq*、1823 ExploreQuickFinishReq、1825 ExploreRewardReq、1827 ExploreCancelReq、1841 FriendIntervalInitReq*、1843 SelfIntervalInitReq* |
| UIExploreSystem | 1821 ExploreReq、1829 ExploreCollectRewardReq、1833 IntervalStartReq、1835 IntervalJoinReq、1837 IntervalQuickReq、1839 IntervalRewardReq |
| ManualSystem | 1861 ManualListReq*、1863 ManualFlagUnlockReq、1997 GetCollectItemIdsReq* |
| UIManualSystem | 1865 ManualFlagRewardReq |
| SceneAreaSystem | 1901 AreaCloseReq*、1907 AreaAchieveListReq* |
| SceneAreaEntity | 1903 AreaUnlockReq* |
| UIAreaSettingSystem | 1905 AreaLevelUpReq |
| CampsiteSystem | 1913 PlaceCampFireReq、1923 RemoveCampFireReq |
| ChatSystem | 1930 ChangeChatChannelReq* |
| UIChatSystem | 1933 SendChatMsgReq*、2013 PrivateChatMsgRecordReq* |
| VoiceRegionEntity | 1951 CollectVoiceRegionReq |
| TeamEntity | 1953 CharacterDeadReq |
| UIBagSystem | 1959 ItemUseReq |
| UIArmorStrengSystem | 1963 ArmorStrengthReq |
| UnitEmotionSystem | 1967 SendActionReq* |
| MultiplayerActionSystem | 1977 MultipleActionAcceptReq |
| UIActivitySystem | 1985 ActivitySignInReq、1995 ActivityRegularRewardReq、2121 ActivityGiftRewardReq、2133 GetActivityBattlePassLevelRewardReq、2252 ActivityInviteRewardClaimReq、2254 InviteCodeUseReq、2635 ActivityTurntableDrawReq、2637 ActivityTurntableRecordReq、4511 ActivityRecallRewardClaimReq、4513 RecallCodeUseReq、4515 ActivityUploadWishTxtReq、4519 ActivityKoiUseItemReq |
| ActivitySystem | 1991 ActivityQuestRewardReq、2131 ActivityBattlePassBuyExpReq、2135 GetActivityBattlePassQuestRewardReq、4501 ActivitySandbagRewardReq、4503 ActivitySandbagHitReq |
| TogetherMovementSystem | 1999 ClapReq |
| DiceGameSystem | 2003 ThrowDiceReq |
| AbyssSystem | 2111 AbyssInfoReq*、2115 AbyssTeamUpdateReq |
| UIAbyssSystem | 2113 AbyssQuestRewardReq |
| UIRankListSystem | 2119 AbyssFriendRankReq、2661 AbyssAllServerRankReq |
| UISettingSystem | 2151 UseRedemptionCodeReq |
| CharacterEntity | 2213 CharacterReviveReq |
| SceneCollectionsSystem | 2303 GenericGameBReq*、2577 GetCollectMoonInfoReq* |
| OnlineTeamSystem | 2305 GenericSceneAReq、2406 MPTeamCreateReq、2414 MPBeaconActionReq、2445 MPTeamJoinReq、2447 MPTeamQuitReq |
| TeamGamePlaySystem | 2412 MPSwapCharacterReq、2422 MPTeamInfoReq、2441 MPTeamRenameReq、2443 MPTeamInviteReq、2449 MPTeamKickMemberReq、2517 MPPlayRoomExitReq、2531 BuyGameCoinReq |
| UITeamGamePlaySystem | 2419 MPTeamGameModeChangeReq、2451 MPTeamDismissReq |
| MiniGameSystem | 2501 MPPlayRoomEventReq、2505 MPPlayRoomReadyReq |
| UIPhotoShareSystem | 2551 UploadPhotoShareReq、2555 PhotoShareSearchReq*、2557 PhotoShareDetailReq、2559 OperatePhotoShareReq |
| PhotoShareSystem | 2553 ChangePhotoShareTitleReq |
| UIShopSystem | 2571 GetExpressAddressReq、2573 UploadExpressAddressReq、2681 WishListOperateReq、2683 WishListByShopIdReq、2689 WishListBuyReq |
| CollectionSmallMoonEntity | 2575 CollectMoonReq* |
| CollectionBigMoonEntity | 2581 EmotionMoonInteractReq |
| BlessingTreeSystem | 2587 BlessTreeUnlockReq |
| QuestSystem | 2602 DailyTaskExchangeReq |
| UIHomeSystem | 2611 GardenFurnitureRemoveAllReq*、2615 GardenFurnitureSaveReq*、2617 GardenFurnitureApplySchemeReq*、2619 GardenSchemeFurnitureListReq*、2621 GardenFurnitureSchemeSetNameReq*、2623 GardenFurnitureSchemeReq* |
| UIFriendSystem | 2651 UseItemFriendIntimacyReq |
| MusicInstrumentController | 2671 ChangeMusicalItemReq* |
| MusicalInstrumentSystem | 2673 PlayMusicNoteReq* |
| ShopModel | 2685 WishListByFriendIdReq* |
| BossRushSystem | 2701 BossRushInfoReq*、2703 BossRushStartChallengeReq、2715 BossRushTerminateChallengeReq |
| BossRushModel | 2705 BossRushEnterStageReq、2709 BossRushLeaveStageReq |
| UIBossRushSystem | 2711 BossRushFriendRankReq、2717 BossRushQuestRewardReq |
| UIRestaurantGameSystem | 2731 RestaurantGameInfoReq、2733 RestaurantGameOperateReq、2735 RestaurantGameEatReq、2739 RestaurantGameQuestRewardReq |
| VehicleSystem | 2765 SceneSitVehicleReq* |
| GatherItemEntity | 4521 TreasureDigReq |
| ArchaeologySystem | 4523 TreasureIllustrationInfoReq* |
| PetSystem | 4531 PetCaptureReq |
| UIPetSystem | 4533 PetRenameReq |
| UIGomokuSystem | 4600 GomokuSinglePlayerRewardReq |
| GomokuSystem | 4602 GomokuOnlinePlayerReadyReq、4604 GomokuOnlinePlayerWatchReq、4609 GetGomokuOnlineChessboardReq、4611 GomokuOnlinePlayerPlaceStoneReq、4614 GomokuOnlinePlayerLeaveReq、4617 GomokuOnlinePlayerStartPlayReq |
| TeamPlayCollaseController | 4700 CollapseFailReq |
| 其他 | 1007 PlayerPingReq*、1009 PlayerOfflineReq、1203 PlayerSceneRecordReq*、1211 SetArchiveInfoReq*、1391 GetLifeAchieveReq、1413 TempPackWearEquipReq、1427 LockEquipReq、1531 PlayerLevelRewardReq、1653 OutfitRandomColorReq、1803 NpcTalkReq*、1851 MonsterDeadReq*、1919 SceneInterActionPlayStatusReq*、2170 ReportReq、2201 ClientLogAuthReq*、2203 ClientLogMessageReq*、2263 AccountNameBindingCancelReq、2431 MPGameStartReq、2469 MPBeaconPlayerReadyReq、2511 SceneTransmitReq、2515 MPPlayRoomRewardInfoReq、2521 UploadFGBytesReq、2563 PhotoSharePaymentAddUploadGridReq、2585 BlessTreeReq、2695 SPOperateReq、2713 BossRushFriendDetailReq、3001 InheritCodeUseReq |

## 推送

服务端推送 127 个（`cmdid.csv` 中的 `*Notice`）。客户端消费方尚未建索引，实现所属系统时按协议与客户端代码核对发送时机与字段：

1016 SceneDataNotice、1022 NeedLoginNotice、1044 TeamCharExpUpdateNotice、1128 ReceiveMailNotice、1206 PlayerSceneSyncDataNotice、1208 ServerSceneSyncDataNotice、1352 PlayerTimeOffsetNotice、1368 LifeProficiencyNotice、1374 FishingResultNotice、1376 LifeSkillLevelUpNotice、1378 LifeAchieveNotice、1400 PackNotice、1442 PlayerTagUpdateNotice、1454 ModuleCloseNotice、1520 PlayerLevelExpNotice、1534 ForbiddenInfoNotice、1536 ClientIFIxGMNotice、1538 SpeechRecyclingNotice、1580 FlagUnlockAddNotice、1582 FlagUnLockRemoveNotice、1600 PlayerUnlockFunctionNotice、1616 PlayerAbilityUnlockNotice、1650 OutfitPresetUpdateNotice、1674 CharacterAppearanceUpdateNotice、1680 ShopRefreshNotice、1696 SceneGardenFurnitureUpdateNotice、1698 SceneGardenFurnitureRemoveNotice、1706 ShopInitNotice、1718 QuestNotice、1720 AchieveNotice、1722 GmNotice、1738 FriendHandleNotice、1754 GatherSceneLimitRecoveryNotice、1764 AchieveFinishNotice、1832 IntervalChangeNotice、1880 PlayerBuffNotice、1910 PopEmojiNotice、1916 PlaceCampFireNotice、1918 SceneWeatherChangeNotice、1922 SceneInterActionPlayStatusNotice、1926 RemoveCampFireNotice、1936 ChatMsgNotice、1938 ChatMsgRecordInitNotice、1940 ChatUnLockExpressionNotice、1970 SendActionNotice、1972 SendActionStudyNotice、1974 SendActionAddNotice、1976 SendMultipleActionNotice、1980 SendMultipleActionCompleteNotice、1982 ActivityChangeNotice、1984 ActivitySignInDataNotice、1988 PlayerNotInSceneChannelNotice、1990 ActivityQuestDataNotice、1994 ActivityRegularDataNotice、2002 ClapResultNotice、2006 ThrowDiceNotice、2008 ThrowDiceResultNotice、2010 AchieveActionUnLockNotice、2012 PrivateChatOfflineNotice、2016 SystemNotice、2018 PlayerTiredDropNotice、2020 NewQuestionnaireNotice、2104 PaySendGoodsNotice、2106 PackNotShowNotice、2110 AbyssSeasonNotice、2124 ActivityGiftDataNotice、2126 MonthCardNotice、2128 CharacterBpBuyNotice、2130 ActivityBattlePassBuyNotice、2138 ActivityBattlePassInfoNotice、2140 ActivityGiftBuyNotice、2160 FireworksStartNotice、2212 ActionStudyNotice、2231 QuestionnaireRemoveNotice、2232 ChangeChatChannelNotice、2251 ActivityInviteNotice、2256 ActivityInviteCountUpdateNotice、2312 FurnitureItemChangeNotice、2320 GameRefreshNotice、2418 MPTeamBeaconNotice、2421 MPTeamGameModeChangeNotice、2424 SceneMPBeaconNotice、2430 MPBeaconCanStartNotice、2453 MPTeamDismissNotice、2454 MPTeamInviteNotice、2455 MPTeamPlayerKickedNotice、2456 PlayerNotInMPTeamNotice、2458 MPTeamMemberNotice、2460 MPTeamRenameNotice、2462 PlayerMPGameNotice、2465 MPRoomAssignHostNotice、2467 MPRoomDismissNotice、2468 MPRoomEnterNotice、2504 MPPlayRoomEventNotice、2514 MPPlayRoomSettlementNotice、2519 MPPlayRoomExitNotice、2520 UploadFGBytesNotice、2534 FreezePlayerActionNotice、2580 CollectMoonInfoUpdateNotice、2584 BlessTreeNotice、2601 DailyTaskUpdateNotice、2604 MPGameWeaponUpdateNotice、2614 GardenFurnitureBatchUpdateNotice、2628 GardenPlaceCharacterNotice、2630 GMRecommendChannelNotice、2634 ActivityTurntableDataNotice、2648 AbyssServerRankNotice、2658 IntimacyGiftDayCountNotice、2688 WishListBuyNotice、2738 RestaurantGameDayNotice、2742 MoonQuestGamePlayRewardNotice、2746 PlaceMoonSpotNotice、2750 RoomDecorNotice、2760 DungeonOpenFinalBoxNotice、4500 ActivitySandbagNotice、4508 ActivitySandbagDropNotice、4510 ActivityRecallNotice、4518 ActivityKoiNotice、4538 PetWarehouseLimitNotice、4606 GomokuOnlinePlayerNotice、4607 GomokuOnlinePlayerWatchNotice、4608 GomokuOnlinePlayerRemoveNotice、4613 GomokuOnlinePlayerPlaceStoneNotice、4616 GomokuOnlinePlayerLeaveNotice、4619 GomokuOnlinePlayerStartPlayNotice、4620 GomokuOnlineSettlementNotice、4702 CollapseFailNotice
