package game

import (
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) ChatUnLockExpressionNotice(s *model.Player) {
	notice := &proto.ChatUnLockExpressionNotice{
		Status:       proto.StatusCode_StatusCode_Ok,
		ExpressionId: s.GetChatModel().GetUnLockExpression(),
	}
	defer g.send(s, 0, notice)
}

func (g *Game) PrivateChatOfflineNotice(s *model.Player) {
	notice := &proto.PrivateChatOfflineNotice{
		Status:     proto.StatusCode_StatusCode_Ok,
		OfflineMsg: make([]*proto.PrivateChatOffline, 0),
	}
	defer g.send(s, 0, notice)
	privates, err := db.GetAllChatPrivate(s.UserId)
	if err != nil {
		notice.Status = proto.StatusCode_StatusCode_ChatChannelNotExist
		log.Game.Warnf("UserID:%v func db.GetAllChatPrivate err:%v", s.UserId, err)
		return
	}
	for _, private := range privates {
		alg.AddList(&notice.OfflineMsg, s.GetPrivateChatOffline(private))
	}
	// 添加bot
	g.botCache.Range(func(_ uint32, bot BotInterface) bool {
		alg.AddList(&notice.OfflineMsg, bot.GetBotInfo().GetPrivateChatOffline())
		return true
	})
}

func (g *Game) PrivateChatMsgRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PrivateChatMsgRecordReq)
	rsp := &proto.PrivateChatMsgRecordRsp{
		Status:    proto.StatusCode_StatusCode_Ok,
		MsgRecord: make([]*proto.ChatMsgData, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	// bot判断
	if bot, ok := g.botCache.Get(req.TargetPlayerId); ok {
		rsp.MsgRecord = bot.GetBotInfo().MsgRecords(s, bot.GetMsgRecords(s.UserId))
		return
	}
	// 好友判断
	if count, err := db.GetIsFiend(s.UserId, req.TargetPlayerId); err != nil {
		log.Game.Warnf("UserId:%v db.GetIsFiend err:%v", s.UserId, err)
		return
	} else if count == 0 {
		return
	}
	// 获取聊天内容
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
		Status:    proto.StatusCode_StatusCode_Ok,
		ChannelId: req.ChannelId,
	}
	defer g.send(s, msg.PacketId, rsp)
	chatChannel := g.getChatInfo().getChannelUser(s)
	if chatChannel.channel != nil {
		chatChannel.channel.delUserChan <- s.UserId
	}
	channel := g.getChatInfo().getChatChannel(req.ChannelId)
	if channel == nil {
		rsp.Status = proto.StatusCode_StatusCode_ChatChannelNotExist
		log.Game.Errorf("UserId:%v ChatChannel:%v 聊天房间不存在chatChannel.channel", s.UserId, req.ChannelId)
		return
	}
	channel.addUserChan <- chatChannel
}

func (g *Game) SendChatMsg(s *model.Player, msg *alg.GameMsg) {
	var isBot bool
	req := msg.Body.(*proto.SendChatMsgReq)
	rsp := &proto.SendChatMsgRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Text:   req.Text,
	}
	defer func() {
		if !isBot {
			g.send(s, msg.PacketId, rsp)
		}
	}()
	chatMsg := &db.OFChatMsg{
		SendTime:   time.Now().UnixMilli(),
		Text:       req.Text,
		Expression: req.Expression,
	}
	chatMsgData := model.GetUserChatMsgData(chatMsg, s.UserId)
	switch req.Type {
	case proto.ChatChannelType_ChatChannelType_ChatChannelDefault: // 默认消息是房间消息
		scenePlayer := g.getWordInfo().getScenePlayer(s)
		if scenePlayer != nil &&
			scenePlayer.channelInfo == nil &&
			scenePlayer.channelInfo.chatChannel != nil {
			scenePlayer.channelInfo.chatChannel.allSendMsgChan <- chatMsgData
		}
	case proto.ChatChannelType_ChatChannelType_ChatChannelChatRoom: // 聊天房间
		chatChannel := g.getChatInfo().getChannelUser(s)
		if chatChannel.channel == nil {
			log.Game.Warnf("User:%v 玩家没加入聊天房间", s.UserId)
			return
		}
		chatChannel.channel.allSendMsgChan <- chatMsgData
	case proto.ChatChannelType_ChatChannelType_ChatChannelPrivate: // 私聊
		// bot判断
		bot, ok := g.botCache.Get(req.PlayerId)
		if ok {
			if req.Text == "" {
				return
			}
			isBot = true
			g.send(s, msg.PacketId, rsp)
			bot.Handle(s, req.Text)
			return
		}
		// 好友判断
		if count, err := db.GetIsFiend(s.UserId, req.PlayerId); err != nil {
			log.Game.Warnf("UserId:%v db.GetIsFiend err:%v", s.UserId, err)
			return
		} else if count == 0 {
			return
		}
		// 写入数据库
		privateMsg := &db.OFChatPrivateMsg{
			UserId:    s.UserId,
			OFChatMsg: chatMsg,
		}
		if err := db.CreateChatPrivateMsg(req.PlayerId, privateMsg); err != nil {
			log.Game.Warnf("UserId:%v db.CreateChatPrivateMsg err:%v", s.UserId, err)
			return
		}
		// 如果在线就通知过去
		if user := g.GetUser(req.PlayerId); user != nil {
			go g.ChatPrivateMsgNotice(user, chatMsgData)
		}
	}
}

// 历史消息同步通知
func (g *Game) ChatMsgRecordInitNotice(s *model.Player, msgs []*proto.ChatMsgData, t proto.ChatChannelType) {
	notice := &proto.ChatMsgRecordInitNotice{
		Status: proto.StatusCode_StatusCode_Ok,
		Type:   t,
		Msg:    msgs,
	}
	g.send(s, 0, notice)
}

// 实时消息通知
func (g *Game) ChatPrivateMsgNotice(s *model.Player, msg *proto.ChatMsgData) {
	notice := &proto.ChatMsgNotice{
		Status: proto.StatusCode_StatusCode_Ok,
		Type:   proto.ChatChannelType_ChatChannelType_ChatChannelPrivate,
		Msg:    msg,
	}
	g.send(s, 0, notice)
}
