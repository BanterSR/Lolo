package game

import (
	"errors"
	"net"
	"testing"

	protobuf "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
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
		Status: proto.StatusCode_StatusCode_Ok, LifeType: proto.LifeType_LifeType_Cooking, Success: true,
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
		proto.LifeType_LifeType_Cooking, proto.LifeType_LifeType_Fishing,
		proto.LifeType_LifeType_Making, proto.LifeType_LifeType_Handicraft,
	} {
		if !model.ValidLifeType(lifeType) {
			t.Fatalf("有效生活类型被拒绝: %v", lifeType)
		}
	}
	for _, lifeType := range []proto.LifeType{proto.LifeType_LifeType_None, proto.LifeType_LifeType_Sewing, proto.LifeType_LifeType_Gathering} {
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
	if !ok || rsp.GetStatus() != proto.StatusCode_StatusCode_Ok || !rsp.GetIsCancel() || rsp.GetFishingEventResult() != nil {
		t.Fatalf("取消钓鱼响应错误: %+v", conn.sent[0])
	}
	if player.Life != nil || len(player.Item.ItemBaseInfo) != 0 {
		t.Fatal("取消钓鱼修改了玩家状态")
	}
}
