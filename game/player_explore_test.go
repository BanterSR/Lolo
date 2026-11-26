package game

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/config"
	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

const gameExploreFixture = `{
  "explore_area":{"datas":[{"i_d":10101,"need_time":600,"quick_need_item":102,"quick_need_item_count":1,"need_scene_i_d":1,"need_player_level":2,"need_item":50,"need_item_count":1,"explore_area_item":[{"rewar_item_i_d":101},{"rewar_item_i_d":109}]}]},
  "explore_activity":{"datas":[{"i_d":901,"start_time":"2029-01-01 00:00:00","end_time":"2031-01-01 00:00:00","need_time":60,"max_times":2,"quick_need_item":102,"quick_need_item_count":1,"special_reward_item_i_d":102,"explore_activity_item":[{"rewar_item_i_d":101}]}]},
  "explore_interval":{"datas":[{"i_d":101,"retention_time":172800,"time":14400,"need_item_i_d":10,"captain_count":20,"member_count":10,"quick_need_item":102,"quick_need_item_count":10}]}
}`

func writeGameExploreFixture(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	excel := filepath.Join(root, "Excel")
	if err := os.MkdirAll(excel, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"Explore.json":      gameExploreFixture,
		"GlobalConfig.json": `{"config":{"datas":[{"key":"Explore_ProcessIng_MaxCount","values":["1"]}]}}`,
	} {
		if err := os.WriteFile(filepath.Join(excel, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := config.CONF
	config.CONF = &config.Config{Resources: &config.Resources{ResourcePath: root}}
	t.Cleanup(func() { config.CONF = old })
}

func newExploreTestPlayer() *model.Player {
	player := &model.Player{UserId: 42}
	player.GetCharacterModel().CharacterMap[1] = &model.CharacterInfo{CharacterId: 1}
	player.GetCharacterModel().CharacterMap[2] = &model.CharacterInfo{CharacterId: 2}
	player.GetItemModel().ItemBaseInfo = map[uint32]*model.ItemBaseInfo{
		102: {ItemId: 102, Num: 5, ItemType: proto.EBagItemTag_EBagItemTag_Currency, PackType: proto.PackType_PackType_Inventory},
		50:  {ItemId: 50, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Quest, PackType: proto.PackType_PackType_Inventory},
	}
	return player
}

func exploreTestDeps() (exploreDeps, map[uint32]int64, *int) {
	granted := make(map[uint32]int64)
	consumptions := 0
	deps := exploreDeps{
		loadConfig: model.LoadExploreConfig,
		itemExists: func(uint32) bool { return true },
		hasItem: func(s *model.Player, id uint32, count int64) bool {
			item := s.GetItemModel().GetItemBaseInfo(id)
			return item != nil && item.Num >= count
		},
		consumeItem: func(s *model.Player, id uint32, count int64) (*proto.ItemDetail, bool) {
			item := s.GetItemModel().GetItemBaseInfo(id)
			if item == nil || item.Num < count {
				return nil, false
			}
			item.Num -= count
			consumptions++
			return item.ItemDetail(), true
		},
		grantItem: func(_ *model.Player, id uint32, count int64) *proto.ItemDetail {
			granted[id] += count
			return &proto.ItemDetail{MainItem: &proto.ItemInfo{
				ItemId: id, ItemTag: proto.EBagItemTag_EBagItemTag_Material,
				Item: &proto.ItemInfo_BaseItem{BaseItem: &proto.BaseItem{ItemId: id, Num: count}},
			}, PackType: proto.PackType_PackType_Inventory}
		},
	}
	return deps, granted, &consumptions
}

func TestExploreCompletionSettlesOnlyOnce(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps, granted, _ := exploreTestDeps()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	team := [3]uint32{1, 2, 0}
	started, status := startExplore(player, deps, 10101, false, 1, 2, team, now)
	if status != proto.StatusCode_StatusCode_Ok || started.Times != 0 {
		t.Fatalf("start status = %v times = %d", status, started.Times)
	}
	if _, _, status := quickFinishExplore(player, deps, 10101, false); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("quick status = %v", status)
	}
	_, items, status := claimExploreReward(player, deps, 10101, false, now)
	if status != proto.StatusCode_StatusCode_Ok || len(items) != 2 {
		t.Fatalf("claim status=%v items=%d", status, len(items))
	}
	_, again, status := claimExploreReward(player, deps, 10101, false, now)
	if status != proto.StatusCode_StatusCode_ExploreNotFinish || len(again) != 0 {
		t.Fatalf("duplicate claim status=%v items=%d", status, len(again))
	}
	if granted[101] != 1 || granted[109] != 1 {
		t.Fatalf("completion rewards = %v, want one grant each", granted)
	}
	_, items, status = claimExploreCollectReward(player, deps, 10101, false)
	if status != proto.StatusCode_StatusCode_Ok || len(items) != 2 {
		t.Fatalf("collect status=%v items=%d", status, len(items))
	}
	_, again, status = claimExploreCollectReward(player, deps, 10101, false)
	if status != proto.StatusCode_StatusCode_RepeatGetReward || len(again) != 0 {
		t.Fatalf("duplicate collect status=%v items=%d", status, len(again))
	}
}

func TestExploreCancelPreventsAllRewards(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps, granted, _ := exploreTestDeps()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1, 2, 0}, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, status := cancelExplore(player, 10101, false); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, _, status := claimExploreReward(player, deps, 10101, false, now.Add(time.Hour)); status != proto.StatusCode_StatusCode_ExploreNotFinish {
		t.Fatalf("reward after cancel status = %v", status)
	}
	if _, _, status := claimExploreCollectReward(player, deps, 10101, false); status != proto.StatusCode_StatusCode_ExploreCollectNotFinish {
		t.Fatalf("collect after cancel status = %v", status)
	}
	if len(granted) != 0 {
		t.Fatalf("cancel granted rewards: %v", granted)
	}
}

func TestExploreQuickFinishChargesOnlyOnce(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps, _, consumptions := exploreTestDeps()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1, 0, 0}, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, _, status := quickFinishExplore(player, deps, 10101, false); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, item, status := quickFinishExplore(player, deps, 10101, false); status != proto.StatusCode_StatusCode_Ok || item != nil {
		t.Fatalf("repeat quick status=%v item=%v", status, item)
	}
	if *consumptions != 1 || player.GetItemModel().GetItemBaseInfo(102).Num != 4 {
		t.Fatalf("consumptions=%d balance=%d", *consumptions, player.GetItemModel().GetItemBaseInfo(102).Num)
	}

	other := newExploreTestPlayer()
	other.GetItemModel().GetItemBaseInfo(102).Num = 0
	if _, status := startExplore(other, deps, 10101, false, 1, 2, [3]uint32{1, 0, 0}, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, _, status := quickFinishExplore(other, deps, 10101, false); status != proto.StatusCode_StatusCode_ItemNotEnough {
		t.Fatalf("insufficient quick item status = %v", status)
	}
	state, _ := other.LoadExploreModel()
	if state.Areas[10101].QuickFinish {
		t.Fatal("failed quick finish mutated state")
	}
}

func TestExploreStartValidatesSceneLevelTeamAndItems(t *testing.T) {
	writeGameExploreFixture(t)
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	tests := []struct {
		name   string
		scene  uint32
		level  uint32
		team   [3]uint32
		mutate func(*model.Player, exploreDeps) exploreDeps
		want   proto.StatusCode
	}{
		{name: "wrong scene", scene: 2, level: 2, team: [3]uint32{1}, want: proto.StatusCode_StatusCode_SceneNotUnlock},
		{name: "low level", scene: 1, level: 1, team: [3]uint32{1}, want: proto.StatusCode_StatusCode_PlayerLevelLimit},
		{name: "missing captain", scene: 1, level: 2, team: [3]uint32{}, want: proto.StatusCode_StatusCode_BadReq},
		{name: "duplicate member", scene: 1, level: 2, team: [3]uint32{1, 1}, want: proto.StatusCode_StatusCode_BadReq},
		{name: "unknown member", scene: 1, level: 2, team: [3]uint32{1, 99}, want: proto.StatusCode_StatusCode_BadReq},
		{name: "missing entry item", scene: 1, level: 2, team: [3]uint32{1}, mutate: func(p *model.Player, d exploreDeps) exploreDeps {
			p.GetItemModel().GetItemBaseInfo(50).Num = 0
			return d
		}, want: proto.StatusCode_StatusCode_ItemNotEnough},
		{name: "missing reward config", scene: 1, level: 2, team: [3]uint32{1}, mutate: func(_ *model.Player, d exploreDeps) exploreDeps {
			d.itemExists = func(id uint32) bool { return id != 109 }
			return d
		}, want: proto.StatusCode_StatusCode_CfgNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			player := newExploreTestPlayer()
			deps, _, _ := exploreTestDeps()
			if test.mutate != nil {
				deps = test.mutate(player, deps)
			}
			if _, status := startExplore(player, deps, 10101, false, test.scene, test.level, test.team, now); status != test.want {
				t.Fatalf("status = %v, want %v", status, test.want)
			}
		})
	}
}

