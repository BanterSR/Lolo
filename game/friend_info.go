package game

import (
	"gucooing/lolo/db"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetFriendBriefInfo(userId uint32, friend *db.OFFriend) *proto.FriendBriefInfo {
	basic, err := db.GetGameBasic(userId)
	if err != nil {
		log.Game.Warnf("GetGameBasic:%v func db.GetGameBasic:%v", userId, err)
		return nil
	}
	info := &proto.FriendBriefInfo{
		Alias:            "", // 别名
		Info:             g.PlayerBriefInfo(basic),
		FriendTag:        0,
		FriendIntimacy:   0,
		FriendBackground: 0,
	}
	if friend != nil {
		info.Alias = friend.Alias
		info.FriendTag = friend.FriendTag
		info.FriendIntimacy = friend.FriendIntimacy
		info.FriendBackground = friend.FriendBackground
	}

	return info
}

func (g *Game) PlayerBriefInfo(b *db.OFGameBasic) *proto.PlayerBriefInfo {
	return &proto.PlayerBriefInfo{
		PlayerId:        b.UserId,
		NickName:        b.NickName,
		Level:           b.Level,
		Head:            b.Head,
		LastLoginTime:   b.LastLoginTime,
		TeamLeaderBadge: 0,
		Sex:             b.Sex,
		PhoneBackground: b.PhoneBackground,
		IsOnline:        g.GetUser(b.UserId) != nil,
		Sign:            b.Sign,
		GuildName:       "",
		CharacterId:     0,
		CreateTime:      uint32(b.CreatedAt.Unix()),
		PlayerLabel:     0,
		GardenLikeNum:   0,
		AccountType:     0,
		Birthday:        b.Birthday,
		HideValue:       0,
		AvatarFrame:     b.AvatarFrame,
	}
}
