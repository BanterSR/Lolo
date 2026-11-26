package game

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

type activityTestHarness struct {
	service *activityService
	player  *model.Player
	now     int64
	saves   int
	saveErr error
}

func newActivityTestHarness(t *testing.T) *activityTestHarness {
	t.Helper()
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC).Unix()
	h := &activityTestHarness{now: now}
	conf := newActivityConfigs()
	conf.Items[100] = itemConfigRow{ID: 100, Tag: int32(proto.EBagItemTag_EBagItemTag_Currency)}
	conf.Items[200] = itemConfigRow{ID: 200, Tag: int32(proto.EBagItemTag_EBagItemTag_Material)}
	conf.Items[300] = itemConfigRow{ID: 300, Tag: int32(proto.EBagItemTag_EBagItemTag_Material)}
	h.player = &model.Player{UserId: 1, Created: time.Unix(now-3600, 0).UTC()}
	h.player.GetItemModel().GetItemBaseMap()[100] = &model.ItemBaseInfo{ItemId: 100, Num: 1000, ItemType: proto.EBagItemTag_EBagItemTag_Currency}
	h.service = &activityService{
		conf:        conf,
		now:         func() int64 { return h.now },
		randomInt:   func(n int) int { return 0 },
		playerLevel: func(*model.Player) uint32 { return 1 },
		location:    time.UTC,
		save: func(*model.Player) error {
			h.saves++
			return h.saveErr
		},
		grant: func(s *model.Player, items []activityItem) ([]*proto.ItemDetail, error) {
			details := make([]*proto.ItemDetail, 0, len(items))
			for _, item := range items {
				info := s.GetItemModel().GetItemBaseInfo(item.ID)
				if info == nil {
					info = &model.ItemBaseInfo{ItemId: item.ID, ItemType: proto.EBagItemTag(item.ID % 10)}
					s.GetItemModel().GetItemBaseMap()[item.ID] = info
				}
				info.Num += item.Num
				details = append(details, info.ItemDetail())
			}
			return details, nil
		},
	}
	return h
}

func (h *activityTestHarness) addActivity(id, template uint32, activityType proto.EActivityType, open int64) {
	h.service.conf.Activities[id] = activityConfigRow{
		ID: id, ActivityID: template, NewActivityType: int32(activityType), openUnix: open, closeUnix: open + 30*86400,
	}
}

func (h *activityTestHarness) itemCount(id uint32) int64 {
	info := h.player.GetItemModel().GetItemBaseInfo(id)
	if info == nil {
		return 0
	}
	return info.Num
}

func (h *activityTestHarness) restart(t *testing.T) {
	t.Helper()
	data, err := sonic.Marshal(h.player)
	if err != nil {
		t.Fatal(err)
	}
	var restored model.Player
	if err := sonic.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.UserId = h.player.UserId
	restored.Created = h.player.Created
	h.player = &restored
}

func TestActivitySignInPeriodDuplicateAndRestart(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(1, 10, proto.EActivityType_EActivityType_SevenDaySign, open)
	h.addActivity(2, 10, proto.EActivityType_EActivityType_SevenDaySign, open+14*86400)
	h.service.conf.Signs[10] = activitySignConfig{ID: 10, Items: []activitySignDayConfig{
		{Day: 1, Rewards: []activitySignReward{{ItemID: 200, ItemNum: 1}}},
		{Day: 2, Rewards: []activitySignReward{{ItemID: 200, ItemNum: 2}}},
	}}
	if err := h.service.touchActivityLogin(h.player, h.now); err != nil {
		t.Fatal(err)
	}

	rsp, _ := h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 1, Day: 1})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 1 {
		t.Fatalf("first sign-in failed: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	rsp, _ = h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 1, Day: 1})
	if rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD || h.itemCount(200) != 1 {
		t.Fatalf("duplicate sign-in was accepted: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	h.now = open + 86400 + 1
	if err := h.service.touchActivityLogin(h.player, h.now); err != nil {
		t.Fatal(err)
	}
	rsp, _ = h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 1, Day: 2})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 3 {
		t.Fatalf("second day sign-in failed: status=%v count=%d", rsp.Status, h.itemCount(200))
	}

	h.now = open + 14*86400 + 1
	if err := h.service.touchActivityLogin(h.player, h.now); err != nil {
		t.Fatal(err)
	}
	rsp, _ = h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 2, Day: 1})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 4 {
		t.Fatalf("new activity period did not reset claims: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	h.restart(t)
	rsp, _ = h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 2, Day: 1})
	if rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD || h.itemCount(200) != 4 {
		t.Fatalf("restart lost period claim: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
}

func TestActivitySignInRejectsFutureClosedAndMissingConfig(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(1, 10, proto.EActivityType_EActivityType_SevenDaySign, open)
	h.service.conf.Signs[10] = activitySignConfig{ID: 10, Items: []activitySignDayConfig{{Day: 1, Rewards: []activitySignReward{{ItemID: 200, ItemNum: 1}}}}}

	rsp, _ := h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 1, Day: 2})
	if rsp.Status != proto.StatusCode_StatusCode_ChallengeNotStart {
		t.Fatalf("future day accepted: %v", rsp.Status)
	}
	rsp, _ = h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 99, Day: 1})
	if rsp.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND {
		t.Fatalf("missing activity accepted: %v", rsp.Status)
	}
	h.now = open + 31*86400
	rsp, _ = h.service.signIn(h.player, &proto.ActivitySignInReq{ActivityId: 1, Day: 1})
	if rsp.Status != proto.StatusCode_StatusCode_Activity_Close {
		t.Fatalf("closed activity accepted: %v", rsp.Status)
	}
}