func TestExploreReconnectRestoresActiveState(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps, granted, _ := exploreTestDeps()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	blob, err := sonic.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	restored := new(model.Player)
	if err := sonic.Unmarshal(blob, restored); err != nil {
		t.Fatal(err)
	}
	if _, _, status := quickFinishExplore(restored, deps, 10101, false); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("quick after reconnect status = %v", status)
	}
	if _, _, status := claimExploreReward(restored, deps, 10101, false, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("claim after reconnect status = %v", status)
	}
	if granted[101] != 1 || granted[109] != 1 {
		t.Fatalf("reconnect rewards = %v", granted)
	}
}

func TestExploreLimitsConcurrentAndActivityAttempts(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps, _, _ := exploreTestDeps()
	now := time.Date(2030, 5, 1, 12, 0, 0, 0, time.Local)
	if _, status := startExplore(player, deps, 901, true, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_ExploreInProgress {
		t.Fatalf("concurrent start status = %v", status)
	}
	if _, _, status := quickFinishExplore(player, deps, 901, true); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, _, status := claimExploreReward(player, deps, 901, true, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, status := startExplore(player, deps, 901, true, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, status := cancelExplore(player, 901, true); status != proto.StatusCode_StatusCode_Ok {
		t.Fatal(status)
	}
	if _, status := startExplore(player, deps, 901, true, 1, 2, [3]uint32{1}, now); status != proto.StatusCode_StatusCode_ExploreMaxTimes {
		t.Fatalf("third activity attempt status = %v", status)
	}
}

func TestExploreMissingConfigurationFailsClosed(t *testing.T) {
	writeGameExploreFixture(t)
	player := newExploreTestPlayer()
	deps, _, _ := exploreTestDeps()
	deps.loadConfig = func() (*model.ExploreConfig, error) { return nil, errors.New("missing") }
	if _, status := startExplore(player, deps, 10101, false, 1, 2, [3]uint32{1}, time.Now()); status != proto.StatusCode_StatusCode_CfgNotFound {
		t.Fatalf("status = %v", status)
	}
}
