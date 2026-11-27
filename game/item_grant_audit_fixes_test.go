package game

import (
	"errors"
	"math"
	"testing"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func TestRewardBatchPreflightRejectsOverflowWithoutMutation(t *testing.T) {
	player := &model.Player{Item: &model.ItemModel{ItemBaseInfo: map[uint32]*model.ItemBaseInfo{
		5001: {ItemId: 5001, Num: math.MaxInt64 - 1, ItemType: proto.EBagItemTag_EBagItemTag_Material},
	}}}
	if status := prepareRewardItems(player, []rewardItem{{id: 5001, num: 1}, {id: 5001, num: 1}}); status == proto.StatusCode_StatusCode_OK {
		t.Fatal("combined reward overflow was accepted")
	}
	if got := player.Item.ItemBaseInfo[5001].Num; got != math.MaxInt64-1 {
		t.Fatalf("preflight mutated balance: %d", got)
	}
	if status := prepareRewardItems(player, []rewardItem{{id: 999999, num: 1}}); status == proto.StatusCode_StatusCode_OK {
		t.Fatal("unknown reward item was accepted")
	}
}

func TestRewardSnapshotSaveFailureRollsBackGrantAndClaim(t *testing.T) {
	oldSave := rewardSavePlayer
	rewardSavePlayer = func(*model.Player) error { return errors.New("disk full") }
	t.Cleanup(func() { rewardSavePlayer = oldSave })
	player := &model.Player{Item: &model.ItemModel{ItemBaseInfo: make(map[uint32]*model.ItemBaseInfo)}}
	player.GetArchive().SetArchiveMap("claim", "0")
	status := commitPlayerReward(player, func() proto.StatusCode {
		if _, status := grantPreparedRewards(player, []rewardItem{{id: 5001, num: 2}}); status != proto.StatusCode_StatusCode_OK {
			return status
		}
		player.GetArchive().SetArchiveMap("claim", "1")
		return proto.StatusCode_StatusCode_OK
	})
	if status != proto.StatusCode_StatusCode_PLAYER_NOT_FOUND {
		t.Fatalf("save failure status = %v", status)
	}
	if player.GetItemModel().GetItemBaseInfo(5001) != nil || player.GetArchive().GetArchiveValue("claim") != "0" {
		t.Fatal("failed commit retained reward or claim")
	}
}

func TestMailClaimSaveFailureRollsBackAllAttachments(t *testing.T) {
	stubRewardSave(t)
	oldSave := rewardSavePlayer
	rewardSavePlayer = func(*model.Player) error { return errors.New("disk full") }
	t.Cleanup(func() { rewardSavePlayer = oldSave })
	oldMail := model.TestMails[1]
	model.TestMails[1] = &proto.MailBriefData{MailId: 1, Items: []*proto.BaseItem{{ItemId: 5001, Num: 2}, {ItemId: 5001, Num: 3}}}
	t.Cleanup(func() { model.TestMails[1] = oldMail })
	conn := &lifeTestConn{}
	player := newMailPlayer(conn)
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{OperateType: proto.MailOperateType_MailOperateType_CLAIM, MailId: 1}))
	rsp := lastMailResponse(t, conn)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_PLAYER_NOT_FOUND || len(rsp.GetClaimResult()) != 0 {
		t.Fatalf("failed mail claim status=%v items=%d", rsp.GetStatus(), len(rsp.GetClaimResult()))
	}
	if player.GetItemModel().GetItemBaseInfo(5001) != nil || player.GetMailModel().GetState(1).Claimed {
		t.Fatal("failed mail claim retained attachments or claimed state")
	}
}

func TestMailClaimRejectsOverflowBeforeFirstAttachment(t *testing.T) {
	stubRewardSave(t)
	oldMail := model.TestMails[1]
	model.TestMails[1] = &proto.MailBriefData{MailId: 1, Items: []*proto.BaseItem{{ItemId: 5001, Num: 1}, {ItemId: 5001, Num: 1}}}
	t.Cleanup(func() { model.TestMails[1] = oldMail })
	player := newMailPlayer(&lifeTestConn{})
	player.GetItemModel().GetItemBaseMap()[5001] = &model.ItemBaseInfo{ItemId: 5001, Num: math.MaxInt64, ItemType: proto.EBagItemTag_EBagItemTag_Material}
	items, status := claimMail(player, 1)
	if status == proto.StatusCode_StatusCode_OK || items != nil {
		t.Fatalf("overflow claim status=%v items=%d", status, len(items))
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001); got == nil || got.Num != math.MaxInt64 || player.GetMailModel().GetState(1).Claimed {
		t.Fatal("overflow claim partially granted or marked claimed")
	}
}

