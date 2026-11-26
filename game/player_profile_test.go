package game

import (
	"errors"
	"math"
	"net"
	"testing"
	"time"

	protobuf "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

type profileTestConn struct {
	sent []protobuf.Message
}

func (c *profileTestConn) Read() (*alg.GameMsg, error)         { return nil, errors.New("not implemented") }
func (c *profileTestConn) SendBin(uint32, uint32, []byte)      {}
func (c *profileTestConn) Send(_ uint32, msg protobuf.Message) { c.sent = append(c.sent, msg) }
func (c *profileTestConn) SetUID(uint32)                       {}
func (c *profileTestConn) GetSeqId() uint32                    { return 0 }
func (c *profileTestConn) SetServerTag(string)                 {}
func (c *profileTestConn) Close()                              {}
func (c *profileTestConn) LocalAddr() net.Addr                 { return &net.TCPAddr{} }
func (c *profileTestConn) RemoteAddr() net.Addr                { return &net.TCPAddr{} }
func (c *profileTestConn) last(t *testing.T) protobuf.Message {
	t.Helper()
	if len(c.sent) == 0 {
		t.Fatal("no response sent")
	}
	return c.sent[len(c.sent)-1]
}

type profileFixture struct {
	player   *model.Player
	profile  *model.ProfileModel
	game     *Game
	conn     *profileTestConn
	now      time.Time
	level    uint32
	saves    int
	saveErr  error
	items    map[uint32]*excel.ItemConfigure
	globals  map[string]*excel.GlobalConfigConfigure
	levels   *excel.AllLevelDatas
	heads    *excel.AllHeadDatas
	phones   *excel.AllPhoneDatas
	actions  *excel.AllActionDatas
	exprs    *excel.AllExpressionDatas
	pools    map[uint32]*excel.RewardPoolConfigure
	itemPool map[uint32]*excel.RewardItemPoolConfigure
}

func newProfileFixture(t *testing.T) *profileFixture {
	t.Helper()
	f := &profileFixture{
		now:   time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC),
		level: 5,
		items: map[uint32]*excel.ItemConfigure{
			11:    {ID: 11, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Energy)},
			102:   {ID: 102, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Currency)},
			200:   {ID: 200, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Material)},
			41101: {ID: 41101, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Head)},
			7000:  {ID: 7000, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_PhoneCase)},
			2020:  {ID: 2020, NewBagItemTag: int32(proto.EBagItemTag_EBagItemTag_Item)},
		},
		globals: map[string]*excel.GlobalConfigConfigure{
			"Player_Recover_Vitality_Second": {Key: "Player_Recover_Vitality_Second", Values: []string{"600"}},
			"Player_Energy_Buy_Times": {Key: "Player_Energy_Buy_Times", Values: []string{
				"102:30:200", "102:40:200",
			}},
		},
		levels: &excel.AllLevelDatas{
			PlayerLevel: &excel.PlayerLevelDatas{Datas: []*excel.PlayerLevelConfigure{
				{ID: 1, PlayerEnergy: 600},
				{ID: 2, PlayerEnergy: 600, RewardID: 10},
				{ID: 5, PlayerEnergy: 620},
			}},
			WorldLevel: &excel.WorldLevelDatas{Datas: []*excel.WorldLevelConfigure{
				{ID: 1, Level: 0},
				{ID: 2, Level: 1, WorldLevelInfo: []*excel.WorldLevelInfo{{AchieveID: 100, FlagID: 200}}},
				{ID: 3, Level: 2},
			}},
		},
		heads: &excel.AllHeadDatas{Head: &excel.HeadDatas{Datas: []*excel.HeadConfigure{
			{ID: 41101, HeadID: 41101, IsShow: true},
		}}},
		phones: &excel.AllPhoneDatas{Phone: &excel.PhoneDatas{Datas: []*excel.PhoneConfigure{
			{ID: 7000, AssetPath: "PhoneCases/Test"},
		}}},
		actions: &excel.AllActionDatas{ActionInfo: &excel.ActionInfoDatas{Datas: []*excel.ActionInfoConfigure{
			{ID: 3000, AnimPrefab: "Action/Emoji/测试/ActionPlayer@测试"},
			{ID: 1130, AnimPrefab: "Action/Motion/拍手/ActionPlayer@拍手"},
			{ID: 1004, AnimPrefab: "Action/Motion/丢骰子/ActionPlayer@丢骰子"},
		}}},
		exprs: &excel.AllExpressionDatas{Expression: &excel.ExpressionDatas{Datas: []*excel.ExpressionConfigure{
			{ID: 1020, ExpressionGroupInfo: []*excel.ExpressionGroupInfo{{NeedItemID: 2020, ActionID: 1004}}},
		}}},
		pools: map[uint32]*excel.RewardPoolConfigure{
			10: {ID: 10, RewardPoolGroup: []*excel.RewardPoolGroupInfo{{RewardPoolGroupID: 100}}},
		},
		itemPool: map[uint32]*excel.RewardItemPoolConfigure{
			100: {ID: 100, RewardItemPoolGroup: []*excel.RewardItemPoolGroupInfo{{ItemID: 200, ItemMinCount: 2, ItemMaxCount: 2}}},
		},
	}
	f.profile = model.DefaultProfileModel()
	f.player = &model.Player{
		UserId: 7, NickName: "tester",
		Item: &model.ItemModel{
			ItemBaseInfo: map[uint32]*model.ItemBaseInfo{
				102:  {ItemId: 102, Num: 100, ItemType: proto.EBagItemTag_EBagItemTag_Currency},
				2020: {ItemId: 2020, Num: 1, ItemType: proto.EBagItemTag_EBagItemTag_Item},
			},
			ItemHeadMap: map[uint32]*model.ItemHeadInfo{},
		},
		Character: &model.CharacterModel{CharacterMap: map[uint32]*model.CharacterInfo{
			1: {CharacterId: 1},
		}},
		Scene: &model.SceneModel{},
	}
	f.conn = new(profileTestConn)
	f.player.Conn = f.conn
	f.game = &Game{}
	channel := &ChannelInfo{ChannelId: 1, allPlayer: make(map[uint32]*ScenePlayer)}
	scenePlayer := &ScenePlayer{
		Player: f.player, ChannelId: 1, channelInfo: channel,
		CurScene: &model.ScenePlayerInfo{Player: f.player, SceneId: 1},
	}
	channel.allPlayer[f.player.UserId] = scenePlayer
	f.game.wordInfo = &WordInfo{}
	f.game.wordInfo.allScenePlayer.Store(f.player.UserId, scenePlayer)

	oldLevelProvider := profileLevelConfigProvider
	oldHeadProvider := profileHeadConfigProvider
	oldPhoneProvider := profilePhoneConfigProvider
	oldActionProvider := profileActionConfigProvider
	oldExpressionProvider := profileExpressionProvider
	oldItemConfigure := profileItemConfigure
	oldGlobalConfigure := profileGlobalConfigure
	oldAchieveConfigure := profileAchieveConfigure
	oldRewardPool := profileRewardPool
	oldRewardItemPool := profileRewardItemPool
	oldPlayerLevel := profilePlayerLevel
	oldAccessor := profileAccessor
	oldSave := profileSavePlayer
	oldNow := profileNow
	oldRoll := profileRollInt64N
	t.Cleanup(func() {
		profileLevelConfigProvider = oldLevelProvider
		profileHeadConfigProvider = oldHeadProvider
		profilePhoneConfigProvider = oldPhoneProvider
		profileActionConfigProvider = oldActionProvider
		profileExpressionProvider = oldExpressionProvider
		profileItemConfigure = oldItemConfigure
		profileGlobalConfigure = oldGlobalConfigure
		profileAchieveConfigure = oldAchieveConfigure
		profileRewardPool = oldRewardPool
		profileRewardItemPool = oldRewardItemPool
		profilePlayerLevel = oldPlayerLevel
		profileAccessor = oldAccessor
		profileSavePlayer = oldSave
		profileNow = oldNow
		profileRollInt64N = oldRoll
	})
	profileLevelConfigProvider = func() (*excel.AllLevelDatas, error) { return f.levels, nil }
	profileHeadConfigProvider = func() (*excel.AllHeadDatas, error) { return f.heads, nil }
	profilePhoneConfigProvider = func() (*excel.AllPhoneDatas, error) { return f.phones, nil }
	profileActionConfigProvider = func() (*excel.AllActionDatas, error) { return f.actions, nil }
	profileExpressionProvider = func() (*excel.AllExpressionDatas, error) { return f.exprs, nil }
	profileItemConfigure = func(id uint32) *excel.ItemConfigure { return f.items[id] }
	profileGlobalConfigure = func(key string) *excel.GlobalConfigConfigure { return f.globals[key] }
	profileAchieveConfigure = func(id uint32) *excel.AchieveConfigure {
		if id == 100 {
			return &excel.AchieveConfigure{ID: 100, CountParam: 1}
		}
		return nil
	}
	profileRewardPool = func(id uint32) *excel.RewardPoolConfigure { return f.pools[id] }
	profileRewardItemPool = func(id uint32) *excel.RewardItemPoolConfigure { return f.itemPool[id] }
	profilePlayerLevel = func(*model.Player) (uint32, bool) { return f.level, true }
	profileAccessor = func(*model.Player) *model.ProfileModel { return f.profile }
	profileSavePlayer = func(*model.Player) error { f.saves++; return f.saveErr }
	profileNow = func() time.Time { return f.now }
	profileRollInt64N = func(n int64) int64 { return 0 }
	return f
}

