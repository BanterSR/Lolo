package game

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gookit/slog"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/config"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "lolo-authority-config-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	excelDir := filepath.Join(root, "Resource", "Excel")
	configDir := filepath.Join(root, "Resource", "Config")
	dataDir := filepath.Join(root, "data")
	for _, dir := range []string{excelDir, configDir, dataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	files := map[string]string{
		filepath.Join(dataDir, "Constant.json"): `{
			"DefaultCharacter":[101],"DefaultBadge":4001,"DefaultUmbrellaId":4002,
			"EquipmentPresetNum":3,"OutfitPresetNum":1,"DefaultSceneId":1,"DefaultChannelId":1
		}`,
		filepath.Join(excelDir, "GlobalConfig.json"): `{"Config":{"Datas":[{
			"ID":906,"Key":"GatherLuckyFinishBaseProbability","Values":["0","0","0","0","0","0"]
		}]}}`,
		filepath.Join(excelDir, "Gather.json"): `{
			"Gather":{"Datas":[
				{"ID":100,"GatherGroupInfo":[{"Reward":1000,"NewGatherType":1,"NewWeaponType":5}]},
				{"ID":200,"GatherGroupInfo":[{"Reward":2000,"NewGatherType":0,"NewWeaponType":4}]},
				{"ID":300,"GatherGroupInfo":[{"Reward":3000,"NewGatherType":1,"NewWeaponType":5}]}
			]},
			"GatherReward":{"Datas":[
				{"ID":1000,"GatherRewardGroupInfo":[
					{"Index":1,"ItemID":5001,"Count":1,"Lucky":false},
					{"Index":2,"ItemID":5001,"Count":3,"Lucky":true}
				]},
				{"ID":2000,"GatherRewardGroupInfo":[{"Index":1,"ItemID":5001,"Count":2,"Lucky":false}]},
				{"ID":3000,"GatherRewardGroupInfo":[{"Index":1,"ItemID":5001,"Count":0,"Lucky":false}]},
				{"ID":4000,"GatherRewardGroupInfo":[{"Index":1,"ItemID":5001,"Count":1,"Lucky":false}]}
			]},
			"GatherRule":{"Datas":[
				{"ID":1,"WoodNum":50,"FlowerNum":15,"StoneNum":15,"ButterflyNum":15,"WaterNum":15,"PickNum":15,
				 "LuckyFinishwoodNum":50,"LuckyFinishflowerNum":10,"LuckyFinishstoneNum":10,"LuckyFinishbutterflyNum":10,"LuckyFinishwaterNum":10,"LuckyFinishpickNum":0},
				{"ID":2,"WoodNum":15,"FlowerNum":50,"StoneNum":15,"ButterflyNum":15,"WaterNum":15,"PickNum":15,
				 "LuckyFinishwoodNum":10,"LuckyFinishflowerNum":50,"LuckyFinishstoneNum":10,"LuckyFinishbutterflyNum":10,"LuckyFinishwaterNum":10,"LuckyFinishpickNum":0},
				{"ID":3,"WoodNum":15,"FlowerNum":15,"StoneNum":50,"ButterflyNum":15,"WaterNum":15,"PickNum":15,
				 "LuckyFinishwoodNum":10,"LuckyFinishflowerNum":10,"LuckyFinishstoneNum":50,"LuckyFinishbutterflyNum":10,"LuckyFinishwaterNum":10,"LuckyFinishpickNum":0},
				{"ID":4,"WoodNum":15,"FlowerNum":15,"StoneNum":15,"ButterflyNum":50,"WaterNum":15,"PickNum":15,
				 "LuckyFinishwoodNum":10,"LuckyFinishflowerNum":10,"LuckyFinishstoneNum":10,"LuckyFinishbutterflyNum":50,"LuckyFinishwaterNum":10,"LuckyFinishpickNum":0},
				{"ID":5,"WoodNum":15,"FlowerNum":15,"StoneNum":15,"ButterflyNum":15,"WaterNum":50,"PickNum":15,
				 "LuckyFinishwoodNum":10,"LuckyFinishflowerNum":10,"LuckyFinishstoneNum":10,"LuckyFinishbutterflyNum":10,"LuckyFinishwaterNum":50,"LuckyFinishpickNum":0},
				{"ID":6,"WoodNum":30,"FlowerNum":30,"StoneNum":30,"ButterflyNum":30,"WaterNum":30,"PickNum":15,
				 "LuckyFinishwoodNum":20,"LuckyFinishflowerNum":20,"LuckyFinishstoneNum":20,"LuckyFinishbutterflyNum":20,"LuckyFinishwaterNum":20,"LuckyFinishpickNum":0}
			]}
		}`,
		filepath.Join(excelDir, "Scene.json"): `{"Scene":{"Datas":[
			{"ID":1,"GatherRuleID":1},{"ID":2,"GatherRuleID":2},{"ID":3,"GatherRuleID":3},
			{"ID":4,"GatherRuleID":4},{"ID":5,"GatherRuleID":5},{"ID":6,"GatherRuleID":6},
			{"ID":7,"GatherRuleID":999},{"ID":8,"GatherRuleID":0}
		]}}`,
		filepath.Join(configDir, "ScenesConfigAsset.json"): `{"Scenes":[{"ID":1,
			"GatherPointSetInfo":[{"GatherPointSets":[{"LifeGatherPointers":[
				{"Index":10,"GatherID":100},{"Index":20,"GatherID":200},{"Index":30,"GatherID":300}
			]}]}]
		},{"ID":7,
			"GatherPointSetInfo":[{"GatherPointSets":[{"LifeGatherPointers":[
				{"Index":10,"GatherID":100}
			]}]}]
		}]}`,
		filepath.Join(excelDir, "Item.json"): `{"Item":{"Datas":[
			{"ID":1001,"NewBagItemTag":2},{"ID":1002,"NewBagItemTag":2},
			{"ID":1003,"NewBagItemTag":2},{"ID":1004,"NewBagItemTag":2},
			{"ID":1005,"NewBagItemTag":2},{"ID":1006,"NewBagItemTag":2},
			{"ID":1007,"NewBagItemTag":2},{"ID":1008,"NewBagItemTag":2},
			{"ID":1009,"NewBagItemTag":2},{"ID":2001,"NewBagItemTag":3},
			{"ID":2002,"NewBagItemTag":3},{"ID":3001,"NewBagItemTag":5},
			{"ID":4001,"NewBagItemTag":33},{"ID":4002,"NewBagItemTag":31},
			{"ID":5001,"NewBagItemTag":8}
		]}}`,
		filepath.Join(excelDir, "Weapon.json"): `{"Weapon":{"Datas":[
			{"ID":1001,"ItemID":1001,"NewWeaponSystemType":4,"NewWeaponType":13},
			{"ID":1002,"ItemID":1002,"NewWeaponSystemType":6,"NewWeaponType":5},
			{"ID":1003,"ItemID":1003,"NewWeaponSystemType":6,"NewWeaponType":7},
			{"ID":1004,"ItemID":1004,"NewWeaponSystemType":6,"NewWeaponType":8},
			{"ID":1005,"ItemID":1005,"NewWeaponSystemType":6,"NewWeaponType":4},
			{"ID":1006,"ItemID":1006,"NewWeaponSystemType":6,"NewWeaponType":6},
			{"ID":1007,"ItemID":1007,"NewWeaponSystemType":6,"NewWeaponType":18},
			{"ID":1008,"ItemID":1008,"NewWeaponSystemType":6,"NewWeaponType":21},
			{"ID":1009,"ItemID":1009,"NewWeaponSystemType":6,"NewWeaponType":22}
		]}}`,
		filepath.Join(excelDir, "Armor.json"): `{"Armor":{"Datas":[
			{"ID":2001,"ItemID":2001,"NewWeaponSystemType":6,"NewEquipType":0},
			{"ID":2002,"ItemID":2002,"NewWeaponSystemType":6,"NewEquipType":1}
		]}}`,
		filepath.Join(excelDir, "Poster.json"): `{
			"Poster":{"Datas":[{"ID":3001,"ItemID":3001,"IllustrationID":3001}]},
			"PosterIllustration":{"Datas":[{"ID":3001,"IsShow":true}]}
		}`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	config.CONF = &config.Config{
		Resources: &config.Resources{ResourcePath: filepath.Join(root, "Resource"), DataPath: dataDir},
		Log:       &config.Log{Level: slog.PanicLevel, LogFile: false, AppName: "authority-test"},
	}
	log.NewApp()
	log.Game = log.App
	gdconf.LoadGameConfig()
	if gdconf.GetSceneInfo(1) == nil || gdconf.GetGatherConfigure(100) == nil ||
		gdconf.GetGatherRuleConfigure(6) == nil || gdconf.GetSceneGatherRuleConfigure(1) == nil ||
		gdconf.GetWeaponAllInfo(1002) == nil {
		fmt.Fprintln(os.Stderr, "authority test fixtures were not loaded")
		os.RemoveAll(root)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(root)
	os.Exit(code)
}

type authorityMessage struct {
	packetId uint32
	message  pb.Message
}

type authorityConn struct {
	mu       sync.Mutex
	messages []authorityMessage
}

func (c *authorityConn) Read() (*alg.GameMsg, error)    { return nil, errors.New("not implemented") }
func (c *authorityConn) SendBin(uint32, uint32, []byte) {}
func (c *authorityConn) Send(packetId uint32, message pb.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, authorityMessage{packetId: packetId, message: pb.Clone(message)})
}
func (c *authorityConn) SetUID(uint32)        {}
func (c *authorityConn) GetSeqId() uint32     { return 0 }
func (c *authorityConn) SetServerTag(string)  {}
func (c *authorityConn) Close()               {}
func (c *authorityConn) LocalAddr() net.Addr  { return &net.TCPAddr{} }
func (c *authorityConn) RemoteAddr() net.Addr { return &net.TCPAddr{} }
func (c *authorityConn) last(packetId uint32) pb.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.messages) - 1; i >= 0; i-- {
		if c.messages[i].packetId == packetId {
			return c.messages[i].message
		}
	}
	return nil
}

