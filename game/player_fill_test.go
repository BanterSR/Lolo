package game

import (
	"testing"

	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

func newFillTestPlayer(conn *lifeTestConn) *model.Player {
	return &model.Player{
		UserId:   1001,
		NickName: "tester",
		Conn:     conn,
		Item:     &model.ItemModel{ItemBaseInfo: make(map[uint32]*model.ItemBaseInfo)},
	}
}

func TestGetAllCharacterEquipReturnsWornItems(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	player.Item.ItemWeaponMap = map[uint32]*model.ItemWeaponInfo{
		1: {
			ItemBaseInfo: &model.ItemBaseInfo{ItemId: 1001, ItemType: proto.EBagItemTag_EBagItemTag_Weapon},
			WeaponId:     1001,
			InstanceId:   1,
		},
	}
	player.Item.ItemArmorMap = map[uint32]*model.ItemArmorInfo{
		2: {
			ItemBaseInfo: &model.ItemBaseInfo{ItemId: 2001, ItemType: proto.EBagItemTag_EBagItemTag_Armor},
			ArmorId:      2001,
			InstanceId:   2,
		},
	}
	player.Character = &model.CharacterModel{CharacterMap: map[uint32]*model.CharacterInfo{
		101001: {
			CharacterId: 101001,
			EquipmentPresetList: map[uint32]*model.EquipmentPreset{
				0: {
					PresetIndex:      0,
					WeaponInstanceId: 1,
					Armors: map[proto.EEquipType]*model.ArmorInfo{
						proto.EEquipType_EEquipType_Head: {EquipType: proto.EEquipType_EEquipType_Head, InstanceId: 2},
					},
				},
			},
		},
	}}

	(&Game{}).GetAllCharacterEquip(player, mailMsg(&proto.GetAllCharacterEquipReq{}))
	if len(conn.sent) != 1 {
		t.Fatalf("装备总览响应数量错误: %d", len(conn.sent))
	}
	rsp, ok := conn.sent[0].(*proto.GetAllCharacterEquipRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("装备总览响应错误: %+v", conn.sent[0])
	}
	if len(rsp.GetItems()) != 2 {
		t.Fatalf("装备总览数量错误: %d", len(rsp.GetItems()))
	}
}

func TestPlayerVitalityReturnsOwnedEnergyItems(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	player.Item.ItemBaseInfo[2001] = &model.ItemBaseInfo{
		ItemId: 2001, Num: 3, ItemType: proto.EBagItemTag_EBagItemTag_Vitality,
	}
	player.Item.ItemBaseInfo[2002] = &model.ItemBaseInfo{
		ItemId: 2002, Num: 5, ItemType: proto.EBagItemTag_EBagItemTag_Energy,
	}
	player.Item.ItemBaseInfo[101] = &model.ItemBaseInfo{
		ItemId: 101, Num: 100, ItemType: proto.EBagItemTag_EBagItemTag_Currency,
	}

	(&Game{}).PlayerVitality(player, mailMsg(&proto.PlayerVitalityReq{}))
	rsp, ok := conn.sent[0].(*proto.PlayerVitalityRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("体力响应错误: %+v", conn.sent[0])
	}
	if len(rsp.GetItems()) != 2 {
		t.Fatalf("体力物品数量错误: %d", len(rsp.GetItems()))
	}
}

func TestPhotoShareSearchFiltersStoredPhotos(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	g := &Game{}

	g.UploadPhotoShare(player, mailMsg(&proto.UploadPhotoShareReq{
		File:  []byte("fake-image"),
		Title: "海边日落",
		Tag1:  "风景",
	}))
	if _, ok := conn.sent[0].(*proto.UploadPhotoShareRsp); !ok {
		t.Fatalf("上传照片响应类型错误: %T", conn.sent[0])
	}

	conn.sent = nil
	g.PhotoShareSearch(player, mailMsg(&proto.PhotoShareSearchReq{
		SearchType: proto.PhotoSearchType_PhotoSearchType_MyShare,
		TextType:   proto.PhotoSearchTextType_PhotoSearchTextType_Text,
		Text:       "日落",
	}))
	rsp, ok := conn.sent[0].(*proto.PhotoShareSearchRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("照片搜索响应错误: %+v", conn.sent[0])
	}
	if rsp.GetTotalNum() != 1 || len(rsp.GetPhotos()) != 1 {
		t.Fatalf("照片搜索数量错误: total=%d photos=%d", rsp.GetTotalNum(), len(rsp.GetPhotos()))
	}

	conn.sent = nil
	g.PhotoShareSearch(player, mailMsg(&proto.PhotoShareSearchReq{
		SearchType: proto.PhotoSearchType_PhotoSearchType_MyShare,
		TextType:   proto.PhotoSearchTextType_PhotoSearchTextType_Text,
		Text:       "不存在",
	}))
	empty, ok := conn.sent[0].(*proto.PhotoShareSearchRsp)
	if !ok || empty.GetTotalNum() != 0 {
		t.Fatalf("无匹配照片时结果错误: %+v", conn.sent[0])
	}
}

func TestSPOperatePasswordFlow(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	g := &Game{}

	g.SPOperate(player, mailMsg(&proto.SPOperateReq{
		OperateType: proto.SPOperateType_SPOperateType_Open,
		Password:    "1234",
	}))
	openRsp, ok := conn.sent[len(conn.sent)-1].(*proto.SPOperateRsp)
	if !ok || !openRsp.GetSpOpenStatus() {
		t.Fatalf("开启空间失败: %+v", conn.sent)
	}

	conn.sent = nil
	g.SPOperate(player, mailMsg(&proto.SPOperateReq{
		OperateType: proto.SPOperateType_SPOperateType_Check,
		Password:    "bad",
	}))
	badRsp, ok := conn.sent[len(conn.sent)-1].(*proto.SPOperateRsp)
	if !ok || badRsp.GetStatus() != proto.StatusCode_StatusCode_BadReq || badRsp.GetSpCheckStatus() {
		t.Fatalf("错误密码未被拒绝: %+v", badRsp)
	}

	conn.sent = nil
	g.SPOperate(player, mailMsg(&proto.SPOperateReq{
		OperateType: proto.SPOperateType_SPOperateType_Check,
		Password:    "1234",
	}))
	checkRsp, ok := conn.sent[len(conn.sent)-1].(*proto.SPOperateRsp)
	if !ok || !checkRsp.GetSpCheckStatus() {
		t.Fatalf("正确密码校验失败: %+v", checkRsp)
	}

	conn.sent = nil
	g.SPOperate(player, mailMsg(&proto.SPOperateReq{
		OperateType: proto.SPOperateType_SPOperateType_Close,
	}))
	closeRsp, ok := conn.sent[len(conn.sent)-1].(*proto.SPOperateRsp)
	if !ok || closeRsp.GetSpOpenStatus() {
		t.Fatalf("关闭空间失败: %+v", closeRsp)
	}
}

func TestDoLikesDailyLimit(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	g := &Game{}
	var last *proto.DoLikesRsp
	for i := 0; i < 12; i++ {
		conn.sent = nil
		g.DoLikes(player, mailMsg(&proto.DoLikesReq{}))
		rsp, ok := conn.sent[0].(*proto.DoLikesRsp)
		if !ok {
			t.Fatalf("点赞响应类型错误: %T", conn.sent[0])
		}
		last = rsp
	}
	if last.GetLikeNums() != 10 || last.GetLeftLikeNums() != 0 {
		t.Fatalf("每日点赞上限错误: like=%d left=%d", last.GetLikeNums(), last.GetLeftLikeNums())
	}
}

func TestPhotoUploadLimitUsesPurchasedGrids(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	player.GetArchive().SetArchiveMap("photo:uploadMax", "1")
	g := &Game{}

	g.UploadPhotoShare(player, mailMsg(&proto.UploadPhotoShareReq{File: []byte("a"), Title: "first"}))
	first, ok := conn.sent[0].(*proto.UploadPhotoShareRsp)
	if !ok || first.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("第一张照片上传失败: %+v", conn.sent[0])
	}

	conn.sent = nil
	g.UploadPhotoShare(player, mailMsg(&proto.UploadPhotoShareReq{File: []byte("b"), Title: "second"}))
	second, ok := conn.sent[0].(*proto.UploadPhotoShareRsp)
	if !ok || second.GetStatus() != proto.StatusCode_StatusCode_PsUploadLimit {
		t.Fatalf("上传上限未生效: %+v", conn.sent[0])
	}

	conn.sent = nil
	g.PhotoSharePaymentAddUploadGrid(player, mailMsg(&proto.PhotoSharePaymentAddUploadGridReq{}))
	if player.GetArchive().GetArchiveValue("photo:uploadMax") != "11" {
		t.Fatalf("扩容后上限错误: %s", player.GetArchive().GetArchiveValue("photo:uploadMax"))
	}
}

