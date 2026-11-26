package game

import (
	"errors"
	"testing"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

type dungeonTestFixture struct {
	service *dungeonService
	player  *model.Player
	config  *dungeonConfig
	now     time.Time
	grants  []dungeonReward
}

func newDungeonTestFixture() *dungeonTestFixture {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	cfg := &dungeonConfig{
		ID:              1,
		NeedPlayerLevel: 1,
		TeamSize:        2,
		DailyLimit:      2,
		CanReplay:       true,
		FightTime:       60,
		RewardID:        10,
		StarRewardID:    11,
		StarTimer1:      30,
		StarTimer2:      20,
		StarTimer3:      10,
		Monsters: map[uint32]dungeonMonster{
			7: {UnitID: 100, RewardPoolID: 20},
		},
		MonstersByIndex: true,
		Treasures:       map[uint32]uint32{3: 30},
	}
	f := &dungeonTestFixture{
		config: cfg,
		now:    now,
		player: &model.Player{
			UserId: 99,
			Item:   &model.ItemModel{ItemBaseInfo: make(map[uint32]*model.ItemBaseInfo)},
			Character: &model.CharacterModel{CharacterMap: map[uint32]*model.CharacterInfo{
				1: {CharacterId: 1},
				2: {CharacterId: 2},
			}},
		},
	}
	f.service = &dungeonService{
		loadConfig: func(id uint32) (*dungeonConfig, error) {
			if id != cfg.ID {
				return nil, errors.New("missing config")
			}
			return cfg, nil
		},
		loadReward: func(id uint32) ([]dungeonReward, error) {
			switch id {
			case 0:
				return nil, nil
			case 10, 11, 20, 30:
				return []dungeonReward{{ItemID: id * 10, MinCount: 1, MaxCount: 1, ItemType: proto.EBagItemTag_EBagItemTag_Material}}, nil
			default:
				return nil, errors.New("missing reward")
			}
		},
		loadTreasureReward: func(boxID uint32) ([]dungeonReward, error) {
			if boxID != 30 {
				return nil, errors.New("missing treasure")
			}
			return []dungeonReward{{ItemID: 300, MinCount: 1, MaxCount: 1, ItemType: proto.EBagItemTag_EBagItemTag_Material}}, nil
		},
		playerLevel: func(*model.Player) (uint32, bool) { return 10, true },
		addReward: func(player *model.Player, reward dungeonReward) *proto.ItemDetail {
			count := reward.Count()
			reward.MinCount = count
			reward.MaxCount = count
			f.grants = append(f.grants, reward)
			item := player.GetItemModel().GetItemBaseInfo(reward.ItemID)
			if item == nil {
				item = &model.ItemBaseInfo{
					ItemId:   reward.ItemID,
					ItemType: proto.EBagItemTag_EBagItemTag_Material,
					PackType: proto.PackType_PackType_Inventory,
				}
				player.GetItemModel().ItemBaseInfo[reward.ItemID] = item
			}
			item.Num += count
			return item.ItemDetail()
		},
		now: func() time.Time { return f.now },
	}
	info := f.player.GetDungeonModel().GetDungeonInfo(cfg.ID)
	info.TreasureBoxes = map[uint32]*model.TreasureBox{
		3: {Index: 3, BoxId: 30, State: proto.TreasureBoxState_TreasureBoxState_Close},
	}
	return f
}

func (f *dungeonTestFixture) grantTotal(itemID uint32) int64 {
	item := f.player.GetItemModel().GetItemBaseInfo(itemID)
	if item == nil {
		return 0
	}
	return item.Num
}

func (f *dungeonTestFixture) enter(t *testing.T) (*model.SceneDungeon, *model.DungeonRuntime) {
	t.Helper()
	runtime, status := f.service.enterDungeon(f.player, nil, f.config.ID, model.TeamInfo{Char1: 1, Char2: 2})
	if status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("enter failed: %v", status)
	}
	return f.scene(runtime), runtime
}

func (f *dungeonTestFixture) scene(runtime *model.DungeonRuntime) *model.SceneDungeon {
	info := f.player.GetDungeonModel().GetDungeonInfo(f.config.ID)
	return &model.SceneDungeon{
		ScenePlayerInfo: &model.ScenePlayerInfo{
			Player:   f.player,
			Team:     &info.TeamInfo,
			TempPack: model.NewSceneTempPack(),
		},
		Info:    info,
		Runtime: runtime,
	}
}

