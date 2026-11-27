package game

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/config"
	"gucooing/lolo/protocol/proto"
)

func TestGatherSaveFailureRollsBackRewardAndCount(t *testing.T) {
	oldSave := collectSavePlayer
	collectSavePlayer = func(*model.Player) error { return errors.New("disk full") }
	t.Cleanup(func() { collectSavePlayer = oldSave })
	player := newAuthorityPlayer()
	player.Online = true
	prepareWoodGather(player)
	g := newGatherGame(player, 1)
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() == proto.StatusCode_StatusCode_OK {
		t.Fatal("save failure reported success")
	}
	if player.GetItemModel().GetItemBaseInfo(5001) != nil || player.Scene.SceneMap[1].GatherLimits[1].GatherNum != 0 {
		t.Fatal("save failure retained reward or consumed gather count")
	}
	collectSavePlayer = func(*model.Player) error { return nil }
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("retry after save recovery: %v", rsp.GetStatus())
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001).Num; got != 1 {
		t.Fatalf("retry reward = %d", got)
	}
}

func inventoryExploreDeps() exploreDeps {
	deps, _, _ := exploreTestDeps()
	deps.mu = new(sync.Mutex)
	deps.grantItem = func(s *model.Player, id uint32, count int64) *proto.ItemDetail {
		item := s.GetItemModel().GetItemBaseInfo(id)
		if item == nil {
			item = &model.ItemBaseInfo{ItemId: id, ItemType: proto.EBagItemTag_EBagItemTag_Material}
			s.GetItemModel().GetItemBaseMap()[id] = item
		}
		item.Num += count
		return item.ItemDetail()
	}
	return deps
}

func TestExploreSaveFailureRollsBackChargeStateAndRewards(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps := inventoryExploreDeps()
	deps.save = func(*model.Player) error { return errors.New("disk full") }
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_PLAYER_NOT_FOUND {
		t.Fatalf("start save failure status = %v", status)
	}
	state, _ := player.LoadExploreModel()
	if state.Areas[10101] != nil || player.GetItemModel().GetItemBaseInfo(50).Num != 1 {
		t.Fatal("failed start persisted a round or consumed entry item")
	}
	deps.save = func(*model.Player) error { return nil }
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_OK {
		t.Fatal(status)
	}
	if _, _, status := quickFinishExplore(player, deps, 10101, false); status != proto.StatusCode_StatusCode_OK {
		t.Fatal(status)
	}
	deps.save = func(*model.Player) error { return errors.New("disk full") }
	if _, items, status := claimExploreReward(player, deps, 10101, false, now); status != proto.StatusCode_StatusCode_PLAYER_NOT_FOUND || len(items) != 0 {
		t.Fatalf("claim save failure status=%v items=%d", status, len(items))
	}
	state, _ = player.LoadExploreModel()
	if state.Areas[10101].StartTime == 0 || player.GetItemModel().GetItemBaseInfo(101) != nil {
		t.Fatal("failed claim consumed round or retained rewards")
	}
	deps.save = func(*model.Player) error { return nil }
	if _, items, status := claimExploreReward(player, deps, 10101, false, now); status != proto.StatusCode_StatusCode_OK || len(items) != 2 {
		t.Fatalf("claim retry status=%v items=%d", status, len(items))
	}
}

func TestExplorePartialGrantAndConcurrentClaimsAreAtomic(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps := inventoryExploreDeps()
	deps.grantItem = func(s *model.Player, id uint32, count int64) *proto.ItemDetail {
		if id == 109 {
			return nil
		}
		item := &model.ItemBaseInfo{ItemId: id, Num: count, ItemType: proto.EBagItemTag_EBagItemTag_Material}
		s.GetItemModel().GetItemBaseMap()[id] = item
		return item.ItemDetail()
	}
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_OK {
		t.Fatal(status)
	}
	if _, _, status := quickFinishExplore(player, deps, 10101, false); status != proto.StatusCode_StatusCode_OK {
		t.Fatal(status)
	}
	if _, _, status := claimExploreReward(player, deps, 10101, false, now); status != proto.StatusCode_StatusCode_Item_Get_Fail {
		t.Fatalf("partial grant status = %v", status)
	}
	if player.GetItemModel().GetItemBaseInfo(101) != nil {
		t.Fatal("partial grant was not rolled back")
	}

	deps = inventoryExploreDeps()
	const requests = 12
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, status := claimExploreReward(player, deps, 10101, false, now); status == proto.StatusCode_StatusCode_OK {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || player.GetItemModel().GetItemBaseInfo(101).Num != 1 || player.GetItemModel().GetItemBaseInfo(109).Num != 1 {
		t.Fatalf("concurrent claims successes=%d rewards=%v", successes.Load(), player.GetItemModel().ItemBaseInfo)
	}
}

func TestExploreCollectRewardIsPerCompletedRound(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps := inventoryExploreDeps()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	for round := 1; round <= 2; round++ {
		if round > 1 {
			player.GetItemModel().GetItemBaseInfo(50).Num = 1
		}
		if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_OK {
			t.Fatal(status)
		}
		if _, _, status := quickFinishExplore(player, deps, 10101, false); status != proto.StatusCode_StatusCode_OK {
			t.Fatal(status)
		}
		if _, _, status := claimExploreReward(player, deps, 10101, false, now); status != proto.StatusCode_StatusCode_OK {
			t.Fatal(status)
		}
		if _, _, status := claimExploreCollectReward(player, deps, 10101, false); status != proto.StatusCode_StatusCode_OK {
			t.Fatalf("round %d collect: %v", round, status)
		}
		if _, _, status := claimExploreCollectReward(player, deps, 10101, false); status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
			t.Fatalf("round %d duplicate collect: %v", round, status)
		}
	}
	if got := player.GetItemModel().GetItemBaseInfo(101).Num; got != 4 {
		t.Fatalf("two rounds completion+collect reward = %d, want 4", got)
	}
}

