package model

import (
	"gucooing/lolo/db"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type Expression uint32

func (e Expression) ItemDetail() *proto.ItemDetail {
	info := &proto.ItemDetail{
		MainItem: &proto.ItemInfo{
			ItemId:  uint32(e),
			ItemTag: proto.EBagItemTag_EBagItemTag_Expression,
			Item: &proto.ItemInfo_BaseItem{
				BaseItem: &proto.BaseItem{
					ItemId: uint32(e),
					Num:    1,
				},
			},
		},
		PackType: proto.PackType_PackType_Inventory,
	}
	return info
}

type ChatModel struct {
	UnLockExpression []uint32 // 已解锁的表情
}

func DefaultChatModel() *ChatModel {
	info := &ChatModel{
		UnLockExpression: make([]uint32, 0),
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
		c.UnLockExpression = make([]uint32, 0)
	}
	return c.UnLockExpression
}

func (c *ChatModel) AddUnExpression(expression uint32) Expression {
	alg.AddSlice(&c.UnLockExpression, expression)
	return Expression(expression)
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