func (f *dungeonTestFixture) start(t *testing.T, sd *model.SceneDungeon) {
	t.Helper()
	if status := f.service.start(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("start failed: %v", status)
	}
}

func TestDungeonRejectsEndAndFinishBeforeStart(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	if status := f.service.end(f.player, sd); status != proto.StatusCode_StatusCode_ChallengeNotStart {
		t.Fatalf("end before start status = %v", status)
	}
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_ChallengeNotEnd {
		t.Fatalf("finish before start status = %v", status)
	}
	f.start(t, sd)
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_ChallengeNotEnd {
		t.Fatalf("finish before end status = %v", status)
	}
}

func TestDungeonFinishRequiresAllAuthoritativeMonsterDeaths(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	if status := f.service.end(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("end failed: %v", status)
	}
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_ExploreNotFinish {
		t.Fatalf("finish with living monster status = %v", status)
	}
}

func TestDungeonMonsterDeathIsWhitelistedAndSettledOnce(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)

	if drop, status := f.service.monsterDead(f.player, sd, 101, 7); status != proto.StatusCode_StatusCode_DungeonCheat || drop != nil {
		t.Fatalf("forged monster accepted: drop=%v status=%v", drop, status)
	}
	if drop, status := f.service.monsterDead(f.player, sd, 100, 8); status != proto.StatusCode_StatusCode_DungeonCheat || drop != nil {
		t.Fatalf("forged index accepted: drop=%v status=%v", drop, status)
	}
	drop, status := f.service.monsterDead(f.player, sd, 100, 7)
	if status != proto.StatusCode_StatusCode_Ok || drop == nil {
		t.Fatalf("valid monster death failed: drop=%v status=%v", drop, status)
	}
	if duplicate, status := f.service.monsterDead(f.player, sd, 100, 7); status != proto.StatusCode_StatusCode_DungeonCheat || duplicate != nil {
		t.Fatalf("duplicate death generated loot: drop=%v status=%v", duplicate, status)
	}
	if len(runtime.Drops) != 1 || !runtime.DeadMonsters[7] {
		t.Fatalf("unexpected runtime death state: %+v", runtime)
	}
}

func TestDungeonMonsterIDOnlyConfigSettlesEachMonsterOnce(t *testing.T) {
	f := newDungeonTestFixture()
	f.config.MonstersByIndex = false
	f.config.Monsters = map[uint32]dungeonMonster{100: {UnitID: 100, RewardPoolID: 20}}
	sd, runtime := f.enter(t)
	f.start(t, sd)
	if drop, status := f.service.monsterDead(f.player, sd, 100, 999); status != proto.StatusCode_StatusCode_Ok || drop == nil {
		t.Fatalf("configured monster ID rejected: drop=%v status=%v", drop, status)
	}
	if drop, status := f.service.monsterDead(f.player, sd, 100, 1); status != proto.StatusCode_StatusCode_DungeonCheat || drop != nil {
		t.Fatalf("monster ID settled twice: drop=%v status=%v", drop, status)
	}
	if !runtime.DeadMonsters[100] {
		t.Fatal("monster ID was not recorded in runtime")
	}
	if _, status := f.service.encounterState(f.player, sd, 100); status != proto.StatusCode_StatusCode_DungeonCheat {
		t.Fatalf("monster-only config invented encounter: %v", status)
	}
}

func TestDungeonUsesMonsterSnapshotFromEnteredRound(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	f.config.Monsters[7] = dungeonMonster{UnitID: 999, RewardPoolID: 20}
	f.service.loadConfig = func(uint32) (*dungeonConfig, error) { return nil, errors.New("changed") }
	if drop, status := f.service.monsterDead(f.player, sd, 100, 7); status != proto.StatusCode_StatusCode_Ok || drop == nil {
		t.Fatalf("entered-round monster was not authoritative: drop=%v status=%v", drop, status)
	}
	if drop, status := f.service.monsterDead(f.player, sd, 999, 7); status != proto.StatusCode_StatusCode_DungeonCheat || drop != nil {
		t.Fatalf("changed config affected active round: drop=%v status=%v", drop, status)
	}
}

