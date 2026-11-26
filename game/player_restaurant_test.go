package game

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	protobuf "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

type coreTestConn struct {
	sent []protobuf.Message
}

func (c *coreTestConn) Read() (*alg.GameMsg, error)         { return nil, errors.New("not implemented") }
func (c *coreTestConn) SendBin(uint32, uint32, []byte)      {}
func (c *coreTestConn) Send(_ uint32, msg protobuf.Message) { c.sent = append(c.sent, msg) }
func (c *coreTestConn) SetUID(uint32)                       {}
func (c *coreTestConn) GetSeqId() uint32                    { return 0 }
func (c *coreTestConn) SetServerTag(string)                 {}
func (c *coreTestConn) Close()                              {}
func (c *coreTestConn) LocalAddr() net.Addr                 { return &net.TCPAddr{} }
func (c *coreTestConn) RemoteAddr() net.Addr                { return &net.TCPAddr{} }

func (c *coreTestConn) last(t *testing.T) protobuf.Message {
	t.Helper()
	if len(c.sent) == 0 {
		t.Fatal("no response sent")
	}
	return c.sent[len(c.sent)-1]
}

func coreTestMessage(body protobuf.Message) *alg.GameMsg {
	return &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 9001}, Body: body}
}

func newCoreTestPlayer() *model.Player {
	p := &model.Player{
		UserId: 7,
		Item: &model.ItemModel{
			ItemBaseInfo:   make(map[uint32]*model.ItemBaseInfo),
			ItemPetInfoMap: make(map[uint32]*model.ItemPetInfo),
		},
		Character: &model.CharacterModel{CharacterMap: map[uint32]*model.CharacterInfo{
			1: {CharacterId: 1},
		}},
		Team:  &model.TeamModel{TeamInfo: &model.TeamInfo{Char1: 1}},
		Scene: &model.SceneModel{SceneMap: make(map[uint32]*model.SceneInfo)},
	}
	p.Scene.SceneMap[1] = &model.SceneInfo{
		SceneId:   1,
		AreaDatas: map[uint32]*model.AreaData{10: {AreaId: 10, AreaState: proto.AreaData_Unlock}},
	}
	return p
}

func newCoreTestGame(p *model.Player) (*Game, *coreTestConn) {
	conn := new(coreTestConn)
	p.Conn = conn
	g := &Game{}
	g.wordInfo = &WordInfo{}
	g.wordInfo.allScenePlayer.Store(p.UserId, &ScenePlayer{
		Player:   p,
		CurScene: &model.ScenePlayerInfo{Player: p, SceneId: 1},
	})
	return g, conn
}

func addCoreBase(p *model.Player, id uint32, count int64) {
	items := p.GetItemModel().GetItemBaseMap()
	if items[id] == nil {
		items[id] = &model.ItemBaseInfo{ItemId: id, Num: count, ItemType: proto.EBagItemTag_EBagItemTag_Item, PackType: proto.ItemDetail_PackType_Inventory}
		return
	}
	items[id].Num += count
}

func coreBaseCount(p *model.Player, id uint32) int64 {
	if info := p.GetItemModel().GetItemBaseInfo(id); info != nil {
		return info.Num
	}
	return 0
}