func newAuthorityPlayer() *model.Player {
	player := &model.Player{
		UserId:    1,
		Item:      model.DefaultItemModel(),
		Character: model.DefaultCharacterModel(),
		Team:      &model.TeamModel{TeamInfo: &model.TeamInfo{Char1: 101}},
		Scene: &model.SceneModel{SceneMap: map[uint32]*model.SceneInfo{
			1: {SceneId: 1},
		}},
		Conn: new(authorityConn),
	}
	addAuthorityCharacter(player, 101)
	addAuthorityCharacter(player, 102)
	return player
}

func addAuthorityCharacter(player *model.Player, characterId uint32) *model.CharacterInfo {
	character := &model.CharacterInfo{
		CharacterId:         characterId,
		CharacterAppearance: &model.CharacterAppearance{},
		EquipmentPresetList: make(map[uint32]*model.EquipmentPreset),
	}
	player.GetCharacterModel().GetCharacterMap()[characterId] = character
	return character
}

func addAuthorityWeapon(player *model.Player, instanceId, weaponId uint32) *model.ItemWeaponInfo {
	conf := gdconf.GetWeaponAllInfo(weaponId).WeaponInfo
	item := &model.ItemWeaponInfo{
		ItemBaseInfo: &model.ItemBaseInfo{
			ItemId: weaponId, ItemType: proto.EBagItemTag_EBagItemTag_Weapon,
			PackType: proto.PackType_PackType_Inventory,
		},
		WeaponId: weaponId, InstanceId: instanceId,
		WeaponSystemType: proto.EWeaponSystemType(conf.GetNewWeaponSystemType()),
	}
	player.GetItemModel().GetItemWeaponMap()[instanceId] = item
	return item
}