func TestDungeonRejectsMissingOrDuplicateCharactersAndRepeatedEntry(t *testing.T) {
	f := newDungeonTestFixture()
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1, Char2: 999}); status != proto.StatusCode_StatusCode_CharNotExist {
		t.Fatalf("missing character status = %v", status)
	}
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1, Char2: 1}); status != proto.StatusCode_StatusCode_DungeonTeamError {
		t.Fatalf("duplicate character status = %v", status)
	}
	f.player.Character.CharacterMap[1].CharacterId = 999
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_CharNotExist {
		t.Fatalf("mismatched character status = %v", status)
	}
	f.player.Character.CharacterMap[1].CharacterId = 1
	sd, runtime := f.enter(t)
	if next, status := f.service.enterDungeon(f.player, sd, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_ExploreInProgress || next != nil {
		t.Fatalf("repeated enter accepted: runtime=%v status=%v", next, status)
	}
	if s := f.scene(runtime); s.Runtime.Token != runtime.Token {
		t.Fatal("test scene changed runtime token")
	}
}

func TestDungeonRewardClaimIsIdempotent(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)
	if _, status := f.service.monsterDead(f.player, sd, 100, 7); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("monster death failed: %v", status)
	}
	if status := f.service.end(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("end failed: %v", status)
	}
	items, status := f.service.finish(f.player, sd)
	if status != proto.StatusCode_StatusCode_Ok || len(items) != 2 {
		t.Fatalf("finish rewards: items=%v status=%v", items, status)
	}
	if len(f.grants) != 2 || runtime.State != model.DungeonRuntimeClaimed {
		t.Fatalf("unexpected first claim: grants=%v state=%v", f.grants, runtime.State)
	}
	duplicate, status := f.service.finish(f.player, sd)
	if status != proto.StatusCode_StatusCode_RepeatGetReward || len(duplicate) != 0 {
		t.Fatalf("duplicate claim: items=%v status=%v", duplicate, status)
	}
	info := f.player.GetDungeonModel().GetDungeonInfo(1)
	if len(f.grants) != 2 || info.FinishTimes != 1 || info.RewardClaimedToken != runtime.Token || !info.FirstClearRewardClaimed {
		t.Fatalf("duplicate claim mutated progress: grants=%v info=%+v", f.grants, info)
	}
}

func TestDungeonTokenSkipsPersistedTokensAfterRestart(t *testing.T) {
	f := newDungeonTestFixture()
	reserved := nextDungeonToken()
	info := f.player.GetDungeonModel().GetDungeonInfo(1)
	info.LastRunToken = reserved
	info.RewardClaimedToken = reserved + 1
	_, runtime := f.enter(t)
	if runtime.Token == 0 || runtime.Token == reserved || runtime.Token == reserved+1 {
		t.Fatalf("new round reused persisted token: %d", runtime.Token)
	}
}

func TestDungeonReplayUsesNewTokenAndNonReplayCannotReenter(t *testing.T) {
	f := newDungeonTestFixture()
	sd, first := f.enter(t)
	f.start(t, sd)
	_, _ = f.service.monsterDead(f.player, sd, 100, 7)
	_ = f.service.end(f.player, sd)
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("first finish failed: %v", status)
	}
	if status := f.service.exit(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("exit failed: %v", status)
	}
	secondScene, second := f.enter(t)
	if first.Token == second.Token {
		t.Fatal("new run reused token")
	}
	f.start(t, secondScene)
	_, _ = f.service.monsterDead(f.player, secondScene, 100, 7)
	_ = f.service.end(f.player, secondScene)
	if _, status := f.service.finish(f.player, secondScene); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("replay finish failed: %v", status)
	}
	if len(f.grants) != 3 { // first run: clear + star; replay: clear only
		t.Fatalf("replay grants = %d", len(f.grants))
	}

	f.config.CanReplay = false
	if status := f.service.exit(f.player, secondScene); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("second exit failed: %v", status)
	}
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_DungeonCantReward {
		t.Fatalf("non-replay reenter status = %v", status)
	}
}

func TestDungeonExitInvalidatesRuntimeOperations(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	_, _ = f.service.monsterDead(f.player, sd, 100, 7)
	if status := f.service.exit(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("exit failed: %v", status)
	}
	if sd.Runtime != nil {
		t.Fatal("exit did not clear runtime")
	}
	if status := f.service.end(f.player, sd); status != proto.StatusCode_StatusCode_PlayerNotInDungeon {
		t.Fatalf("end after exit status = %v", status)
	}
	if _, status := f.service.monsterDead(f.player, sd, 100, 7); status != proto.StatusCode_StatusCode_PlayerNotInDungeon {
		t.Fatalf("death after exit status = %v", status)
	}
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_PlayerNotInDungeon {
		t.Fatalf("finish after exit status = %v", status)
	}
	if _, status := f.service.pickup(f.player, sd, 1, -1); status != proto.StatusCode_StatusCode_PlayerNotInDungeon {
		t.Fatalf("pickup after exit status = %v", status)
	}
}

