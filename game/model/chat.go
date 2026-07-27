package model

import (
	"gucooing/lolo/db"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type ChatModel struct {
	UnLockExpression map[uint32]*ItemBaseInfo `json:"unLockExpression,omitempty"` // 已解锁的表情
}

func DefaultChatModel() *ChatModel {
	info := &ChatModel{
		UnLockExpression: make(map[uint32]*ItemBaseInfo),
	}
	return info
}

func (s *Player) GetChatModel() *ChatModel {
	if s.Chat == nil {
		s.Chat = DefaultChatModel()
	}
	return s.Chat
}

func (c *ChatModel) GetUnLockExpression() []uint32 {
	if c.UnLockExpression == nil {
		c.UnLockExpression = make(map[uint32]*ItemBaseInfo)
	}
	list := make([]uint32, 0, len(c.UnLockExpression))
	for _, v := range c.UnLockExpression {
		list = append(list, v.ItemId)
	}
	return list
}

func (c *ChatModel) AddUnExpression(expression uint32) *ItemBaseInfo {
	if c.UnLockExpression == nil {
		c.UnLockExpression = make(map[uint32]*ItemBaseInfo)
	}
	item, ok := c.UnLockExpression[expression]
	if !ok {
		item = &ItemBaseInfo{
			ItemId:   expression,
			Num:      1,
			ItemType: proto.EBagItemTag_EBagItemTag_Expression,
			PackType: proto.PackType_PackType_Inventory,
		}
		c.UnLockExpression[expression] = item
	}
	return item
}

func (s *Player) GetPrivateChatOffline(private *db.OFChatPrivate) *proto.PrivateChatOffline {
	userId := private.GetSubUserID(s.UserId)
	basic, ok := db.GetGameBasic(userId)
	if !ok {
		log.Game.Warnf("UserId:%v func GetPrivateChatOffline 玩家不存在", userId)
		return nil
	}
	return &proto.PrivateChatOffline{
		PlayerId:    basic.UserId,
		Name:        basic.NickName,
		Head:        basic.Head,
		IsNewMsg:    private.IsNewMsg,
		AvatarFrame: basic.AvatarFrame,
	}
}

func GetUserChatMsgData(chatMsg *db.OFChatMsg, userId uint32) *proto.ChatMsgData {
	basic, ok := db.GetGameBasic(userId)
	if !ok {
		log.Game.Warnf("UserId:%v func GetUserChatMsgData 玩家不存在", userId)
		return nil
	}
	return &proto.ChatMsgData{
		PlayerId:    basic.UserId,
		Head:        basic.Head,
		Badge:       basic.Head,
		Name:        basic.NickName,
		Text:        chatMsg.Text,
		Expression:  chatMsg.Expression,
		SendTime:    chatMsg.SendTime,
		AvatarFrame: basic.AvatarFrame,
	}
}