func setupCoreGlobals(t *testing.T) map[uint32]*excel.ItemConfigure {
	t.Helper()
	oldRestaurantProvider := restaurantConfigProvider
	oldActivityProvider := activityConfigProvider
	oldPetProvider := petConfigProvider
	oldExpressionProvider := expressionConfigProvider
	oldSceneProvider := sceneConfigProvider
	oldItemConfigure := coreItemConfigure
	oldGrantItem := coreGrantItem
	oldSavePlayer := coreSavePlayer
	oldRoll := coreRollIntN
	oldNow := coreNow
	oldSceneArea := coreSceneAreaID
	oldCapacity := corePetCapacity
	oldNameMax := corePetNameMaxLength
	t.Cleanup(func() {
		restaurantConfigProvider = oldRestaurantProvider
		activityConfigProvider = oldActivityProvider
		petConfigProvider = oldPetProvider
		expressionConfigProvider = oldExpressionProvider
		sceneConfigProvider = oldSceneProvider
		coreItemConfigure = oldItemConfigure
		coreGrantItem = oldGrantItem
		coreSavePlayer = oldSavePlayer
		coreRollIntN = oldRoll
		coreNow = oldNow
		coreSceneAreaID = oldSceneArea
		corePetCapacity = oldCapacity
		corePetNameMaxLength = oldNameMax
	})

	items := map[uint32]*excel.ItemConfigure{
		101:       {ID: 101, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
		1600:      {ID: 1600, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
		1601:      {ID: 1601, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
		200100001: {ID: 200100001, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Catcher)},
		200001001: {ID: 200001001, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Pet)},
		200101002: {ID: 200101002, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_PetMmaterials)},
	}
	coreItemConfigure = func(id uint32) *excel.ItemConfigure { return items[id] }
	coreGrantItem = func(s *model.Player, id uint32, count int64) *model.AddItemCtx {
		conf := items[id]
		if conf == nil {
			return nil
		}
		if proto.EBagItemTag(conf.NewBagItemTag) == proto.EBagItemTag_EBagItemTag_Pet {
			pet := s.GetItemModel().AddItemPet(id)
			return &model.AddItemCtx{EBagItemTag: pet, Num: count}
		}
		info := s.GetItemModel().GetItemBaseInfo(id)
		if info == nil {
			info = &model.ItemBaseInfo{ItemId: id, ItemType: proto.EBagItemTag(conf.NewBagItemTag), PackType: proto.ItemDetail_PackType_Inventory}
			s.GetItemModel().GetItemBaseMap()[id] = info
		}
		info.Num += count
		return &model.AddItemCtx{EBagItemTag: info, Num: count}
	}
	coreSavePlayer = func(*model.Player) error { return nil }
	coreRollIntN = func(int) int { return 0 }
	coreNow = func() time.Time { return time.Date(2026, 6, 1, 12, 0, 0, 0, time.Local) }
	coreSceneAreaID = func(sceneID uint32) (uint32, bool) {
		if sceneID == 1 {
			return 10, true
		}
		return 0, false
	}
	corePetCapacity = func() (int, error) { return 100, nil }
	corePetNameMaxLength = func() (int, error) { return 4, nil }
	return items
}

func setupRestaurantConfig(t *testing.T) {
	t.Helper()
	restaurantConfigProvider = func() (*excel.AllRestaurantGameDatas, error) {
		return &excel.AllRestaurantGameDatas{
			RestaurantDungeon: &excel.RestaurantDungeonDatas{Datas: []*excel.RestaurantDungeonConfigure{
				{ID: 1001, ActivityID: 12000, RestaurantDungeonGroupInfo: []*excel.RestaurantDungeonGroupInfos{
					{FoodLevel: 1, FoodType: 1, OrderQuantity: 1},
				}},
			}},
			RestaurantDeskBag: &excel.RestaurantDeskBagDatas{Datas: []*excel.RestaurantDeskBagConfigure{
				{ID: 12000, RestaurantDeskBagGroupInfo: []*excel.RestaurantDeskBagGroupInfos{
					{BagID: 1, UnlockDungonID: 0},
					{BagID: 2, UnlockDungonID: 1001},
					{BagID: 3, UnlockDungonID: 0},
				}},
			}},
			RestaurantGameFood: &excel.RestaurantGameFoodDatas{Datas: []*excel.RestaurantGameFoodConfigure{
				{ID: 12000, RestaurantGameFoodGroupInfo: []*excel.RestaurantGameFoodGroupInfos{
					{FoodID: 1, FoodLevel: 1, FoodType: 1, SyntheticTarget: 2, RecyclingPriceMin: 1, RecyclingPriceMax: 1},
					{FoodID: 2, FoodLevel: 2, FoodType: 1, RecyclingPriceMin: 3, RecyclingPriceMax: 3},
				}},
			}},
			RestaurantRandomPool: &excel.RestaurantRandomPoolDatas{Datas: []*excel.RestaurantRandomPoolConfigure{
				{ID: 1001, ActivityID: 12000, RateMin: 2, RateMax: 2, RestaurantRandomPoolGroupInfo: []*excel.RestaurantRandomPoolGroupInfos{{FoodId: 1}}},
			}},
			RestaurantGameQuest: &excel.RestaurantGameQuestDatas{Datas: []*excel.RestaurantGameQuestConfigure{
				{ID: 12000, RestaurantGameQuestGroupInfo: []*excel.RestaurantGameQuestGroupInfo{
					{GameQuestID: 1, GameCoin: 10, GameIntegralItem: 1600, GameIntegral: 1, Param: 1001, GameCoinID: 1601},
					{GameQuestID: 101, GameCoin: 5, Param: 1, IsRefresh: true, NewRestaurantGameQuestType: 1, GameCoinID: 1601},
				}},
			}},
		}, nil
	}
	activityConfigProvider = func() (*excel.AllActivityDatas, error) {
		return &excel.AllActivityDatas{Activity: &excel.ActivityDatas{Datas: []*excel.ActivityConfigure{
			{ID: 12000, OpeningTime: "2026-01-01 00:00:00", ClosingTime: "2026-12-31 23:59:59"},
		}}}, nil
	}
}