func TestDungeonEntryEnforcesLevelPrerequisiteAndDailyLimit(t *testing.T) {
	f := newDungeonTestFixture()
	f.config.NeedPlayerLevel = 11
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_PlayerLevelLimit {
		t.Fatalf("level status = %v", status)
	}
	f.config.NeedPlayerLevel = 1
	f.config.PreDungeonID = 2
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_DungeonUnlock {
		t.Fatalf("prerequisite status = %v", status)
	}
	previous := f.player.GetDungeonModel().GetDungeonInfo(2)
	previous.FinishTimes = 1
	f.config.CheckQuestID = 50
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_DungeonUnlock {
		t.Fatalf("quest prerequisite status = %v", status)
	}
	f.player.GetQuestModel().GetQuestMap()[50] = &proto.Quest{QuestId: 50, Status: proto.QuestStatus_QuestStatus_Completed}
	f.config.CheckQuestID = 0

	first, firstRuntime := f.enter(t)
	if status := f.service.exit(f.player, first); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("first exit failed: %v", status)
	}
	second, secondRuntime := f.enter(t)
	if firstRuntime.Token == secondRuntime.Token {
		t.Fatal("daily runs reused token")
	}
	if status := f.service.exit(f.player, second); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("second exit failed: %v", status)
	}
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_ExploreNumLimit {
		t.Fatalf("daily limit status = %v", status)
	}
	f.now = f.now.Add(24 * time.Hour)
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("daily reset status = %v", status)
	}
}

func TestDungeonFinishSaveFailureRollsBackClaim(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)
	_, _ = f.service.monsterDead(f.player, sd, 100, 7)
	_ = f.service.end(f.player, sd)
	saveCalled := false
	f.service.save = func(player *model.Player) error {
		saveCalled = true
		info := player.GetDungeonModel().GetDungeonInfo(1)
		if f.grantTotal(100) != 1 || f.grantTotal(110) != 1 || info.RewardClaimedToken != runtime.Token {
			t.Fatalf("save snapshot did not contain both rewards and claim: info=%+v", info)
		}
		return errors.New("save failed")
	}
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_PlayerNotFound {
		t.Fatalf("save failure status = %v", status)
	}
	if !saveCalled {
		t.Fatal("finish did not attempt to persist the complete settlement")
	}
	info := f.player.GetDungeonModel().GetDungeonInfo(1)
	if info.FinishTimes != 0 || info.RewardClaimedToken != 0 || f.grantTotal(100) != 0 || f.grantTotal(110) != 0 || runtime.State != model.DungeonRuntimeFinished {
		t.Fatalf("failed save partially claimed: info=%+v state=%v", info, runtime.State)
	}
	f.service.save = nil
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("retry finish status = %v", status)
	}
	if info.FinishTimes != 1 || f.grantTotal(100) != 1 || f.grantTotal(110) != 1 {
		t.Fatalf("successful retry info=%+v", info)
	}
}

func TestDungeonMissingConfigurationRejectsWithoutProgress(t *testing.T) {
	f := newDungeonTestFixture()
	f.service.loadConfig = func(uint32) (*dungeonConfig, error) { return nil, errors.New("missing") }
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_CfgNotFound {
		t.Fatalf("missing config status = %v", status)
	}
	f = newDungeonTestFixture()
	f.service.loadReward = func(uint32) ([]dungeonReward, error) { return nil, errors.New("missing") }
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_CfgNotFound {
		t.Fatalf("missing reward status = %v", status)
	}
	if info := f.player.GetDungeonModel().GetDungeonInfo(1); info.EnterTimes != 0 || info.LastRunToken != 0 {
		t.Fatalf("failed enter mutated progress: %+v", info)
	}
}

