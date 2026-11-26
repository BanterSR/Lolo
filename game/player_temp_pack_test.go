package game

import (
	"errors"
	"testing"

	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
)

func tempTestPack() *model.SceneTempPack {
	return &model.SceneTempPack{
		TempPack: make([]model.EBagItemTag, 0), TempAreaPack: make(map[uint32]*model.DropItem),
	}
}

func tempTestBase(id uint32, count int64) *model.ItemBaseInfo {
	return &model.ItemBaseInfo{
		ItemId: id, Num: count, ItemType: proto.EBagItemTag_EBagItemTag_Material,
		PackType: proto.PackType_PackType_TempStorageArea,
	}
}

func tempTestWeapon(system proto.EWeaponSystemType) *model.ItemWeaponInfo {
	return &model.ItemWeaponInfo{
		ItemBaseInfo: &model.ItemBaseInfo{
			ItemId: 1001, ItemType: proto.EBagItemTag_EBagItemTag_Weapon,
			PackType: proto.PackType_PackType_TempStorageArea,
		},
		WeaponId: 2001, WeaponSystemType: system,
	}
}

func tempTestArmor(id uint32, system proto.EWeaponSystemType) *model.ItemArmorInfo {
	return &model.ItemArmorInfo{
		ItemBaseInfo: &model.ItemBaseInfo{
			ItemId: id + 1000, ItemType: proto.EBagItemTag_EBagItemTag_Armor,
			PackType: proto.PackType_PackType_TempStorageArea,
		},
		ArmorId: id, WeaponSystemType: system,
	}
}

func tempTestPlayer() *model.Player {
	player := &model.Player{UserId: 7}
	player.GetCharacterModel().CharacterMap[1] = &model.CharacterInfo{CharacterId: 1}
	return player
}

func tempTestDeps() tempPackDeps {
	return tempPackDeps{
		characterWeaponType: func(uint32) (proto.EWeaponSystemType, bool) {
			return proto.EWeaponSystemType_EWeaponSystemType_Chopping, true
		},
		armorEquipType: func(id uint32) (proto.EEquipType, bool) {
			return proto.EEquipType(id % 4), true
		},
	}
}

func TestTempPackPickupMovesObjectsAndEnforcesCapacity(t *testing.T) {
	pack := tempTestPack()
	first, second := tempTestBase(10, 1), tempTestBase(11, 1)
	drop := pack.AddDropItems([]model.EBagItemTag{first, second})
	items, err := pickupTempPackItems(pack, drop.Index, 2)
	if err != nil || len(items) != 1 {
		t.Fatalf("single pickup items=%d err=%v", len(items), err)
	}
	if len(pack.TempPack) != 1 || pack.TempPack[0] != second || len(pack.GetTempAreaPack(drop.Index).Items) != 1 {
		t.Fatal("single pickup did not move the selected object")
	}
	items, err = pickupTempPackItems(pack, drop.Index, -1)
	if err != nil || len(items) != 1 || pack.TempPack[1] != first || pack.GetTempAreaPack(drop.Index) != nil {
		t.Fatalf("pickup-all did not move remaining object: items=%d err=%v", len(items), err)
	}

	full := tempTestPack()
	for i := 0; i < tempPackMaxSize; i++ {
		full.TempPack = append(full.TempPack, tempTestBase(uint32(i+1), 1))
	}
	drop = full.AddDropItems([]model.EBagItemTag{tempTestBase(100, 1)})
	if _, err := pickupTempPackItems(full, drop.Index, -1); !errors.Is(err, errTempPackFull) {
		t.Fatalf("full pack error = %v", err)
	}
	if len(full.TempPack) != tempPackMaxSize || len(full.GetTempAreaPack(drop.Index).Items) != 1 {
		t.Fatal("full pickup must not mutate either owner")
	}
}