func TestChooseGachaPoolUsesOnlyConfiguredChildren(t *testing.T) {
	parent := &gdconf.GachaData{Conf: &excel.GachaInfoConfigure{ID: 10, NewUIGachaType: 9, CardPoolRange: []int32{20, 21}}}
	child := &gdconf.GachaData{Conf: &excel.GachaInfoConfigure{ID: 20, NewUIGachaType: 9}}
	if !validChooseGachaPool(parent, child) {
		t.Fatal("configured child rejected")
	}
	for _, invalid := range []*gdconf.GachaData{
		{Conf: &excel.GachaInfoConfigure{ID: 22, NewUIGachaType: 9}},
		{Conf: &excel.GachaInfoConfigure{ID: 21, NewUIGachaType: 9, IsFree: true}},
		{Conf: &excel.GachaInfoConfigure{ID: 21, NewUIGachaType: 10}},
		{Conf: &excel.GachaInfoConfigure{ID: 10, NewUIGachaType: 9}},
	} {
		if validChooseGachaPool(parent, invalid) {
			t.Fatalf("invalid child accepted: %+v", invalid)
		}
	}
}

func TestItemSweepRejectsZeroOversizedAndOverflowingCounts(t *testing.T) {
	player := &model.Player{}
	for _, count := range []uint32{0, 10001, math.MaxUint32} {
		conn := &lifeTestConn{}
		player.Conn = conn
		(&Game{}).ItemSweep(player, &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 1}, Body: &proto.ItemSweepReq{ItemId: 5001, Count: count}})
		rsp := conn.sent[len(conn.sent)-1].(*proto.ItemSweepRsp)
		if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ {
			t.Fatalf("sweep count %d status = %v", count, rsp.GetStatus())
		}
	}
	if _, ok := checkedRewardMul(math.MaxInt64, 2); ok {
		t.Fatal("multiplication overflow accepted")
	}
	if value, ok := checkedRewardMul(3, 4); !ok || value != 12 {
		t.Fatalf("checked multiplication = %d, %v", value, ok)
	}
}

func TestValidatePlayerEconomyRejectsCorruptInventory(t *testing.T) {
	base := func(num int64) *model.Player {
		return &model.Player{Item: &model.ItemModel{ItemBaseInfo: map[uint32]*model.ItemBaseInfo{
			5001: {ItemId: 5001, Num: num, ItemType: proto.EBagItemTag_EBagItemTag_Material},
		}}}
	}
	if err := ValidatePlayerEconomy(base(1)); err != nil {
		t.Fatalf("valid inventory rejected: %v", err)
	}
	if err := ValidatePlayerEconomy(base(-1)); err == nil {
		t.Fatal("negative balance accepted")
	}
	unknown := base(1)
	unknown.Item.ItemBaseInfo[999999] = &model.ItemBaseInfo{ItemId: 999999, Num: 1}
	if err := ValidatePlayerEconomy(unknown); err == nil {
		t.Fatal("unknown item accepted")
	}
	conflict := base(1)
	conflict.Item.InstanceIndex = 1
	conflict.Item.ItemWeaponMap = map[uint32]*model.ItemWeaponInfo{1: {ItemBaseInfo: &model.ItemBaseInfo{ItemId: 1001, ItemType: proto.EBagItemTag_EBagItemTag_Weapon}, WeaponId: 1001, InstanceId: 1}}
	conflict.Item.ItemArmorMap = map[uint32]*model.ItemArmorInfo{1: {ItemBaseInfo: &model.ItemBaseInfo{ItemId: 2001, ItemType: proto.EBagItemTag_EBagItemTag_Armor}, ArmorId: 2001, InstanceId: 1}}
	if err := ValidatePlayerEconomy(conflict); err == nil {
		t.Fatal("duplicate instance ID accepted")
	}
}

func TestSandbagProgressComesFromServerArchive(t *testing.T) {
	player := &model.Player{}
	player.GetArchive().SetArchiveMap("sandbagHit:7:1", "3")
	player.GetArchive().SetArchiveMap("sandbagHit:7:2", "4")
	player.GetArchive().SetArchiveMap("sandbagHit:8:1", "100")
	if got := sandbagHitCount(player, 7); got != 7 {
		t.Fatalf("sandbag progress = %d, want 7", got)
	}
}