func TestDungeonEncounterFinishRequiresServerDeath(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	if status := f.service.updateEncounter(f.player, sd, 7, proto.BattleState_BattleState_Finish); status != proto.StatusCode_StatusCode_DungeonCheat {
		t.Fatalf("forged encounter finish status = %v", status)
	}
	if _, status := f.service.monsterDead(f.player, sd, 100, 7); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("monster death failed: %v", status)
	}
	state, status := f.service.encounterState(f.player, sd, 7)
	if status != proto.StatusCode_StatusCode_Ok || state != proto.BattleState_BattleState_Finish {
		t.Fatalf("authoritative encounter state=%v status=%v", state, status)
	}
	if status := f.service.updateEncounter(f.player, sd, 7, proto.BattleState_BattleState_Reset); status != proto.StatusCode_StatusCode_DungeonCheat {
		t.Fatalf("dead encounter reset status = %v", status)
	}
}

func TestDungeonDropCanOnlyBePickedOnce(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	drop, status := f.service.monsterDead(f.player, sd, 100, 7)
	if status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("monster death failed: %v", status)
	}
	items, status := f.service.pickup(f.player, sd, drop.Index, -1)
	if status != proto.StatusCode_StatusCode_Ok || len(items) != 1 {
		t.Fatalf("pickup items=%v status=%v", items, status)
	}
	if len(f.grants) != 1 {
		t.Fatalf("inventory grants = %d", len(f.grants))
	}
	if duplicate, status := f.service.pickup(f.player, sd, drop.Index, -1); status != proto.StatusCode_StatusCode_GachaAlreadyFullPick || len(duplicate) != 0 {
		t.Fatalf("duplicate pickup items=%v status=%v", duplicate, status)
	}
	if len(f.grants) != 1 {
		t.Fatalf("duplicate pickup changed inventory grants: %d", len(f.grants))
	}
}

func TestSceneTempPackPickupRemovesOnlySelectedItem(t *testing.T) {
	pack := model.NewSceneTempPack()
	drop := pack.NewDrop([]model.EBagItemTag{
		&model.ItemBaseInfo{ItemId: 1, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Material},
		&model.ItemBaseInfo{ItemId: 2, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Material},
		&model.ItemBaseInfo{ItemId: 3, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Material},
	})
	items := pack.Pickup(drop.Index, 2)
	if len(items) != 1 || items[0].MainItem.ItemId != 2 {
		t.Fatalf("selected pickup = %v", items)
	}
	if len(drop.Items) != 2 || drop.Items[0].(*model.ItemBaseInfo).ItemId != 1 || drop.Items[1].(*model.ItemBaseInfo).ItemId != 3 {
		t.Fatalf("remaining drop = %+v", drop.Items)
	}
	pack.Pickup(drop.Index, -1)
	if pack.GetTempAreaPack(drop.Index) != nil {
		t.Fatal("fully picked drop remained in temp pack")
	}
}

func TestDungeonTreasureHelpersValidateIndexAndOneTimeClaim(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	if _, status := f.service.openTreasure(f.player, sd, 99); status != proto.StatusCode_StatusCode_TreasureBoxNotExist {
		t.Fatalf("invalid treasure index status = %v", status)
	}
	preview, status := f.service.openTreasure(f.player, sd, 3)
	if status != proto.StatusCode_StatusCode_Ok || len(preview) != 1 {
		t.Fatalf("open treasure items=%v status=%v", preview, status)
	}
	again, status := f.service.openTreasure(f.player, sd, 3)
	if status != proto.StatusCode_StatusCode_Ok || len(again) != 1 {
		t.Fatalf("repeat open items=%v status=%v", again, status)
	}
	items, status := f.service.pickupTreasure(f.player, sd, 3, -1)
	if status != proto.StatusCode_StatusCode_Ok || len(items) != 1 {
		t.Fatalf("treasure pickup items=%v status=%v", items, status)
	}
	if _, status := f.service.pickupTreasure(f.player, sd, 3, -1); status != proto.StatusCode_StatusCode_AlreadyGotReward {
		t.Fatalf("duplicate treasure pickup status = %v", status)
	}
	if len(f.grants) != 1 {
		t.Fatalf("treasure inventory grants = %d", len(f.grants))
	}
}

func TestDungeonRejectedEntryDoesNotResetDailyProgress(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	if status := f.service.exit(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("exit failed: %v", status)
	}
	info := f.player.GetDungeonModel().GetDungeonInfo(1)
	previousDay := info.LastEnterDay
	f.now = f.now.Add(24 * time.Hour)
	f.config.NeedPlayerLevel = 11
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_PlayerLevelLimit {
		t.Fatalf("rejected entry status = %v", status)
	}
	if info.LastEnterDay != previousDay || info.DailyEnterCount != 1 || info.EnterTimes != 1 {
		t.Fatalf("rejected entry mutated counters: %+v", info)
	}
}

