package game

import (
	"net"
	"testing"

	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

type equipmentTestConn struct {
	messages []struct {
		packetID uint32
		body     pb.Message
	}
}

func (c *equipmentTestConn) Read() (*alg.GameMsg, error)    { return nil, nil }
func (c *equipmentTestConn) SendBin(uint32, uint32, []byte) {}
func (c *equipmentTestConn) Send(packetID uint32, body pb.Message) {
	c.messages = append(c.messages, struct {
		packetID uint32
		body     pb.Message
	}{packetID: packetID, body: body})
}
func (c *equipmentTestConn) SetUID(uint32)        {}
func (c *equipmentTestConn) GetSeqId() uint32     { return 0 }
func (c *equipmentTestConn) SetServerTag(string)  {}
func (c *equipmentTestConn) Close()               {}
func (c *equipmentTestConn) LocalAddr() net.Addr  { return nil }
func (c *equipmentTestConn) RemoteAddr() net.Addr { return nil }

func newEquipmentHandlerTestPlayer() (*model.Player, *equipmentTestConn) {
	conn := new(equipmentTestConn)
	return &model.Player{
		Conn: conn,
		Item: &model.ItemModel{
			ItemWeaponMap: make(map[uint32]*model.ItemWeaponInfo),
			ItemArmorMap:  make(map[uint32]*model.ItemArmorInfo),
		},
		Character: &model.CharacterModel{CharacterMap: make(map[uint32]*model.CharacterInfo)},
	}, conn
}

func lastEquipmentResponse(t *testing.T, conn *equipmentTestConn, packetID uint32) pb.Message {
	t.Helper()
	for i := len(conn.messages) - 1; i >= 0; i-- {
		if conn.messages[i].packetID == packetID {
			return conn.messages[i].body
		}
	}
	t.Fatalf("no response for packet %d", packetID)
	return nil
}

func TestEquipmentHandlersRejectGhostInstances(t *testing.T) {
	g := new(Game)
	p, conn := newEquipmentHandlerTestPlayer()
	g.WeaponStrength(p, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 1962},
		Body:       &proto.WeaponStrengthReq{WeaponInstanceId: 999},
	})
	rsp := lastEquipmentResponse(t, conn, 1962).(*proto.WeaponStrengthRsp)
	if rsp.Status == proto.StatusCode_StatusCode_Ok {
		t.Fatal("ghost weapon strengthening returned success")
	}

	g.EquipDisassemble(p, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 1958},
		Body:       &proto.EquipDisassembleReq{WeaponIds: []uint32{999}},
	})
	rspDisassemble := lastEquipmentResponse(t, conn, 1958).(*proto.EquipDisassembleRsp)
	if rspDisassemble.Status == proto.StatusCode_StatusCode_Ok {
		t.Fatal("ghost weapon disassembly returned success")
	}
}

func TestLockEquipHandlerRejectsWrongInstanceType(t *testing.T) {
	g := new(Game)
	p, conn := newEquipmentHandlerTestPlayer()
	p.Item.ItemArmorMap[7] = &model.ItemArmorInfo{
		ItemBaseInfo: &model.ItemBaseInfo{ItemId: 2, ItemType: proto.EBagItemTag_EBagItemTag_Armor},
		ArmorId:      2,
		InstanceId:   7,
	}
	g.LockEquip(p, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 1428},
		Body: &proto.LockEquipReq{
			InstanceId: 7,
			EquipType:  proto.LockEquip_EquipType_LockEquip_EquipType_Weapon,
		},
	})
	rsp := lastEquipmentResponse(t, conn, 1428).(*proto.LockEquipRsp)
	if rsp.Status == proto.StatusCode_StatusCode_Ok {
		t.Fatal("armor instance was locked through the weapon type")
	}
}

func TestCharacterEquipPresetSwitchHandlerRequiresExistingPreset(t *testing.T) {
	g := new(Game)
	p, conn := newEquipmentHandlerTestPlayer()
	character := &model.CharacterInfo{
		CharacterId:         10,
		EquipmentPresetList: map[uint32]*model.EquipmentPreset{0: {PresetIndex: 0}},
	}
	p.Character.CharacterMap[10] = character

	g.CharacterEquipPresetSwitch(p, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 1420},
		Body:       &proto.CharacterEquipPresetSwitchReq{CharId: 10, PresetIndex: 1},
	})
	rsp := lastEquipmentResponse(t, conn, 1420).(*proto.CharacterEquipPresetSwitchRsp)
	if rsp.Status == proto.StatusCode_StatusCode_Ok || character.InUseEquipmentPresetIndex != 0 {
		t.Fatal("missing equipment preset was accepted")
	}

	character.EquipmentPresetList[1] = &model.EquipmentPreset{PresetIndex: 1}
	g.CharacterEquipPresetSwitch(p, &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 1421},
		Body:       &proto.CharacterEquipPresetSwitchReq{CharId: 10, PresetIndex: 1},
	})
	rsp = lastEquipmentResponse(t, conn, 1421).(*proto.CharacterEquipPresetSwitchRsp)
	if rsp.Status != proto.StatusCode_StatusCode_Ok || character.InUseEquipmentPresetIndex != 1 {
		t.Fatal("existing equipment preset was not activated")
	}
}