func addAuthorityArmor(player *model.Player, instanceId, armorId uint32) *model.ItemArmorInfo {
	conf := gdconf.GetArmorAllInfo(armorId).ArmorInfo
	item := &model.ItemArmorInfo{
		ItemBaseInfo: &model.ItemBaseInfo{
			ItemId: armorId, ItemType: proto.EBagItemTag_EBagItemTag_Armor,
			PackType: proto.PackType_PackType_Inventory,
		},
		ArmorId: armorId, InstanceId: instanceId,
		WeaponSystemType: proto.EWeaponSystemType(conf.GetNewWeaponSystemType()),
	}
	player.GetItemModel().GetItemArmorMap()[instanceId] = item
	return item
}

func addAuthorityPoster(player *model.Player, instanceId, posterId uint32) *model.ItemPosterInfo {
	item := &model.ItemPosterInfo{
		ItemBaseInfo: &model.ItemBaseInfo{
			ItemId: posterId, ItemType: proto.EBagItemTag_EBagItemTag_Poster,
			PackType: proto.PackType_PackType_Inventory,
		},
		PosterId: posterId, InstanceId: instanceId,
	}
	player.GetItemModel().GetItemPosterMap()[instanceId] = item
	return item
}

func newGatherGame(player *model.Player, sceneId uint32) *Game {
	team := player.GetTeamModel().GetTeamInfo()
	scene := model.NewScenePlayerInfo(player, &sceneId, team, &proto.Vector3{}, &proto.Vector3{})
	g := &Game{wordInfo: &WordInfo{}}
	g.wordInfo.allScenePlayer.Store(player.UserId, &ScenePlayer{Player: player, CurScene: scene})
	return g
}