func TestTempPackRejectsForgedIndicesAndDuplicateReferences(t *testing.T) {
	pack := tempTestPack()
	item := tempTestBase(10, 1)
	drop := pack.AddDropItems([]model.EBagItemTag{item, tempTestBase(11, 1)})
	for _, test := range []struct {
		drop uint32
		pick int32
	}{{0, -1}, {drop.Index, 0}, {drop.Index, 3}, {999, -1}} {
		if _, err := pickupTempPackItems(pack, test.drop, test.pick); !errors.Is(err, errTempPackIndex) {
			t.Fatalf("drop=%d pick=%d err=%v", test.drop, test.pick, err)
		}
	}
	pack.TempPack = []model.EBagItemTag{item}
	for _, index := range []uint32{0, 2, 999} {
		if _, err := dropTempPackItem(pack, index); !errors.Is(err, errTempPackIndex) {
			t.Fatalf("drop index %d err=%v", index, err)
		}
	}
	pack.TempPack = append(pack.TempPack, item)
	if _, err := tempPackSnapshot(pack); err == nil {
		t.Fatal("duplicate object references must be rejected")
	}
	if _, err := storeTempPackItems(tempTestPlayer(), pack, 7); err == nil {
		t.Fatal("duplicate store references must be rejected")
	}
}

func TestTempPackStoreMovesWithoutDuplicatingItems(t *testing.T) {
	player := tempTestPlayer()
	pack := tempTestPack()
	base, weapon := tempTestBase(10, 3), tempTestWeapon(proto.EWeaponSystemType_EWeaponSystemType_Chopping)
	drop := pack.AddDropItems([]model.EBagItemTag{base, weapon})
	if _, err := pickupTempPackItems(pack, drop.Index, -1); err != nil {
		t.Fatal(err)
	}
	items, err := storeTempPackItems(player, pack, player.UserId)
	if err != nil || len(items) != 2 {
		t.Fatalf("store items=%d err=%v", len(items), err)
	}
	if got := player.GetItemModel().GetItemBaseInfo(10); got == nil || got.Num != 3 {
		t.Fatalf("stored base = %+v", got)
	}
	if len(player.GetItemModel().GetItemWeaponMap()) != 1 || player.GetItemModel().GetItemWeaponInfo(weapon.InstanceId) != weapon {
		t.Fatal("weapon was copied or received a second instance")
	}
	if len(pack.TempPack) != 0 {
		t.Fatal("store did not clear the temporary owner")
	}
	items, err = storeTempPackItems(player, pack, player.UserId)
	if err != nil || len(items) != 0 || player.GetItemModel().GetItemBaseInfo(10).Num != 3 {
		t.Fatalf("second store duplicated items: items=%d err=%v", len(items), err)
	}
}

func TestTempPackStorePreflightsOwnershipAndInstanceCollisions(t *testing.T) {
	player := tempTestPlayer()
	pack := tempTestPack()
	unowned := tempTestBase(10, 2)
	owned := tempTestWeapon(proto.EWeaponSystemType_EWeaponSystemType_Chopping)
	owned.InstanceId = 1
	player.GetItemModel().GetItemWeaponMap()[1] = owned
	pack.TempPack = []model.EBagItemTag{unowned, owned}
	if _, err := storeTempPackItems(player, pack, 7); err == nil {
		t.Fatal("an already-owned temp item must fail the whole store")
	}
	if player.GetItemModel().GetItemBaseInfo(10) != nil || len(pack.TempPack) != 2 {
		t.Fatal("failed store preflight must not partially commit")
	}

	pack.TempPack = []model.EBagItemTag{tempTestWeapon(proto.EWeaponSystemType_EWeaponSystemType_Chopping)}
	if _, err := storeTempPackItems(player, pack, 7); err != nil {
		t.Fatal(err)
	}
	if len(player.GetItemModel().GetItemWeaponMap()) != 2 {
		t.Fatal("new weapon must receive a distinct server instance index")
	}
}

func TestTempPackStoreAndDropCheckOwnership(t *testing.T) {
	player := tempTestPlayer()
	pack := tempTestPack()
	pack.TempPack = []model.EBagItemTag{tempTestBase(10, 2)}
	if _, err := storeTempPackItems(player, pack, 8); !errors.Is(err, errTempPackOwner) {
		t.Fatalf("wrong owner error = %v", err)
	}
	if len(pack.TempPack) != 1 || player.GetItemModel().GetItemBaseInfo(10) != nil {
		t.Fatal("wrong-owner store mutated state")
	}
	if _, err := dropTempPackItem(pack, 1); err != nil {
		t.Fatal(err)
	}
	if len(pack.TempPack) != 0 || player.GetItemModel().GetItemBaseInfo(10) != nil {
		t.Fatal("drop must remove only the temporary object")
	}
}

