package game

import (
	"testing"

	"github.com/bytedance/sonic"

	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

func captureRequest() *proto.PetCaptureReq {
	return &proto.PetCaptureReq{MonsterId: 1001, Hp: 1, MaxHp: 50, CatcherId: 120000, CatchAreaId: 10}
}

func TestPetCaptureSuccessIsServerDecidedAndReplaySafe(t *testing.T) {
	setupCoreGlobals(t)
	setupPetConfig(t, 10000)
	p := newCoreTestPlayer()
	addCoreBase(p, 200100001, 1)
	g, conn := newCoreTestGame(p)
	req := captureRequest()

	g.PetCapture(p, coreTestMessage(req))
	rsp := conn.last(t).(*proto.PetCaptureRsp)
	if rsp.Status != proto.StatusCode_StatusCode_Ok || !rsp.Success || len(p.GetItemModel().GetItemPetInfoMap()) != 1 {
		t.Fatalf("capture response = %v/%v pets=%d", rsp.Status, rsp.Success, len(p.GetItemModel().GetItemPetInfoMap()))
	}
	if coreBaseCount(p, 200100001) != 0 {
		t.Fatal("capture did not consume configured catcher material")
	}
	pet := p.GetItemModel().GetItemPetInfoMap()[rsp.Items[len(rsp.Items)-1].GetMainItem().GetPet().GetInstanceId()]
	if pet == nil || pet.CatchSceneId != 1 || pet.CatchAreaId != 10 {
		t.Fatal("captured pet metadata not persisted")
	}

	g.PetCapture(p, coreTestMessage(req))
	if !conn.last(t).(*proto.PetCaptureRsp).Success || len(p.GetItemModel().GetItemPetInfoMap()) != 1 || coreBaseCount(p, 200100001) != 0 {
		t.Fatal("duplicate capture created another pet or charged again")
	}
}

func TestPetCaptureFailureCreatesNoPetAndIsReplaySafe(t *testing.T) {
	setupCoreGlobals(t)
	setupPetConfig(t, 0)
	p := newCoreTestPlayer()
	addCoreBase(p, 200100001, 1)
	g, conn := newCoreTestGame(p)
	req := captureRequest()

	g.PetCapture(p, coreTestMessage(req))
	if rsp := conn.last(t).(*proto.PetCaptureRsp); rsp.Status != proto.StatusCode_StatusCode_Ok || rsp.Success {
		t.Fatalf("failed capture response = %v/%v", rsp.Status, rsp.Success)
	}
	if len(p.GetItemModel().GetItemPetInfoMap()) != 0 || coreBaseCount(p, 200100001) != 0 {
		t.Fatal("failed capture produced a pet or did not consume material")
	}
	g.PetCapture(p, coreTestMessage(req))
	if len(p.GetItemModel().GetItemPetInfoMap()) != 0 || coreBaseCount(p, 200100001) != 0 {
		t.Fatal("duplicate failed capture repeated consumption")
	}
}

func TestPetCaptureRejectsInsufficientMaterialsAndWrongScene(t *testing.T) {
	setupCoreGlobals(t)
	setupPetConfig(t, 10000)
	p := newCoreTestPlayer()
	g, conn := newCoreTestGame(p)

	g.PetCapture(p, coreTestMessage(captureRequest()))
	if rsp := conn.last(t).(*proto.PetCaptureRsp); rsp.Status != proto.StatusCode_StatusCode_ItemNotEnough {
		t.Fatalf("material status = %v", rsp.Status)
	}
	if len(p.GetPetModel().Attempts) != 0 || len(p.GetItemModel().GetItemPetInfoMap()) != 0 {
		t.Fatal("rejected capture changed attempt ledger or pets")
	}

	addCoreBase(p, 200100001, 1)
	req := captureRequest()
	req.CatchAreaId = 11
	g.PetCapture(p, coreTestMessage(req))
	if rsp := conn.last(t).(*proto.PetCaptureRsp); rsp.Status != proto.StatusCode_StatusCode_SceneNotUnlock {
		t.Fatalf("scene status = %v", rsp.Status)
	}
	if coreBaseCount(p, 200100001) != 1 {
		t.Fatal("wrong-scene capture consumed material")
	}
}

func TestPetRenameLengthSensitiveContentAndUniqueness(t *testing.T) {
	setupCoreGlobals(t)
	setupPetConfig(t, 10000)
	p := newCoreTestPlayer()
	first := p.GetItemModel().AddItemPet(200001001)
	second := p.GetItemModel().AddItemPet(200001001)
	g, conn := newCoreTestGame(p)

	g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: first.InstanceId, Name: "小可爱"}))
	if rsp := conn.last(t).(*proto.PetRenameRsp); rsp.Status != proto.StatusCode_StatusCode_Ok || first.Name != "小可爱" {
		t.Fatalf("valid rename = %v/%q", rsp.Status, first.Name)
	}
	g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: first.InstanceId, Name: "小可爱"}))
	if conn.last(t).(*proto.PetRenameRsp).Status != proto.StatusCode_StatusCode_Ok {
		t.Fatal("renaming to own current name should be idempotent")
	}
	g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: second.InstanceId, Name: "小可爱"}))
	if conn.last(t).(*proto.PetRenameRsp).Status != proto.StatusCode_StatusCode_DuplicatedName {
		t.Fatal("duplicate pet name was accepted")
	}
	g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: second.InstanceId, Name: "一二三四"}))
	if conn.last(t).(*proto.PetRenameRsp).Status != proto.StatusCode_StatusCode_Ok {
		t.Fatal("name at configured rune limit was rejected")
	}

	cases := []struct {
		name   string
		status proto.StatusCode
	}{
		{"", proto.StatusCode_StatusCode_IllegalLen},
		{"一二三四五", proto.StatusCode_StatusCode_IllegalLen},
		{"www.", proto.StatusCode_StatusCode_BadWord},
		{"傻逼", proto.StatusCode_StatusCode_BadWord},
		{"a\x00b", proto.StatusCode_StatusCode_IllegalChar},
		{string([]byte{0xff}), proto.StatusCode_StatusCode_IllegalLen},
	}
	for _, tc := range cases {
		g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: second.InstanceId, Name: tc.name}))
		if rsp := conn.last(t).(*proto.PetRenameRsp); rsp.Status != tc.status {
			t.Fatalf("rename %q status = %v, want %v", tc.name, rsp.Status, tc.status)
		}
	}
	g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: 999, Name: "有效"}))
	if conn.last(t).(*proto.PetRenameRsp).Status != proto.StatusCode_StatusCode_PetNotFound {
		t.Fatal("nonexistent pet rename was accepted")
	}
}