func TestTempPackStoreSaveFailureRestoresOwnership(t *testing.T) {
	oldSave := tempPackSavePlayer
	t.Cleanup(func() { tempPackSavePlayer = oldSave })
	player := tempTestPlayer()
	player.Online = true
	player.Conn = new(authorityConn)
	pack := tempTestPack()
	base := tempTestBase(10, 3)
	pack.TempPack = []model.EBagItemTag{base}
	g := newGatherGame(player, 1)
	g.getWordInfo().getScenePlayer(player).CurScene.(*model.ScenePlayerInfo).TempPack = pack
	tempPackSavePlayer = func(*model.Player) error { return errors.New("disk full") }
	g.TempPackItemStore(player, &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 77}})
	rsp := player.Conn.(*authorityConn).last(77).(*proto.TempPackItemStoreRsp)
	if rsp.GetStatus() == proto.StatusCode_StatusCode_OK || len(pack.TempPack) != 1 || pack.TempPack[0] != base || player.GetItemModel().GetItemBaseInfo(10) != nil {
		t.Fatal("failed store duplicated or lost the temporary item")
	}
	tempPackSavePlayer = func(*model.Player) error { return nil }
	g.TempPackItemStore(player, &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 78}})
	if got := player.GetItemModel().GetItemBaseInfo(10); got == nil || got.Num != 3 || len(pack.TempPack) != 0 {
		t.Fatal("store retry did not transfer ownership exactly once")
	}
}

func TestOrdinaryMonsterDeadRejectsClientInventoryAndDuplicates(t *testing.T) {
	oldLevel := ordinaryMonsterPlayerLevel
	ordinaryMonsterPlayerLevel = func(*model.Player) (uint32, bool) { return 1, true }
	t.Cleanup(func() { ordinaryMonsterPlayerLevel = oldLevel })
	player := newAuthorityPlayer()
	scene := model.NewScenePlayerInfo(player, new(uint32), player.GetTeamModel().GetTeamInfo(), &proto.Vector3{}, &proto.Vector3{})
	sceneConf := gdconf.GetSceneInfo(1)
	oldMonsters := sceneConf.Info.MonsterInfos
	sceneConf.Info.MonsterInfos = []*config.DungeonMonsterInfo{{Index: 9, UnitID: 999999}}
	t.Cleanup(func() { sceneConf.Info.MonsterInfos = oldMonsters })
	if _, status := ordinaryMonsterDead(player, scene, 1, 999999, 8); status != proto.StatusCode_StatusCode_DungeonCheat {
		t.Fatalf("forged monster index status = %v", status)
	}
	if _, status := ordinaryMonsterDead(player, scene, 1, 999998, 9); status != proto.StatusCode_StatusCode_DungeonCheat {
		t.Fatalf("forged monster id status = %v", status)
	}
	// Missing monster/reward configuration must fail closed and not mark it dead.
	if _, status := ordinaryMonsterDead(player, scene, 1, 999999, 9); status != proto.StatusCode_StatusCode_CFG_NOT_FOUND {
		t.Fatalf("missing drop configuration status = %v", status)
	}
	if len(scene.DeadMonsters) != 0 || len(scene.GetTempPack().TempAreaPack) != 0 {
		t.Fatal("rejected death mutated authoritative state")
	}
}

func TestTempPackArchiveSurvivesSceneRecreationWithoutDuplication(t *testing.T) {
	player := newAuthorityPlayer()
	pack := tempTestPack()
	item := tempTestBase(10, 2)
	drop := pack.AddDropItems([]model.EBagItemTag{item})
	if _, err := pickupTempPackItems(pack, drop.Index, -1); err != nil {
		t.Fatal(err)
	}
	if err := player.SaveTempPackArchive(pack); err != nil {
		t.Fatal(err)
	}
	restored, err := player.LoadTempPackArchive()
	if err != nil || len(restored.TempPack) != 1 || len(restored.TempAreaPack) != 0 {
		t.Fatalf("restored pack items=%d drops=%d err=%v", len(restored.TempPack), len(restored.TempAreaPack), err)
	}
	if restored.TempPack[0].ItemDetail().MainItem.GetBaseItem().Num != 2 {
		t.Fatal("restored item quantity changed")
	}
	scene := model.NewScenePlayerInfo(player, new(uint32), player.GetTeamModel().GetTeamInfo(), &proto.Vector3{}, &proto.Vector3{})
	if len(scene.TempPack.TempPack) != 1 {
		t.Fatal("new scene did not restore the player's temporary pack")
	}
}

func TestTempPackRejectsCrossPlayerPickup(t *testing.T) {
	player := tempTestPlayer()
	other := &model.Player{UserId: 8}
	pack := tempTestPack()
	drop := pack.AddDropItems([]model.EBagItemTag{tempTestBase(10, 1)})
	g := newGatherGame(player, 1)
	sp := g.getWordInfo().getScenePlayer(player)
	sp.Player = other
	sp.CurScene.(*model.ScenePlayerInfo).TempPack = pack
	status, items := g.pickupTempPack(player, &proto.PickupReq{DropItemIndex: drop.Index, PickIndex: -1})
	if status == proto.StatusCode_StatusCode_OK || len(items) != 0 || len(pack.TempPack) != 0 || pack.GetTempAreaPack(drop.Index) == nil {
		t.Fatal("cross-player pickup accessed another player's drop")
	}
}
