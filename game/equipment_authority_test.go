package game

import (
	"testing"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func equipRequest(characterId, presetIndex, weaponId uint32, armors ...*proto.ArmorInfo) *proto.CharacterEquipUpdateReq {
	return &proto.CharacterEquipUpdateReq{
		CharId: characterId,
		EquipmentPreset: &proto.EquipmentPreset{
			PresetIndex: presetIndex, Weapon: weaponId, Armors: armors,
		},
	}
}

func applyEquip(player *model.Player, req *proto.CharacterEquipUpdateReq) *proto.CharacterEquipUpdateRsp {
	rsp := &proto.CharacterEquipUpdateRsp{}
	applyCharacterEquipment(player, req, rsp)
	return rsp
}

func TestCharacterEquipRejectsGhostAndWrongTypeAtomically(t *testing.T) {
	player := newAuthorityPlayer()
	character := player.GetCharacterModel().GetCharacterInfo(101)
	oldWeapon := addAuthorityWeapon(player, 1, 1001)
	preset := character.GetEquipmentPreset(0)
	preset.WeaponInstanceId = oldWeapon.InstanceId
	oldWeapon.SetWearerId(101, 0)
	armor := addAuthorityArmor(player, 2, 2001)

	rsp := applyEquip(player, equipRequest(101, 0, 999,
		&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Head, ArmorId: armor.InstanceId}))
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ {
		t.Fatalf("ghost weapon: %v", rsp.GetStatus())
	}
	if preset.WeaponInstanceId != oldWeapon.InstanceId || preset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId != 0 ||
		oldWeapon.WearerId != 101 || armor.WearerId != 0 {
		t.Fatal("invalid request partially changed equipment")
	}

	rsp = applyEquip(player, equipRequest(101, 0, armor.InstanceId))
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ || preset.WeaponInstanceId != oldWeapon.InstanceId {
		t.Fatalf("armor instance in weapon slot: status=%v", rsp.GetStatus())
	}
}

func TestCharacterEquipRejectsInvalidSlotsDuplicatesAndPreset(t *testing.T) {
	player := newAuthorityPlayer()
	armor := addAuthorityArmor(player, 1, 2001)
	poster := addAuthorityPoster(player, 2, 3001)

	tests := []struct {
		name string
		req  *proto.CharacterEquipUpdateReq
	}{
		{name: "nil preset", req: &proto.CharacterEquipUpdateReq{CharId: 101}},
		{name: "missing character", req: equipRequest(999, 0, 0)},
		{name: "invalid preset", req: equipRequest(101, 3, 0)},
		{name: "invalid armor slot", req: equipRequest(101, 0, 0,
			&proto.ArmorInfo{EquipType: proto.EEquipType(99), ArmorId: armor.InstanceId})},
		{name: "wrong armor slot", req: equipRequest(101, 0, 0,
			&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Chest, ArmorId: armor.InstanceId})},
		{name: "duplicate armor slot", req: equipRequest(101, 0, 0,
			&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Head, ArmorId: armor.InstanceId},
			&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Head, ArmorId: 0})},
		{name: "duplicate armor instance", req: equipRequest(101, 0, 0,
			&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Head, ArmorId: armor.InstanceId},
			&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Chest, ArmorId: armor.InstanceId})},
		{name: "invalid poster slot", req: &proto.CharacterEquipUpdateReq{CharId: 101,
			EquipmentPreset: &proto.EquipmentPreset{Posters: []*proto.PosterInfo{{PosterIndex: proto.PosterInfo_PosterIndex(99), PosterId: poster.InstanceId}}}}},
		{name: "ghost poster", req: &proto.CharacterEquipUpdateReq{CharId: 101,
			EquipmentPreset: &proto.EquipmentPreset{Posters: []*proto.PosterInfo{{PosterIndex: proto.PosterInfo_PosterIndex_First, PosterId: 999}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if rsp := applyEquip(player, test.req); rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ {
				t.Fatalf("status = %v", rsp.GetStatus())
			}
		})
	}
	if len(player.GetCharacterModel().GetCharacterInfo(101).EquipmentPresetList) != 0 {
		t.Fatal("rejected requests must not create presets")
	}
}

