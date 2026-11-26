package game

import (
	"errors"
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) ChatUnLockExpressionNotice(s *model.Player) {
	notice := &proto.ChatUnLockExpressionNotice{
		Status:       proto.StatusCode_StatusCode_OK,
		ExpressionId: s.GetChatModel().GetUnLockExpression(),
	}
	defer g.send(s, 0, notice)
}

func (g *Game) PrivateChatOfflineNotice(s *model.Player) {
	notice := &proto.PrivateChatOfflineNotice{
		Status:     proto.StatusCode_StatusCode_OK,
		OfflineMsg: make([]*proto.PrivateChatOffline, 0),
	}
	defer g.send(s, 0, notice)
	privates, err := db.GetAllChatPrivate(s.UserId)
	if err != nil {
		notice.Status = proto.StatusCode_StatusCode_CHAT_CHANNEL_NOT_EXIST
		log.Game.Warnf("UserID:%v func db.GetAllChatPrivate err:%v", s.UserId, err)
		return
	}
	for _, private := range privates {
		alg.AddList(&notice.OfflineMsg, s.GetPrivateChatOffline(private))
	}
}

func (g *Game) PrivateChatMsgRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PrivateChatMsgRecordReq)
	rsp := &proto.PrivateChatMsgRecordRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		MsgRecord: make([]*proto.ChatMsgData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	// 历史只按会话双方鉴权，删除好友或拉黑后仍保留可查询的历史。
	privateMsgs, err := db.GetAllChatPrivateMsg(s.UserId, req.TargetPlayerId)
	if err != nil {
		log.Game.Warnf("UserId:%v db.GetAllChatPrivateMsg err:%v", s.UserId, err)
		return
	}
	for _, privateMsg := range privateMsgs {
		alg.AddList(&rsp.MsgRecord,
			model.GetUserChatMsgData(privateMsg.OFChatMsg, privateMsg.UserId))
	}
}

func (g *Game) ChangeChatChannel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeChatChannelReq)
	rsp := &proto.ChangeChatChannelRsp{
		Status:    proto.StatusCode_StatusCode_OK,
		ChannelId: req.ChannelId,
	}
	defer g.send(s, msg.PacketId, rsp)
	chatChannel := g.getChatInfo().getChannelUser(s)
	if chatChannel.channel != nil {
		chatChannel.channel.delUserChan <- s.UserId
	}
	channel := g.getChatInfo().getChatChannel(req.ChannelId)
	if channel == nil {
		rsp.Status = proto.StatusCode_StatusCode_CHAT_CHANNEL_NOT_EXIST
		log.Game.Errorf("UserId:%v ChatChannel:%v 聊天房间不存在chatChannel.channel", s.UserId, req.ChannelId)
		return
	}
	channel.addUserChan <- chatChannel
}

func (g *Game) SendChatMsg(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SendChatMsgReq)
	rsp := &proto.SendChatMsgRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Text:   req.Text,
	}
	defer g.send(s, msg.PacketId, rsp)
	chatMsg := &db.OFChatMsg{
		SendTime:   time.Now().UnixMilli(),
		Text:       req.Text,
		Expression: req.Expression,
	}
	chatMsgData := model.GetUserChatMsgData(chatMsg, s.UserId)
	switch req.Type {
	case proto.ChatChannelType_ChatChannel_Default: // 默认消息是房间消息
		scenePlayer := g.getWordInfo().getScenePlayer(s)
		if scenePlayer != nil &&
			scenePlayer.channelInfo != nil &&
			scenePlayer.channelInfo.chatChannel != nil {
			scenePlayer.channelInfo.chatChannel.allSendMsgChan <- chatMsgData
		}
	case proto.ChatChannelType_ChatChannel_ChatRoom: // 聊天房间
		chatChannel := g.getChatInfo().getChannelUser(s)
		if chatChannel.channel == nil {
			log.Game.Warnf("User:%v 玩家没加入聊天房间", s.UserId)
			return
		}
		chatChannel.channel.allSendMsgChan <- chatMsgData
	case proto.ChatChannelType_ChatChannel_Private: // 私聊
		// 黑名单、双向好友、消息写入和接收者未读更新由数据库在一个事务内完成。
		privateMsg := &db.OFChatPrivateMsg{
			UserId:    s.UserId,
			OFChatMsg: chatMsg,
		}
		if err := db.CreateChatPrivateMsg(req.PlayerId, privateMsg); err != nil {
			rsp.Status = privateChatErrorStatus(err)
			log.Game.Warnf("UserId:%v db.CreateChatPrivateMsg err:%v", s.UserId, err)
			return
		}
		// 如果在线就通知过去(在主循环上直接通知,避免 go 起协程碰 live Player 造成竞态)
		if user := g.GetUser(req.PlayerId); user != nil {
			g.ChatPrivateMsgNotice(user, chatMsgData)
		}
	}
}

func privateChatErrorStatus(err error) proto.StatusCode {
	switch {
	case errors.Is(err, db.ErrPrivateChatBlocked):
		return proto.StatusCode_StatusCode_FRIEND_BLACK
	case errors.Is(err, db.ErrPrivateChatNotAllowed), errors.Is(err, db.ErrInvalidSocialTarget):
		return proto.StatusCode_StatusCode_FRIEND_NOT_EXIST
	default:
		return proto.StatusCode_StatusCode_CHAT_CHANNEL_NOT_EXIST
	}
}

// 历史消息同步通知
func (g *Game) ChatMsgRecordInitNotice(s *model.Player, msgs []*proto.ChatMsgData, t proto.ChatChannelType) {
	notice := &proto.ChatMsgRecordInitNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Type:   t,
		Msg:    msgs,
	}
	g.send(s, 0, notice)
}

// 实时消息通知
func (g *Game) ChatPrivateMsgNotice(s *model.Player, msg *proto.ChatMsgData) {
	notice := &proto.ChatMsgNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Type:   proto.ChatChannelType_ChatChannel_Private,
		Msg:    msg,
	}
	g.send(s, 0, notice)
}