func callGather(g *Game, player *model.Player, index uint32, reward int32, lucky bool) *proto.GatherRsp {
	g.Gather(player, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 42},
		Body: &proto.GatherReq{GatherItem: &proto.GatherItem{
			Index: index, Reward: reward, IsLucky: lucky,
		}},
	})
	return player.Conn.(*authorityConn).last(42).(*proto.GatherRsp)
}

func prepareWoodGather(player *model.Player) {
	character := player.GetCharacterModel().GetCharacterInfo(101)
	axe := addAuthorityWeapon(player, 1, 1002)
	character.GatherWeapon = axe.WeaponId
	preset := character.GetEquipmentPreset(0)
	oldWeapon := addAuthorityWeapon(player, 2, 1001)
	preset.WeaponInstanceId = oldWeapon.InstanceId
	oldWeapon.SetWearerId(character.CharacterId, 0)
}

func TestGatherRejectsMissingScenePointAndModelData(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	g := newGatherGame(player, 999)
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("missing configured scene: %v", rsp.GetStatus())
	}

	g = newGatherGame(player, 1)
	if rsp := callGather(g, player, 999, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("missing gather point: %v", rsp.GetStatus())
	}
	player.Scene.SceneMap[1] = nil
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("nil player scene data: %v", rsp.GetStatus())
	}
}

func TestGatherRejectsMissingCurrentScene(t *testing.T) {
	player := newAuthorityPlayer()
	g := &Game{wordInfo: &WordInfo{}}
	g.Gather(player, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 42},
		Body:       &proto.GatherReq{GatherItem: &proto.GatherItem{Index: 10, Reward: 1000}},
	})
	rsp := player.Conn.(*authorityConn).last(42).(*proto.GatherRsp)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("missing current scene: %v", rsp.GetStatus())
	}
}

func TestGatherIgnoresClientLuckyAndAcceptsConfiguredZeroType(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	g := newGatherGame(player, 1)
	rsp := callGather(g, player, 10, 1000, true)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_Ok || len(rsp.GetItems()) != 1 {
		t.Fatalf("forged lucky changed reward: status=%v items=%d", rsp.GetStatus(), len(rsp.GetItems()))
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001).Num; got != 1 {
		t.Fatalf("normal reward count = %d, want 1", got)
	}

	flower := addAuthorityWeapon(player, 3, 1005)
	player.GetCharacterModel().GetCharacterInfo(101).GatherWeapon = flower.WeaponId
	rsp = callGather(g, player, 20, 2000, false)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("configured gather type zero must be valid: %v", rsp.GetStatus())
	}
	limit := player.Scene.SceneMap[1].GatherLimits[0]
	if limit == nil || limit.GatherNum != 1 {
		t.Fatalf("type zero limit was not incremented: %#v", limit)
	}
}

func TestGatherUsesServerProbabilityForLuckyReward(t *testing.T) {
	probability := gdconf.GetGlobalConfigConfigure(gatherLuckyProbabilityConfig)
	old := probability.Values[1]
	probability.Values[1] = "100"
	defer func() { probability.Values[1] = old }()
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	g := newGatherGame(player, 1)
	rsp := callGather(g, player, 10, 1000, false)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("lucky gather failed: %v", rsp.GetStatus())
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001).Num; got != 3 {
		t.Fatalf("server lucky reward count = %d, want 3", got)
	}
}