func profileMessage(body protobuf.Message) *alg.GameMsg {
	return &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 9001}, Body: body}
}

func addProfileBase(f *profileFixture, id uint32, count int64, tag proto.EBagItemTag) {
	f.player.GetItemModel().GetItemBaseMap()[id] = &model.ItemBaseInfo{
		ItemId: id, Num: count, ItemType: tag, PackType: proto.PackType_PackType_Inventory,
	}
}

func TestProfileEnergyRecoveryUsesElapsedTimeAndCap(t *testing.T) {
	f := newProfileFixture(t)
	f.level = 1
	rules, ok := loadProfileEnergyRules(f.level)
	if !ok {
		t.Fatal("missing test energy rules")
	}
	addProfileBase(f, 11, 0, proto.EBagItemTag_EBagItemTag_Energy)
	f.profile.EnergyLastRecoverAt = f.now.Unix() - 2*600 - 59
	info, dirty, status := recoverProfileEnergy(f.player, f.profile, rules, f.now.Unix())
	if status != proto.StatusCode_StatusCode_Ok || !dirty || info.Num != 2 {
		t.Fatalf("recovery = %d dirty=%v status=%v", info.Num, dirty, status)
	}
	if f.profile.EnergyLastRecoverAt != f.now.Unix()-59 {
		t.Fatalf("remainder was banked incorrectly: %d", f.profile.EnergyLastRecoverAt)
	}
	f.now = f.now.Add(601 * time.Second)
	info, _, status = recoverProfileEnergy(f.player, f.profile, rules, f.now.Unix())
	if status != proto.StatusCode_StatusCode_Ok || info.Num != 3 {
		t.Fatalf("second recovery = %d status=%v", info.Num, status)
	}

	addProfileBase(f, 11, 599, proto.EBagItemTag_EBagItemTag_Energy)
	f.profile.EnergyLastRecoverAt = f.now.Unix() - 10*600
	info, _, status = recoverProfileEnergy(f.player, f.profile, rules, f.now.Unix())
	if status != proto.StatusCode_StatusCode_Ok || info.Num != rules.maxEnergy || f.profile.EnergyLastRecoverAt != f.now.Unix() {
		t.Fatalf("cap recovery = %d last=%d status=%v", info.Num, f.profile.EnergyLastRecoverAt, status)
	}
	f.now = f.now.Add(600 * time.Second)
	info, _, _ = recoverProfileEnergy(f.player, f.profile, rules, f.now.Unix())
	if info.Num != rules.maxEnergy {
		t.Fatal("capped energy exceeded the configured maximum")
	}
}