func setupPetConfig(t *testing.T, rate int32) {
	t.Helper()
	petConfigProvider = func() (*excel.AllPetDatas, error) {
		return &excel.AllPetDatas{
			Pet: &excel.PetDatas{Datas: []*excel.PetConfigure{
				{ID: 200001001, MonsterID: 1001, DecisionCapture: true, Parma: []int32{rate}, ReleaseReward: []int32{10002}},
			}},
			ReleaseReward: &excel.ReleaseRewardDatas{Datas: []*excel.ReleaseRewardConfigure{
				{ID: 10002, ReleaseRewardItem: []*excel.ReleaseRewardItem{{ReleaseRewardID: 200101002, ReleaseRewardCount: 2}}},
			}},
		}, nil
	}
	expressionConfigProvider = func() (*excel.AllExpressionDatas, error) {
		return &excel.AllExpressionDatas{Catcher: &excel.CatcherDatas{Datas: []*excel.CatcherConfigure{
			{ID: 120000, NeedItemID: 200100001, NeedItemCount: 1},
		}}}, nil
	}
}

func restaurantState(t *testing.T, p *model.Player) (*model.RestaurantActivityState, *model.RestaurantDungeonState) {
	t.Helper()
	activity := p.GetRestaurantModel().Activities[12000]
	if activity == nil || activity.Dungeons[1001] == nil {
		t.Fatal("restaurant state missing")
	}
	return activity, activity.Dungeons[1001]
}

func TestRestaurantOperateEatAndRewardAreIdempotent(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	addCoreBase(p, 1601, 10)
	g, conn := newCoreTestGame(p)

	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))
	if rsp := conn.last(t).(*proto.RestaurantGameInfoRsp); rsp.Status != proto.StatusCode_StatusCode_OK {
		t.Fatalf("info status = %v", rsp.Status)
	}
	buy := &proto.RestaurantGameOperateReq{ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Buy, BuyPoolId: 1001, Grid: []uint32{1}}
	g.RestaurantGameOperate(p, coreTestMessage(buy))
	if coreBaseCount(p, 1601) != 8 {
		t.Fatalf("coins after buy = %d, want 8", coreBaseCount(p, 1601))
	}
	g.RestaurantGameOperate(p, coreTestMessage(buy))
	if coreBaseCount(p, 1601) != 8 {
		t.Fatalf("duplicate buy charged again: %d", coreBaseCount(p, 1601))
	}

	eat := &proto.RestaurantGameEatReq{ActivityId: 12000, CustomerIndex: 0, Grid: 1}
	g.RestaurantGameEat(p, coreTestMessage(eat))
	_, dungeon := restaurantState(t, p)
	if !dungeon.Cleared || dungeon.Grids[1] != 0 {
		t.Fatal("eat did not consume food and clear configured customer")
	}
	g.RestaurantGameEat(p, coreTestMessage(eat))
	if rsp := conn.last(t).(*proto.RestaurantGameEatRsp); rsp.Status != proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_CUSTOMER_FINISH {
		t.Fatalf("duplicate eat status = %v", rsp.Status)
	}

	reward := &proto.RestaurantGameQuestRewardReq{ActivityId: 12000, QuestId: 1}
	g.RestaurantGameQuestReward(p, coreTestMessage(reward))
	if coreBaseCount(p, 1601) != 18 || coreBaseCount(p, 1600) != 1 {
		t.Fatalf("reward counts = %d/%d, want 18/1", coreBaseCount(p, 1601), coreBaseCount(p, 1600))
	}
	g.RestaurantGameQuestReward(p, coreTestMessage(reward))
	if rsp := conn.last(t).(*proto.RestaurantGameQuestRewardRsp); rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("duplicate reward status = %v", rsp.Status)
	}
	if coreBaseCount(p, 1601) != 18 || coreBaseCount(p, 1600) != 1 {
		t.Fatal("duplicate reward granted items")
	}
}