func TestGatherRejectsUnmatchedRewardWrongToolAndInvalidCount(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	g := newGatherGame(player, 1)
	if rsp := callGather(g, player, 10, 4000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("reward outside gather group: %v", rsp.GetStatus())
	}
	player.GetCharacterModel().GetCharacterInfo(101).GatherWeapon = 1003
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("wrong tool type: %v", rsp.GetStatus())
	}
	player.GetCharacterModel().GetCharacterInfo(101).GatherWeapon = 1002
	if rsp := callGather(g, player, 30, 3000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("zero reward count: %v", rsp.GetStatus())
	}
	if player.GetItemModel().GetItemBaseInfo(5001) != nil {
		t.Fatal("rejected gathers must not grant items")
	}
}

func TestGatherLimitAndConcurrentDuplicateProtection(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	limit := player.Scene.SceneMap[1].GetGatherLimit(1)
	rule := gdconf.GetGatherRuleConfigure(1)
	oldNormal, oldLucky := rule.FlowerNum, rule.LuckyFinishflowerNum
	rule.FlowerNum, rule.LuckyFinishflowerNum = 1, 1
	defer func() { rule.FlowerNum, rule.LuckyFinishflowerNum = oldNormal, oldLucky }()
	g := newGatherGame(player, 1)

	const requests = 8
	var okCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			packetId := uint32(100 + index)
			g.Gather(player, &alg.GameMsg{
				PacketHead: &proto.PacketHead{PacketId: packetId},
				Body: &proto.GatherReq{GatherItem: &proto.GatherItem{
					Index: 10, Reward: 1000, IsLucky: true,
				}},
			})
			rsp := player.Conn.(*authorityConn).last(packetId).(*proto.GatherRsp)
			if rsp.GetStatus() == proto.StatusCode_StatusCode_Ok {
				okCount.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if got := okCount.Load(); got != 1 {
		t.Fatalf("successful duplicate gathers = %d, want 1", got)
	}
	if limit.GatherNum != 1 || player.GetItemModel().GetItemBaseInfo(5001).Num != 1 {
		t.Fatalf("duplicate protection failed: limit=%d items=%d", limit.GatherNum, player.GetItemModel().GetItemBaseInfo(5001).Num)
	}
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("gather beyond configured limit: %v", rsp.GetStatus())
	}
}

func TestGatherRejectsCounterOverflow(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	limit := player.Scene.SceneMap[1].GetGatherLimit(1)
	limit.GatherNum = math.MaxUint32
	g := newGatherGame(player, 1)
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("counter overflow must be rejected: %v", rsp.GetStatus())
	}
}

func TestGatherRejectsRewardQuantityOverflow(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	item := &model.ItemBaseInfo{
		ItemId: 5001, Num: math.MaxInt64, ItemType: proto.EBagItemTag_EBagItemTag_Material,
	}
	player.GetItemModel().GetItemBaseMap()[5001] = item
	g := newGatherGame(player, 1)
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("reward quantity overflow must be rejected: %v", rsp.GetStatus())
	}
	if item.Num != math.MaxInt64 || player.Scene.SceneMap[1].GatherLimits[1].GatherNum != 0 {
		t.Fatal("rejected overflow changed inventory or gather count")
	}
}