func TestProfileEnergyBuyChecksPriceCountAndLimit(t *testing.T) {
	f := newProfileFixture(t)
	addProfileBase(f, 11, 0, proto.EBagItemTag_EBagItemTag_Energy)
	f.game.PlayerEnergyBuy(f.player, profileMessage(&proto.PlayerEnergyBuyReq{}))
	first := f.conn.last(t).(*proto.PlayerEnergyBuyRsp)
	if first.Status != proto.StatusCode_StatusCode_Ok || first.EnergyBuyNum != 1 ||
		f.player.GetItemModel().GetItemBaseInfo(102).Num != 70 || f.player.GetItemModel().GetItemBaseInfo(11).Num != 200 {
		t.Fatalf("unexpected first purchase: %+v currency=%d energy=%d", first,
			f.player.GetItemModel().GetItemBaseInfo(102).Num, f.player.GetItemModel().GetItemBaseInfo(11).Num)
	}
	f.game.PlayerEnergyBuy(f.player, profileMessage(&proto.PlayerEnergyBuyReq{}))
	second := f.conn.last(t).(*proto.PlayerEnergyBuyRsp)
	if second.Status != proto.StatusCode_StatusCode_Ok || second.EnergyBuyNum != 2 ||
		f.player.GetItemModel().GetItemBaseInfo(102).Num != 30 || f.player.GetItemModel().GetItemBaseInfo(11).Num != 400 {
		t.Fatalf("unexpected second purchase: %+v", second)
	}
	f.game.PlayerEnergyBuy(f.player, profileMessage(&proto.PlayerEnergyBuyReq{}))
	third := f.conn.last(t).(*proto.PlayerEnergyBuyRsp)
	if third.Status != proto.StatusCode_StatusCode_EnergyBuyTimesLimit || third.EnergyBuyNum != 2 ||
		f.player.GetItemModel().GetItemBaseInfo(102).Num != 30 || f.player.GetItemModel().GetItemBaseInfo(11).Num != 400 {
		t.Fatalf("duplicate/over-limit purchase changed state: %+v", third)
	}
}