func TestPetDecomposeValidatesAllInstancesAndUnloadsCarried(t *testing.T) {
	setupCoreGlobals(t)
	setupPetConfig(t, 10000)
	p := newCoreTestPlayer()
	pet := p.GetItemModel().AddItemPet(200001001)
	p.GetSceneModel().CurPetInstanceId = pet.InstanceId
	g, conn := newCoreTestGame(p)

	g.PetDecompose(p, coreTestMessage(&proto.PetDecomposeReq{PetInstanceIds: []uint32{pet.InstanceId, 999}}))
	if rsp := conn.last(t).(*proto.PetDecomposeRsp); rsp.Status != proto.StatusCode_StatusCode_PetNotFound {
		t.Fatalf("missing pet status = %v", rsp.Status)
	}
	if p.GetItemModel().GetItemPetInfoMap()[pet.InstanceId] == nil || p.GetSceneModel().CurPetInstanceId != pet.InstanceId || coreBaseCount(p, 200101002) != 0 {
		t.Fatal("failed batch decompose partially mutated player")
	}

	g.PetDecompose(p, coreTestMessage(&proto.PetDecomposeReq{PetInstanceIds: []uint32{pet.InstanceId}}))
	if rsp := conn.last(t).(*proto.PetDecomposeRsp); rsp.Status != proto.StatusCode_StatusCode_Ok {
		t.Fatalf("decompose status = %v", rsp.Status)
	}
	if p.GetItemModel().GetItemPetInfoMap()[pet.InstanceId] != nil || p.GetSceneModel().CurPetInstanceId != 0 || coreBaseCount(p, 200101002) != 2 {
		t.Fatal("decompose did not delete, unload, and grant configured output")
	}
	g.PetDecompose(p, coreTestMessage(&proto.PetDecomposeReq{PetInstanceIds: []uint32{pet.InstanceId}}))
	if conn.last(t).(*proto.PetDecomposeRsp).Status != proto.StatusCode_StatusCode_PetNotFound || coreBaseCount(p, 200101002) != 2 {
		t.Fatal("duplicate decompose granted output again")
	}
}

func TestPetCaptureAndRenameSurviveRelogin(t *testing.T) {
	setupCoreGlobals(t)
	setupPetConfig(t, 10000)
	p := newCoreTestPlayer()
	addCoreBase(p, 200100001, 1)
	g, _ := newCoreTestGame(p)
	req := captureRequest()
	g.PetCapture(p, coreTestMessage(req))
	pet := p.GetItemModel().GetItemPetInfoMap()[1]
	if pet == nil {
		t.Fatal("captured pet missing")
	}
	g.PetRename(p, coreTestMessage(&proto.PetRenameReq{PetInstanceId: pet.InstanceId, Name: "宠物"}))

	bin, err := sonic.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	restored := &model.Player{UserId: p.UserId}
	if err := sonic.Unmarshal(bin, restored); err != nil {
		t.Fatal(err)
	}
	g2, _ := newCoreTestGame(restored)
	g2.PetCapture(restored, coreTestMessage(req))
	if len(restored.GetItemModel().GetItemPetInfoMap()) != 1 || coreBaseCount(restored, 200100001) != 0 {
		t.Fatal("relogin capture replay was not idempotent")
	}
	if restored.GetItemModel().GetItemPetInfoMap()[1].Name != "宠物" {
		t.Fatal("pet name did not survive relogin")
	}
}