func TestGatherRuleLimitsByType(t *testing.T) {
	names := [...]string{"Wood", "Flower", "Stone", "Butterfly", "Water", "Pick"}
	primaryNormal := [...]uint32{50, 50, 50, 50, 50, 15}
	primaryLucky := [...]uint32{50, 50, 50, 50, 50, 0}
	for ruleID := int32(1); ruleID <= 6; ruleID++ {
		rule := gdconf.GetGatherRuleConfigure(ruleID)
		if rule == nil || gdconf.GetSceneGatherRuleConfigure(uint32(ruleID)) != rule {
			t.Fatalf("scene %d did not load GatherRule %d", ruleID, ruleID)
		}
		for gatherType, name := range names {
			wantNormal, wantLucky := uint32(15), uint32(10)
			if ruleID == 6 {
				wantNormal, wantLucky = 30, 20
			}
			if int32(gatherType) == ruleID-1 {
				wantNormal, wantLucky = primaryNormal[gatherType], primaryLucky[gatherType]
			}
			if gatherType == 5 {
				wantLucky = 0
			}
			normal, lucky, ok := gdconf.GatherRuleLimits(rule, uint32(gatherType))
			if !ok || normal != wantNormal || lucky != wantLucky {
				t.Fatalf("rule %d type %d (%s): got (%d,%d,%v), want (%d,%d,true)",
					ruleID, gatherType, name, normal, lucky, ok, wantNormal, wantLucky)
			}
		}
	}
	for _, gatherType := range []uint32{6, math.MaxUint32} {
		if _, _, ok := gdconf.GatherRuleLimits(gdconf.GetGatherRuleConfigure(1), gatherType); ok {
			t.Fatalf("out-of-range gather type %d accepted", gatherType)
		}
	}
	if _, _, ok := gdconf.GatherRuleLimits(nil, 0); ok {
		t.Fatal("missing gather rule accepted")
	}
}

func TestSceneInfoInitializesAllGatherLimits(t *testing.T) {
	sceneModel := new(model.SceneModel)
	info := sceneModel.GetSceneInfo(1)
	if len(info.GetGatherLimits()) != gdconf.GatherTypeCount {
		t.Fatalf("initialized gather types = %d, want %d", len(info.GetGatherLimits()), gdconf.GatherTypeCount)
	}
	for gatherType := uint32(0); gatherType < gdconf.GatherTypeCount; gatherType++ {
		normal, lucky, _ := gdconf.GatherRuleLimits(gdconf.GetGatherRuleConfigure(1), gatherType)
		limit := info.GetGatherLimit(gatherType)
		if limit.GatherType != gatherType || limit.GatherLimitNum != normal ||
			limit.LuckyGatherLimitNum != lucky || limit.GatherNum != 0 {
			t.Fatalf("type %d initialized incorrectly: %+v", gatherType, limit)
		}
	}
	if got := len(info.SceneGatherLimit().GetGatherLimits()); got != gdconf.GatherTypeCount {
		t.Fatalf("scene gather limit response contains %d types", got)
	}
}

func TestOldSaveRestoresLimitsWithoutClearingExceededCount(t *testing.T) {
	player := newAuthorityPlayer()
	player.Scene.SceneMap[1] = &model.SceneInfo{
		SceneId: 1,
		GatherLimits: map[uint32]*model.GatherLimit{
			1: {GatherType: 1, GatherNum: 60, GatherLimitNum: 9999, LuckyGatherLimitNum: 9999},
		},
	}
	recovered := player.GetSceneModel().GetSceneInfo(1)
	if limit := recovered.GatherLimits[1]; limit == nil || limit.GatherNum != 60 ||
		limit.GatherLimitNum != 15 || limit.LuckyGatherLimitNum != 10 {
		t.Fatalf("old save was not restored on scene access: %+v", limit)
	}
	prepareWoodGather(player)
	g := newGatherGame(player, 1)
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("old save exceeded tightened limit: %v", rsp.GetStatus())
	}
	limit := player.Scene.SceneMap[1].GatherLimits[1]
	if limit.GatherNum != 60 || limit.GatherLimitNum != 15 || limit.LuckyGatherLimitNum != 10 {
		t.Fatalf("old save was not authoritatively restored: %+v", limit)
	}
	if player.GetItemModel().GetItemBaseInfo(5001) != nil {
		t.Fatal("rejected old save granted items")
	}
}

func TestGatherLimitDailyResetPreservesSameDayCount(t *testing.T) {
	today := model.GatherServerDay(time.Now())
	info := &model.SceneInfo{
		SceneId: 1,
		GatherLimits: map[uint32]*model.GatherLimit{
			1: {GatherType: 1, GatherNum: 15, LastGatherDay: today},
		},
	}
	info.RefreshGatherLimits(today)
	if got := info.GatherLimits[1].GatherNum; got != 15 {
		t.Fatalf("same-day refresh cleared count to %d", got)
	}
	info.RefreshGatherLimits(today + 1)
	limit := info.GatherLimits[1]
	if limit.GatherNum != 0 || limit.LastGatherDay != today+1 ||
		limit.GatherLimitNum != 15 || limit.LuckyGatherLimitNum != 10 {
		t.Fatalf("cross-day reset incorrect: %+v", limit)
	}
}