func TestRestaurantRejectsLockedGridAndInsufficientCoins(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	addCoreBase(p, 1601, 1)
	g, conn := newCoreTestGame(p)
	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))

	g.RestaurantGameOperate(p, coreTestMessage(&proto.RestaurantGameOperateReq{
		ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Buy, BuyPoolId: 1001, Grid: []uint32{2},
	}))
	if rsp := conn.last(t).(*proto.RestaurantGameOperateRsp); rsp.Status != proto.StatusCode_StatusCode_MINIGAME_RESTAURANT_GRID_NOT_EXIST {
		t.Fatalf("locked grid status = %v", rsp.Status)
	}
	g.RestaurantGameOperate(p, coreTestMessage(&proto.RestaurantGameOperateReq{
		ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Buy, BuyPoolId: 1001, Grid: []uint32{1},
	}))
	if rsp := conn.last(t).(*proto.RestaurantGameOperateRsp); rsp.Status != proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH {
		t.Fatalf("insufficient coin status = %v", rsp.Status)
	}
	_, dungeon := restaurantState(t, p)
	if dungeon.Grids[1] != 0 || len(dungeon.Operations) != 0 || coreBaseCount(p, 1601) != 1 {
		t.Fatal("failed buy mutated grid, ledger, or coins")
	}
}

func TestRestaurantMixAndRecycleUseConfiguredRules(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	addCoreBase(p, 1601, 10)
	g, _ := newCoreTestGame(p)
	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))
	_, dungeon := restaurantState(t, p)
	dungeon.Grids[1], dungeon.Grids[3] = 1, 1

	g.RestaurantGameOperate(p, coreTestMessage(&proto.RestaurantGameOperateReq{
		ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Mix, Grid: []uint32{1, 3},
	}))
	if dungeon.Grids[1] != 2 || dungeon.Grids[3] != 0 {
		t.Fatalf("mix grids = %v, want [2 0]", dungeon.Grids)
	}
	g.RestaurantGameOperate(p, coreTestMessage(&proto.RestaurantGameOperateReq{
		ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Recycle, Grid: []uint32{1},
	}))
	if dungeon.Grids[1] != 0 || coreBaseCount(p, 1601) != 13 {
		t.Fatalf("recycle grid/coins = %d/%d, want 0/13", dungeon.Grids[1], coreBaseCount(p, 1601))
	}
}

func TestRestaurantStateSurvivesReloginAndReplay(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	addCoreBase(p, 1601, 10)
	g, _ := newCoreTestGame(p)
	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))
	buy := &proto.RestaurantGameOperateReq{ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Buy, BuyPoolId: 1001, Grid: []uint32{1}}
	eat := &proto.RestaurantGameEatReq{ActivityId: 12000, CustomerIndex: 0, Grid: 1}
	reward := &proto.RestaurantGameQuestRewardReq{ActivityId: 12000, QuestId: 1}
	g.RestaurantGameOperate(p, coreTestMessage(buy))
	g.RestaurantGameEat(p, coreTestMessage(eat))
	g.RestaurantGameQuestReward(p, coreTestMessage(reward))

	bin, err := sonic.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	restored := &model.Player{UserId: p.UserId}
	if err := sonic.Unmarshal(bin, restored); err != nil {
		t.Fatal(err)
	}
	g2, conn := newCoreTestGame(restored)
	g2.RestaurantGameOperate(restored, coreTestMessage(buy))
	g2.RestaurantGameEat(restored, coreTestMessage(eat))
	g2.RestaurantGameQuestReward(restored, coreTestMessage(reward))
	if coreBaseCount(restored, 1601) != 18 || coreBaseCount(restored, 1600) != 1 {
		t.Fatal("relogin replay changed reward balances")
	}
	if rsp := conn.last(t).(*proto.RestaurantGameQuestRewardRsp); rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("restored duplicate reward status = %v", rsp.Status)
	}
}