func TestProfileEnergyBuyRejectsMissingConfiguration(t *testing.T) {
	f := newProfileFixture(t)
	delete(f.globals, "Player_Energy_Buy_Times")
	f.game.PlayerEnergyBuy(f.player, profileMessage(&proto.PlayerEnergyBuyReq{}))
	rsp := f.conn.last(t).(*proto.PlayerEnergyBuyRsp)
	if rsp.Status != proto.StatusCode_StatusCode_EnergyBuyTimesLimit || f.saves != 0 ||
		f.player.GetItemModel().GetItemBaseInfo(102).Num != 100 {
		t.Fatalf("missing configuration was not rejected: %+v", rsp)
	}
}

func TestProfileLevelRewardIsOneTime(t *testing.T) {
	f := newProfileFixture(t)
	f.level = 2
	f.game.PlayerLevelReward(f.player, profileMessage(&proto.PlayerLevelRewardReq{Level: 2}))
	first := f.conn.last(t).(*proto.PlayerLevelRewardRsp)
	if first.Status != proto.StatusCode_StatusCode_Ok || len(first.Items) != 1 || f.player.GetItemModel().GetItemBaseInfo(200).Num != 2 {
		t.Fatalf("first reward failed: %+v", first)
	}
	f.game.PlayerLevelReward(f.player, profileMessage(&proto.PlayerLevelRewardReq{Level: 2}))
	second := f.conn.last(t).(*proto.PlayerLevelRewardRsp)
	if second.Status != proto.StatusCode_StatusCode_RepeatGetReward || f.player.GetItemModel().GetItemBaseInfo(200).Num != 2 {
		t.Fatalf("duplicate reward was granted: %+v", second)
	}
}

func TestProfileGhostHeadAndWrongItemTypeAreRejected(t *testing.T) {
	f := newProfileFixture(t)
	if validOwnedProfileHead(f.player, 41101) {
		t.Fatal("unowned head was accepted")
	}
	f.player.GetItemModel().ItemHeadMap[41101] = &model.ItemHeadInfo{
		ItemBaseInfo: &model.ItemBaseInfo{ItemId: 41101, ItemType: proto.EBagItemTag_EBagItemTag_Item},
		HeadId:       41101,
	}
	if validOwnedProfileHead(f.player, 41101) {
		t.Fatal("head with wrong item type was accepted")
	}
	f.player.GetItemModel().ItemHeadMap[41101].ItemType = proto.EBagItemTag_EBagItemTag_Head
	if !validOwnedProfileHead(f.player, 41101) {
		t.Fatal("owned configured head was rejected")
	}
}

