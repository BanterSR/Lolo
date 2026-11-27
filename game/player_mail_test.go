package game

import (
	"testing"

	protobuf "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func newMailPlayer(conn *lifeTestConn) *model.Player {
	return &model.Player{
		UserId: 1001,
		Conn:   conn,
		Item:   &model.ItemModel{ItemBaseInfo: make(map[uint32]*model.ItemBaseInfo)},
	}
}

func mailMsg(body protobuf.Message) *alg.GameMsg {
	return &alg.GameMsg{
		PacketHead: &proto.PacketHead{PacketId: 1, MsgId: 1, SeqId: 1},
		Body:       body,
	}
}

func lastMailResponse(t *testing.T, conn *lifeTestConn) *proto.OperateMailsRsp {
	t.Helper()
	for i := len(conn.sent) - 1; i >= 0; i-- {
		if rsp, ok := conn.sent[i].(*proto.OperateMailsRsp); ok {
			return rsp
		}
	}
	t.Fatal("未收到邮件操作响应")
	return nil
}

func TestGetMailsReturnsBuiltinTemplates(t *testing.T) {
	conn := &lifeTestConn{}
	player := newMailPlayer(conn)
	(&Game{}).GetMails(player, mailMsg(&proto.GetMailsReq{}))

	if len(conn.sent) != 1 {
		t.Fatalf("获取邮件应只返回一个响应, got %d", len(conn.sent))
	}
	rsp, ok := conn.sent[0].(*proto.GetMailsRsp)
	if !ok {
		t.Fatalf("响应类型错误: %T", conn.sent[0])
	}
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("获取邮件失败: %v", rsp.GetStatus())
	}
	if len(rsp.GetMails()) != len(model.TestMails) {
		t.Fatalf("邮件数量错误: got %d want %d", len(rsp.GetMails()), len(model.TestMails))
	}
	first := rsp.GetMails()[0]
	if first.GetMailId() != 1 || first.GetMailState() != proto.MailState_MailState_UNCLAIM {
		t.Fatalf("补偿邮件状态错误: %+v", first)
	}
	if len(first.GetItems()) != 2 {
		t.Fatalf("补偿邮件附件数量错误: %+v", first.GetItems())
	}
}

func stubRewardSave(t *testing.T) {
	t.Helper()
	oldSave := rewardSavePlayer
	rewardSavePlayer = func(*model.Player) error { return nil }
	t.Cleanup(func() { rewardSavePlayer = oldSave })
}

func TestClaimMailGrantsAttachmentsOnce(t *testing.T) {
	stubRewardSave(t)
	oldMail := model.TestMails[1]
	model.TestMails[1] = &proto.MailBriefData{MailId: 1, Items: []*proto.BaseItem{{ItemId: 5001, Num: 2}, {ItemId: 5001, Num: 3}}}
	t.Cleanup(func() { model.TestMails[1] = oldMail })

	conn := &lifeTestConn{}
	player := newMailPlayer(conn)
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{
		OperateType: proto.MailOperateType_MailOperateType_CLAIM,
		MailId:      1,
	}))

	rsp := lastMailResponse(t, conn)
	if rsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("领取邮件失败: %v", rsp.GetStatus())
	}
	if len(rsp.GetClaimResult()) != 2 {
		t.Fatalf("领取结果数量错误: %d", len(rsp.GetClaimResult()))
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001); got == nil || got.Num != 5 {
		t.Fatalf("附件一发放错误: %+v", got)
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001); got == nil || got.Num != 5 {
		t.Fatalf("附件二发放错误: %+v", got)
	}

	state := player.GetMailModel().GetState(1)
	if !state.Claimed || !state.Read {
		t.Fatalf("邮件状态未更新: %+v", state)
	}
	for _, mail := range rsp.GetMails() {
		if mail.GetMailId() == 1 && mail.GetMailState() != proto.MailState_MailState_CLAIMED {
			t.Fatalf("领取后邮件状态错误: %v", mail.GetMailState())
		}
	}

	// 第二次领取必须拒绝，且不能重复发货。
	conn.sent = nil
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{
		OperateType: proto.MailOperateType_MailOperateType_CLAIM,
		MailId:      1,
	}))
	again := lastMailResponse(t, conn)
	if again.GetStatus() != proto.StatusCode_StatusCode_REPEAT_GET_REWARD {
		t.Fatalf("重复领取未被拒绝: %v", again.GetStatus())
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001); got == nil || got.Num != 5 {
		t.Fatalf("重复领取重复发货: %+v", got)
	}
}

func TestReadAndDeleteMail(t *testing.T) {
	conn := &lifeTestConn{}
	player := newMailPlayer(conn)
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{
		OperateType: proto.MailOperateType_MailOperateType_READ,
		MailId:      2,
	}))
	readRsp := lastMailResponse(t, conn)
	if readRsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("读取邮件失败: %v", readRsp.GetStatus())
	}
	foundRead := false
	for _, mail := range readRsp.GetMails() {
		if mail.GetMailId() == 2 {
			foundRead = mail.GetMailState() == proto.MailState_MailState_READED
		}
	}
	if !foundRead {
		t.Fatal("公告邮件未标记为已读")
	}

	conn.sent = nil
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{
		OperateType: proto.MailOperateType_MailOperateType_DELETE_ALL_READED,
	}))
	deleteRsp := lastMailResponse(t, conn)
	if deleteRsp.GetStatus() != proto.StatusCode_StatusCode_OK {
		t.Fatalf("删除已读邮件失败: %v", deleteRsp.GetStatus())
	}
	if len(deleteRsp.GetDeletedMailIds()) != 1 || deleteRsp.GetDeletedMailIds()[0] != 2 {
		t.Fatalf("删除列表错误: %+v", deleteRsp.GetDeletedMailIds())
	}
	for _, mail := range deleteRsp.GetMails() {
		if mail.GetMailId() == 2 {
			t.Fatal("已删除邮件仍出现在列表中")
		}
	}
}

func TestClaimAllSkipsClaimedMail(t *testing.T) {
	stubRewardSave(t)
	oldMail := model.TestMails[1]
	model.TestMails[1] = &proto.MailBriefData{MailId: 1, Items: []*proto.BaseItem{{ItemId: 5001, Num: 2}, {ItemId: 5001, Num: 3}}}
	t.Cleanup(func() { model.TestMails[1] = oldMail })

	conn := &lifeTestConn{}
	player := newMailPlayer(conn)
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{
		OperateType: proto.MailOperateType_MailOperateType_CLAIM_ALL,
	}))
	first := lastMailResponse(t, conn)
	if first.GetStatus() != proto.StatusCode_StatusCode_OK || len(first.GetClaimResult()) != 2 {
		t.Fatalf("一键领取失败: status=%v items=%d", first.GetStatus(), len(first.GetClaimResult()))
	}

	conn.sent = nil
	(&Game{}).OperateMails(player, mailMsg(&proto.OperateMailsReq{
		OperateType: proto.MailOperateType_MailOperateType_CLAIM_ALL,
	}))
	second := lastMailResponse(t, conn)
	if second.GetStatus() != proto.StatusCode_StatusCode_MAIL_NO_REWARD {
		t.Fatalf("无邮件可领时状态错误: %v", second.GetStatus())
	}
	if got := player.GetItemModel().GetItemBaseInfo(5001); got == nil || got.Num != 5 {
		t.Fatalf("一键领取重复发货: %+v", got)
	}
}
