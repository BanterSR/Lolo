package cmd

import (
	"gucooing/lolo/protocol/proto"
)

const (
	InValid                                 = -1
	VerifyLoginTokenReq                     = 1001
	VerifyLoginTokenRsp                     = 1002
	PlayerLoginReq                          = 1003
	PlayerLoginRsp                          = 1004
	PlayerMainDataReq                       = 1005
	PlayerMainDataRsp                       = 1006
	PlayerPingReq                           = 1007
	PlayerPingRsp                           = 1008
	PlayerOfflineReq                        = 1009
	PlayerOfflineRsp                        = 1010
	GmCodeReq                               = 1013
	GmCodeRsp                               = 1014
	SceneDataNotice                         = 1016
	NeedLoginNotice                         = 1022
	CharacterSkillLevelUpReq                = 1035
	CharacterSkillLevelUpRsp                = 1036
	CharacterStarUpReq                      = 1037
	CharacterStarUpRsp                      = 1038
	CharacterLevelUpReq                     = 1039
	CharacterLevelUpRsp                     = 1040
	CharacterLevelBreakReq                  = 1041
	CharacterLevelBreakRsp                  = 1042
	TeamCharExpUpdateNotice                 = 1044
	PlayerEnergyInfoReq                     = 1051
	PlayerEnergyInfoRsp                     = 1052
	PlayerEnergyBuyReq                      = 1053
	PlayerEnergyBuyRsp                      = 1054
	GetMailsReq                             = 1121
	GetMailsRsp                             = 1122
	OperateMailsReq                         = 1125
	OperateMailsRsp                         = 1126
	ReceiveMailNotice                       = 1128
	ChangeSceneChannelReq                   = 1201
	ChangeSceneChannelRsp                   = 1202
	PlayerSceneRecordReq                    = 1203
	PlayerSceneRecordRsp                    = 1204
	PlayerSceneSyncDataNotice               = 1206
	ServerSceneSyncDataNotice               = 1208
	SetArchiveInfoReq                       = 1211
	SetArchiveInfoRsp                       = 1212
	GetArchiveInfoReq                       = 1213
	GetArchiveInfoRsp                       = 1214
	ChallengeFriendRankReq                  = 1301
	ChallengeFriendRankRsp                  = 1302
	ChallengeStateUpdateReq                 = 1321
	ChallengeStateUpdateRsp                 = 1322
	RiddleStateUpdateReq                    = 1323
	RiddleStateUpdateRsp                    = 1324
	FlagBattleStateUpdateReq                = 1325
	FlagBattleStateUpdateRsp                = 1326
	BattleEncounterStateUpdateReq           = 1327
	BattleEncounterStateUpdateRsp           = 1328
	BattleEncounterInfoReq                  = 1329
	BattleEncounterInfoRsp                  = 1330
	DungeonRepeatEnterReq                   = 1331
	DungeonRepeatEnterRsp                   = 1332
	DungeonViewReq                          = 1333
	DungeonViewRsp                          = 1334
	DungeonEnterReq                         = 1335
	DungeonEnterRsp                         = 1336
	DungeonExitReq                          = 1337
	DungeonExitRsp                          = 1338
	DungeonFinishReq                        = 1339
	DungeonFinishRsp                        = 1340
	CollectCoinReq                          = 1341
	CollectCoinRsp                          = 1342
	DungeonTaskFinishRewardReq              = 1343
	DungeonTaskFinishRewardRsp              = 1344
	DungeonStarRewardReq                    = 1345
	DungeonStarRewardRsp                    = 1346
	DungeonOperateReq                       = 1347
	DungeonOperateRsp                       = 1348
	PlayerTimeOffsetNotice                  = 1352
	DungeonSweepReq                         = 1353
	DungeonSweepRsp                         = 1354
	ItemSweepReq                            = 1355
	ItemSweepRsp                            = 1356
	GetOneTypeLifePictorialBookCountReq     = 1365
	GetOneTypeLifePictorialBookCountRsp     = 1366
	LifeProficiencyNotice                   = 1368
	GetLifeInfoReq                          = 1369
	GetLifeInfoRsp                          = 1370
	FishingResultNotice                     = 1374
	LifeSkillLevelUpNotice                  = 1376
	LifeAchieveNotice                       = 1378
	CookingFoodReq                          = 1379
	CookingFoodRsp                          = 1380
	MakePropReq                             = 1381
	MakePropRsp                             = 1382
	HandicraftReq                           = 1383
	HandicraftRsp                           = 1384
	SewingReq                               = 1385
	SewingRsp                               = 1386
	GetLifeAchievementRewardReq             = 1387
	GetLifeAchievementRewardRsp             = 1388
	LifeLeveUpReq                           = 1389
	LifeLeveUpRsp                           = 1390
	GetLifeAchieveReq                       = 1391
	GetLifeAchieveRsp                       = 1392
	LifeSkillLevelUpReq                     = 1393
	LifeSkillLevelUpRsp                     = 1394
	GetLifeSkillReq                         = 1395
	GetLifeSkillRsp                         = 1396
	FishingReq                              = 1397
	FishingRsp                              = 1398
	PackNotice                              = 1400
	GetWeaponReq                            = 1401
	GetWeaponRsp                            = 1402
	GetArmorReq                             = 1403
	GetArmorRsp                             = 1404
	GetPosterReq                            = 1405
	GetPosterRsp                            = 1406
	TempPackItemDropReq                     = 1407
	TempPackItemDropRsp                     = 1408
	TempPackItemStoreReq                    = 1409
	TempPackItemStoreRsp                    = 1410
	TempPackSortReq                         = 1411
	TempPackSortRsp                         = 1412
	TempPackWearEquipReq                    = 1413
	TempPackWearEquipRsp                    = 1414
	GetAllCharacterEquipReq                 = 1415
	GetAllCharacterEquipRsp                 = 1416
	CharacterEquipUpdateReq                 = 1417
	CharacterEquipUpdateRsp                 = 1418
	CharacterEquipPresetSwitchReq           = 1419
	CharacterEquipPresetSwitchRsp           = 1420
	PosterStarUpReq                         = 1421
	PosterStarUpRsp                         = 1422
	PosterIllustrationListReq               = 1423
	PosterIllustrationListRsp               = 1424
	PosterIllustrationRewardReq             = 1425
	PosterIllustrationRewardRsp             = 1426
	LockEquipReq                            = 1427
	LockEquipRsp                            = 1428
	PlayerTagUpdateNotice                   = 1442
	GachaListReq                            = 1443
	GachaListRsp                            = 1444
	GachaReq                                = 1445
	GachaRsp                                = 1446
	GachaFullPickReq                        = 1447
	GachaFullPickRsp                        = 1448
	GachaRecordReq                          = 1449
	GachaRecordRsp                          = 1450
	GamePlayRewardReq                       = 1451
	GamePlayRewardRsp                       = 1452
	ModuleCloseNotice                       = 1454
	GetCharacterAchievementListReq          = 1479
	GetCharacterAchievementListRsp          = 1480
	GetCharacterAchievementAwardReq         = 1481
	GetCharacterAchievementAwardRsp         = 1482
	GetCharacterAchievementUnlockPaymentReq = 1483
	GetCharacterAchievementUnlockPaymentRsp = 1484
	GetCharacterAchievementBadgeAwardReq    = 1485
	GetCharacterAchievementBadgeAwardRsp    = 1486
	ChangeNickNameReq                       = 1511
	ChangeNickNameRsp                       = 1512
	ChangeSignReq                           = 1513
	ChangeSignRsp                           = 1514
	ChangeHeadReq                           = 1515
	ChangeHeadRsp                           = 1516
	ChangePhoneBackgroundReq                = 1517
	ChangePhoneBackgroundRsp                = 1518
	PlayerLevelExpNotice                    = 1520
	ChangePlayerSexReq                      = 1521
	ChangePlayerSexRsp                      = 1522
	WorldLevelAchieveListReq                = 1523
	WorldLevelAchieveListRsp                = 1524
	UnlockWorldLevelReq                     = 1525
	UnlockWorldLevelRsp                     = 1526
	ChangeWorldLevelReq                     = 1527
	ChangeWorldLevelRsp                     = 1528
	UnlockHeadListReq                       = 1529
	UnlockHeadListRsp                       = 1530
	PlayerLevelRewardReq                    = 1531
	PlayerLevelRewardRsp                    = 1532
	ForbiddenInfoNotice                     = 1534
	ClientIFIxGMNotice                      = 1536
	SpeechRecyclingNotice                   = 1538
	FlagUnlockAddNotice                     = 1580
	FlagUnLockRemoveNotice                  = 1582
	TutorialReq                             = 1589
	TutorialRsp                             = 1590
	PlayerUnlockFunctionNotice              = 1600
	PlayerAbilityListReq                    = 1611
	PlayerAbilityListRsp                    = 1612
	PlayerAbilityLevelUpReq                 = 1613
	PlayerAbilityLevelUpRsp                 = 1614
	PlayerAbilityUnlockNotice               = 1616
	PlayerVitalityReq                       = 1621
	PlayerVitalityRsp                       = 1622
	PlayerVitalityBuyReq                    = 1623
	PlayerVitalityBuyRsp                    = 1624
	AbilityBadgeListReq                     = 1631
	AbilityBadgeListRsp                     = 1632
	AbilityBadgePageBoxActiveReq            = 1633
	AbilityBadgePageBoxActiveRsp            = 1634
	AbilityBadgePageRewardReq               = 1635
	AbilityBadgePageRewardRsp               = 1636
	AbilityBadgeAchieveRewardReq            = 1637
	AbilityBadgeAchieveRewardRsp            = 1638
	UpdateTeamReq                           = 1641
	UpdateTeamRsp                           = 1642
	OutfitDyeUnlockIndexReq                 = 1645
	OutfitDyeUnlockIndexRsp                 = 1646
	OutfitDyeReq                            = 1647
	OutfitDyeRsp                            = 1648
	OutfitPresetUpdateNotice                = 1650
	OutfitColorantSelectReq                 = 1651
	OutfitColorantSelectRsp                 = 1652
	OutfitRandomColorReq                    = 1653
	OutfitRandomColorRsp                    = 1654
	OutFitDyeSaveReq                        = 1655
	OutFitDyeSaveRsp                        = 1656
	OutfitPresetUpdateReq                   = 1657
	OutfitPresetUpdateRsp                   = 1658
	OutfitPresetSwitchReq                   = 1659
	OutfitPresetSwitchRsp                   = 1660
	OutfitHideSwitchReq                     = 1661
	OutfitHideSwitchRsp                     = 1662
	UpdateCharacterAppearanceReq            = 1671
	UpdateCharacterAppearanceRsp            = 1672
	CharacterAppearanceUpdateNotice         = 1674
	ShopInfoReq                             = 1675
	ShopInfoRsp                             = 1676
	ShopBuyReq                              = 1677
	ShopBuyRsp                              = 1678
	ShopRefreshNotice                       = 1680
	PlaceFurnitureReq                       = 1681
	PlaceFurnitureRsp                       = 1682
	TakeOutFurnitureReq                     = 1683
	TakeOutFurnitureRsp                     = 1684
	GetGardenInfoReq                        = 1685
	GetGardenInfoRsp                        = 1686
	SwitchGardenStatusReq                   = 1687
	SwitchGardenStatusRsp                   = 1688
	DoLikesReq                              = 1693
	DoLikesRsp                              = 1694
	SceneGardenFurnitureUpdateNotice        = 1696
	SceneGardenFurnitureRemoveNotice        = 1698
	WeaponInscriptionInlaidReq              = 1701
	WeaponInscriptionInlaidRsp              = 1702
	WeaponInscriptionRemoveReq              = 1703
	WeaponInscriptionRemoveRsp              = 1704
	ShopInitNotice                          = 1706
	QuestNotice                             = 1718
	AchieveNotice                           = 1720
	GmNotice                                = 1722
	AcceptQuestReq                          = 1723
	AcceptQuestRsp                          = 1724
	ClaimQuestRewardReq                     = 1725
	ClaimQuestRewardRsp                     = 1726
	FriendSearchReq                         = 1727
	FriendSearchRsp                         = 1728
	FriendAddReq                            = 1729
	FriendAddRsp                            = 1730
	FriendDelReq                            = 1731
	FriendDelRsp                            = 1732
	FriendBlackReq                          = 1733
	FriendBlackRsp                          = 1734
	FriendHandleReq                         = 1735
	FriendHandleRsp                         = 1736
	FriendHandleNotice                      = 1738
	FriendReq                               = 1739
	FriendRsp                               = 1740
	CollectingReq                           = 1741
	CollectingRsp                           = 1742
	CollectionRewardReq                     = 1743
	CollectionRewardRsp                     = 1744
	TreasureBoxOpenReq                      = 1745
	TreasureBoxOpenRsp                      = 1746
	TreasureBoxPickupReq                    = 1747
	TreasureBoxPickupRsp                    = 1748
	PickupReq                               = 1749
	PickupRsp                               = 1750
	GatherReq                               = 1751
	GatherRsp                               = 1752
	GatherSceneLimitRecoveryNotice          = 1754
	GetAchieveGroupAwardReq                 = 1755
	GetAchieveGroupAwardRsp                 = 1756
	GetAchieveGroupListReq                  = 1757
	GetAchieveGroupListRsp                  = 1758
	GetOneAchieveAwardReq                   = 1759
	GetOneAchieveAwardRsp                   = 1760
	GetAchieveOneGroupReq                   = 1761
	GetAchieveOneGroupRsp                   = 1762
	AchieveFinishNotice                     = 1764
	SetFriendExtInfoReq                     = 1781
	SetFriendExtInfoRsp                     = 1782
	SceneSitChairReq                        = 1801
	SceneSitChairRsp                        = 1802
	NpcTalkReq                              = 1803
	NpcTalkRsp                              = 1804
	ExploreInitReq                          = 1819
	ExploreInitRsp                          = 1820
	ExploreReq                              = 1821
	ExploreRsp                              = 1822
	ExploreQuickFinishReq                   = 1823
	ExploreQuickFinishRsp                   = 1824
	ExploreRewardReq                        = 1825
	ExploreRewardRsp                        = 1826
	ExploreCancelReq                        = 1827
	ExploreCancelRsp                        = 1828
	ExploreCollectRewardReq                 = 1829
	ExploreCollectRewardRsp                 = 1830
	IntervalChangeNotice                    = 1832
	IntervalStartReq                        = 1833
	IntervalStartRsp                        = 1834
	IntervalJoinReq                         = 1835
	IntervalJoinRsp                         = 1836
	IntervalQuickReq                        = 1837
	IntervalQuickRsp                        = 1838
	IntervalRewardReq                       = 1839
	IntervalRewardRsp                       = 1840
	FriendIntervalInitReq                   = 1841
	FriendIntervalInitRsp                   = 1842
	SelfIntervalInitReq                     = 1843
	SelfIntervalInitRsp                     = 1844
	MonsterDeadReq                          = 1851
	MonsterDeadRsp                          = 1852
	ManualListReq                           = 1861
	ManualListRsp                           = 1862
	ManualFlagUnlockReq                     = 1863
	ManualFlagUnlockRsp                     = 1864
	ManualFlagRewardReq                     = 1865
	ManualFlagRewardRsp                     = 1866
	SceneProcessListReq                     = 1871
	SceneProcessListRsp                     = 1872
	PlayerBuffNotice                        = 1880
	SupplyBoxInfoReq                        = 1891
	SupplyBoxInfoRsp                        = 1892
	SupplyBoxRewardReq                      = 1893
	SupplyBoxRewardRsp                      = 1894
	AreaCloseReq                            = 1901
	AreaCloseRsp                            = 1902
	AreaUnlockReq                           = 1903
	AreaUnlockRsp                           = 1904
	AreaLevelUpReq                          = 1905
	AreaLevelUpRsp                          = 1906
	AreaAchieveListReq                      = 1907
	AreaAchieveListRsp                      = 1908
	PopEmojiNotice                          = 1910
	PopEmojiReq                             = 1911
	PopEmojiRsp                             = 1912
	PlaceCampFireReq                        = 1913
	PlaceCampFireRsp                        = 1914
	PlaceCampFireNotice                     = 1916
	SceneWeatherChangeNotice                = 1918
	SceneInterActionPlayStatusReq           = 1919
	SceneInterActionPlayStatusRsp           = 1920
	SceneInterActionPlayStatusNotice        = 1922
	RemoveCampFireReq                       = 1923
	RemoveCampFireRsp                       = 1924
	RemoveCampFireNotice                    = 1926
	ChangeChatChannelReq                    = 1930
	ChangeChatChannelRsp                    = 1931
	SendChatMsgReq                          = 1933
	SendChatMsgRsp                          = 1934
	ChatMsgNotice                           = 1936
	ChatMsgRecordInitNotice                 = 1938
	ChatUnLockExpressionNotice              = 1940
	CollectVoiceRegionReq                   = 1951
	CollectVoiceRegionRsp                   = 1952
	CharacterDeadReq                        = 1953
	CharacterDeadRsp                        = 1954
	CharacterGatherWeaponUpdateReq          = 1955
	CharacterGatherWeaponUpdateRsp          = 1956
	EquipDisassembleReq                     = 1957
	EquipDisassembleRsp                     = 1958
	ItemUseReq                              = 1959
	ItemUseRsp                              = 1960
	WeaponStrengthReq                       = 1961
	WeaponStrengthRsp                       = 1962
	ArmorStrengthReq                        = 1963
	ArmorStrengthRsp                        = 1964
	OtherPlayerInfoReq                      = 1965
	OtherPlayerInfoRsp                      = 1966
	SendActionReq                           = 1967
	SendActionRsp                           = 1968
	SendActionNotice                        = 1970
	SendActionStudyNotice                   = 1972
	SendActionAddNotice                     = 1974
	SendMultipleActionNotice                = 1976
	MultipleActionAcceptReq                 = 1977
	MultipleActionAcceptRsp                 = 1978
	SendMultipleActionCompleteNotice        = 1980
	ActivityChangeNotice                    = 1982
	ActivitySignInDataNotice                = 1984
	ActivitySignInReq                       = 1985
	ActivitySignInRsp                       = 1986
	PlayerNotInSceneChannelNotice           = 1988
	ActivityQuestDataNotice                 = 1990
	ActivityQuestRewardReq                  = 1991
	ActivityQuestRewardRsp                  = 1992
	ActivityRegularDataNotice               = 1994
	ActivityRegularRewardReq                = 1995
	ActivityRegularRewardRsp                = 1996
	GetCollectItemIdsReq                    = 1997
	GetCollectItemIdsRsp                    = 1998
	ClapReq                                 = 1999
	ClapRsp                                 = 2000
	ClapResultNotice                        = 2002
	ThrowDiceReq                            = 2003
	ThrowDiceRsp                            = 2004
	ThrowDiceNotice                         = 2006
	ThrowDiceResultNotice                   = 2008
	AchieveActionUnLockNotice               = 2010
	PrivateChatOfflineNotice                = 2012
	PrivateChatMsgRecordReq                 = 2013
	PrivateChatMsgRecordRsp                 = 2014
	SystemNotice                            = 2016
	PlayerTiredDropNotice                   = 2018
	NewQuestionnaireNotice                  = 2020
	CreatePayOrderReq                       = 2101
	CreatePayOrderRsp                       = 2102
	PaySendGoodsNotice                      = 2104
	PackNotShowNotice                       = 2106
	AbyssSeasonNotice                       = 2110
	AbyssInfoReq                            = 2111
	AbyssInfoRsp                            = 2112
	AbyssQuestRewardReq                     = 2113
	AbyssQuestRewardRsp                     = 2114
	AbyssTeamUpdateReq                      = 2115
	AbyssTeamUpdateRsp                      = 2116
	AbyssTeamSwitchReq                      = 2117
	AbyssTeamSwitchRsp                      = 2118
	AbyssFriendRankReq                      = 2119
	AbyssFriendRankRsp                      = 2120
	ActivityGiftRewardReq                   = 2121
	ActivityGiftRewardRsp                   = 2122
	ActivityGiftDataNotice                  = 2124
	MonthCardNotice                         = 2126
	CharacterBpBuyNotice                    = 2128
	ActivityBattlePassBuyNotice             = 2130
	ActivityBattlePassBuyExpReq             = 2131
	ActivityBattlePassBuyExpRsp             = 2132
	GetActivityBattlePassLevelRewardReq     = 2133
	GetActivityBattlePassLevelRewardRsp     = 2134
	GetActivityBattlePassQuestRewardReq     = 2135
	GetActivityBattlePassQuestRewardRsp     = 2136
	ActivityBattlePassInfoNotice            = 2138
	ActivityGiftBuyNotice                   = 2140
	UseRedemptionCodeReq                    = 2151
	UseRedemptionCodeRsp                    = 2152
	FireworksStartNotice                    = 2160
	ReportReq                               = 2170
	ReportRsp                               = 2171
	ClientLogAuthReq                        = 2201
	ClientLogAuthRsp                        = 2202
	ClientLogMessageReq                     = 2203
	ClientLogMessageRsp                     = 2204
	GardenLikeRecordReq                     = 2205
	GardenLikeRecordRsp                     = 2206
	OptionalUpPoolItemReq                   = 2207
	OptionalUpPoolItemRsp                   = 2208
	ActionStudyReq                          = 2209
	ActionStudyRsp                          = 2210
	ActionStudyNotice                       = 2212
	CharacterReviveReq                      = 2213
	CharacterReviveRsp                      = 2214
	QuestionnaireRemoveNotice               = 2231
	ChangeChatChannelNotice                 = 2232
	ActivityInviteNotice                    = 2251
	ActivityInviteRewardClaimReq            = 2252
	ActivityInviteRewardClaimRsp            = 2253
	InviteCodeUseReq                        = 2254
	InviteCodeUseRsp                        = 2255
	ActivityInviteCountUpdateNotice         = 2256
	AccountNameBindingCancelReq             = 2263
	AccountNameBindingCancelRsp             = 2264
	GenericGameAReq                         = 2301
	GenericGameARsp                         = 2302
	GenericGameBReq                         = 2303
	GenericGameBRsp                         = 2304
	GenericSceneAReq                        = 2305
	GenericSceneARsp                        = 2306
	GenericSceneBReq                        = 2307
	GenericSceneBRsp                        = 2308
	MonthCardRewardReq                      = 2309
	MonthCardRewardRsp                      = 2310
	FurnitureItemChangeNotice               = 2312
	ChangeIsHideBirthdayReq                 = 2313
	ChangeIsHideBirthdayRsp                 = 2314
	TransmitSceneReq                        = 2315
	TransmitSceneRsp                        = 2316
	ChangeHideTypeReq                       = 2317
	ChangeHideTypeRsp                       = 2318
	GameRefreshNotice                       = 2320
	MPTeamCreateReq                         = 2406
	MPTeamCreateRsp                         = 2407
	MPSwapCharacterReq                      = 2412
	MPSwapCharacterRsp                      = 2413
	MPBeaconActionReq                       = 2414
	MPBeaconActionRsp                       = 2415
	MPTeamBeaconNotice                      = 2418
	MPTeamGameModeChangeReq                 = 2419
	MPTeamGameModeChangeRsp                 = 2420
	MPTeamGameModeChangeNotice              = 2421
	MPTeamInfoReq                           = 2422
	MPTeamInfoRsp                           = 2423
	SceneMPBeaconNotice                     = 2424
	MPBeaconCanStartNotice                  = 2430
	MPGameStartReq                          = 2431
	MPGameStartRsp                          = 2432
	MPTeamRenameReq                         = 2441
	MPTeamRenameRsp                         = 2442
	MPTeamInviteReq                         = 2443
	MPTeamInviteRsp                         = 2444
	MPTeamJoinReq                           = 2445
	MPTeamJoinRsp                           = 2446
	MPTeamQuitReq                           = 2447
	MPTeamQuitRsp                           = 2448
	MPTeamKickMemberReq                     = 2449
	MPTeamKickMemberRsp                     = 2450
	MPTeamDismissReq                        = 2451
	MPTeamDismissRsp                        = 2452
	MPTeamDismissNotice                     = 2453
	MPTeamInviteNotice                      = 2454
	MPTeamPlayerKickedNotice                = 2455
	PlayerNotInMPTeamNotice                 = 2456
	MPTeamMemberNotice                      = 2458
	MPTeamRenameNotice                      = 2460
	PlayerMPGameNotice                      = 2462
	MPRoomAssignHostNotice                  = 2465
	MPRoomDismissNotice                     = 2467
	MPRoomEnterNotice                       = 2468
	MPBeaconPlayerReadyReq                  = 2469
	MPBeaconPlayerReadyRsp                  = 2470
	MPPlayRoomEventReq                      = 2501
	MPPlayRoomEventRsp                      = 2502
	MPPlayRoomEventNotice                   = 2504
	MPPlayRoomReadyReq                      = 2505
	MPPlayRoomReadyRsp                      = 2506
	HandingFurnitureReq                     = 2507
	HandingFurnitureRsp                     = 2508
	TakeOutHandingFurnitureReq              = 2509
	TakeOutHandingFurnitureRsp              = 2510
	SceneTransmitReq                        = 2511
	SceneTransmitRsp                        = 2512
	MPPlayRoomSettlementNotice              = 2514
	MPPlayRoomRewardInfoReq                 = 2515
	MPPlayRoomRewardInfoRsp                 = 2516
	MPPlayRoomExitReq                       = 2517
	MPPlayRoomExitRsp                       = 2518
	MPPlayRoomExitNotice                    = 2519
	UploadFGBytesNotice                     = 2520
	UploadFGBytesReq                        = 2521
	UploadFGBytesRsp                        = 2522
	BuyGameCoinReq                          = 2531
	BuyGameCoinRsp                          = 2532
	FreezePlayerActionNotice                = 2534
	UnfreezePlayerActionReq                 = 2535
	UnfreezePlayerActionRsp                 = 2536
	UploadPhotoShareReq                     = 2551
	UploadPhotoShareRsp                     = 2552
	ChangePhotoShareTitleReq                = 2553
	ChangePhotoShareTitleRsp                = 2554
	PhotoShareSearchReq                     = 2555
	PhotoShareSearchRsp                     = 2556
	PhotoShareDetailReq                     = 2557
	PhotoShareDetailRsp                     = 2558
	OperatePhotoShareReq                    = 2559
	OperatePhotoShareRsp                    = 2560
	PhotoShareCopyFashionDyeReq             = 2561
	PhotoShareCopyFashionDyeRsp             = 2562
	PhotoSharePaymentAddUploadGridReq       = 2563
	PhotoSharePaymentAddUploadGridRsp       = 2564
	GetExpressAddressReq                    = 2571
	GetExpressAddressRsp                    = 2572
	UploadExpressAddressReq                 = 2573
	UploadExpressAddressRsp                 = 2574
	CollectMoonReq                          = 2575
	CollectMoonRsp                          = 2576
	GetCollectMoonInfoReq                   = 2577
	GetCollectMoonInfoRsp                   = 2578
	CollectMoonInfoUpdateNotice             = 2580
	EmotionMoonInteractReq                  = 2581
	EmotionMoonInteractRsp                  = 2582
	BlessTreeNotice                         = 2584
	BlessTreeReq                            = 2585
	BlessTreeRsp                            = 2586
	BlessTreeUnlockReq                      = 2587
	BlessTreeUnlockRsp                      = 2588
	DailyTaskUpdateNotice                   = 2601
	DailyTaskExchangeReq                    = 2602
	DailyTaskExchangeRsp                    = 2603
	MPGameWeaponUpdateNotice                = 2604
	GardenFurnitureRemoveAllReq             = 2611
	GardenFurnitureRemoveAllRsp             = 2612
	GardenFurnitureBatchUpdateNotice        = 2614
	GardenFurnitureSaveReq                  = 2615
	GardenFurnitureSaveRsp                  = 2616
	GardenFurnitureApplySchemeReq           = 2617
	GardenFurnitureApplySchemeRsp           = 2618
	GardenSchemeFurnitureListReq            = 2619
	GardenSchemeFurnitureListRsp            = 2620
	GardenFurnitureSchemeSetNameReq         = 2621
	GardenFurnitureSchemeSetNameRsp         = 2622
	GardenFurnitureSchemeReq                = 2623
	GardenFurnitureSchemeRsp                = 2624
	GardenPlaceCharacterReq                 = 2625
	GardenPlaceCharacterRsp                 = 2626
	GardenPlaceCharacterNotice              = 2628
	GMRecommendChannelNotice                = 2630
	UpdatePlayerAppearanceReq               = 2631
	UpdatePlayerAppearanceRsp               = 2632
	ActivityTurntableDataNotice             = 2634
	ActivityTurntableDrawReq                = 2635
	ActivityTurntableDrawRsp                = 2636
	ActivityTurntableRecordReq              = 2637
	ActivityTurntableRecordRsp              = 2638
	AbyssServerRankNotice                   = 2648
	UseItemFriendIntimacyReq                = 2651
	UseItemFriendIntimacyRsp                = 2652
	IntimacyGiftDayCountNotice              = 2658
	AbyssAllServerRankReq                   = 2661
	AbyssAllServerRankRsp                   = 2662
	ChangeMusicalItemReq                    = 2671
	ChangeMusicalItemRsp                    = 2672
	PlayMusicNoteReq                        = 2673
	PlayMusicNoteRsp                        = 2674
	WishListOperateReq                      = 2681
	WishListOperateRsp                      = 2682
	WishListByShopIdReq                     = 2683
	WishListByShopIdRsp                     = 2684
	WishListByFriendIdReq                   = 2685
	WishListByFriendIdRsp                   = 2686
	WishListBuyNotice                       = 2688
	WishListBuyReq                          = 2689
	WishListBuyRsp                          = 2690
	SPOperateReq                            = 2695
	SPOperateRsp                            = 2696
	BossRushInfoReq                         = 2701
	BossRushInfoRsp                         = 2702
	BossRushStartChallengeReq               = 2703
	BossRushStartChallengeRsp               = 2704
	BossRushEnterStageReq                   = 2705
	BossRushEnterStageRsp                   = 2706
	BossRushStartBattleReq                  = 2707
	BossRushStartBattleRsp                  = 2708
	BossRushLeaveStageReq                   = 2709
	BossRushLeaveStageRsp                   = 2710
	BossRushFriendRankReq                   = 2711
	BossRushFriendRankRsp                   = 2712
	BossRushFriendDetailReq                 = 2713
	BossRushFriendDetailRsp                 = 2714
	BossRushTerminateChallengeReq           = 2715
	BossRushTerminateChallengeRsp           = 2716
	BossRushQuestRewardReq                  = 2717
	BossRushQuestRewardRsp                  = 2718
	RestaurantGameInfoReq                   = 2731
	RestaurantGameInfoRsp                   = 2732
	RestaurantGameOperateReq                = 2733
	RestaurantGameOperateRsp                = 2734
	RestaurantGameEatReq                    = 2735
	RestaurantGameEatRsp                    = 2736
	RestaurantGameDayNotice                 = 2738
	RestaurantGameQuestRewardReq            = 2739
	RestaurantGameQuestRewardRsp            = 2740
	MoonQuestGamePlayRewardNotice           = 2742
	PlaceMoonSpotReq                        = 2743
	PlaceMoonSpotRsp                        = 2744
	PlaceMoonSpotNotice                     = 2746
	SetRoomDecorReq                         = 2747
	SetRoomDecorRsp                         = 2748
	RoomDecorNotice                         = 2750
)