func TestChallengeFriendRankReturnsLocalRecords(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	player.GetArchive().SetArchiveMap("challenge:1001", "2:35")
	g := &Game{}
	g.ChallengeFriendRank(player, mailMsg(&proto.ChallengeFriendRankReq{}))
	rsp, ok := conn.sent[0].(*proto.ChallengeFriendRankRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("挑战排行响应错误: %+v", conn.sent[0])
	}
	if len(rsp.GetSelfChallenge().GetChallengeInfos()) != 1 ||
		rsp.GetSelfChallenge().GetChallengeInfos()[0].GetChallengeId() != 1001 ||
		rsp.GetSelfChallenge().GetChallengeInfos()[0].GetUseTime() != 35 {
		t.Fatalf("挑战记录错误: %+v", rsp.GetSelfChallenge())
	}
}

func TestWishListByShopIdReturnsLocalWish(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	player.GetArchive().SetArchiveMap("wish:10:20", "1")
	g := &Game{}
	g.WishListByShopId(player, mailMsg(&proto.WishListByShopIdReq{ShopId: 10, GridId: 20}))
	rsp, ok := conn.sent[0].(*proto.WishListByShopIdRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok || len(rsp.GetInfo()) != 1 {
		t.Fatalf("心愿单响应错误: %+v", conn.sent[0])
	}
}

func TestMPTeamDismissClearsTeamState(t *testing.T) {
	conn := &lifeTestConn{}
	player := newFillTestPlayer(conn)
	archive := player.GetArchive()
	archive.SetArchiveMap("mp:name", "测试小队")
	archive.SetArchiveMap("mp:mode", "2")
	archive.SetArchiveMap("mp:ready", "1")

	(&Game{}).MPTeamDismiss(player, mailMsg(&proto.MPTeamDismissReq{}))
	if archive.GetArchiveValue("mp:name") != "" || archive.GetArchiveValue("mp:ready") != "" {
		t.Fatal("解散队伍后仍残留队伍状态")
	}
	rsp, ok := conn.sent[0].(*proto.MPTeamDismissRsp)
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("解散队伍响应错误: %+v", conn.sent[0])
	}
}
