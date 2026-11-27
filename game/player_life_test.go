package game

import (
	"errors"
	"net"
	"testing"

	protobuf "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func TestLifeRequestTokenUsesPacketSequence(t *testing.T) {
	player := &model.Player{LoginUUID: "login-a"}
	msg := &alg.GameMsg{PacketHead: &proto.PacketHead{MsgId: 1, SeqId: 9}}
	first := lifeRequestToken(player, msg, "produce", 10)
	if again := lifeRequestToken(player, msg, "produce", 10); again != first {
		t.Fatal("同一封包应生成相同token")
	}
	msg.SeqId = 10
	if next := lifeRequestToken(player, msg, "produce", 10); next == first {
		t.Fatal("不同封包序号共用了token")
	}
	player.LoginUUID = "login-b"
	msg.SeqId = 0
	if reconnect := lifeRequestToken(player, msg, "produce", 10); reconnect == first {
		t.Fatal("重连封包共用了旧token")
	}
}

func TestLifeCommonResultUsesConfiguredItems(t *testing.T) {
	receipt := model.LifeReceipt{
		Status: proto.StatusCode_StatusCode_OK, LifeType: proto.LifeType_LIFE_TYPE_Cooking, Success: true,
		ResultItems:      []model.LifeItem{{ItemID: 200, Count: 2, Tag: proto.EBagItemTag_EBagItemTag_Food}},
		SkillReturnItems: []model.LifeItem{{ItemID: 100, Count: 1, Tag: proto.EBagItemTag_EBagItemTag_Material}},
		IsLuckyReturn:    true,
	}
	result := lifeCommonResult(receipt)
	if !result.GetIsSuccess() || result.GetLifeType() != receipt.LifeType || len(result.GetResultItems()) != 1 || result.GetResultItems()[0].GetMainItem().GetItemId() != 200 {
		t.Fatalf("生产结果转换错误: %+v", result)
	}
	if len(result.GetSkillReturnItems()) != 1 || result.GetSkillReturnItems()[0].GetItemId() != 100 || result.GetSkillReturnItems()[0].GetNum() != 1 {
		t.Fatalf("技能返还转换错误: %+v", result)
	}
}

func TestValidLifeType(t *testing.T) {
	for _, lifeType := range []proto.LifeType{
		proto.LifeType_LIFE_TYPE_Cooking, proto.LifeType_LIFE_TYPE_Fishing,
		proto.LifeType_LIFE_TYPE_Making, proto.LifeType_LIFE_TYPE_Handicraft,
	} {
		if !model.ValidLifeType(lifeType) {
			t.Fatalf("有效生活类型被拒绝: %v", lifeType)
		}
	}
	for _, lifeType := range []proto.LifeType{proto.LifeType_LIFE_TYPE_NONE, proto.LifeType_LIFE_TYPE_Sewing, proto.LifeType_LIFE_TYPE_Gathering} {
		if model.ValidLifeType(lifeType) {
			t.Fatalf("未实现生活类型被接受: %v", lifeType)
		}
	}
}

type lifeTestConn struct {
	sent []protobuf.Message
}

func (c *lifeTestConn) Read() (*alg.GameMsg, error)    { return nil, errors.New("not implemented") }
func (c *lifeTestConn) SendBin(uint32, uint32, []byte) {}
func (c *lifeTestConn) Send(_ uint32, msg protobuf.Message) {
	c.sent = append(c.sent, msg)
}
func (c *lifeTestConn) SetUID(uint32)        {}
func (c *lifeTestConn) GetSeqId() uint32     { return 0 }
func (c *lifeTestConn) SetServerTag(string)  {}
func (c *lifeTestConn) Close()               {}
func (c *lifeTestConn) LocalAddr() net.Addr  { return &net.TCPAddr{} }
func (c *lifeTestConn) RemoteAddr() net.Addr { return &net.TCPAddr{} }

func TestFishingCancelDoesNotConsumeOrLoadConfig(t *testing.T) {
	conn := &lifeTestConn{}
	player := &model.Player{Item: &model.ItemModel{ItemBaseInfo: make(map[uint32]*model.ItemBaseInfo)}, Conn: conn}
	msg := &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 77, MsgId: 10, SeqId: 1},
		Body:       &proto.FishingReq{IsCancel: true},
	}
	(&Game{}).Fishing(player, msg)
	if len(conn.sent) != 1 {
		t.Fatalf("取消钓鱼产生了额外通知: %d", len(conn.sent))
	}
	rsp, ok := conn.sent[0].(*proto.FishingRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_OK || !rsp.GetIsCancel() || rsp.GetFishingEventResult() != nil {
		t.Fatalf("取消钓鱼响应错误: %+v", conn.sent[0])
	}
	if player.Life != nil || len(player.Item.ItemBaseInfo) != 0 {
		t.Fatal("取消钓鱼修改了玩家状态")
	}
}