func TestActivityQuestRequiresServerProgress(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(1, 20, proto.EActivityType_EActivityType_SevenDayQuest, open)
	h.service.conf.Quests[20] = activityQuestConfig{ID: 20, AchieveItems: []activityQuestAchieveConfig{{
		Day: 1, AchieveID: 1001, Rewards: []activityDirectItem{{ItemID: 200, ItemNum: 2}},
	}}}
	h.service.conf.Achieves[1001] = achieveConfig{ID: 1001, CountParam: 2}
	h.service.conf.buildAchievementSpecs()

	rsp, _ := h.service.activityQuestReward(h.player, &proto.ActivityQuestRewardReq{ActivityId: 1, Param: 1})
	if rsp.Status != proto.StatusCode_StatusCode_Quest_Not_Finish || h.saves != 0 || h.itemCount(200) != 0 {
		t.Fatalf("unfinished quest rewarded: status=%v saves=%d count=%d", rsp.Status, h.saves, h.itemCount(200))
	}
	if err := h.service.recordAchievement(h.player, 1001, 2, h.now); err != nil {
		t.Fatal(err)
	}
	rsp, _ = h.service.activityQuestReward(h.player, &proto.ActivityQuestRewardReq{ActivityId: 1, Param: 1})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 2 {
		t.Fatalf("completed quest not rewarded: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	rsp, _ = h.service.activityQuestReward(h.player, &proto.ActivityQuestRewardReq{ActivityId: 1, Param: 1})
	if rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("duplicate quest reward accepted: %v", rsp.Status)
	}
}

func TestActivityRegularDailyRewardResetsByServerDay(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(5, 30, proto.EActivityType_EActivityType_ActivityRegular, open)
	h.service.conf.Regulars[30] = activityRegularConfig{ID: 30, DailyPoolID: 31}
	h.service.conf.RegularPools[31] = activityRegularPoolConfig{ID: 31, Items: []activityRegularAchieveConfig{{AchieveID: 1001, Rewards: []activityDirectItem{{ItemID: 200, ItemNum: 1}}}}}
	h.service.conf.Achieves[1001] = achieveConfig{ID: 1001, CountParam: 1}
	h.service.conf.buildAchievementSpecs()
	if err := h.service.recordAchievement(h.player, 1001, 1, h.now); err != nil {
		t.Fatal(err)
	}
	req := &proto.ActivityRegularRewardReq{ActivityId: 5, Param: 1001, PoolType: 1}
	rsp, _ := h.service.activityRegularReward(h.player, req)
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 1 {
		t.Fatalf("first daily reward failed: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	rsp, _ = h.service.activityRegularReward(h.player, req)
	if rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("duplicate daily reward accepted: %v", rsp.Status)
	}
	h.now += 86400
	rsp, _ = h.service.activityRegularReward(h.player, req)
	if rsp.Status != proto.StatusCode_StatusCode_Quest_Not_Finish {
		t.Fatalf("previous day progress satisfied new day: %v", rsp.Status)
	}
	if err := h.service.recordAchievement(h.player, 1001, 1, h.now); err != nil {
		t.Fatal(err)
	}
	rsp, _ = h.service.activityRegularReward(h.player, req)
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 2 {
		t.Fatalf("new daily reward failed: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
}

func TestActivityQuestMissingRewardRollsBackClaim(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(1, 20, proto.EActivityType_EActivityType_SevenDayQuest, open)
	h.service.conf.Quests[20] = activityQuestConfig{ID: 20, AchieveItems: []activityQuestAchieveConfig{{
		Day: 1, AchieveID: 1001, Rewards: []activityDirectItem{{ItemID: 999, ItemNum: 1}},
	}}}
	h.service.conf.Achieves[1001] = achieveConfig{ID: 1001, CountParam: 1}
	h.service.conf.buildAchievementSpecs()
	if err := h.service.recordAchievement(h.player, 1001, 1, h.now); err != nil {
		t.Fatal(err)
	}
	rsp, _ := h.service.activityQuestReward(h.player, &proto.ActivityQuestRewardReq{ActivityId: 1, Param: 1})
	if rsp.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || h.saves != 0 {
		t.Fatalf("missing reward accepted: status=%v saves=%d", rsp.Status, h.saves)
	}
	state := h.player.GetActivityModel().GetPeriod(activityPeriodKey(h.service.conf.Activities[1]))
	if state.IsClaimed("quest:achieve:1001") {
		t.Fatal("failed reward persisted its claim")
	}
}

func TestBattlePassBuyExpAndLevelRewards(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(30, 100, proto.EActivityType_EActivityType_BattlePass, open)
	h.service.conf.BattlePasses[100] = battlePassConfig{
		ID: 100, MaxExp: 1000,
		Levels: []battlePassLevelConfig{{Level: 1, Exp: 1000, Rewards: []activityDirectItem{{ItemID: 200, ItemNum: 1}, {ItemID: 300, ItemNum: 2}}}},
	}
	h.service.conf.Globals["BattlePass_BuyExp_Currency"] = []string{"100:50:1000"}
	h.player.GetItemModel().GetItemBaseInfo(100).Num = 50

	rsp, _ := h.service.battlePassBuyExp(h.player, &proto.ActivityBattlePassBuyExpReq{ActivityBattlePassId: 30})
	if rsp.Status != proto.StatusCode_StatusCode_OK || rsp.BattlePassLv != 1 || rsp.TotalExp != 1000 || h.itemCount(100) != 0 {
		t.Fatalf("buy exp failed: status=%v lv=%d total=%d currency=%d", rsp.Status, rsp.BattlePassLv, rsp.TotalExp, h.itemCount(100))
	}
	free, _ := h.service.battlePassLevelReward(h.player, &proto.GetActivityBattlePassLevelRewardReq{ActivityBattlePassId: 30, Level: 1, IsGetFreeReward: true})
	if free.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 1 {
		t.Fatalf("free level reward failed: status=%v count=%d", free.Status, h.itemCount(200))
	}
	free, _ = h.service.battlePassLevelReward(h.player, &proto.GetActivityBattlePassLevelRewardReq{ActivityBattlePassId: 30, Level: 1, IsGetFreeReward: true})
	if free.Status != proto.StatusCode_StatusCode_BattlePass_LvRewarded {
		t.Fatalf("duplicate level reward accepted: %v", free.Status)
	}
	paid, _ := h.service.battlePassLevelReward(h.player, &proto.GetActivityBattlePassLevelRewardReq{ActivityBattlePassId: 30, Level: 1})
	if paid.Status != proto.StatusCode_StatusCode_BattlePass_Not_Buy {
		t.Fatalf("paid reward accepted without purchase: %v", paid.Status)
	}
	if err := h.service.recordBattlePassPurchased(h.player, 30, h.now); err != nil {
		t.Fatal(err)
	}
	paid, _ = h.service.battlePassLevelReward(h.player, &proto.GetActivityBattlePassLevelRewardReq{ActivityBattlePassId: 30, Level: 1})
	if paid.Status != proto.StatusCode_StatusCode_OK || h.itemCount(300) != 2 {
		t.Fatalf("paid level reward failed: status=%v count=%d", paid.Status, h.itemCount(300))
	}
}

func TestBattlePassBuyExpInsufficientAndSaveFailureAreAtomic(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(30, 100, proto.EActivityType_EActivityType_BattlePass, open)
	h.service.conf.BattlePasses[100] = battlePassConfig{
		ID: 100, MaxExp: 1000,
		Levels: []battlePassLevelConfig{{Level: 1, Exp: 1000, Rewards: []activityDirectItem{{ItemID: 200, ItemNum: 1}, {ItemID: 300, ItemNum: 2}}}},
	}
	h.service.conf.Globals["BattlePass_BuyExp_Currency"] = []string{"100:50:1000"}
	h.player.GetItemModel().GetItemBaseInfo(100).Num = 49
	rsp, _ := h.service.battlePassBuyExp(h.player, &proto.ActivityBattlePassBuyExpReq{ActivityBattlePassId: 30})
	if rsp.Status != proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH || h.itemCount(100) != 49 || h.saves != 0 {
		t.Fatalf("insufficient currency changed state: status=%v currency=%d saves=%d", rsp.Status, h.itemCount(100), h.saves)
	}

	h.player.GetItemModel().GetItemBaseInfo(100).Num = 50
	h.saveErr = errors.New("save failed")
	rsp, _ = h.service.battlePassBuyExp(h.player, &proto.ActivityBattlePassBuyExpReq{ActivityBattlePassId: 30})
	if rsp.Status == proto.StatusCode_StatusCode_OK || h.itemCount(100) != 50 {
		t.Fatalf("save failure was not rolled back: status=%v currency=%d", rsp.Status, h.itemCount(100))
	}
	state := h.player.GetActivityModel().GetPeriod(activityPeriodKey(h.service.conf.Activities[30])).GetBattlePass()
	if state.TotalExp != 0 || state.Level != 0 {
		t.Fatalf("save failure persisted battle pass exp: %+v", state)
	}
}

func TestActivityTurntableLimitsPersistenceAndMaterials(t *testing.T) {
	h := newActivityTestHarness(t)
	open := h.now - 3600
	h.addActivity(40, 200, proto.EActivityType_EActivityType_ActivityTurntable, open)
	h.service.conf.Turntables[200] = activityTurntableConfig{ID: 200, PoolID: 201, CostItemID: 100, FirstOneNeedItemNum: 6, OneNeedItemNum: 1}
	h.service.conf.TurntablePools[201] = activityTurntablePoolConfig{ID: 201, Items: []activityTurntablePoolItem{{Index: 1, ItemID: 200, ItemNum: 1, LimitNum: 1, Weight: 1}}}
	h.player.GetItemModel().GetItemBaseInfo(100).Num = 5
	rsp, _ := h.service.activityTurntableDraw(h.player, &proto.ActivityTurntableDrawReq{ActivityId: 40, DrawTimes: 1})
	if rsp.Status != proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH || h.itemCount(100) != 5 {
		t.Fatalf("insufficient turntable material changed state: status=%v currency=%d", rsp.Status, h.itemCount(100))
	}

	h.player.GetItemModel().GetItemBaseInfo(100).Num = 6
	rsp, _ = h.service.activityTurntableDraw(h.player, &proto.ActivityTurntableDrawReq{ActivityId: 40, DrawTimes: 1})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(100) != 0 || h.itemCount(200) != 1 {
		t.Fatalf("turntable draw failed: status=%v currency=%d reward=%d", rsp.Status, h.itemCount(100), h.itemCount(200))
	}
	h.restart(t)
	rsp, _ = h.service.activityTurntableDraw(h.player, &proto.ActivityTurntableDrawReq{ActivityId: 40, DrawTimes: 1})
	if rsp.Status != proto.StatusCode_StatusCode_TURNTABLE_NO_REWARD_REMAIN || h.itemCount(200) != 1 {
		t.Fatalf("turntable limit reset after restart: status=%v reward=%d", rsp.Status, h.itemCount(200))
	}
	records := h.service.activityTurntableRecord(h.player, &proto.ActivityTurntableRecordReq{ActivityId: 40, Page: 1})
	if records.Status != proto.StatusCode_StatusCode_OK || len(records.Records) != 1 {
		t.Fatalf("turntable record missing after restart: status=%v records=%d", records.Status, len(records.Records))
	}
}

func TestSupplyBoxCooldownPersistenceAndMissingReward(t *testing.T) {
	h := newActivityTestHarness(t)
	h.service.conf.Globals["SupplyBox_Random_Time"] = []string{"10", "10"}
	h.service.conf.Globals["SupplyBox_Reward"] = []string{"200:1"}
	h.player.Created = time.Unix(h.now-20*60, 0).UTC()
	info, _ := h.service.supplyBoxInfo(h.player)
	if info.Status != proto.StatusCode_StatusCode_OK || info.NextRewardTime != h.now-10*60 {
		t.Fatalf("supply info failed: status=%v next=%d", info.Status, info.NextRewardTime)
	}
	rsp, _ := h.service.supplyBoxReward(h.player)
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 1 || rsp.NextRewardTime != h.now+10*60 {
		t.Fatalf("supply reward failed: status=%v count=%d next=%d", rsp.Status, h.itemCount(200), rsp.NextRewardTime)
	}
	h.restart(t)
	rsp, _ = h.service.supplyBoxReward(h.player)
	if rsp.Status != proto.StatusCode_StatusCode_Interval_Reward_Is_Get || h.itemCount(200) != 1 {
		t.Fatalf("supply cooldown reset after restart: status=%v count=%d", rsp.Status, h.itemCount(200))
	}

	delete(h.service.conf.Globals, "SupplyBox_Reward")
	h.now += 11 * 60
	rsp, _ = h.service.supplyBoxReward(h.player)
	if rsp.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND || h.itemCount(200) != 1 {
		t.Fatalf("missing supply reward was fabricated: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
}

func TestMonthCardDailyClaimAndRestart(t *testing.T) {
	h := newActivityTestHarness(t)
	h.service.conf.Globals["MoonCard_Rveryday_Currency"] = []string{"200:2"}
	state := h.player.GetActivityModel().GetMonthCard()
	state.OverDueTime = h.now + 3*86400
	rsp, _ := h.service.monthCardReward(h.player)
	if rsp.Status != proto.StatusCode_StatusCode_OK || rsp.MonthCardRewardDays != 1 || h.itemCount(200) != 2 {
		t.Fatalf("month card claim failed: status=%v days=%d count=%d", rsp.Status, rsp.MonthCardRewardDays, h.itemCount(200))
	}
	h.restart(t)
	rsp, _ = h.service.monthCardReward(h.player)
	if rsp.Status != proto.StatusCode_StatusCode_MOON_CARD_DAYS_LIMIT || h.itemCount(200) != 2 {
		t.Fatalf("duplicate month card claim accepted: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	h.now += 86400
	rsp, _ = h.service.monthCardReward(h.player)
	if rsp.Status != proto.StatusCode_StatusCode_OK || rsp.MonthCardRewardDays != 2 || h.itemCount(200) != 4 {
		t.Fatalf("next day month card claim failed: status=%v days=%d count=%d", rsp.Status, rsp.MonthCardRewardDays, h.itemCount(200))
	}
}

func TestAchieveGroupAwardRequiresAllRealProgress(t *testing.T) {
	h := newActivityTestHarness(t)
	h.service.conf.Achieves[1001] = achieveConfig{ID: 1001, CountParam: 1}
	h.service.conf.Achieves[1002] = achieveConfig{ID: 1002, CountParam: 2}
	h.service.conf.AchieveQuestByGroup[500] = achieveQuestConfig{GroupID: 500, RewardID: 600}
	h.service.conf.AchieveQuestGroups[500] = achieveQuestGroupConfig{ID: 500, Items: []achieveQuestGroupItem{{AchieveID: 1001, RewardID: 601}, {AchieveID: 1002, RewardID: 602}}}
	h.service.conf.RewardPools[600] = rewardPoolConfig{ID: 600, Groups: []rewardPoolGroupConfig{{ID: 601}}}
	h.service.conf.RewardItemPools[601] = rewardItemPoolConfig{ID: 601, Groups: []rewardItemPoolGroupConfig{{ItemID: 200, ItemMinCount: 1, ItemMaxCount: 1}}}
	h.player.GetActivityModel().AddAchievementProgress(1001, 1)

	rsp, _ := h.service.getAchieveGroupAward(h.player, &proto.GetAchieveGroupAwardReq{GroupId: 500})
	if rsp.Status != proto.StatusCode_StatusCode_Quest_Not_Finish || h.itemCount(200) != 0 {
		t.Fatalf("incomplete group rewarded: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	h.player.GetActivityModel().AddAchievementProgress(1002, 2)
	rsp, _ = h.service.getAchieveGroupAward(h.player, &proto.GetAchieveGroupAwardReq{GroupId: 500})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 1 {
		t.Fatalf("complete group not rewarded: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	rsp, _ = h.service.getAchieveGroupAward(h.player, &proto.GetAchieveGroupAwardReq{GroupId: 500})
	if rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("duplicate group reward accepted: %v", rsp.Status)
	}
}

func TestAbyssQuestRewardRequiresSeasonProgress(t *testing.T) {
	h := newActivityTestHarness(t)
	h.service.conf.AbyssSeasons[1] = abyssSeasonConfig{ID: 1, SeasonAchieve: 10, openUnix: h.now - 3600, closeUnix: h.now + 3600}
	h.service.conf.AbyssAchieves[10] = abyssSeasonAchieveConfig{ID: 10, Items: []abyssSeasonAchieveItem{{AchieveID: 1001, RewardID: 600}}}
	h.service.conf.Achieves[1001] = achieveConfig{ID: 1001, CountParam: 3}
	h.service.conf.RewardItemPools[600] = rewardItemPoolConfig{ID: 600, Groups: []rewardItemPoolGroupConfig{{ItemID: 200, ItemMinCount: 1, ItemMaxCount: 1}}}

	rsp, _ := h.service.abyssQuestReward(h.player, &proto.AbyssQuestRewardReq{SeasonId: 1, ConditionId: 1001})
	if rsp.Status != proto.StatusCode_StatusCode_Quest_Not_Finish {
		t.Fatalf("unfinished abyss achievement rewarded: %v", rsp.Status)
	}
	h.player.GetActivityModel().AddAchievementProgress(1001, 3)
	rsp, _ = h.service.abyssQuestReward(h.player, &proto.AbyssQuestRewardReq{SeasonId: 1, ConditionId: 1001})
	if rsp.Status != proto.StatusCode_StatusCode_OK || h.itemCount(200) != 1 {
		t.Fatalf("complete abyss achievement not rewarded: status=%v count=%d", rsp.Status, h.itemCount(200))
	}
	rsp, _ = h.service.abyssQuestReward(h.player, &proto.AbyssQuestRewardReq{SeasonId: 1, ConditionId: 1001})
	if rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("duplicate abyss reward accepted: %v", rsp.Status)
	}
}

func TestActivityDefaultGranterUsesConfiguredStackableItem(t *testing.T) {
	h := newActivityTestHarness(t)
	h.service.grant = nil
	items, err := h.service.grantItems(h.player, []activityItem{{ID: 200, Num: 3}})
	if err != nil || len(items) != 1 || h.itemCount(200) != 3 {
		t.Fatalf("configured stackable reward failed: items=%d count=%d err=%v", len(items), h.itemCount(200), err)
	}
}

func TestLoadActivityConfigsUsesAuthoritativeSnakeCaseSchema(t *testing.T) {
	dir := t.TempDir()
	activity := `{"activity":{"datas":[{"i_d":900,"activity_i_d":10,"new_activity_type":1,"opening_time":"2026-11-20 00:00:00","closing_time":"2026-11-21 00:00:00"}]}}`
	sign := `{"activity_sign":{"datas":[{"i_d":10,"activity_sign_item":[{"days":1,"activity_sign_reward_item":[{"item_i_d":200,"item_num":2}]}]}]}}`
	if err := os.WriteFile(filepath.Join(dir, "Activity.json"), []byte(activity), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ActivitySign.json"), []byte(sign), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := loadActivityConfigs(dir)
	row, ok := conf.Activities[900]
	if !ok || row.templateID() != 10 || row.openUnix == 0 {
		t.Fatalf("activity config not loaded: %+v", row)
	}
	signConf, ok := conf.Signs[10]
	if !ok || len(signConf.Items) != 1 || signConf.Items[0].Rewards[0].ItemNum != 2 {
		t.Fatalf("sign config not loaded: %+v", signConf)
	}
}