func TestCharacterEquipZeroUnequipsAndClearsWearer(t *testing.T) {
	player := newAuthorityPlayer()
	character := player.GetCharacterModel().GetCharacterInfo(101)
	weapon := addAuthorityWeapon(player, 1, 1001)
	armor := addAuthorityArmor(player, 2, 2001)
	poster := addAuthorityPoster(player, 3, 3001)
	preset := character.GetEquipmentPreset(0)
	preset.WeaponInstanceId = weapon.InstanceId
	preset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId = armor.InstanceId
	preset.GetPoster(proto.PosterInfo_PosterIndex_First).InstanceId = poster.InstanceId
	weapon.SetWearerId(101, 0)
	armor.SetWearer(101, 0)
	poster.SetWearer(101, 0)

	req := equipRequest(101, 0, 0,
		&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Head, ArmorId: 0})
	req.EquipmentPreset.Posters = []*proto.PosterInfo{{PosterIndex: proto.PosterInfo_PosterIndex_First, PosterId: 0}}
	rsp := applyEquip(player, req)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("unequip failed: %v", rsp.GetStatus())
	}
	if preset.WeaponInstanceId != 0 || preset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId != 0 ||
		preset.GetPoster(proto.PosterInfo_PosterIndex_First).InstanceId != 0 {
		t.Fatal("zero IDs must clear equipment slots")
	}
	if weapon.WearerId != 0 || armor.WearerId != 0 || poster.WearerId != 0 {
		t.Fatal("unequip must clear old wearer metadata")
	}
}

func TestCharacterEquipZeroClearsMissingOldInstance(t *testing.T) {
	player := newAuthorityPlayer()
	character := player.GetCharacterModel().GetCharacterInfo(101)
	preset := character.GetEquipmentPreset(0)
	preset.WeaponInstanceId = 999
	if rsp := applyEquip(player, equipRequest(101, 0, 0)); rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("unequip missing old instance: %v", rsp.GetStatus())
	}
	if preset.WeaponInstanceId != 0 {
		t.Fatal("zero request must clear a stale preset reference")
	}
}

func TestCharacterEquipCrossCharacterMoveClearsOldPreset(t *testing.T) {
	player := newAuthorityPlayer()
	first := player.GetCharacterModel().GetCharacterInfo(101)
	second := player.GetCharacterModel().GetCharacterInfo(102)
	firstPreset := first.GetEquipmentPreset(0)
	secondPreset := second.GetEquipmentPreset(1)
	oldWeapon := addAuthorityWeapon(player, 1, 1001)
	movedWeapon := addAuthorityWeapon(player, 2, 1002)
	firstPreset.WeaponInstanceId = oldWeapon.InstanceId
	secondPreset.WeaponInstanceId = movedWeapon.InstanceId
	oldWeapon.SetWearerId(101, 0)
	movedWeapon.SetWearerId(102, 1)

	rsp := applyEquip(player, equipRequest(101, 0, movedWeapon.InstanceId))
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("cross-character equip: %v", rsp.GetStatus())
	}
	if firstPreset.WeaponInstanceId != movedWeapon.InstanceId || movedWeapon.WearerId != 101 || movedWeapon.WearerIndex != 0 {
		t.Fatal("new weapon was not assigned to target character/preset")
	}
	if secondPreset.WeaponInstanceId != 0 || oldWeapon.WearerId != 0 {
		t.Fatal("old preset or replaced weapon wearer was not cleared")
	}
	characters := make(map[uint32]bool)
	for _, character := range rsp.GetCharacter() {
		characters[character.GetCharacterId()] = true
	}
	if !characters[101] || !characters[102] {
		t.Fatalf("response must contain both affected characters: %v", characters)
	}
}