func lifeProtocolFixture(t *testing.T, save func(*model.Player) error) (*Game, *model.Player, *lifeTestConn, *model.LifeConfig) {
	t.Helper()
	previousLoader, previousSave := lifeConfigLoader, lifeSavePlayer
	t.Cleanup(func() {
		lifeConfigLoader, lifeSavePlayer = previousLoader, previousSave
	})
	cfg := &model.LifeConfig{
		Levels:       map[proto.LifeType]map[uint32]*model.LifeLevelConfig{},
		Recipes:      map[proto.LifeType]map[uint32]*model.LifeRecipe{},
		Skills:       map[proto.LifeType]map[uint32]*model.LifeSkillConfig{},
		Achievements: map[uint32]*model.LifeAchievementConfig{},
		Items: map[uint32]*excel.ItemConfigure{
			1: {ID: 1, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
			2: {ID: 2, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)},
			3: {ID: 3, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Food)},
		},
	}
	cfg.Levels[proto.LifeType_LIFE_TYPE_Cooking] = map[uint32]*model.LifeLevelConfig{
		1: {Level: 1, ProficiencyPerAction: 10},
	}
	cfg.Recipes[proto.LifeType_LIFE_TYPE_Cooking] = map[uint32]*model.LifeRecipe{
		1: {ID: 1, Level: 1, IsShow: true, NeedTime: 1, EnergyID: 1, EnergyCount: 1,
			Materials: []model.LifeMaterial{{ItemID: 2, Count: 2}}, Probability: 1, OutputID: 3},
	}
	lifeConfigLoader = func() (*model.LifeConfig, error) { return cfg, nil }
	lifeSavePlayer = save
	conn := &lifeTestConn{}
	player := &model.Player{
		UserId: 7, LoginUUID: "life-session", Conn: conn,
		Item: &model.ItemModel{ItemBaseInfo: map[uint32]*model.ItemBaseInfo{
			1: {ItemId: 1, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Currency},
			2: {ItemId: 2, Num: 2, ItemType: proto.EBagItemTag_EBagItemTag_Material},
		}},
	}
	return &Game{}, player, conn, cfg
}

func lifeProtocolCount(player *model.Player, id uint32) int64 {
	if item := player.GetItemModel().GetItemBaseInfo(id); item != nil {
		return item.Num
	}
	return 0
}

func TestCookingFoodProtocolSaveFailureRollsBackAndRetries(t *testing.T) {
	saves := 0
	game, player, conn, _ := lifeProtocolFixture(t, func(*model.Player) error {
		saves++
		if saves == 1 {
			return errors.New("injected save failure")
		}
		return nil
	})
	msg := &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 1, MsgId: 10, SeqId: 9}, Body: &proto.CookingFoodReq{FoodId: 1}}
	game.CookingFood(player, msg)
	rsp, ok := conn.sent[0].(*proto.CookingFoodRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_SYS_ERR || rsp.GetLifeCommonResult() != nil {
		t.Fatalf("save failure response=%+v", conn.sent)
	}
	if saves != 1 || lifeProtocolCount(player, 1) != 1 || lifeProtocolCount(player, 2) != 2 || lifeProtocolCount(player, 3) != 0 ||
		player.GetLifeModel().GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency != 0 || len(player.GetLifeModel().Receipts) != 0 {
		t.Fatalf("failed save partially committed: saves=%d player=%+v", saves, player)
	}
	if len(conn.sent) != 1 {
		t.Fatalf("failed production emitted success notices: %d", len(conn.sent))
	}

	game.CookingFood(player, msg)
	rsp = conn.sent[len(conn.sent)-1].(*proto.CookingFoodRsp)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK || !rsp.GetLifeCommonResult().GetIsSuccess() || saves != 2 ||
		lifeProtocolCount(player, 1) != 0 || lifeProtocolCount(player, 2) != 0 || lifeProtocolCount(player, 3) != 1 ||
		player.GetLifeModel().GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency != 10 {
		t.Fatalf("retry did not commit exactly once: rsp=%+v saves=%d player=%+v", rsp, saves, player)
	}
}

func TestCookingFoodProtocolRejectsMaterialAndRecipeFaults(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*model.Player, *model.LifeConfig)
		recipe uint32
		want   proto.StatusCode
	}{
		{name: "insufficient material", mutate: func(player *model.Player, _ *model.LifeConfig) {
			player.GetItemModel().GetItemBaseInfo(2).Num = 1
		}, recipe: 1, want: proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH},
		{name: "invalid recipe", mutate: func(*model.Player, *model.LifeConfig) {}, recipe: 999, want: proto.StatusCode_StatusCode_CFG_NOT_FOUND},
	} {
		t.Run(test.name, func(t *testing.T) {
			saves := 0
			game, player, conn, cfg := lifeProtocolFixture(t, func(*model.Player) error { saves++; return nil })
			test.mutate(player, cfg)
			beforeEnergy, beforeMaterial := lifeProtocolCount(player, 1), lifeProtocolCount(player, 2)
			game.CookingFood(player, &alg.GameMsg{
				PacketHead: &proto.PacketHead{PacketId: 2, MsgId: 10, SeqId: 1},
				Body:       &proto.CookingFoodReq{FoodId: test.recipe},
			})
			rsp, ok := conn.sent[0].(*proto.CookingFoodRsp)
			if !ok || rsp.GetStatus() != test.want || rsp.GetLifeCommonResult() != nil || saves != 0 ||
				lifeProtocolCount(player, 1) != beforeEnergy || lifeProtocolCount(player, 2) != beforeMaterial || lifeProtocolCount(player, 3) != 0 ||
				player.GetLifeModel().GetState(proto.LifeType_LIFE_TYPE_Cooking).Proficiency != 0 || len(conn.sent) != 1 {
				t.Fatalf("fault was not atomic: rsp=%+v saves=%d player=%+v messages=%d", rsp, saves, player, len(conn.sent))
			}
		})
	}
}