func TestRestaurantAllowsSameBuyAfterStateChanges(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	addCoreBase(p, 1601, 10)
	g, _ := newCoreTestGame(p)
	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))
	buy := &proto.RestaurantGameOperateReq{ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Buy, BuyPoolId: 1001, Grid: []uint32{1}}
	recycle := &proto.RestaurantGameOperateReq{ActivityId: 12000, Type: proto.RestaurantGameOperateType_RestaurantGameOperateType_Recycle, Grid: []uint32{1}}
	g.RestaurantGameOperate(p, coreTestMessage(buy))
	g.RestaurantGameOperate(p, coreTestMessage(recycle))
	g.RestaurantGameOperate(p, coreTestMessage(buy))
	_, dungeon := restaurantState(t, p)
	if coreBaseCount(p, 1601) != 7 || dungeon.Grids[1] != 1 {
		t.Fatalf("buy after recycle = coins %d food %d, want 7/1", coreBaseCount(p, 1601), dungeon.Grids[1])
	}
	g.RestaurantGameOperate(p, coreTestMessage(buy))
	if coreBaseCount(p, 1601) != 7 {
		t.Fatal("immediate duplicate buy after state change charged again")
	}
}

func TestRestaurantDailyRewardDoesNotResetOnSameDayInfo(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	g, conn := newCoreTestGame(p)
	info := &proto.RestaurantGameInfoReq{ActivityId: 12000}
	reward := &proto.RestaurantGameQuestRewardReq{ActivityId: 12000, QuestId: 101}
	g.RestaurantGameInfo(p, coreTestMessage(info))
	g.RestaurantGameQuestReward(p, coreTestMessage(reward))
	if coreBaseCount(p, 1601) != 5 {
		t.Fatal("daily login reward not granted")
	}
	g.RestaurantGameInfo(p, coreTestMessage(info))
	g.RestaurantGameQuestReward(p, coreTestMessage(reward))
	if rsp := conn.last(t).(*proto.RestaurantGameQuestRewardRsp); rsp.Status != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("same-day daily reward status = %v", rsp.Status)
	}
	if coreBaseCount(p, 1601) != 5 {
		t.Fatal("same-day daily reward granted twice")
	}
}

func TestRestaurantRejectsClosedOrMissingAuthority(t *testing.T) {
	setupCoreGlobals(t)
	setupRestaurantConfig(t)
	p := newCoreTestPlayer()
	g, conn := newCoreTestGame(p)
	activityConfigProvider = func() (*excel.AllActivityDatas, error) {
		return &excel.AllActivityDatas{Activity: &excel.ActivityDatas{Datas: []*excel.ActivityConfigure{
			{ID: 12000, OpeningTime: "2025-01-01 00:00:00", ClosingTime: "2025-12-31 23:59:59"},
		}}}, nil
	}
	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))
	if rsp := conn.last(t).(*proto.RestaurantGameInfoRsp); rsp.Status != proto.StatusCode_StatusCode_Activity_Close {
		t.Fatalf("closed activity status = %v", rsp.Status)
	}
	activityConfigProvider = func() (*excel.AllActivityDatas, error) { return nil, errors.New("missing") }
	g.RestaurantGameInfo(p, coreTestMessage(&proto.RestaurantGameInfoReq{ActivityId: 12000}))
	if rsp := conn.last(t).(*proto.RestaurantGameInfoRsp); rsp.Status != proto.StatusCode_StatusCode_CFG_NOT_FOUND {
		t.Fatalf("missing authority status = %v", rsp.Status)
	}
}