func TestCharacterEquipCrossCharacterArmorAndPoster(t *testing.T) {
	player := newAuthorityPlayer()
	first := player.GetCharacterModel().GetCharacterInfo(101)
	second := player.GetCharacterModel().GetCharacterInfo(102)
	firstPreset := first.GetEquipmentPreset(0)
	secondPreset := second.GetEquipmentPreset(0)
	armor := addAuthorityArmor(player, 1, 2001)
	poster := addAuthorityPoster(player, 2, 3001)
	secondPreset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId = armor.InstanceId
	secondPreset.GetPoster(proto.PosterInfo_PosterIndex_Second).InstanceId = poster.InstanceId
	armor.SetWearer(102, 0)
	poster.SetWearer(102, 0)

	req := equipRequest(101, 0, 0,
		&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Head, ArmorId: armor.InstanceId})
	req.EquipmentPreset.Posters = []*proto.PosterInfo{{PosterIndex: proto.PosterInfo_PosterIndex_Second, PosterId: poster.InstanceId}}
	if rsp := applyEquip(player, req); rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("cross-character armor/poster: %v", rsp.GetStatus())
	}
	if secondPreset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId != 0 ||
		secondPreset.GetPoster(proto.PosterInfo_PosterIndex_Second).InstanceId != 0 {
		t.Fatal("source armor/poster slots were not cleared")
	}
	if firstPreset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId != armor.InstanceId ||
		firstPreset.GetPoster(proto.PosterInfo_PosterIndex_Second).InstanceId != poster.InstanceId ||
		armor.WearerId != 101 || poster.WearerId != 101 {
		t.Fatal("target armor/poster or wearer metadata is wrong")
	}
}

func TestCharacterEquipRejectsDuplicateWithUnchangedSlot(t *testing.T) {
	player := newAuthorityPlayer()
	character := player.GetCharacterModel().GetCharacterInfo(101)
	armor := addAuthorityArmor(player, 1, 2001)
	preset := character.GetEquipmentPreset(0)
	preset.GetArmor(proto.EEquipType_EEquipType_Head).InstanceId = armor.InstanceId
	armor.SetWearer(101, 0)
	req := equipRequest(101, 0, 0,
		&proto.ArmorInfo{EquipType: proto.EEquipType_EEquipType_Chest, ArmorId: armor.InstanceId})
	if rsp := applyEquip(player, req); rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ {
		t.Fatalf("same armor in two final slots: %v", rsp.GetStatus())
	}
	if preset.GetArmor(proto.EEquipType_EEquipType_Chest).InstanceId != 0 {
		t.Fatal("rejected duplicate changed target preset")
	}
}

func callAppearance(player *model.Player, req *proto.UpdateCharacterAppearanceReq) *proto.UpdateCharacterAppearanceRsp {
	player.Team.TeamInfo.Char1 = 0 // 外观单测不触发数据库中的队长基础信息同步。
	g := &Game{}
	g.UpdateCharacterAppearance(player, &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 43}, Body: req})
	return player.Conn.(*authorityConn).last(43).(*proto.UpdateCharacterAppearanceRsp)
}