func TestDungeonMissingTreasureRewardRejectsOpen(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	f.service.loadTreasureReward = func(uint32) ([]dungeonReward, error) {
		return nil, errors.New("missing treasure reward")
	}
	if items, status := f.service.openTreasure(f.player, sd, 3); status != proto.StatusCode_StatusCode_CfgNotFound || len(items) != 0 {
		t.Fatalf("missing treasure reward items=%v status=%v", items, status)
	}
	box := sd.Info.GetTreasureBoxes()[3]
	if box.State != proto.TreasureBoxState_TreasureBoxState_Close || sd.Runtime.Treasures[3] != nil || len(f.grants) != 0 {
		t.Fatalf("failed open mutated treasure state: box=%+v runtime=%+v", box, sd.Runtime.Treasures)
	}
}

func TestDungeonFinishRejectsPersistedClaimToken(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)
	_, _ = f.service.monsterDead(f.player, sd, 100, 7)
	_ = f.service.end(f.player, sd)
	f.player.GetDungeonModel().GetDungeonInfo(1).RewardClaimedToken = runtime.Token
	if items, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_RepeatGetReward || len(items) != 0 {
		t.Fatalf("persisted claim was settled twice: items=%v status=%v", items, status)
	}
	if len(f.grants) != 0 {
		t.Fatalf("duplicate persisted claim granted rewards: %v", f.grants)
	}
}

func TestDungeonEncounterRejectsUnknownStateAndRemainsReadableAfterEnd(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	if status := f.service.updateEncounter(f.player, sd, 7, proto.BattleState(99)); status != proto.StatusCode_StatusCode_DungeonCheat {
		t.Fatalf("unknown encounter state status = %v", status)
	}
	_, _ = f.service.monsterDead(f.player, sd, 100, 7)
	if status := f.service.end(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("end failed: %v", status)
	}
	state, status := f.service.encounterState(f.player, sd, 7)
	if status != proto.StatusCode_StatusCode_Ok || state != proto.BattleState_BattleState_Finish {
		t.Fatalf("encounter state after end=%v status=%v", state, status)
	}
}

func TestDungeonPickupFailureRestoresDrop(t *testing.T) {
	f := newDungeonTestFixture()
	sd, _ := f.enter(t)
	f.start(t, sd)
	drop, status := f.service.monsterDead(f.player, sd, 100, 7)
	if status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("monster death failed: %v", status)
	}
	addReward := f.service.addReward
	f.service.addReward = func(*model.Player, dungeonReward) *proto.ItemDetail { return nil }
	if items, status := f.service.pickup(f.player, sd, drop.Index, -1); status != proto.StatusCode_StatusCode_CfgNotFound || len(items) != 0 {
		t.Fatalf("failed pickup items=%v status=%v", items, status)
	}
	if sd.GetTempPack().GetTempAreaPack(drop.Index) != drop || len(drop.Items) != 1 {
		t.Fatal("failed pickup consumed the authoritative drop")
	}
	f.service.addReward = addReward
	if items, status := f.service.pickup(f.player, sd, drop.Index, -1); status != proto.StatusCode_StatusCode_Ok || len(items) != 1 {
		t.Fatalf("pickup retry items=%v status=%v", items, status)
	}
}

func TestDungeonRuntimeRejectsForeignToken(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)
	f.player.GetDungeonModel().GetDungeonInfo(1).LastRunToken = runtime.Token + 1
	if drop, status := f.service.monsterDead(f.player, sd, 100, 7); status != proto.StatusCode_StatusCode_PlayerNotInDungeon || drop != nil {
		t.Fatalf("foreign runtime token accepted: drop=%v status=%v", drop, status)
	}
}

func TestDungeonDisconnectCleanupIsIdempotent(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)
	drop, _ := f.service.monsterDead(f.player, sd, 100, 7)
	pack := sd.GetTempPack()
	game := &Game{}
	game.getWordInfo().allScenePlayer.Store(f.player.UserId, &ScenePlayer{Player: f.player, CurScene: sd})
	game.cleanupDungeonPlayer(f.player)
	game.cleanupDungeonPlayer(f.player)
	info := f.player.GetDungeonModel().GetDungeonInfo(1)
	if sd.Runtime != nil || runtime.Token != 0 || info.ExitTimes != 1 {
		t.Fatalf("disconnect cleanup state=%v token=%d exits=%d", sd.Runtime, runtime.Token, info.ExitTimes)
	}
	if pack.GetTempAreaPack(drop.Index) != nil || sd.GetTempPack() == pack {
		t.Fatal("disconnect cleanup retained dungeon drops")
	}
}