func TestProfileWorldLevelBoundsAndPrerequisites(t *testing.T) {
	f := newProfileFixture(t)
	if status := unlockProfileWorldLevel(f.player, f.profile, 99, 5); status != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("out-of-range world level status = %v", status)
	}
	if status := unlockProfileWorldLevel(f.player, f.profile, 1, 0); status != proto.StatusCode_StatusCode_PlayerLevelLimit {
		t.Fatalf("player level prerequisite status = %v", status)
	}
	if status := unlockProfileWorldLevel(f.player, f.profile, 1, 1); status != proto.StatusCode_StatusCode_WorldLevelAchieveNotFinish {
		t.Fatalf("missing achievement status = %v", status)
	}
	f.profile.CompletedAchievements[100] = true
	if status := unlockProfileWorldLevel(f.player, f.profile, 1, 1); status != proto.StatusCode_StatusCode_WorldLevelAchieveNotFinish {
		t.Fatalf("missing flag status = %v", status)
	}
	f.profile.CompletedFlags[200] = true
	if status := unlockProfileWorldLevel(f.player, f.profile, 1, 1); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("valid unlock status = %v", status)
	}
	if status := unlockProfileWorldLevel(f.player, f.profile, 1, 1); status != proto.StatusCode_StatusCode_WorldLevelAlreadyUnlock {
		t.Fatalf("duplicate unlock status = %v", status)
	}
	if status := changeProfileWorldLevel(f.player, f.profile, 2, 5); status != proto.StatusCode_StatusCode_WorldLevelLocked {
		t.Fatalf("locked switch status = %v", status)
	}
	if status := changeProfileWorldLevel(f.player, f.profile, 1, 1); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("valid switch status = %v", status)
	}
	if status := changeProfileWorldLevel(f.player, f.profile, 1, 1); status != proto.StatusCode_StatusCode_WorldLevelNowIn {
		t.Fatalf("same-level switch status = %v", status)
	}
}

func TestProfileOutfitHideOnlyChangesSupportedFields(t *testing.T) {
	f := newProfileFixture(t)
	if status := applyProfileOutfitHide(f.player, 1, nil); status != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("nil hide info status = %v", status)
	}
	if status := applyProfileOutfitHide(f.player, 999, &proto.OutfitHideInfo{}); status != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("missing character status = %v", status)
	}
	if status := applyProfileOutfitHide(f.player, 1, &proto.OutfitHideInfo{HideOrn: true, HideBraid: true}); status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("valid hide status = %v", status)
	}
	preset := f.player.GetCharacterModel().GetCharacterInfo(1).GetOutfitPreset(0)
	if !preset.OutfitHideInfo.HideOrn || !preset.OutfitHideInfo.HideBraid {
		t.Fatal("supported hide fields were not updated")
	}
}

func TestProfileSocialActionsValidateFrequencyAndParameters(t *testing.T) {
	f := newProfileFixture(t)
	f.game.PopEmoji(f.player, profileMessage(&proto.PopEmojiReq{EmojiId: 3000}))
	if rsp := f.conn.last(t).(*proto.PopEmojiRsp); rsp.Status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("valid emoji status = %v", rsp.Status)
	}
	f.game.PopEmoji(f.player, profileMessage(&proto.PopEmojiReq{EmojiId: 3000}))
	if rsp := f.conn.last(t).(*proto.PopEmojiRsp); rsp.Status != proto.StatusCode_StatusCode_RequestsTooFrequent {
		t.Fatalf("emoji frequency status = %v", rsp.Status)
	}
	f.now = f.now.Add(socialActionCooldown)
	f.game.PopEmoji(f.player, profileMessage(&proto.PopEmojiReq{EmojiId: 1130}))
	if rsp := f.conn.last(t).(*proto.PopEmojiRsp); rsp.Status != proto.StatusCode_StatusCode_ActionNotUnlock {
		t.Fatalf("non-emoji action status = %v", rsp.Status)
	}

	f.game.Clap(f.player, profileMessage(&proto.ClapReq{MovementId: 1004}))
	if rsp := f.conn.last(t).(*proto.ClapRsp); rsp.Status != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("wrong clap action status = %v", rsp.Status)
	}
	clap := &proto.ClapReq{MovementId: 1130, Position: &proto.Vector3{X: 1}, Forward: &proto.Vector3{Z: 1}}
	f.game.Clap(f.player, profileMessage(clap))
	if rsp := f.conn.last(t).(*proto.ClapRsp); rsp.Status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("valid clap status = %v", rsp.Status)
	}
	f.game.Clap(f.player, profileMessage(clap))
	if rsp := f.conn.last(t).(*proto.ClapRsp); rsp.Status != proto.StatusCode_StatusCode_RequestsTooFrequent {
		t.Fatalf("clap frequency status = %v", rsp.Status)
	}
	f.now = f.now.Add(socialActionCooldown)
	badClap := &proto.ClapReq{MovementId: 1130, Position: &proto.Vector3{X: 1_000_001}, Forward: &proto.Vector3{Z: 1}}
	f.game.Clap(f.player, profileMessage(badClap))
	if rsp := f.conn.last(t).(*proto.ClapRsp); rsp.Status != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("out-of-range clap position status = %v", rsp.Status)
	}

	for _, pips := range []uint32{0, 7} {
		f.game.ThrowDice(f.player, profileMessage(&proto.ThrowDiceReq{DiceId: 2020, Pips: pips, Position: &proto.Vector3{X: 1}}))
		if rsp := f.conn.last(t).(*proto.ThrowDiceRsp); rsp.Status != proto.StatusCode_StatusCode_BadReq {
			t.Fatalf("invalid pips %d status = %v", pips, rsp.Status)
		}
	}
	f.game.ThrowDice(f.player, profileMessage(&proto.ThrowDiceReq{DiceId: 2020, Pips: 3}))
	if rsp := f.conn.last(t).(*proto.ThrowDiceRsp); rsp.Status != proto.StatusCode_StatusCode_BadReq {
		t.Fatalf("nil dice position status = %v", rsp.Status)
	}
	dice := &proto.ThrowDiceReq{DiceId: 2020, Pips: 3, Position: &proto.Vector3{X: 1}}
	f.game.ThrowDice(f.player, profileMessage(dice))
	if rsp := f.conn.last(t).(*proto.ThrowDiceRsp); rsp.Status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("valid dice status = %v", rsp.Status)
	}
	f.game.ThrowDice(f.player, profileMessage(dice))
	if rsp := f.conn.last(t).(*proto.ThrowDiceRsp); rsp.Status != proto.StatusCode_StatusCode_RequestsTooFrequent {
		t.Fatalf("dice frequency status = %v", rsp.Status)
	}
}

