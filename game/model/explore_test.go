package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	appconfig "gucooing/lolo/config"
)

func writeExploreFixture(t *testing.T, explore string) string {
	t.Helper()
	root := t.TempDir()
	excel := filepath.Join(root, "Excel")
	if err := os.MkdirAll(excel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(excel, "Explore.json"), []byte(explore), 0o600); err != nil {
		t.Fatal(err)
	}
	global := `{"config":{"datas":[{"key":"Explore_ProcessIng_MaxCount","values":["1"]}]}}`
	if err := os.WriteFile(filepath.Join(excel, "GlobalConfig.json"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	old := appconfig.CONF
	appconfig.CONF = &appconfig.Config{Resources: &appconfig.Resources{ResourcePath: root}}
	t.Cleanup(func() { appconfig.CONF = old })
	return root
}

const validExploreFixture = `{
  "explore_area":{"datas":[{"i_d":10101,"need_time":600,"quick_need_item":102,"quick_need_item_count":1,"need_scene_i_d":1,"need_player_level":1,"explore_area_item":[{"rewar_item_i_d":101},{"rewar_item_i_d":109}]}]},
  "explore_activity":{"datas":[{"i_d":901,"start_time":"2029-01-01 00:00:00","end_time":"2031-01-01 00:00:00","need_time":60,"max_times":2,"quick_need_item":102,"quick_need_item_count":1,"special_reward_item_i_d":102,"explore_activity_item":[{"rewar_item_i_d":101}]}]},
  "explore_interval":{"datas":[{"i_d":101,"retention_time":172800,"time":14400,"need_item_i_d":10,"captain_count":20,"member_count":10,"quick_need_item":102,"quick_need_item_count":10}]}
}`

func TestLoadExploreConfigReadsAllExploreTables(t *testing.T) {
	writeExploreFixture(t, validExploreFixture)
	cfg, err := LoadExploreConfig()
	if err != nil {
		t.Fatal(err)
	}
	area, ok := cfg.Area(10101)
	if !ok || area.NeedTime != 10*time.Minute || len(area.RewardItemIDs) != 2 || area.NeedSceneID != 1 {
		t.Fatalf("unexpected area config: %+v, ok=%v", area, ok)
	}
	activity, ok := cfg.Activity(901)
	if !ok || activity.MaxTimes != 2 || activity.SpecialItemID != 102 || activity.NeedTime != time.Minute {
		t.Fatalf("unexpected activity config: %+v, ok=%v", activity, ok)
	}
	interval, ok := cfg.Interval(101)
	if !ok || interval.Time != 4*time.Hour || interval.NeedItemID != 10 || interval.CaptainCount != 20 {
		t.Fatalf("unexpected interval config: %+v, ok=%v", interval, ok)
	}
	if cfg.MaxConcurrent != 1 {
		t.Fatalf("max concurrent = %d, want 1", cfg.MaxConcurrent)
	}
}

func TestLoadExploreConfigRejectsMissingRewards(t *testing.T) {
	fixture := strings.Replace(validExploreFixture, `"explore_area_item":[{"rewar_item_i_d":101},{"rewar_item_i_d":109}]`, `"explore_area_item":[]`, 1)
	writeExploreFixture(t, fixture)
	if _, err := LoadExploreConfig(); err == nil {
		t.Fatal("an explore row without configured rewards must fail")
	}
}

func TestExploreStateSurvivesSnapshotAndInvalidatesBrokenToken(t *testing.T) {
	player := &Player{UserId: 7}
	state := newExploreModel()
	state.Ensure(10101, false)
	active := state.Areas[10101]
	active.Times = 1
	active.StartTime = 1000
	active.EndTime = 2000
	active.Token = "server-token"
	if err := player.SaveExploreModel(state); err != nil {
		t.Fatal(err)
	}
	blob, err := sonic.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	restored := new(Player)
	if err := sonic.Unmarshal(blob, restored); err != nil {
		t.Fatal(err)
	}
	loaded, err := restored.LoadExploreModel()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Areas[10101]; got == nil || got.StartTime != 1000 || got.EndTime != 2000 || got.Token != "server-token" {
		t.Fatalf("active explore was not restored: %+v", got)
	}

	loaded.Areas[10101].Token = ""
	if err := restored.SaveExploreModel(loaded); err != nil {
		t.Fatal(err)
	}
	loaded, err = restored.LoadExploreModel()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Areas[10101]; got.StartTime != 0 || got.EndTime != 0 || got.Token != "" {
		t.Fatalf("broken active state must fail closed: %+v", got)
	}
}

func TestResolveDropRewardsUsesConfiguredWorldPoolAndCount(t *testing.T) {
	root := writeExploreFixture(t, validExploreFixture)
	reward := `{
	  "reward_pool":{"datas":[]},
	  "reward_item_pool":{"datas":[{"i_d":10,"reward_item_pool_group":[{"item_i_d":100,"item_min_count":2,"item_max_count":4,"min_level_limit":1,"max_level_limit":10}]}]},
	  "world_level_pool":{"datas":[{"i_d":99,"world_level_pool_group":[{"world_level":2,"pool_i_d":10}]}]}
	}`
	if err := os.WriteFile(filepath.Join(root, "Excel", "Reward.json"), []byte(reward), 0o600); err != nil {
		t.Fatal(err)
	}
	rewards, err := ResolveDropRewards(99, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(rewards) != 1 || rewards[0].ItemID != 100 || rewards[0].Count < 2 || rewards[0].Count > 4 {
		t.Fatalf("unexpected resolved drops: %+v", rewards)
	}
	if _, err := ResolveDropRewards(1000, 2, 5); err == nil {
		t.Fatal("missing reward pool must not fabricate a drop")
	}
}
