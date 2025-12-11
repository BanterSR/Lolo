package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

var (
	maxCapacity          = 200 // 历史消息最大容量
	syncRecordInitMsgNum = 100 // 同步历史消息数量
)

// 初始化玩家聊天
func (g *Game) chatInit(s *model.Player) {
	// 同步解锁的表情
	g.ChatUnLockExpressionNotice(s)
	// 获取私聊情况
	g.PrivateChatOfflineNotice(s)
	// 公共聊天频道
	// - 系统频道
	// - 聊天房间
	g.getChatInfo().joinChatChannel(s)
}

type ChatInfo struct {
	allChatChannel map[uint32]*ChatChannel // 聊天房间集合
	allChannelUser map[uint32]*ChannelUser // 全部房间用户
}

type ChannelUser struct {
	*model.Player
	channel *ChatChannel
}

func (g *Game) getChatInfo() *ChatInfo {
	if g.chatInfo == nil {
		chatInfo := &ChatInfo{
			allChatChannel: make(map[uint32]*ChatChannel),
			allChannelUser: make(map[uint32]*ChannelUser),
		}
		g.chatInfo = chatInfo
	}
	return g.chatInfo
}

func (c *ChatInfo) getChatChannelMap() map[uint32]*ChatChannel {
	if c.allChatChannel == nil {
		c.allChatChannel = make(map[uint32]*ChatChannel)
	}
	return c.allChatChannel
}

func (c *ChatInfo) getChatChannel(channelId uint32) *ChatChannel {
	all := c.getChatChannelMap()
	channel, ok := all[channelId]
	if !ok {
		channel = newChatChannel()
		channel.Type = proto.ChatChannelType_ChatChannel_ChatRoom
		channel.channelId = channelId
		all[channelId] = channel
		go channel.channelMainLoop()
	}
	return channel
}

func (c *ChatInfo) getChannelUserMap() map[uint32]*ChannelUser {
	if c.allChannelUser == nil {
		c.allChannelUser = make(map[uint32]*ChannelUser)
	}
	return c.allChannelUser
}

func (c *ChatInfo) getChannelUser(s *model.Player) *ChannelUser {
	all := c.getChannelUserMap()
	user, ok := all[s.UserId]
	if !ok {
		user = &ChannelUser{
			Player: s,
		}
		all[s.UserId] = user
	}
	return user
}

// 进入聊天房间
func (c *ChatInfo) joinChatChannel(s *model.Player) {
	defaultChannelId := gdconf.GetConstant().DefaultChatChannelId
	channel := c.getChatChannel(defaultChannelId)
	if channel == nil {
		log.Game.Warnf("ChannelId:%v 聊天房间获取失败,请检查默认聊天房间配置", defaultChannelId)
		return
	}
	user := c.getChannelUser(s)

	channel.addUserChan <- user
}

// ChatChannel 聊天房间对象
type ChatChannel struct {
	Type           proto.ChatChannelType   // 房间类型
	channelId      uint32                  //  房间id
	userNum        int                     // 房间人数
	userMap        map[uint32]*ChannelUser // 频道玩家列表
	msgList        []*proto.ChatMsgData    // 历史消息列表
	doneChan       chan struct{}           // 停止通道
	addUserChan    chan *ChannelUser       // 加入通道
	delUserChan    chan uint32             // 退出通道
	allSendMsgChan chan *proto.ChatMsgData // 广播消息通道
}

func newChatChannel() *ChatChannel {
	info := &ChatChannel{
		msgList:        make([]*proto.ChatMsgData, 0, maxCapacity),
		addUserChan:    make(chan *ChannelUser, 100),
		allSendMsgChan: make(chan *proto.ChatMsgData, 100),
		userMap:        make(map[uint32]*ChannelUser),
	}
	return info
}

func (c *ChatChannel) Close() {
	close(c.doneChan)
}

// 聊天房间主线程
func (c *ChatChannel) channelMainLoop() {
	for {
		select {
		case <-c.doneChan:
			return
		case user := <-c.addUserChan:
			c.addUser(user)
		case userId := <-c.delUserChan:
			c.delUser(userId)
		case msg := <-c.allSendMsgChan:
			c.allSendMsg(msg)
		}
	}
}

func (c *ChatChannel) addUser(s *ChannelUser) {
	if c.userMap[s.UserId] != nil {
		return
	}
	s.channel = c
	c.userMap[s.UserId] = s
	c.userNum++
	log.Game.Debugf("UserId:%v ChannelId:%v 进入聊天房间:%s", s.UserId, c.channelId, c.Type.String())
	// 进入通知
	if c.Type == proto.ChatChannelType_ChatChannel_ChatRoom {
		s.Conn.Send(0, &proto.ChangeChatChannelNotice{
			Status:    proto.StatusCode_StatusCode_OK,
			ChannelId: c.channelId,
		})
	}
	// 同步历史消息
	msgNum := alg.MinInt(syncRecordInitMsgNum, len(c.msgList))
	msgList := make([]*proto.ChatMsgData, 0, msgNum)
	for i := 0; i < msgNum; i++ {
		alg.AddList(&msgList, c.msgList[i])
	}
	c.ChatMsgRecordInitNotice(s.Player, msgList)
}

func (c *ChatChannel) delUser(userId uint32) {
	if c.userMap[userId] == nil {
		return
	}
	delete(c.userMap, userId)
	c.userNum--
	log.Game.Debugf("UserId:%v ChannelId:%v 退出聊天房间:%s", userId, c.channelId, c.Type.String())
}

func (c *ChatChannel) allSendMsg(msg *proto.ChatMsgData) {
	if c.userMap[msg.PlayerId] == nil {
		return
	}
	if len(c.msgList) >= maxCapacity {
		retainCount := maxCapacity * 8 / 10
		newList := make([]*proto.ChatMsgData, retainCount, maxCapacity)
		copy(newList, c.msgList[:retainCount])
		c.msgList = newList
	}
	alg.AddList(&c.msgList, msg)
	notice := &proto.ChatMsgNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Type:   c.Type,
		Msg:    msg,
	}
	for _, s := range c.userMap {
		s.Conn.Send(0, notice)
	}
}

func (c *ChatChannel) ChatMsgRecordInitNotice(s *model.Player, msgs []*proto.ChatMsgData) {
	notice := &proto.ChatMsgRecordInitNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Type:   c.Type,
		Msg:    msgs,
	}
	s.Conn.Send(0, notice)
}