func TestGatherRejectsMissingRuleAndOutOfRangeType(t *testing.T) {
	player := newAuthorityPlayer()
	prepareWoodGather(player)
	g := newGatherGame(player, 7)
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("missing scene gather rule accepted: %v", rsp.GetStatus())
	}
	if player.GetItemModel().GetItemBaseInfo(5001) != nil {
		t.Fatal("missing rule granted items")
	}

	g = newGatherGame(player, 1)
	group := gdconf.GetGatherConfigure(100).GatherGroupInfo[0]
	oldType := group.NewGatherType
	group.NewGatherType = int32(gdconf.GatherTypeCount)
	defer func() { group.NewGatherType = oldType }()
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("out-of-range gather type accepted: %v", rsp.GetStatus())
	}
}

func TestGatherZeroLimitsAreForbidden(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit *model.GatherLimit
		lucky bool
		want  bool
	}{
		{name: "nil", lucky: false, want: false},
		{name: "normal zero", limit: &model.GatherLimit{GatherLimitNum: 0, LuckyGatherLimitNum: 10}, want: false},
		{name: "normal zero lucky", limit: &model.GatherLimit{GatherLimitNum: 0, LuckyGatherLimitNum: 10}, lucky: true, want: false},
		{name: "lucky zero normal", limit: &model.GatherLimit{GatherLimitNum: 10, LuckyGatherLimitNum: 0}, want: true},
		{name: "lucky zero lucky", limit: &model.GatherLimit{GatherLimitNum: 10, LuckyGatherLimitNum: 0}, lucky: true, want: false},
		{name: "normal reached", limit: &model.GatherLimit{GatherNum: 10, GatherLimitNum: 10, LuckyGatherLimitNum: 20}, want: false},
		{name: "lucky reached", limit: &model.GatherLimit{GatherNum: 10, GatherLimitNum: 20, LuckyGatherLimitNum: 10}, lucky: true, want: false},
		{name: "lucky reached normal", limit: &model.GatherLimit{GatherNum: 10, GatherLimitNum: 20, LuckyGatherLimitNum: 10}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := gatherWithinLimit(test.limit, test.lucky); got != test.want {
				t.Fatalf("gatherWithinLimit() = %v, want %v", got, test.want)
			}
		})
	}

	player := newAuthorityPlayer()
	prepareWoodGather(player)
	g := newGatherGame(player, 1)
	rule := gdconf.GetGatherRuleConfigure(1)
	probability := gdconf.GetGlobalConfigConfigure(gatherLuckyProbabilityConfig)
	oldNormal, oldLucky, oldProbability := rule.FlowerNum, rule.LuckyFinishflowerNum, probability.Values[1]
	defer func() {
		rule.FlowerNum, rule.LuckyFinishflowerNum = oldNormal, oldLucky
		probability.Values[1] = oldProbability
	}()
	rule.FlowerNum = 0
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("configured zero normal limit accepted: %v", rsp.GetStatus())
	}
	rule.FlowerNum, rule.LuckyFinishflowerNum = oldNormal, 0
	probability.Values[1] = "100"
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("configured zero lucky limit accepted: %v", rsp.GetStatus())
	}
	probability.Values[1] = "0"
	if rsp := callGather(g, player, 10, 1000, false); rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("zero lucky limit incorrectly blocked normal gather: %v", rsp.GetStatus())
	}
}

func TestCollectionRewardCountBounds(t *testing.T) {
	roll := func(int64) int64 { return 0 }
	if count, ok := collectionRewardCount(0, math.MaxInt32, roll); !ok || count != 0 {
		t.Fatalf("full int32 range: count=%d ok=%v", count, ok)
	}
	if _, ok := collectionRewardCount(-1, 1, roll); ok {
		t.Fatal("negative minimum must be rejected")
	}
	if _, ok := collectionRewardCount(2, 1, roll); ok {
		t.Fatal("reversed bounds must be rejected")
	}
	if _, ok := collectionRewardCount(1, 2, func(int64) int64 { return 2 }); ok {
		t.Fatal("out-of-range random offset must be rejected")
	}
}