func TestCharacterAppearanceOwnershipTypeAndUnload(t *testing.T) {
	player := newAuthorityPlayer()
	character := player.GetCharacterModel().GetCharacterInfo(101)
	axe := addAuthorityWeapon(player, 1, 1002)
	net := addAuthorityWeapon(player, 2, 1003)
	player.GetItemModel().AddItemBase(4001, 1)
	player.GetItemModel().AddItemBase(4002, 1)

	rsp := callAppearance(player, &proto.UpdateCharacterAppearanceReq{CharId: 101, Appearance: &proto.CharacterAppearance{
		Badge: 4001, UmbrellaId: 4002, LoggingAxeInstanceId: axe.InstanceId,
	}})
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK || character.CharacterAppearance.LoggingAxeInstanceId != axe.InstanceId {
		t.Fatalf("valid appearance: status=%v", rsp.GetStatus())
	}
	before := character.GetPbCharacterAppearance()
	rsp = callAppearance(player, &proto.UpdateCharacterAppearanceReq{CharId: 101, Appearance: &proto.CharacterAppearance{
		Badge: 4001, UmbrellaId: 4002, InsectNetInstanceId: axe.InstanceId,
	}})
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ || character.GetPbCharacterAppearance().String() != before.String() {
		t.Fatal("wrong tool type must be rejected without partial update")
	}
	rsp = callAppearance(player, &proto.UpdateCharacterAppearanceReq{CharId: 101, Appearance: &proto.CharacterAppearance{
		Badge: 4001, UmbrellaId: 4002, InsectNetInstanceId: net.InstanceId, LoggingAxeInstanceId: net.InstanceId,
	}})
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ {
		t.Fatal("same appearance tool in two slots must be rejected")
	}
	rsp = callAppearance(player, &proto.UpdateCharacterAppearanceReq{CharId: 101, Appearance: &proto.CharacterAppearance{
		InsectNetInstanceId: 999,
	}})
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ || character.CharacterAppearance.LoggingAxeInstanceId != axe.InstanceId {
		t.Fatal("ghost appearance instance must be rejected")
	}
	rsp = callAppearance(player, &proto.UpdateCharacterAppearanceReq{CharId: 101, Appearance: &proto.CharacterAppearance{}})
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK || character.CharacterAppearance.LoggingAxeInstanceId != 0 {
		t.Fatalf("zero appearance IDs must unload tools: status=%v", rsp.GetStatus())
	}
}

func TestCharacterAppearanceRejectsOtherPlayerInstance(t *testing.T) {
	player := newAuthorityPlayer()
	otherPlayer := newAuthorityPlayer()
	otherPlayer.UserId = 2
	axe := addAuthorityWeapon(otherPlayer, 1, 1002)
	rsp := callAppearance(player, &proto.UpdateCharacterAppearanceReq{CharId: 101, Appearance: &proto.CharacterAppearance{
		LoggingAxeInstanceId: axe.InstanceId,
	}})
	if rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ {
		t.Fatalf("other player's tool: %v", rsp.GetStatus())
	}
}

func callGatherWeaponUpdate(player *model.Player, req *proto.CharacterGatherWeaponUpdateReq) *proto.CharacterGatherWeaponUpdateRsp {
	g := &Game{}
	g.CharacterGatherWeaponUpdate(player, &alg.GameMsg{PacketHead: &proto.PacketHead{PacketId: 44}, Body: req})
	return player.Conn.(*authorityConn).last(44).(*proto.CharacterGatherWeaponUpdateRsp)
}

func TestCharacterGatherWeaponAuthority(t *testing.T) {
	player := newAuthorityPlayer()
	character := player.GetCharacterModel().GetCharacterInfo(101)
	axe := addAuthorityWeapon(player, 1, 1002)
	sword := addAuthorityWeapon(player, 2, 1001)

	if rsp := callGatherWeaponUpdate(player, &proto.CharacterGatherWeaponUpdateReq{CharacterId: 101, WeaponId: 999}); rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ || character.GatherWeapon != 0 {
		t.Fatalf("ghost gather weapon: status=%v", rsp.GetStatus())
	}
	if rsp := callGatherWeaponUpdate(player, &proto.CharacterGatherWeaponUpdateReq{CharacterId: 101, WeaponId: sword.InstanceId}); rsp.GetStatus() != proto.StatusCode_StatusCode_BAD_REQ || character.GatherWeapon != 0 {
		t.Fatalf("battle weapon as gather tool: status=%v", rsp.GetStatus())
	}
	if rsp := callGatherWeaponUpdate(player, &proto.CharacterGatherWeaponUpdateReq{CharacterId: 101, WeaponId: axe.InstanceId}); rsp.GetStatus() != proto.StatusCode_StatusCode_OK || character.GatherWeapon != axe.WeaponId {
		t.Fatalf("valid gather weapon: status=%v", rsp.GetStatus())
	}
	if rsp := callGatherWeaponUpdate(player, &proto.CharacterGatherWeaponUpdateReq{CharacterId: 101, WeaponId: 0}); rsp.GetStatus() != proto.StatusCode_StatusCode_OK || character.GatherWeapon != 0 {
		t.Fatalf("unequip gather weapon: status=%v", rsp.GetStatus())
	}
}
