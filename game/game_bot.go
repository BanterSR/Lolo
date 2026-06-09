package game

import (
	"errors"
	"gucooing/lolo/game/model"
	"gucooing/lolo/protocol/proto"
	"time"
)

// bot 接口实现
type BotInterface interface {
	UUID() string                               // 唯一id,将通过这个值确定bot 请保持唯一且不变
	GetBotInfo() *BotInfo                       // bot 基础信息
	Handle(s *model.Player, text string) string // chat 消息处理函数
}

type CommandInterface interface {
}

// 注册bot 传出bot的userid
func (g *Game) RegisterBot(bot BotInterface) (uint32, error) {
	re := false
	g.botCache.Range(func(_ uint32, v BotInterface) bool {
		if v.UUID() == bot.UUID() {
			re = true
			return false
		}
		return true
	})
	if re {
		return 0, errors.New("重复的Bot注册")
	}
	userId := g.onlyUserId.Add(1)

	botInfo := bot.GetBotInfo()
	botInfo.userId = userId
	g.botCache.Set(userId, bot)

	return userId, nil
}

// bot基础信息
type BotInfo struct {
	userId      uint32
	Head        uint32
	Badge       uint32
	NickName    string
	AvatarFrame uint32
	Sing        string
	GuildName   string
	Level       uint32
}

func (b *BotInfo) GetPrivateChatOffline() *proto.PrivateChatOffline {
	return &proto.PrivateChatOffline{
		PlayerId:    b.userId,
		Name:        b.NickName,
		Head:        b.Head,
		IsNewMsg:    false,
		AvatarFrame: b.AvatarFrame,
	}
}

func (b *BotInfo) GetFriendBriefInfo() *proto.FriendBriefInfo {
	return &proto.FriendBriefInfo{
		Alias:            "",
		Info:             b.GetPlayerBriefInfo(),
		FriendTag:        0,
		FriendIntimacy:   0,
		FriendBackground: 0,
	}
}

func (b *BotInfo) GetPlayerBriefInfo() *proto.PlayerBriefInfo {
	return &proto.PlayerBriefInfo{
		PlayerId:        b.userId,
		NickName:        b.NickName,
		Level:           b.Level,
		Head:            b.Head,
		LastLoginTime:   time.Now().Unix(),
		TeamLeaderBadge: 0,
		Sex:             0,
		PhoneBackground: 0,
		IsOnline:        true,
		Sign:            b.Sing,
		GuildName:       b.GuildName,
		CharacterId:     0,
		CreateTime:      0,
		PlayerLabel:     0,
		GardenLikeNum:   0,
		AccountType:     int32(proto.AccountType_AccountType_GlobalSteam),
		Birthday:        "",
		HideValue:       0,
		AvatarFrame:     b.AvatarFrame,
		PhoneCase:       0,
	}
}

func (b *BotInfo) GetUserChatMsgData(text string) *proto.ChatMsgData {
	return &proto.ChatMsgData{
		PlayerId:    b.userId,
		Head:        b.Head,
		Badge:       b.Badge,
		Name:        b.NickName,
		Text:        text,
		Expression:  0,
		SendTime:    time.Now().Add(500 * time.Millisecond).Unix(),
		AvatarFrame: b.AvatarFrame,
	}
}