func (c *CmdProtoMap) registerAllMessage() {
	c.regMsg(VerifyLoginTokenReq, func() any { return new(proto.VerifyLoginTokenReq) })
	c.regMsg(VerifyLoginTokenRsp, func() any { return new(proto.VerifyLoginTokenRsp) })
	c.regMsg(PlayerLoginReq, func() any { return new(proto.PlayerLoginReq) })
	c.regMsg(PlayerLoginRsp, func() any { return new(proto.PlayerLoginRsp) })
	c.regMsg(PlayerMainDataReq, func() any { return new(proto.PlayerMainDataReq) })
	c.regMsg(PlayerMainDataRsp, func() any { return new(proto.PlayerMainDataRsp) })
	c.regMsg(PlayerPingReq, func() any { return new(proto.PlayerPingReq) })
	c.regMsg(PlayerPingRsp, func() any { return new(proto.PlayerPingRsp) })
	c.regMsg(PlayerOfflineReq, func() any { return new(proto.PlayerOfflineReq) })
	c.regMsg(PlayerOfflineRsp, func() any { return new(proto.PlayerOfflineRsp) })
	c.regMsg(GmCodeReq, func() any { return new(proto.GmCodeReq) })
	c.regMsg(GmCodeRsp, func() any { return new(proto.GmCodeRsp) })
	c.regMsg(SceneDataNotice, func() any { return new(proto.SceneDataNotice) })
	c.regMsg(NeedLoginNotice, func() any { return new(proto.NeedLoginNotice) })
	c.regMsg(CharacterSkillLevelUpReq, func() any { return new(proto.CharacterSkillLevelUpReq) })
	c.regMsg(CharacterSkillLevelUpRsp, func() any { return new(proto.CharacterSkillLevelUpRsp) })
	c.regMsg(CharacterStarUpReq, func() any { return new(proto.CharacterStarUpReq) })
	c.regMsg(CharacterStarUpRsp, func() any { return new(proto.CharacterStarUpRsp) })
	c.regMsg(CharacterLevelUpReq, func() any { return new(proto.CharacterLevelUpReq) })
	c.regMsg(CharacterLevelUpRsp, func() any { return new(proto.CharacterLevelUpRsp) })
	c.regMsg(CharacterLevelBreakReq, func() any { return new(proto.CharacterLevelBreakReq) })
	c.regMsg(CharacterLevelBreakRsp, func() any { return new(proto.CharacterLevelBreakRsp) })
	c.regMsg(TeamCharExpUpdateNotice, func() any { return new(proto.TeamCharExpUpdateNotice) })
	c.regMsg(PlayerEnergyInfoReq, func() any { return new(proto.PlayerEnergyInfoReq) })
	c.regMsg(PlayerEnergyInfoRsp, func() any { return new(proto.PlayerEnergyInfoRsp) })
	c.regMsg(PlayerEnergyBuyReq, func() any { return new(proto.PlayerEnergyBuyReq) })
	c.regMsg(PlayerEnergyBuyRsp, func() any { return new(proto.PlayerEnergyBuyRsp) })
	c.regMsg(GetMailsReq, func() any { return new(proto.GetMailsReq) })
	c.regMsg(GetMailsRsp, func() any { return new(proto.GetMailsRsp) })
	c.regMsg(OperateMailsReq, func() any { return new(proto.OperateMailsReq) })
	c.regMsg(OperateMailsRsp, func() any { return new(proto.OperateMailsRsp) })
	c.regMsg(ReceiveMailNotice, func() any { return new(proto.ReceiveMailNotice) })
	c.regMsg(ChangeSceneChannelReq, func() any { return new(proto.ChangeSceneChannelReq) })
	c.regMsg(ChangeSceneChannelRsp, func() any { return new(proto.ChangeSceneChannelRsp) })
	c.regMsg(PlayerSceneRecordReq, func() any { return new(proto.PlayerSceneRecordReq) })
	c.regMsg(PlayerSceneRecordRsp, func() any { return new(proto.PlayerSceneRecordRsp) })
	c.regMsg(PlayerSceneSyncDataNotice, func() any { return new(proto.PlayerSceneSyncDataNotice) })
	c.regMsg(ServerSceneSyncDataNotice, func() any { return new(proto.ServerSceneSyncDataNotice) })
	c.regMsg(SetArchiveInfoReq, func() any { return new(proto.SetArchiveInfoReq) })
	c.regMsg(SetArchiveInfoRsp, func() any { return new(proto.SetArchiveInfoRsp) })
	c.regMsg(GetArchiveInfoReq, func() any { return new(proto.GetArchiveInfoReq) })
	c.regMsg(GetArchiveInfoRsp, func() any { return new(proto.GetArchiveInfoRsp) })
	c.regMsg(ChallengeFriendRankReq, func() any { return new(proto.ChallengeFriendRankReq) })
	c.regMsg(ChallengeFriendRankRsp, func() any { return new(proto.ChallengeFriendRankRsp) })
	c.regMsg(ChallengeStateUpdateReq, func() any { return new(proto.ChallengeStateUpdateReq) })
	c.regMsg(ChallengeStateUpdateRsp, func() any { return new(proto.ChallengeStateUpdateRsp) })
	c.regMsg(RiddleStateUpdateReq, func() any { return new(proto.RiddleStateUpdateReq) })
	c.regMsg(RiddleStateUpdateRsp, func() any { return new(proto.RiddleStateUpdateRsp) })
	c.regMsg(FlagBattleStateUpdateReq, func() any { return new(proto.FlagBattleStateUpdateReq) })
	c.regMsg(FlagBattleStateUpdateRsp, func() any { return new(proto.FlagBattleStateUpdateRsp) })
	c.regMsg(BattleEncounterStateUpdateReq, func() any { return new(proto.BattleEncounterStateUpdateReq) })
	c.regMsg(BattleEncounterStateUpdateRsp, func() any { return new(proto.BattleEncounterStateUpdateRsp) })
	c.regMsg(BattleEncounterInfoReq, func() any { return new(proto.BattleEncounterInfoReq) })
	c.regMsg(BattleEncounterInfoRsp, func() any { return new(proto.BattleEncounterInfoRsp) })
	c.regMsg(DungeonRepeatEnterReq, func() any { return new(proto.DungeonRepeatEnterReq) })
	c.regMsg(DungeonRepeatEnterRsp, func() any { return new(proto.DungeonRepeatEnterRsp) })
	c.regMsg(DungeonViewReq, func() any { return new(proto.DungeonViewReq) })
	c.regMsg(DungeonViewRsp, func() any { return new(proto.DungeonViewRsp) })
	c.regMsg(DungeonEnterReq, func() any { return new(proto.DungeonEnterReq) })
	c.regMsg(DungeonEnterRsp, func() any { return new(proto.DungeonEnterRsp) })
	c.regMsg(DungeonExitReq, func() any { return new(proto.DungeonExitReq) })
	c.regMsg(DungeonExitRsp, func() any { return new(proto.DungeonExitRsp) })
	c.regMsg(DungeonFinishReq, func() any { return new(proto.DungeonFinishReq) })
	c.regMsg(DungeonFinishRsp, func() any { return new(proto.DungeonFinishRsp) })
	c.regMsg(CollectCoinReq, func() any { return new(proto.CollectCoinReq) })
	c.regMsg(CollectCoinRsp, func() any { return new(proto.CollectCoinRsp) })
	c.regMsg(DungeonTaskFinishRewardReq, func() any { return new(proto.DungeonTaskFinishRewardReq) })
	c.regMsg(DungeonTaskFinishRewardRsp, func() any { return new(proto.DungeonTaskFinishRewardRsp) })
	c.regMsg(DungeonStarRewardReq, func() any { return new(proto.DungeonStarRewardReq) })
	c.regMsg(DungeonStarRewardRsp, func() any { return new(proto.DungeonStarRewardRsp) })
	c.regMsg(DungeonOperateReq, func() any { return new(proto.DungeonOperateReq) })
	c.regMsg(DungeonOperateRsp, func() any { return new(proto.DungeonOperateRsp) })
	c.regMsg(PlayerTimeOffsetNotice, func() any { return new(proto.PlayerTimeOffsetNotice) })
	c.regMsg(DungeonSweepReq, func() any { return new(proto.DungeonSweepReq) })
	c.regMsg(DungeonSweepRsp, func() any { return new(proto.DungeonSweepRsp) })
	c.regMsg(ItemSweepReq, func() any { return new(proto.ItemSweepReq) })
	c.regMsg(ItemSweepRsp, func() any { return new(proto.ItemSweepRsp) })
	c.regMsg(GetOneTypeLifePictorialBookCountReq, func() any { return new(proto.GetOneTypeLifePictorialBookCountReq) })
	c.regMsg(GetOneTypeLifePictorialBookCountRsp, func() any { return new(proto.GetOneTypeLifePictorialBookCountRsp) })
	c.regMsg(LifeProficiencyNotice, func() any { return new(proto.LifeProficiencyNotice) })
	c.regMsg(GetLifeInfoReq, func() any { return new(proto.GetLifeInfoReq) })
	c.regMsg(GetLifeInfoRsp, func() any { return new(proto.GetLifeInfoRsp) })
	c.regMsg(FishingResultNotice, func() any { return new(proto.FishingResultNotice) })
	c.regMsg(LifeSkillLevelUpNotice, func() any { return new(proto.LifeSkillLevelUpNotice) })
	c.regMsg(LifeAchieveNotice, func() any { return new(proto.LifeAchieveNotice) })
	c.regMsg(CookingFoodReq, func() any { return new(proto.CookingFoodReq) })
	c.regMsg(CookingFoodRsp, func() any { return new(proto.CookingFoodRsp) })
	c.regMsg(MakePropReq, func() any { return new(proto.MakePropReq) })
	c.regMsg(MakePropRsp, func() any { return new(proto.MakePropRsp) })
	c.regMsg(HandicraftReq, func() any { return new(proto.HandicraftReq) })
	c.regMsg(HandicraftRsp, func() any { return new(proto.HandicraftRsp) })
	c.regMsg(SewingReq, func() any { return new(proto.SewingReq) })
	c.regMsg(SewingRsp, func() any { return new(proto.SewingRsp) })
	c.regMsg(GetLifeAchievementRewardReq, func() any { return new(proto.GetLifeAchievementRewardReq) })
	c.regMsg(GetLifeAchievementRewardRsp, func() any { return new(proto.GetLifeAchievementRewardRsp) })
	c.regMsg(LifeLeveUpReq, func() any { return new(proto.LifeLeveUpReq) })
	c.regMsg(LifeLeveUpRsp, func() any { return new(proto.LifeLeveUpRsp) })
	c.regMsg(GetLifeAchieveReq, func() any { return new(proto.GetLifeAchieveReq) })
	c.regMsg(GetLifeAchieveRsp, func() any { return new(proto.GetLifeAchieveRsp) })
	c.regMsg(LifeSkillLevelUpReq, func() any { return new(proto.LifeSkillLevelUpReq) })
	c.regMsg(LifeSkillLevelUpRsp, func() any { return new(proto.LifeSkillLevelUpRsp) })
	c.regMsg(GetLifeSkillReq, func() any { return new(proto.GetLifeSkillReq) })
	c.regMsg(GetLifeSkillRsp, func() any { return new(proto.GetLifeSkillRsp) })
	c.regMsg(FishingReq, func() any { return new(proto.FishingReq) })
	c.regMsg(FishingRsp, func() any { return new(proto.FishingRsp) })
	c.regMsg(PackNotice, func() any { return new(proto.PackNotice) })
	c.regMsg(GetWeaponReq, func() any { return new(proto.GetWeaponReq) })
	c.regMsg(GetWeaponRsp, func() any { return new(proto.GetWeaponRsp) })
	c.regMsg(GetArmorReq, func() any { return new(proto.GetArmorReq) })
	c.regMsg(GetArmorRsp, func() any { return new(proto.GetArmorRsp) })
	c.regMsg(GetPosterReq, func() any { return new(proto.GetPosterReq) })
	c.regMsg(GetPosterRsp, func() any { return new(proto.GetPosterRsp) })
	c.regMsg(TempPackItemDropReq, func() any { return new(proto.TempPackItemDropReq) })
	c.regMsg(TempPackItemDropRsp, func() any { return new(proto.TempPackItemDropRsp) })
	c.regMsg(TempPackItemStoreReq, func() any { return new(proto.TempPackItemStoreReq) })
	c.regMsg(TempPackItemStoreRsp, func() any { return new(proto.TempPackItemStoreRsp) })
	c.regMsg(TempPackSortReq, func() any { return new(proto.TempPackSortReq) })
	c.regMsg(TempPackSortRsp, func() any { return new(proto.TempPackSortRsp) })
	c.regMsg(TempPackWearEquipReq, func() any { return new(proto.TempPackWearEquipReq) })
	c.regMsg(TempPackWearEquipRsp, func() any { return new(proto.TempPackWearEquipRsp) })
	c.regMsg(GetAllCharacterEquipReq, func() any { return new(proto.GetAllCharacterEquipReq) })
	c.regMsg(GetAllCharacterEquipRsp, func() any { return new(proto.GetAllCharacterEquipRsp) })
	c.regMsg(CharacterEquipUpdateReq, func() any { return new(proto.CharacterEquipUpdateReq) })
	c.regMsg(CharacterEquipUpdateRsp, func() any { return new(proto.CharacterEquipUpdateRsp) })
	c.regMsg(CharacterEquipPresetSwitchReq, func() any { return new(proto.CharacterEquipPresetSwitchReq) })
	c.regMsg(CharacterEquipPresetSwitchRsp, func() any { return new(proto.CharacterEquipPresetSwitchRsp) })
	c.regMsg(PosterStarUpReq, func() any { return new(proto.PosterStarUpReq) })
	c.regMsg(PosterStarUpRsp, func() any { return new(proto.PosterStarUpRsp) })
	c.regMsg(PosterIllustrationListReq, func() any { return new(proto.PosterIllustrationListReq) })
	c.regMsg(PosterIllustrationListRsp, func() any { return new(proto.PosterIllustrationListRsp) })
	c.regMsg(PosterIllustrationRewardReq, func() any { return new(proto.PosterIllustrationRewardReq) })
	c.regMsg(PosterIllustrationRewardRsp, func() any { return new(proto.PosterIllustrationRewardRsp) })
	c.regMsg(LockEquipReq, func() any { return new(proto.LockEquipReq) })
	c.regMsg(LockEquipRsp, func() any { return new(proto.LockEquipRsp) })
	c.regMsg(PlayerTagUpdateNotice, func() any { return new(proto.PlayerTagUpdateNotice) })
	c.regMsg(GachaListReq, func() any { return new(proto.GachaListReq) })
	c.regMsg(GachaListRsp, func() any { return new(proto.GachaListRsp) })
	c.regMsg(GachaReq, func() any { return new(proto.GachaReq) })
	c.regMsg(GachaRsp, func() any { return new(proto.GachaRsp) })
	c.regMsg(GachaFullPickReq, func() any { return new(proto.GachaFullPickReq) })
	c.regMsg(GachaFullPickRsp, func() any { return new(proto.GachaFullPickRsp) })
	c.regMsg(GachaRecordReq, func() any { return new(proto.GachaRecordReq) })
	c.regMsg(GachaRecordRsp, func() any { return new(proto.GachaRecordRsp) })
	c.regMsg(GamePlayRewardReq, func() any { return new(proto.GamePlayRewardReq) })
	c.regMsg(GamePlayRewardRsp, func() any { return new(proto.GamePlayRewardRsp) })
	c.regMsg(ModuleCloseNotice, func() any { return new(proto.ModuleCloseNotice) })
	c.regMsg(GetCharacterAchievementListReq, func() any { return new(proto.GetCharacterAchievementListReq) })
	c.regMsg(GetCharacterAchievementListRsp, func() any { return new(proto.GetCharacterAchievementListRsp) })
	c.regMsg(GetCharacterAchievementAwardReq, func() any { return new(proto.GetCharacterAchievementAwardReq) })
	c.regMsg(GetCharacterAchievementAwardRsp, func() any { return new(proto.GetCharacterAchievementAwardRsp) })
	c.regMsg(GetCharacterAchievementUnlockPaymentReq, func() any { return new(proto.GetCharacterAchievementUnlockPaymentReq) })
	c.regMsg(GetCharacterAchievementUnlockPaymentRsp, func() any { return new(proto.GetCharacterAchievementUnlockPaymentRsp) })
	c.regMsg(GetCharacterAchievementBadgeAwardReq, func() any { return new(proto.GetCharacterAchievementBadgeAwardReq) })
	c.regMsg(GetCharacterAchievementBadgeAwardRsp, func() any { return new(proto.GetCharacterAchievementBadgeAwardRsp) })
	c.regMsg(ChangeNickNameReq, func() any { return new(proto.ChangeNickNameReq) })
	c.regMsg(ChangeNickNameRsp, func() any { return new(proto.ChangeNickNameRsp) })
	c.regMsg(ChangeSignReq, func() any { return new(proto.ChangeSignReq) })
	c.regMsg(ChangeSignRsp, func() any { return new(proto.ChangeSignRsp) })
	c.regMsg(ChangeHeadReq, func() any { return new(proto.ChangeHeadReq) })
	c.regMsg(ChangeHeadRsp, func() any { return new(proto.ChangeHeadRsp) })
	c.regMsg(ChangePhoneBackgroundReq, func() any { return new(proto.ChangePhoneBackgroundReq) })
	c.regMsg(ChangePhoneBackgroundRsp, func() any { return new(proto.ChangePhoneBackgroundRsp) })
	c.regMsg(PlayerLevelExpNotice, func() any { return new(proto.PlayerLevelExpNotice) })
	c.regMsg(ChangePlayerSexReq, func() any { return new(proto.ChangePlayerSexReq) })
	c.regMsg(ChangePlayerSexRsp, func() any { return new(proto.ChangePlayerSexRsp) })
	c.regMsg(WorldLevelAchieveListReq, func() any { return new(proto.WorldLevelAchieveListReq) })
	c.regMsg(WorldLevelAchieveListRsp, func() any { return new(proto.WorldLevelAchieveListRsp) })
	c.regMsg(UnlockWorldLevelReq, func() any { return new(proto.UnlockWorldLevelReq) })
	c.regMsg(UnlockWorldLevelRsp, func() any { return new(proto.UnlockWorldLevelRsp) })
	c.regMsg(ChangeWorldLevelReq, func() any { return new(proto.ChangeWorldLevelReq) })
	c.regMsg(ChangeWorldLevelRsp, func() any { return new(proto.ChangeWorldLevelRsp) })
	c.regMsg(UnlockHeadListReq, func() any { return new(proto.UnlockHeadListReq) })
	c.regMsg(UnlockHeadListRsp, func() any { return new(proto.UnlockHeadListRsp) })
	c.regMsg(PlayerLevelRewardReq, func() any { return new(proto.PlayerLevelRewardReq) })
	c.regMsg(PlayerLevelRewardRsp, func() any { return new(proto.PlayerLevelRewardRsp) })
	c.regMsg(ForbiddenInfoNotice, func() any { return new(proto.ForbiddenInfoNotice) })
	c.regMsg(ClientIFIxGMNotice, func() any { return new(proto.ClientIFIxGMNotice) })
	c.regMsg(SpeechRecyclingNotice, func() any { return new(proto.SpeechRecyclingNotice) })
	c.regMsg(FlagUnlockAddNotice, func() any { return new(proto.FlagUnlockAddNotice) })
	c.regMsg(FlagUnLockRemoveNotice, func() any { return new(proto.FlagUnLockRemoveNotice) })
	c.regMsg(TutorialReq, func() any { return new(proto.TutorialReq) })
	c.regMsg(TutorialRsp, func() any { return new(proto.TutorialRsp) })
	c.regMsg(PlayerUnlockFunctionNotice, func() any { return new(proto.PlayerUnlockFunctionNotice) })
	c.regMsg(PlayerAbilityListReq, func() any { return new(proto.PlayerAbilityListReq) })
	c.regMsg(PlayerAbilityListRsp, func() any { return new(proto.PlayerAbilityListRsp) })
	c.regMsg(PlayerAbilityLevelUpReq, func() any { return new(proto.PlayerAbilityLevelUpReq) })
	c.regMsg(PlayerAbilityLevelUpRsp, func() any { return new(proto.PlayerAbilityLevelUpRsp) })
	c.regMsg(PlayerAbilityUnlockNotice, func() any { return new(proto.PlayerAbilityUnlockNotice) })
	c.regMsg(PlayerVitalityReq, func() any { return new(proto.PlayerVitalityReq) })
	c.regMsg(PlayerVitalityRsp, func() any { return new(proto.PlayerVitalityRsp) })
	c.regMsg(PlayerVitalityBuyReq, func() any { return new(proto.PlayerVitalityBuyReq) })
	c.regMsg(PlayerVitalityBuyRsp, func() any { return new(proto.PlayerVitalityBuyRsp) })
	c.regMsg(AbilityBadgeListReq, func() any { return new(proto.AbilityBadgeListReq) })
	c.regMsg(AbilityBadgeListRsp, func() any { return new(proto.AbilityBadgeListRsp) })
	c.regMsg(AbilityBadgePageBoxActiveReq, func() any { return new(proto.AbilityBadgePageBoxActiveReq) })
	c.regMsg(AbilityBadgePageBoxActiveRsp, func() any { return new(proto.AbilityBadgePageBoxActiveRsp) })
	c.regMsg(AbilityBadgePageRewardReq, func() any { return new(proto.AbilityBadgePageRewardReq) })
	c.regMsg(AbilityBadgePageRewardRsp, func() any { return new(proto.AbilityBadgePageRewardRsp) })
	c.regMsg(AbilityBadgeAchieveRewardReq, func() any { return new(proto.AbilityBadgeAchieveRewardReq) })
	c.regMsg(AbilityBadgeAchieveRewardRsp, func() any { return new(proto.AbilityBadgeAchieveRewardRsp) })
	c.regMsg(UpdateTeamReq, func() any { return new(proto.UpdateTeamReq) })
	c.regMsg(UpdateTeamRsp, func() any { return new(proto.UpdateTeamRsp) })
	c.regMsg(OutfitDyeUnlockIndexReq, func() any { return new(proto.OutfitDyeUnlockIndexReq) })
	c.regMsg(OutfitDyeUnlockIndexRsp, func() any { return new(proto.OutfitDyeUnlockIndexRsp) })
	c.regMsg(OutfitDyeReq, func() any { return new(proto.OutfitDyeReq) })
	c.regMsg(OutfitDyeRsp, func() any { return new(proto.OutfitDyeRsp) })
	c.regMsg(OutfitPresetUpdateNotice, func() any { return new(proto.OutfitPresetUpdateNotice) })
	c.regMsg(OutfitColorantSelectReq, func() any { return new(proto.OutfitColorantSelectReq) })
	c.regMsg(OutfitColorantSelectRsp, func() any { return new(proto.OutfitColorantSelectRsp) })
	c.regMsg(OutfitRandomColorReq, func() any { return new(proto.OutfitRandomColorReq) })
	c.regMsg(OutfitRandomColorRsp, func() any { return new(proto.OutfitRandomColorRsp) })
	c.regMsg(OutFitDyeSaveReq, func() any { return new(proto.OutFitDyeSaveReq) })
	c.regMsg(OutFitDyeSaveRsp, func() any { return new(proto.OutFitDyeSaveRsp) })
	c.regMsg(OutfitPresetUpdateReq, func() any { return new(proto.OutfitPresetUpdateReq) })
	c.regMsg(OutfitPresetUpdateRsp, func() any { return new(proto.OutfitPresetUpdateRsp) })
	c.regMsg(OutfitPresetSwitchReq, func() any { return new(proto.OutfitPresetSwitchReq) })
	c.regMsg(OutfitPresetSwitchRsp, func() any { return new(proto.OutfitPresetSwitchRsp) })
	c.regMsg(OutfitHideSwitchReq, func() any { return new(proto.OutfitHideSwitchReq) })
	c.regMsg(OutfitHideSwitchRsp, func() any { return new(proto.OutfitHideSwitchRsp) })
	c.regMsg(UpdateCharacterAppearanceReq, func() any { return new(proto.UpdateCharacterAppearanceReq) })
	c.regMsg(UpdateCharacterAppearanceRsp, func() any { return new(proto.UpdateCharacterAppearanceRsp) })
	c.regMsg(CharacterAppearanceUpdateNotice, func() any { return new(proto.CharacterAppearanceUpdateNotice) })
	c.regMsg(ShopInfoReq, func() any { return new(proto.ShopInfoReq) })
	c.regMsg(ShopInfoRsp, func() any { return new(proto.ShopInfoRsp) })
	c.regMsg(ShopBuyReq, func() any { return new(proto.ShopBuyReq) })
	c.regMsg(ShopBuyRsp, func() any { return new(proto.ShopBuyRsp) })
	c.regMsg(ShopRefreshNotice, func() any { return new(proto.ShopRefreshNotice) })
	c.regMsg(PlaceFurnitureReq, func() any { return new(proto.PlaceFurnitureReq) })
	c.regMsg(PlaceFurnitureRsp, func() any { return new(proto.PlaceFurnitureRsp) })
	c.regMsg(TakeOutFurnitureReq, func() any { return new(proto.TakeOutFurnitureReq) })
	c.regMsg(TakeOutFurnitureRsp, func() any { return new(proto.TakeOutFurnitureRsp) })
	c.regMsg(GetGardenInfoReq, func() any { return new(proto.GetGardenInfoReq) })
	c.regMsg(GetGardenInfoRsp, func() any { return new(proto.GetGardenInfoRsp) })
	c.regMsg(SwitchGardenStatusReq, func() any { return new(proto.SwitchGardenStatusReq) })
	c.regMsg(SwitchGardenStatusRsp, func() any { return new(proto.SwitchGardenStatusRsp) })
	c.regMsg(DoLikesReq, func() any { return new(proto.DoLikesReq) })
	c.regMsg(DoLikesRsp, func() any { return new(proto.DoLikesRsp) })
	c.regMsg(SceneGardenFurnitureUpdateNotice, func() any { return new(proto.SceneGardenFurnitureUpdateNotice) })
	c.regMsg(SceneGardenFurnitureRemoveNotice, func() any { return new(proto.SceneGardenFurnitureRemoveNotice) })
	c.regMsg(WeaponInscriptionInlaidReq, func() any { return new(proto.WeaponInscriptionInlaidReq) })
	c.regMsg(WeaponInscriptionInlaidRsp, func() any { return new(proto.WeaponInscriptionInlaidRsp) })
	c.regMsg(WeaponInscriptionRemoveReq, func() any { return new(proto.WeaponInscriptionRemoveReq) })
	c.regMsg(WeaponInscriptionRemoveRsp, func() any { return new(proto.WeaponInscriptionRemoveRsp) })
	c.regMsg(ShopInitNotice, func() any { return new(proto.ShopInitNotice) })
	c.regMsg(QuestNotice, func() any { return new(proto.QuestNotice) })
	c.regMsg(AchieveNotice, func() any { return new(proto.AchieveNotice) })
	c.regMsg(GmNotice, func() any { return new(proto.GmNotice) })
	c.regMsg(AcceptQuestReq, func() any { return new(proto.AcceptQuestReq) })
	c.regMsg(AcceptQuestRsp, func() any { return new(proto.AcceptQuestRsp) })
	c.regMsg(ClaimQuestRewardReq, func() any { return new(proto.ClaimQuestRewardReq) })
	c.regMsg(ClaimQuestRewardRsp, func() any { return new(proto.ClaimQuestRewardRsp) })
	c.regMsg(FriendSearchReq, func() any { return new(proto.FriendSearchReq) })
	c.regMsg(FriendSearchRsp, func() any { return new(proto.FriendSearchRsp) })
	c.regMsg(FriendAddReq, func() any { return new(proto.FriendAddReq) })
	c.regMsg(FriendAddRsp, func() any { return new(proto.FriendAddRsp) })
	c.regMsg(FriendDelReq, func() any { return new(proto.FriendDelReq) })
	c.regMsg(FriendDelRsp, func() any { return new(proto.FriendDelRsp) })
	c.regMsg(FriendBlackReq, func() any { return new(proto.FriendBlackReq) })
	c.regMsg(FriendBlackRsp, func() any { return new(proto.FriendBlackRsp) })
	c.regMsg(FriendHandleReq, func() any { return new(proto.FriendHandleReq) })
	c.regMsg(FriendHandleRsp, func() any { return new(proto.FriendHandleRsp) })
	c.regMsg(FriendHandleNotice, func() any { return new(proto.FriendHandleNotice) })
	c.regMsg(FriendReq, func() any { return new(proto.FriendReq) })
	c.regMsg(FriendRsp, func() any { return new(proto.FriendRsp) })
	c.regMsg(CollectingReq, func() any { return new(proto.CollectingReq) })
	c.regMsg(CollectingRsp, func() any { return new(proto.CollectingRsp) })
	c.regMsg(CollectionRewardReq, func() any { return new(proto.CollectionRewardReq) })
	c.regMsg(CollectionRewardRsp, func() any { return new(proto.CollectionRewardRsp) })
	c.regMsg(TreasureBoxOpenReq, func() any { return new(proto.TreasureBoxOpenReq) })
	c.regMsg(TreasureBoxOpenRsp, func() any { return new(proto.TreasureBoxOpenRsp) })
	c.regMsg(TreasureBoxPickupReq, func() any { return new(proto.TreasureBoxPickupReq) })
	c.regMsg(TreasureBoxPickupRsp, func() any { return new(proto.TreasureBoxPickupRsp) })
	c.regMsg(PickupReq, func() any { return new(proto.PickupReq) })
	c.regMsg(PickupRsp, func() any { return new(proto.PickupRsp) })
	c.regMsg(GatherReq, func() any { return new(proto.GatherReq) })
	c.regMsg(GatherRsp, func() any { return new(proto.GatherRsp) })
	c.regMsg(GatherSceneLimitRecoveryNotice, func() any { return new(proto.GatherSceneLimitRecoveryNotice) })
	c.regMsg(GetAchieveGroupAwardReq, func() any { return new(proto.GetAchieveGroupAwardReq) })
	c.regMsg(GetAchieveGroupAwardRsp, func() any { return new(proto.GetAchieveGroupAwardRsp) })
	c.regMsg(GetAchieveGroupListReq, func() any { return new(proto.GetAchieveGroupListReq) })
	c.regMsg(GetAchieveGroupListRsp, func() any { return new(proto.GetAchieveGroupListRsp) })
	c.regMsg(GetOneAchieveAwardReq, func() any { return new(proto.GetOneAchieveAwardReq) })
	c.regMsg(GetOneAchieveAwardRsp, func() any { return new(proto.GetOneAchieveAwardRsp) })
	c.regMsg(GetAchieveOneGroupReq, func() any { return new(proto.GetAchieveOneGroupReq) })
	c.regMsg(GetAchieveOneGroupRsp, func() any { return new(proto.GetAchieveOneGroupRsp) })
	c.regMsg(AchieveFinishNotice, func() any { return new(proto.AchieveFinishNotice) })
	c.regMsg(SetFriendExtInfoReq, func() any { return new(proto.SetFriendExtInfoReq) })
	c.regMsg(SetFriendExtInfoRsp, func() any { return new(proto.SetFriendExtInfoRsp) })
	c.regMsg(SceneSitChairReq, func() any { return new(proto.SceneSitChairReq) })
	c.regMsg(SceneSitChairRsp, func() any { return new(proto.SceneSitChairRsp) })
	c.regMsg(NpcTalkReq, func() any { return new(proto.NpcTalkReq) })
	c.regMsg(NpcTalkRsp, func() any { return new(proto.NpcTalkRsp) })
	c.regMsg(ExploreInitReq, func() any { return new(proto.ExploreInitReq) })
	c.regMsg(ExploreInitRsp, func() any { return new(proto.ExploreInitRsp) })
	c.regMsg(ExploreReq, func() any { return new(proto.ExploreReq) })
	c.regMsg(ExploreRsp, func() any { return new(proto.ExploreRsp) })
	c.regMsg(ExploreQuickFinishReq, func() any { return new(proto.ExploreQuickFinishReq) })
	c.regMsg(ExploreQuickFinishRsp, func() any { return new(proto.ExploreQuickFinishRsp) })
	c.regMsg(ExploreRewardReq, func() any { return new(proto.ExploreRewardReq) })
	c.regMsg(ExploreRewardRsp, func() any { return new(proto.ExploreRewardRsp) })
	c.regMsg(ExploreCancelReq, func() any { return new(proto.ExploreCancelReq) })
	c.regMsg(ExploreCancelRsp, func() any { return new(proto.ExploreCancelRsp) })
	c.regMsg(ExploreCollectRewardReq, func() any { return new(proto.ExploreCollectRewardReq) })
	c.regMsg(ExploreCollectRewardRsp, func() any { return new(proto.ExploreCollectRewardRsp) })
	c.regMsg(IntervalChangeNotice, func() any { return new(proto.IntervalChangeNotice) })
	c.regMsg(IntervalStartReq, func() any { return new(proto.IntervalStartReq) })
	c.regMsg(IntervalStartRsp, func() any { return new(proto.IntervalStartRsp) })
	c.regMsg(IntervalJoinReq, func() any { return new(proto.IntervalJoinReq) })
	c.regMsg(IntervalJoinRsp, func() any { return new(proto.IntervalJoinRsp) })
	c.regMsg(IntervalQuickReq, func() any { return new(proto.IntervalQuickReq) })
	c.regMsg(IntervalQuickRsp, func() any { return new(proto.IntervalQuickRsp) })
	c.regMsg(IntervalRewardReq, func() any { return new(proto.IntervalRewardReq) })
	c.regMsg(IntervalRewardRsp, func() any { return new(proto.IntervalRewardRsp) })
	c.regMsg(FriendIntervalInitReq, func() any { return new(proto.FriendIntervalInitReq) })
	c.regMsg(FriendIntervalInitRsp, func() any { return new(proto.FriendIntervalInitRsp) })
	c.regMsg(SelfIntervalInitReq, func() any { return new(proto.SelfIntervalInitReq) })
	c.regMsg(SelfIntervalInitRsp, func() any { return new(proto.SelfIntervalInitRsp) })
	c.regMsg(MonsterDeadReq, func() any { return new(proto.MonsterDeadReq) })
	c.regMsg(MonsterDeadRsp, func() any { return new(proto.MonsterDeadRsp) })
	c.regMsg(ManualListReq, func() any { return new(proto.ManualListReq) })
	c.regMsg(ManualListRsp, func() any { return new(proto.ManualListRsp) })
	c.regMsg(ManualFlagUnlockReq, func() any { return new(proto.ManualFlagUnlockReq) })
	c.regMsg(ManualFlagUnlockRsp, func() any { return new(proto.ManualFlagUnlockRsp) })
	c.regMsg(ManualFlagRewardReq, func() any { return new(proto.ManualFlagRewardReq) })
	c.regMsg(ManualFlagRewardRsp, func() any { return new(proto.ManualFlagRewardRsp) })
	c.regMsg(SceneProcessListReq, func() any { return new(proto.SceneProcessListReq) })
	c.regMsg(SceneProcessListRsp, func() any { return new(proto.SceneProcessListRsp) })
	c.regMsg(PlayerBuffNotice, func() any { return new(proto.PlayerBuffNotice) })
	c.regMsg(SupplyBoxInfoReq, func() any { return new(proto.SupplyBoxInfoReq) })
	c.regMsg(SupplyBoxInfoRsp, func() any { return new(proto.SupplyBoxInfoRsp) })
	c.regMsg(SupplyBoxRewardReq, func() any { return new(proto.SupplyBoxRewardReq) })
	c.regMsg(SupplyBoxRewardRsp, func() any { return new(proto.SupplyBoxRewardRsp) })
	c.regMsg(AreaCloseReq, func() any { return new(proto.AreaCloseReq) })
	c.regMsg(AreaCloseRsp, func() any { return new(proto.AreaCloseRsp) })
	c.regMsg(AreaUnlockReq, func() any { return new(proto.AreaUnlockReq) })
	c.regMsg(AreaUnlockRsp, func() any { return new(proto.AreaUnlockRsp) })
	c.regMsg(AreaLevelUpReq, func() any { return new(proto.AreaLevelUpReq) })
	c.regMsg(AreaLevelUpRsp, func() any { return new(proto.AreaLevelUpRsp) })
	c.regMsg(AreaAchieveListReq, func() any { return new(proto.AreaAchieveListReq) })
	c.regMsg(AreaAchieveListRsp, func() any { return new(proto.AreaAchieveListRsp) })
	c.regMsg(PopEmojiNotice, func() any { return new(proto.PopEmojiNotice) })
	c.regMsg(PopEmojiReq, func() any { return new(proto.PopEmojiReq) })
	c.regMsg(PopEmojiRsp, func() any { return new(proto.PopEmojiRsp) })
	c.regMsg(PlaceCampFireReq, func() any { return new(proto.PlaceCampFireReq) })
	c.regMsg(PlaceCampFireRsp, func() any { return new(proto.PlaceCampFireRsp) })
	c.regMsg(PlaceCampFireNotice, func() any { return new(proto.PlaceCampFireNotice) })
	c.regMsg(SceneWeatherChangeNotice, func() any { return new(proto.SceneWeatherChangeNotice) })
	c.regMsg(SceneInterActionPlayStatusReq, func() any { return new(proto.SceneInterActionPlayStatusReq) })
	c.regMsg(SceneInterActionPlayStatusRsp, func() any { return new(proto.SceneInterActionPlayStatusRsp) })
	c.regMsg(SceneInterActionPlayStatusNotice, func() any { return new(proto.SceneInterActionPlayStatusNotice) })
	c.regMsg(RemoveCampFireReq, func() any { return new(proto.RemoveCampFireReq) })
	c.regMsg(RemoveCampFireRsp, func() any { return new(proto.RemoveCampFireRsp) })
	c.regMsg(RemoveCampFireNotice, func() any { return new(proto.RemoveCampFireNotice) })
	c.regMsg(ChangeChatChannelReq, func() any { return new(proto.ChangeChatChannelReq) })
	c.regMsg(ChangeChatChannelRsp, func() any { return new(proto.ChangeChatChannelRsp) })
	c.regMsg(SendChatMsgReq, func() any { return new(proto.SendChatMsgReq) })
	c.regMsg(SendChatMsgRsp, func() any { return new(proto.SendChatMsgRsp) })
	c.regMsg(ChatMsgNotice, func() any { return new(proto.ChatMsgNotice) })
	c.regMsg(ChatMsgRecordInitNotice, func() any { return new(proto.ChatMsgRecordInitNotice) })
	c.regMsg(ChatUnLockExpressionNotice, func() any { return new(proto.ChatUnLockExpressionNotice) })
	c.regMsg(CollectVoiceRegionReq, func() any { return new(proto.CollectVoiceRegionReq) })
	c.regMsg(CollectVoiceRegionRsp, func() any { return new(proto.CollectVoiceRegionRsp) })
	c.regMsg(CharacterDeadReq, func() any { return new(proto.CharacterDeadReq) })
	c.regMsg(CharacterDeadRsp, func() any { return new(proto.CharacterDeadRsp) })
	c.regMsg(CharacterGatherWeaponUpdateReq, func() any { return new(proto.CharacterGatherWeaponUpdateReq) })
	c.regMsg(CharacterGatherWeaponUpdateRsp, func() any { return new(proto.CharacterGatherWeaponUpdateRsp) })
	c.regMsg(EquipDisassembleReq, func() any { return new(proto.EquipDisassembleReq) })
	c.regMsg(EquipDisassembleRsp, func() any { return new(proto.EquipDisassembleRsp) })
	c.regMsg(ItemUseReq, func() any { return new(proto.ItemUseReq) })
	c.regMsg(ItemUseRsp, func() any { return new(proto.ItemUseRsp) })
	c.regMsg(WeaponStrengthReq, func() any { return new(proto.WeaponStrengthReq) })
	c.regMsg(WeaponStrengthRsp, func() any { return new(proto.WeaponStrengthRsp) })
	c.regMsg(ArmorStrengthReq, func() any { return new(proto.ArmorStrengthReq) })
	c.regMsg(ArmorStrengthRsp, func() any { return new(proto.ArmorStrengthRsp) })
	c.regMsg(OtherPlayerInfoReq, func() any { return new(proto.OtherPlayerInfoReq) })
	c.regMsg(OtherPlayerInfoRsp, func() any { return new(proto.OtherPlayerInfoRsp) })
	c.regMsg(SendActionReq, func() any { return new(proto.SendActionReq) })
	c.regMsg(SendActionRsp, func() any { return new(proto.SendActionRsp) })
	c.regMsg(SendActionNotice, func() any { return new(proto.SendActionNotice) })
	c.regMsg(SendActionStudyNotice, func() any { return new(proto.SendActionStudyNotice) })
	c.regMsg(SendActionAddNotice, func() any { return new(proto.SendActionAddNotice) })
	c.regMsg(SendMultipleActionNotice, func() any { return new(proto.SendMultipleActionNotice) })
	c.regMsg(MultipleActionAcceptReq, func() any { return new(proto.MultipleActionAcceptReq) })
	c.regMsg(MultipleActionAcceptRsp, func() any { return new(proto.MultipleActionAcceptRsp) })
	c.regMsg(SendMultipleActionCompleteNotice, func() any { return new(proto.SendMultipleActionCompleteNotice) })
	c.regMsg(ActivityChangeNotice, func() any { return new(proto.ActivityChangeNotice) })
	c.regMsg(ActivitySignInDataNotice, func() any { return new(proto.ActivitySignInDataNotice) })
	c.regMsg(ActivitySignInReq, func() any { return new(proto.ActivitySignInReq) })
	c.regMsg(ActivitySignInRsp, func() any { return new(proto.ActivitySignInRsp) })
	c.regMsg(PlayerNotInSceneChannelNotice, func() any { return new(proto.PlayerNotInSceneChannelNotice) })
	c.regMsg(ActivityQuestDataNotice, func() any { return new(proto.ActivityQuestDataNotice) })
	c.regMsg(ActivityQuestRewardReq, func() any { return new(proto.ActivityQuestRewardReq) })
	c.regMsg(ActivityQuestRewardRsp, func() any { return new(proto.ActivityQuestRewardRsp) })
	c.regMsg(ActivityRegularDataNotice, func() any { return new(proto.ActivityRegularDataNotice) })
	c.regMsg(ActivityRegularRewardReq, func() any { return new(proto.ActivityRegularRewardReq) })
	c.regMsg(ActivityRegularRewardRsp, func() any { return new(proto.ActivityRegularRewardRsp) })
	c.regMsg(GetCollectItemIdsReq, func() any { return new(proto.GetCollectItemIdsReq) })
	c.regMsg(GetCollectItemIdsRsp, func() any { return new(proto.GetCollectItemIdsRsp) })
	c.regMsg(ClapReq, func() any { return new(proto.ClapReq) })
	c.regMsg(ClapRsp, func() any { return new(proto.ClapRsp) })
	c.regMsg(ClapResultNotice, func() any { return new(proto.ClapResultNotice) })
	c.regMsg(ThrowDiceReq, func() any { return new(proto.ThrowDiceReq) })
	c.regMsg(ThrowDiceRsp, func() any { return new(proto.ThrowDiceRsp) })
	c.regMsg(ThrowDiceNotice, func() any { return new(proto.ThrowDiceNotice) })
	c.regMsg(ThrowDiceResultNotice, func() any { return new(proto.ThrowDiceResultNotice) })
	c.regMsg(AchieveActionUnLockNotice, func() any { return new(proto.AchieveActionUnLockNotice) })
	c.regMsg(PrivateChatOfflineNotice, func() any { return new(proto.PrivateChatOfflineNotice) })
	c.regMsg(PrivateChatMsgRecordReq, func() any { return new(proto.PrivateChatMsgRecordReq) })
	c.regMsg(PrivateChatMsgRecordRsp, func() any { return new(proto.PrivateChatMsgRecordRsp) })
	c.regMsg(SystemNotice, func() any { return new(proto.SystemNotice) })
	c.regMsg(PlayerTiredDropNotice, func() any { return new(proto.PlayerTiredDropNotice) })
	c.regMsg(NewQuestionnaireNotice, func() any { return new(proto.NewQuestionnaireNotice) })
	c.regMsg(CreatePayOrderReq, func() any { return new(proto.CreatePayOrderReq) })
	c.regMsg(CreatePayOrderRsp, func() any { return new(proto.CreatePayOrderRsp) })
	c.regMsg(PaySendGoodsNotice, func() any { return new(proto.PaySendGoodsNotice) })
	c.regMsg(PackNotShowNotice, func() any { return new(proto.PackNotShowNotice) })
	c.regMsg(AbyssSeasonNotice, func() any { return new(proto.AbyssSeasonNotice) })
	c.regMsg(AbyssInfoReq, func() any { return new(proto.AbyssInfoReq) })
	c.regMsg(AbyssInfoRsp, func() any { return new(proto.AbyssInfoRsp) })
	c.regMsg(AbyssQuestRewardReq, func() any { return new(proto.AbyssQuestRewardReq) })
	c.regMsg(AbyssQuestRewardRsp, func() any { return new(proto.AbyssQuestRewardRsp) })
	c.regMsg(AbyssTeamUpdateReq, func() any { return new(proto.AbyssTeamUpdateReq) })
	c.regMsg(AbyssTeamUpdateRsp, func() any { return new(proto.AbyssTeamUpdateRsp) })
	c.regMsg(AbyssTeamSwitchReq, func() any { return new(proto.AbyssTeamSwitchReq) })
	c.regMsg(AbyssTeamSwitchRsp, func() any { return new(proto.AbyssTeamSwitchRsp) })
	c.regMsg(AbyssFriendRankReq, func() any { return new(proto.AbyssFriendRankReq) })
	c.regMsg(AbyssFriendRankRsp, func() any { return new(proto.AbyssFriendRankRsp) })
	c.regMsg(ActivityGiftRewardReq, func() any { return new(proto.ActivityGiftRewardReq) })
	c.regMsg(ActivityGiftRewardRsp, func() any { return new(proto.ActivityGiftRewardRsp) })
	c.regMsg(ActivityGiftDataNotice, func() any { return new(proto.ActivityGiftDataNotice) })
	c.regMsg(MonthCardNotice, func() any { return new(proto.MonthCardNotice) })
	c.regMsg(CharacterBpBuyNotice, func() any { return new(proto.CharacterBpBuyNotice) })
	c.regMsg(ActivityBattlePassBuyNotice, func() any { return new(proto.ActivityBattlePassBuyNotice) })
	c.regMsg(ActivityBattlePassBuyExpReq, func() any { return new(proto.ActivityBattlePassBuyExpReq) })
	c.regMsg(ActivityBattlePassBuyExpRsp, func() any { return new(proto.ActivityBattlePassBuyExpRsp) })
	c.regMsg(GetActivityBattlePassLevelRewardReq, func() any { return new(proto.GetActivityBattlePassLevelRewardReq) })
	c.regMsg(GetActivityBattlePassLevelRewardRsp, func() any { return new(proto.GetActivityBattlePassLevelRewardRsp) })
	c.regMsg(GetActivityBattlePassQuestRewardReq, func() any { return new(proto.GetActivityBattlePassQuestRewardReq) })
	c.regMsg(GetActivityBattlePassQuestRewardRsp, func() any { return new(proto.GetActivityBattlePassQuestRewardRsp) })
	c.regMsg(ActivityBattlePassInfoNotice, func() any { return new(proto.ActivityBattlePassInfoNotice) })
	c.regMsg(ActivityGiftBuyNotice, func() any { return new(proto.ActivityGiftBuyNotice) })
	c.regMsg(UseRedemptionCodeReq, func() any { return new(proto.UseRedemptionCodeReq) })
	c.regMsg(UseRedemptionCodeRsp, func() any { return new(proto.UseRedemptionCodeRsp) })
	c.regMsg(FireworksStartNotice, func() any { return new(proto.FireworksStartNotice) })
	c.regMsg(ReportReq, func() any { return new(proto.ReportReq) })
	c.regMsg(ReportRsp, func() any { return new(proto.ReportRsp) })
	c.regMsg(ClientLogAuthReq, func() any { return new(proto.ClientLogAuthReq) })
	c.regMsg(ClientLogAuthRsp, func() any { return new(proto.ClientLogAuthRsp) })
	c.regMsg(ClientLogMessageReq, func() any { return new(proto.ClientLogMessageReq) })
	c.regMsg(ClientLogMessageRsp, func() any { return new(proto.ClientLogMessageRsp) })
	c.regMsg(GardenLikeRecordReq, func() any { return new(proto.GardenLikeRecordReq) })
	c.regMsg(GardenLikeRecordRsp, func() any { return new(proto.GardenLikeRecordRsp) })
	c.regMsg(OptionalUpPoolItemReq, func() any { return new(proto.OptionalUpPoolItemReq) })
	c.regMsg(OptionalUpPoolItemRsp, func() any { return new(proto.OptionalUpPoolItemRsp) })
	c.regMsg(ActionStudyReq, func() any { return new(proto.ActionStudyReq) })
	c.regMsg(ActionStudyRsp, func() any { return new(proto.ActionStudyRsp) })
	c.regMsg(ActionStudyNotice, func() any { return new(proto.ActionStudyNotice) })
	c.regMsg(CharacterReviveReq, func() any { return new(proto.CharacterReviveReq) })
	c.regMsg(CharacterReviveRsp, func() any { return new(proto.CharacterReviveRsp) })
	c.regMsg(QuestionnaireRemoveNotice, func() any { return new(proto.QuestionnaireRemoveNotice) })
	c.regMsg(ChangeChatChannelNotice, func() any { return new(proto.ChangeChatChannelNotice) })
	c.regMsg(ActivityInviteNotice, func() any { return new(proto.ActivityInviteNotice) })
	c.regMsg(ActivityInviteRewardClaimReq, func() any { return new(proto.ActivityInviteRewardClaimReq) })
	c.regMsg(ActivityInviteRewardClaimRsp, func() any { return new(proto.ActivityInviteRewardClaimRsp) })
	c.regMsg(InviteCodeUseReq, func() any { return new(proto.InviteCodeUseReq) })
	c.regMsg(InviteCodeUseRsp, func() any { return new(proto.InviteCodeUseRsp) })
	c.regMsg(ActivityInviteCountUpdateNotice, func() any { return new(proto.ActivityInviteCountUpdateNotice) })
	c.regMsg(AccountNameBindingCancelReq, func() any { return new(proto.AccountNameBindingCancelReq) })
	c.regMsg(AccountNameBindingCancelRsp, func() any { return new(proto.AccountNameBindingCancelRsp) })
	c.regMsg(GenericGameAReq, func() any { return new(proto.GenericGameAReq) })
	c.regMsg(GenericGameARsp, func() any { return new(proto.GenericGameARsp) })
	c.regMsg(GenericGameBReq, func() any { return new(proto.GenericGameBReq) })
	c.regMsg(GenericGameBRsp, func() any { return new(proto.GenericGameBRsp) })
	c.regMsg(GenericSceneAReq, func() any { return new(proto.GenericSceneAReq) })
	c.regMsg(GenericSceneARsp, func() any { return new(proto.GenericSceneARsp) })
	c.regMsg(GenericSceneBReq, func() any { return new(proto.GenericSceneBReq) })
	c.regMsg(GenericSceneBRsp, func() any { return new(proto.GenericSceneBRsp) })
	c.regMsg(MonthCardRewardReq, func() any { return new(proto.MonthCardRewardReq) })
	c.regMsg(MonthCardRewardRsp, func() any { return new(proto.MonthCardRewardRsp) })
	c.regMsg(FurnitureItemChangeNotice, func() any { return new(proto.FurnitureItemChangeNotice) })
	c.regMsg(ChangeIsHideBirthdayReq, func() any { return new(proto.ChangeIsHideBirthdayReq) })
	c.regMsg(ChangeIsHideBirthdayRsp, func() any { return new(proto.ChangeIsHideBirthdayRsp) })
	c.regMsg(TransmitSceneReq, func() any { return new(proto.TransmitSceneReq) })
	c.regMsg(TransmitSceneRsp, func() any { return new(proto.TransmitSceneRsp) })
	c.regMsg(ChangeHideTypeReq, func() any { return new(proto.ChangeHideTypeReq) })
	c.regMsg(ChangeHideTypeRsp, func() any { return new(proto.ChangeHideTypeRsp) })
	c.regMsg(GameRefreshNotice, func() any { return new(proto.GameRefreshNotice) })
	c.regMsg(MPTeamCreateReq, func() any { return new(proto.MPTeamCreateReq) })
	c.regMsg(MPTeamCreateRsp, func() any { return new(proto.MPTeamCreateRsp) })
	c.regMsg(MPSwapCharacterReq, func() any { return new(proto.MPSwapCharacterReq) })
	c.regMsg(MPSwapCharacterRsp, func() any { return new(proto.MPSwapCharacterRsp) })
	c.regMsg(MPBeaconActionReq, func() any { return new(proto.MPBeaconActionReq) })
	c.regMsg(MPBeaconActionRsp, func() any { return new(proto.MPBeaconActionRsp) })
	c.regMsg(MPTeamBeaconNotice, func() any { return new(proto.MPTeamBeaconNotice) })
	c.regMsg(MPTeamGameModeChangeReq, func() any { return new(proto.MPTeamGameModeChangeReq) })
	c.regMsg(MPTeamGameModeChangeRsp, func() any { return new(proto.MPTeamGameModeChangeRsp) })
	c.regMsg(MPTeamGameModeChangeNotice, func() any { return new(proto.MPTeamGameModeChangeNotice) })
	c.regMsg(MPTeamInfoReq, func() any { return new(proto.MPTeamInfoReq) })
	c.regMsg(MPTeamInfoRsp, func() any { return new(proto.MPTeamInfoRsp) })
	c.regMsg(SceneMPBeaconNotice, func() any { return new(proto.SceneMPBeaconNotice) })
	c.regMsg(MPBeaconCanStartNotice, func() any { return new(proto.MPBeaconCanStartNotice) })
	c.regMsg(MPGameStartReq, func() any { return new(proto.MPGameStartReq) })
	c.regMsg(MPGameStartRsp, func() any { return new(proto.MPGameStartRsp) })
	c.regMsg(MPTeamRenameReq, func() any { return new(proto.MPTeamRenameReq) })
	c.regMsg(MPTeamRenameRsp, func() any { return new(proto.MPTeamRenameRsp) })
	c.regMsg(MPTeamInviteReq, func() any { return new(proto.MPTeamInviteReq) })
	c.regMsg(MPTeamInviteRsp, func() any { return new(proto.MPTeamInviteRsp) })
	c.regMsg(MPTeamJoinReq, func() any { return new(proto.MPTeamJoinReq) })
	c.regMsg(MPTeamJoinRsp, func() any { return new(proto.MPTeamJoinRsp) })
	c.regMsg(MPTeamQuitReq, func() any { return new(proto.MPTeamQuitReq) })
	c.regMsg(MPTeamQuitRsp, func() any { return new(proto.MPTeamQuitRsp) })
	c.regMsg(MPTeamKickMemberReq, func() any { return new(proto.MPTeamKickMemberReq) })
	c.regMsg(MPTeamKickMemberRsp, func() any { return new(proto.MPTeamKickMemberRsp) })
	c.regMsg(MPTeamDismissReq, func() any { return new(proto.MPTeamDismissReq) })
	c.regMsg(MPTeamDismissRsp, func() any { return new(proto.MPTeamDismissRsp) })
	c.regMsg(MPTeamDismissNotice, func() any { return new(proto.MPTeamDismissNotice) })
	c.regMsg(MPTeamInviteNotice, func() any { return new(proto.MPTeamInviteNotice) })
	c.regMsg(MPTeamPlayerKickedNotice, func() any { return new(proto.MPTeamPlayerKickedNotice) })
	c.regMsg(PlayerNotInMPTeamNotice, func() any { return new(proto.PlayerNotInMPTeamNotice) })
	c.regMsg(MPTeamMemberNotice, func() any { return new(proto.MPTeamMemberNotice) })
	c.regMsg(MPTeamRenameNotice, func() any { return new(proto.MPTeamRenameNotice) })
	c.regMsg(PlayerMPGameNotice, func() any { return new(proto.PlayerMPGameNotice) })
	c.regMsg(MPRoomAssignHostNotice, func() any { return new(proto.MPRoomAssignHostNotice) })
	c.regMsg(MPRoomDismissNotice, func() any { return new(proto.MPRoomDismissNotice) })
	c.regMsg(MPRoomEnterNotice, func() any { return new(proto.MPRoomEnterNotice) })
	c.regMsg(MPBeaconPlayerReadyReq, func() any { return new(proto.MPBeaconPlayerReadyReq) })
	c.regMsg(MPBeaconPlayerReadyRsp, func() any { return new(proto.MPBeaconPlayerReadyRsp) })
	c.regMsg(MPPlayRoomEventReq, func() any { return new(proto.MPPlayRoomEventReq) })
	c.regMsg(MPPlayRoomEventRsp, func() any { return new(proto.MPPlayRoomEventRsp) })
	c.regMsg(MPPlayRoomEventNotice, func() any { return new(proto.MPPlayRoomEventNotice) })
	c.regMsg(MPPlayRoomReadyReq, func() any { return new(proto.MPPlayRoomReadyReq) })
	c.regMsg(MPPlayRoomReadyRsp, func() any { return new(proto.MPPlayRoomReadyRsp) })
	c.regMsg(HandingFurnitureReq, func() any { return new(proto.HandingFurnitureReq) })
	c.regMsg(HandingFurnitureRsp, func() any { return new(proto.HandingFurnitureRsp) })
	c.regMsg(TakeOutHandingFurnitureReq, func() any { return new(proto.TakeOutHandingFurnitureReq) })
	c.regMsg(TakeOutHandingFurnitureRsp, func() any { return new(proto.TakeOutHandingFurnitureRsp) })
	c.regMsg(SceneTransmitReq, func() any { return new(proto.SceneTransmitReq) })
	c.regMsg(SceneTransmitRsp, func() any { return new(proto.SceneTransmitRsp) })
	c.regMsg(MPPlayRoomSettlementNotice, func() any { return new(proto.MPPlayRoomSettlementNotice) })
	c.regMsg(MPPlayRoomRewardInfoReq, func() any { return new(proto.MPPlayRoomRewardInfoReq) })
	c.regMsg(MPPlayRoomRewardInfoRsp, func() any { return new(proto.MPPlayRoomRewardInfoRsp) })
	c.regMsg(MPPlayRoomExitReq, func() any { return new(proto.MPPlayRoomExitReq) })
	c.regMsg(MPPlayRoomExitRsp, func() any { return new(proto.MPPlayRoomExitRsp) })
	c.regMsg(MPPlayRoomExitNotice, func() any { return new(proto.MPPlayRoomExitNotice) })
	c.regMsg(UploadFGBytesNotice, func() any { return new(proto.UploadFGBytesNotice) })
	c.regMsg(UploadFGBytesReq, func() any { return new(proto.UploadFGBytesReq) })
	c.regMsg(UploadFGBytesRsp, func() any { return new(proto.UploadFGBytesRsp) })
	c.regMsg(BuyGameCoinReq, func() any { return new(proto.BuyGameCoinReq) })
	c.regMsg(BuyGameCoinRsp, func() any { return new(proto.BuyGameCoinRsp) })
	c.regMsg(FreezePlayerActionNotice, func() any { return new(proto.FreezePlayerActionNotice) })
	c.regMsg(UnfreezePlayerActionReq, func() any { return new(proto.UnfreezePlayerActionReq) })
	c.regMsg(UnfreezePlayerActionRsp, func() any { return new(proto.UnfreezePlayerActionRsp) })
	c.regMsg(UploadPhotoShareReq, func() any { return new(proto.UploadPhotoShareReq) })
	c.regMsg(UploadPhotoShareRsp, func() any { return new(proto.UploadPhotoShareRsp) })
	c.regMsg(ChangePhotoShareTitleReq, func() any { return new(proto.ChangePhotoShareTitleReq) })
	c.regMsg(ChangePhotoShareTitleRsp, func() any { return new(proto.ChangePhotoShareTitleRsp) })
	c.regMsg(PhotoShareSearchReq, func() any { return new(proto.PhotoShareSearchReq) })
	c.regMsg(PhotoShareSearchRsp, func() any { return new(proto.PhotoShareSearchRsp) })
	c.regMsg(PhotoShareDetailReq, func() any { return new(proto.PhotoShareDetailReq) })
	c.regMsg(PhotoShareDetailRsp, func() any { return new(proto.PhotoShareDetailRsp) })
	c.regMsg(OperatePhotoShareReq, func() any { return new(proto.OperatePhotoShareReq) })
	c.regMsg(OperatePhotoShareRsp, func() any { return new(proto.OperatePhotoShareRsp) })
	c.regMsg(PhotoShareCopyFashionDyeReq, func() any { return new(proto.PhotoShareCopyFashionDyeReq) })
	c.regMsg(PhotoShareCopyFashionDyeRsp, func() any { return new(proto.PhotoShareCopyFashionDyeRsp) })
	c.regMsg(PhotoSharePaymentAddUploadGridReq, func() any { return new(proto.PhotoSharePaymentAddUploadGridReq) })
	c.regMsg(PhotoSharePaymentAddUploadGridRsp, func() any { return new(proto.PhotoSharePaymentAddUploadGridRsp) })
	c.regMsg(GetExpressAddressReq, func() any { return new(proto.GetExpressAddressReq) })
	c.regMsg(GetExpressAddressRsp, func() any { return new(proto.GetExpressAddressRsp) })
	c.regMsg(UploadExpressAddressReq, func() any { return new(proto.UploadExpressAddressReq) })
	c.regMsg(UploadExpressAddressRsp, func() any { return new(proto.UploadExpressAddressRsp) })
	c.regMsg(CollectMoonReq, func() any { return new(proto.CollectMoonReq) })
	c.regMsg(CollectMoonRsp, func() any { return new(proto.CollectMoonRsp) })
	c.regMsg(GetCollectMoonInfoReq, func() any { return new(proto.GetCollectMoonInfoReq) })
	c.regMsg(GetCollectMoonInfoRsp, func() any { return new(proto.GetCollectMoonInfoRsp) })
	c.regMsg(CollectMoonInfoUpdateNotice, func() any { return new(proto.CollectMoonInfoUpdateNotice) })
	c.regMsg(EmotionMoonInteractReq, func() any { return new(proto.EmotionMoonInteractReq) })
	c.regMsg(EmotionMoonInteractRsp, func() any { return new(proto.EmotionMoonInteractRsp) })
	c.regMsg(BlessTreeNotice, func() any { return new(proto.BlessTreeNotice) })
	c.regMsg(BlessTreeReq, func() any { return new(proto.BlessTreeReq) })
	c.regMsg(BlessTreeRsp, func() any { return new(proto.BlessTreeRsp) })
	c.regMsg(BlessTreeUnlockReq, func() any { return new(proto.BlessTreeUnlockReq) })
	c.regMsg(BlessTreeUnlockRsp, func() any { return new(proto.BlessTreeUnlockRsp) })
	c.regMsg(DailyTaskUpdateNotice, func() any { return new(proto.DailyTaskUpdateNotice) })
	c.regMsg(DailyTaskExchangeReq, func() any { return new(proto.DailyTaskExchangeReq) })
	c.regMsg(DailyTaskExchangeRsp, func() any { return new(proto.DailyTaskExchangeRsp) })
	c.regMsg(MPGameWeaponUpdateNotice, func() any { return new(proto.MPGameWeaponUpdateNotice) })
	c.regMsg(GardenFurnitureRemoveAllReq, func() any { return new(proto.GardenFurnitureRemoveAllReq) })
	c.regMsg(GardenFurnitureRemoveAllRsp, func() any { return new(proto.GardenFurnitureRemoveAllRsp) })
	c.regMsg(GardenFurnitureBatchUpdateNotice, func() any { return new(proto.GardenFurnitureBatchUpdateNotice) })
	c.regMsg(GardenFurnitureSaveReq, func() any { return new(proto.GardenFurnitureSaveReq) })
	c.regMsg(GardenFurnitureSaveRsp, func() any { return new(proto.GardenFurnitureSaveRsp) })
	c.regMsg(GardenFurnitureApplySchemeReq, func() any { return new(proto.GardenFurnitureApplySchemeReq) })
	c.regMsg(GardenFurnitureApplySchemeRsp, func() any { return new(proto.GardenFurnitureApplySchemeRsp) })
	c.regMsg(GardenSchemeFurnitureListReq, func() any { return new(proto.GardenSchemeFurnitureListReq) })
	c.regMsg(GardenSchemeFurnitureListRsp, func() any { return new(proto.GardenSchemeFurnitureListRsp) })
	c.regMsg(GardenFurnitureSchemeSetNameReq, func() any { return new(proto.GardenFurnitureSchemeSetNameReq) })
	c.regMsg(GardenFurnitureSchemeSetNameRsp, func() any { return new(proto.GardenFurnitureSchemeSetNameRsp) })
	c.regMsg(GardenFurnitureSchemeReq, func() any { return new(proto.GardenFurnitureSchemeReq) })
	c.regMsg(GardenFurnitureSchemeRsp, func() any { return new(proto.GardenFurnitureSchemeRsp) })
	c.regMsg(GardenPlaceCharacterReq, func() any { return new(proto.GardenPlaceCharacterReq) })
	c.regMsg(GardenPlaceCharacterRsp, func() any { return new(proto.GardenPlaceCharacterRsp) })
	c.regMsg(GardenPlaceCharacterNotice, func() any { return new(proto.GardenPlaceCharacterNotice) })
	c.regMsg(GMRecommendChannelNotice, func() any { return new(proto.GMRecommendChannelNotice) })
	c.regMsg(UpdatePlayerAppearanceReq, func() any { return new(proto.UpdatePlayerAppearanceReq) })
	c.regMsg(UpdatePlayerAppearanceRsp, func() any { return new(proto.UpdatePlayerAppearanceRsp) })
	c.regMsg(ActivityTurntableDataNotice, func() any { return new(proto.ActivityTurntableDataNotice) })
	c.regMsg(ActivityTurntableDrawReq, func() any { return new(proto.ActivityTurntableDrawReq) })
	c.regMsg(ActivityTurntableDrawRsp, func() any { return new(proto.ActivityTurntableDrawRsp) })
	c.regMsg(ActivityTurntableRecordReq, func() any { return new(proto.ActivityTurntableRecordReq) })
	c.regMsg(ActivityTurntableRecordRsp, func() any { return new(proto.ActivityTurntableRecordRsp) })
	c.regMsg(AbyssServerRankNotice, func() any { return new(proto.AbyssServerRankNotice) })
	c.regMsg(UseItemFriendIntimacyReq, func() any { return new(proto.UseItemFriendIntimacyReq) })
	c.regMsg(UseItemFriendIntimacyRsp, func() any { return new(proto.UseItemFriendIntimacyRsp) })
	c.regMsg(IntimacyGiftDayCountNotice, func() any { return new(proto.IntimacyGiftDayCountNotice) })
	c.regMsg(AbyssAllServerRankReq, func() any { return new(proto.AbyssAllServerRankReq) })
	c.regMsg(AbyssAllServerRankRsp, func() any { return new(proto.AbyssAllServerRankRsp) })
	c.regMsg(ChangeMusicalItemReq, func() any { return new(proto.ChangeMusicalItemReq) })
	c.regMsg(ChangeMusicalItemRsp, func() any { return new(proto.ChangeMusicalItemRsp) })
	c.regMsg(PlayMusicNoteReq, func() any { return new(proto.PlayMusicNoteReq) })
	c.regMsg(PlayMusicNoteRsp, func() any { return new(proto.PlayMusicNoteRsp) })
	c.regMsg(WishListOperateReq, func() any { return new(proto.WishListOperateReq) })
	c.regMsg(WishListOperateRsp, func() any { return new(proto.WishListOperateRsp) })
	c.regMsg(WishListByShopIdReq, func() any { return new(proto.WishListByShopIdReq) })
	c.regMsg(WishListByShopIdRsp, func() any { return new(proto.WishListByShopIdRsp) })
	c.regMsg(WishListByFriendIdReq, func() any { return new(proto.WishListByFriendIdReq) })
	c.regMsg(WishListByFriendIdRsp, func() any { return new(proto.WishListByFriendIdRsp) })
	c.regMsg(WishListBuyNotice, func() any { return new(proto.WishListBuyNotice) })
	c.regMsg(WishListBuyReq, func() any { return new(proto.WishListBuyReq) })
	c.regMsg(WishListBuyRsp, func() any { return new(proto.WishListBuyRsp) })
	c.regMsg(SPOperateReq, func() any { return new(proto.SPOperateReq) })
	c.regMsg(SPOperateRsp, func() any { return new(proto.SPOperateRsp) })
	c.regMsg(BossRushInfoReq, func() any { return new(proto.BossRushInfoReq) })
	c.regMsg(BossRushInfoRsp, func() any { return new(proto.BossRushInfoRsp) })
	c.regMsg(BossRushStartChallengeReq, func() any { return new(proto.BossRushStartChallengeReq) })
	c.regMsg(BossRushStartChallengeRsp, func() any { return new(proto.BossRushStartChallengeRsp) })
	c.regMsg(BossRushEnterStageReq, func() any { return new(proto.BossRushEnterStageReq) })
	c.regMsg(BossRushEnterStageRsp, func() any { return new(proto.BossRushEnterStageRsp) })
	c.regMsg(BossRushStartBattleReq, func() any { return new(proto.BossRushStartBattleReq) })
	c.regMsg(BossRushStartBattleRsp, func() any { return new(proto.BossRushStartBattleRsp) })
	c.regMsg(BossRushLeaveStageReq, func() any { return new(proto.BossRushLeaveStageReq) })
	c.regMsg(BossRushLeaveStageRsp, func() any { return new(proto.BossRushLeaveStageRsp) })
	c.regMsg(BossRushFriendRankReq, func() any { return new(proto.BossRushFriendRankReq) })
	c.regMsg(BossRushFriendRankRsp, func() any { return new(proto.BossRushFriendRankRsp) })
	c.regMsg(BossRushFriendDetailReq, func() any { return new(proto.BossRushFriendDetailReq) })
	c.regMsg(BossRushFriendDetailRsp, func() any { return new(proto.BossRushFriendDetailRsp) })
	c.regMsg(BossRushTerminateChallengeReq, func() any { return new(proto.BossRushTerminateChallengeReq) })
	c.regMsg(BossRushTerminateChallengeRsp, func() any { return new(proto.BossRushTerminateChallengeRsp) })
	c.regMsg(BossRushQuestRewardReq, func() any { return new(proto.BossRushQuestRewardReq) })
	c.regMsg(BossRushQuestRewardRsp, func() any { return new(proto.BossRushQuestRewardRsp) })
	c.regMsg(RestaurantGameInfoReq, func() any { return new(proto.RestaurantGameInfoReq) })
	c.regMsg(RestaurantGameInfoRsp, func() any { return new(proto.RestaurantGameInfoRsp) })
	c.regMsg(RestaurantGameOperateReq, func() any { return new(proto.RestaurantGameOperateReq) })
	c.regMsg(RestaurantGameOperateRsp, func() any { return new(proto.RestaurantGameOperateRsp) })
	c.regMsg(RestaurantGameEatReq, func() any { return new(proto.RestaurantGameEatReq) })
	c.regMsg(RestaurantGameEatRsp, func() any { return new(proto.RestaurantGameEatRsp) })
	c.regMsg(RestaurantGameDayNotice, func() any { return new(proto.RestaurantGameDayNotice) })
	c.regMsg(RestaurantGameQuestRewardReq, func() any { return new(proto.RestaurantGameQuestRewardReq) })
	c.regMsg(RestaurantGameQuestRewardRsp, func() any { return new(proto.RestaurantGameQuestRewardRsp) })
	c.regMsg(MoonQuestGamePlayRewardNotice, func() any { return new(proto.MoonQuestGamePlayRewardNotice) })
	c.regMsg(PlaceMoonSpotReq, func() any { return new(proto.PlaceMoonSpotReq) })
	c.regMsg(PlaceMoonSpotRsp, func() any { return new(proto.PlaceMoonSpotRsp) })
	c.regMsg(PlaceMoonSpotNotice, func() any { return new(proto.PlaceMoonSpotNotice) })
	c.regMsg(SetRoomDecorReq, func() any { return new(proto.SetRoomDecorReq) })
	c.regMsg(SetRoomDecorRsp, func() any { return new(proto.SetRoomDecorRsp) })
	c.regMsg(RoomDecorNotice, func() any { return new(proto.RoomDecorNotice) })
}