func TestProfileSocialActionsRequireScene(t *testing.T) {
	f := newProfileFixture(t)
	f.game.wordInfo = &WordInfo{}
	f.game.PopEmoji(f.player, profileMessage(&proto.PopEmojiReq{EmojiId: 3000}))
	if rsp := f.conn.last(t).(*proto.PopEmojiRsp); rsp.Status != proto.StatusCode_StatusCode_PlayerNotInChannel {
		t.Fatalf("missing scene status = %v", rsp.Status)
	}
}

func TestProfileEnergyPurchaseSaveFailureRollsBack(t *testing.T) {
	f := newProfileFixture(t)
	f.saveErr = errors.New("save failed")
	addProfileBase(f, 11, 0, proto.EBagItemTag_EBagItemTag_Energy)
	f.game.PlayerEnergyBuy(f.player, profileMessage(&proto.PlayerEnergyBuyReq{}))
	rsp := f.conn.last(t).(*proto.PlayerEnergyBuyRsp)
	if rsp.Status != proto.StatusCode_StatusCode_SysErr || f.profile.EnergyBuyNum != 0 ||
		f.player.GetItemModel().GetItemBaseInfo(102).Num != 100 || f.player.GetItemModel().GetItemBaseInfo(11).Num != 0 {
		t.Fatalf("failed save did not roll back purchase: %+v", rsp)
	}
}

func TestProfileLevelRewardSaveFailureRollsBack(t *testing.T) {
	f := newProfileFixture(t)
	f.level = 2
	f.saveErr = errors.New("save failed")
	f.game.PlayerLevelReward(f.player, profileMessage(&proto.PlayerLevelRewardReq{Level: 2}))
	rsp := f.conn.last(t).(*proto.PlayerLevelRewardRsp)
	if rsp.Status != proto.StatusCode_StatusCode_SysErr || f.profile.HasPlayerLevelReward(2) ||
		f.player.GetItemModel().GetItemBaseInfo(200) != nil {
		t.Fatalf("failed save did not roll back reward: %+v", rsp)
	}
}

func TestProfileBaseRewardValidationDoesNotPartiallyGrant(t *testing.T) {
	f := newProfileFixture(t)
	_, err := grantProfileBaseRewards(f.player, []profileItemReward{{itemID: 200, count: 1}, {itemID: 999, count: 1}})
	if err == nil || f.player.GetItemModel().GetItemBaseInfo(200) != nil {
		t.Fatal("invalid reward list partially granted items")
	}
	addProfileBase(f, 200, math.MaxInt64, proto.EBagItemTag_EBagItemTag_Material)
	_, err = grantProfileBaseRewards(f.player, []profileItemReward{{itemID: 200, count: 1}})
	if err == nil || f.player.GetItemModel().GetItemBaseInfo(200).Num != math.MaxInt64 {
		t.Fatal("overflowing reward changed the balance")
	}
}