func TestSceneTempPackRejectsInvalidPickupAndNilDrops(t *testing.T) {
	pack := model.NewSceneTempPack()
	if drop := pack.NewDrop([]model.EBagItemTag{nil}); drop != nil {
		t.Fatalf("nil drop was created: %+v", drop)
	}
	drop := pack.NewDrop([]model.EBagItemTag{
		&model.ItemBaseInfo{ItemId: 1, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Material},
	})
	if items := pack.Pickup(drop.Index, -2); items != nil || pack.GetTempAreaPack(drop.Index) != drop {
		t.Fatalf("invalid pickup consumed drop: items=%v", items)
	}
	if items := pack.Pickup(drop.Index, 0); items != nil || pack.GetTempAreaPack(drop.Index) != drop {
		t.Fatalf("zero-based pickup consumed drop: items=%v", items)
	}
}

func TestDungeonDataUsesPersistedCountersAndMigratesTreasureKeys(t *testing.T) {
	info := &model.DungeonInfo{
		DungeonID:               1,
		EnterTimes:              2,
		ExitTimes:               1,
		FinishTimes:             1,
		LastFinishTime:          12,
		Stars:                   3,
		StarRewardClaimed:       0b111,
		FirstClearRewardClaimed: true,
		TreasureBoxes: map[uint32]*model.TreasureBox{
			30: {Index: 3, BoxId: 30, State: proto.TreasureBoxState_TreasureBoxState_Open},
		},
	}
	boxes := info.GetTreasureBoxes()
	if boxes[3] == nil || boxes[30] != nil {
		t.Fatalf("treasure keys were not migrated: %+v", boxes)
	}
	data := info.DungeonData()
	if data.EnterTimes != 2 || data.ExitTimes != 1 || data.FinishTimes != 1 || data.LastFinishTime != 12 {
		t.Fatalf("persisted counters missing from DungeonData: %+v", data)
	}
	if data.TaskFinishReward != proto.RewardStatus_RewardStatus_Reward || data.StarReward != proto.RewardStatus_RewardStatus_Reward {
		t.Fatalf("claimed reward status missing from DungeonData: %+v", data)
	}
	info.ResetTreasureBoxes()
	if boxes[3].State != proto.TreasureBoxState_TreasureBoxState_Close || boxes[3].NextRefreshTime != 0 {
		t.Fatalf("treasure reset failed: %+v", boxes[3])
	}
}

func TestDungeonEntryEnforcesConfiguredOpenWeekday(t *testing.T) {
	f := newDungeonTestFixture()
	f.config.OpenWeekdays = []int32{5}
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_ActivityClose {
		t.Fatalf("closed weekday status = %v", status)
	}
	f.config.OpenWeekdays = []int32{4}
	if _, status := f.service.enterDungeon(f.player, nil, 1, model.TeamInfo{Char1: 1}); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("open weekday status = %v", status)
	}
}

func TestDungeonFinishGrantFailureRollsBackProgress(t *testing.T) {
	f := newDungeonTestFixture()
	sd, runtime := f.enter(t)
	f.start(t, sd)
	_, _ = f.service.monsterDead(f.player, sd, 100, 7)
	_ = f.service.end(f.player, sd)
	addReward := f.service.addReward
	f.service.addReward = func(*model.Player, dungeonReward) *proto.ItemDetail { return nil }
	if items, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_CfgNotFound || len(items) != 0 {
		t.Fatalf("failed grant items=%v status=%v", items, status)
	}
	info := f.player.GetDungeonModel().GetDungeonInfo(1)
	if info.FinishTimes != 0 || info.RewardClaimedToken != 0 || f.grantTotal(100) != 0 || runtime.State != model.DungeonRuntimeFinished {
		t.Fatalf("failed grant partially settled: info=%+v state=%v", info, runtime.State)
	}
	f.service.addReward = addReward
	if _, status := f.service.finish(f.player, sd); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("finish retry status = %v", status)
	}
}