func TestTempPackSortReindexesEntries(t *testing.T) {
	pack := tempTestPack()
	pack.TempPack = []model.EBagItemTag{tempTestBase(20, 1), tempTestBase(10, 1)}
	snapshot, err := sortTempPack(pack)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.MaxSize != tempPackMaxSize || len(snapshot.ItemInfos) != 2 ||
		snapshot.ItemInfos[0].ItemId != 10 || snapshot.ItemInfos[0].TempPackIndex != 1 ||
		snapshot.ItemInfos[1].ItemId != 20 || snapshot.ItemInfos[1].TempPackIndex != 2 {
		t.Fatalf("unexpected sorted pack: %+v", snapshot)
	}
}

func TestTempPackWearRejectsWrongEquipmentAndMovesWeapon(t *testing.T) {
	player := tempTestPlayer()
	pack := tempTestPack()
	base := tempTestBase(10, 1)
	wrong := tempTestWeapon(proto.EWeaponSystemType_EWeaponSystemType_Strike)
	pack.TempPack = []model.EBagItemTag{base, wrong}
	deps := tempTestDeps()
	if _, _, err := wearTempPackItem(player, pack, 7, 1, 1, deps); !errors.Is(err, errTempPackType) {
		t.Fatalf("base wear error = %v", err)
	}
	if _, _, err := wearTempPackItem(player, pack, 7, 2, 1, deps); !errors.Is(err, errTempPackEquip) {
		t.Fatalf("wrong weapon system error = %v", err)
	}
	if len(pack.TempPack) != 2 || len(player.GetItemModel().GetItemWeaponMap()) != 0 {
		t.Fatal("failed wear mutated inventory or temp pack")
	}

	old := tempTestWeapon(proto.EWeaponSystemType_EWeaponSystemType_Chopping)
	old.InstanceId = 99
	old.SetWearerId(1, 0)
	player.GetItemModel().GetItemWeaponMap()[99] = old
	character := player.GetCharacterModel().GetCharacterInfo(1)
	character.GetEquipmentPreset(0).WeaponInstanceId = 99
	correct := tempTestWeapon(proto.EWeaponSystemType_EWeaponSystemType_Chopping)
	pack.TempPack = []model.EBagItemTag{correct}
	snapshot, changed, err := wearTempPackItem(player, pack, 7, 1, 1, deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.ItemInfos) != 0 || len(changed) != 2 || len(player.GetItemModel().GetItemWeaponMap()) != 2 {
		t.Fatalf("unexpected wear result: temp=%d changed=%d weapons=%d", len(snapshot.ItemInfos), len(changed), len(player.GetItemModel().GetItemWeaponMap()))
	}
	if player.GetItemModel().GetItemWeaponInfo(correct.InstanceId) != correct || correct.WearerId != 1 || old.WearerId != 0 {
		t.Fatal("wear did not transfer the same weapon instance")
	}
	if character.GetEquipmentPreset(0).WeaponInstanceId != correct.InstanceId {
		t.Fatal("character preset does not reference the worn weapon")
	}
}

func TestTempPackWearValidatesArmorSlotAndType(t *testing.T) {
	player := tempTestPlayer()
	pack := tempTestPack()
	armor := tempTestArmor(1, proto.EWeaponSystemType_EWeaponSystemType_Chopping)
	pack.TempPack = []model.EBagItemTag{armor}
	_, changed, err := wearTempPackItem(player, pack, 7, 1, 1, tempTestDeps())
	if err != nil || len(changed) != 1 {
		t.Fatalf("armor wear changed=%d err=%v", len(changed), err)
	}
	if player.GetItemModel().GetItemArmorInfo(armor.InstanceId) != armor || armor.WearerId != 1 {
		t.Fatal("armor was copied or not equipped")
	}
	if got := player.GetCharacterModel().GetCharacterInfo(1).GetEquipmentPreset(0).GetArmor(proto.EEquipType_EEquipType_Chest).InstanceId; got != armor.InstanceId {
		t.Fatalf("armor slot = %d, want %d", got, armor.InstanceId)
	}
}
