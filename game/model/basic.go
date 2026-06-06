package model

import (
	"gucooing/lolo/db"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (s *Player) GetPbPlayerDropRateInfo() *proto.PlayerDropRateInfo {
	info := &proto.PlayerDropRateInfo{
		KillDropRate:     100,
		TreasureDropRate: 100,
	}
	return info
}

func (s *Player) GetUnlockFunctions() []uint32 {
	list := make([]uint32, 0)
	for _, v := range gdconf.GetPlayerUnlockMap() {
		list = append(list, uint32(v.ID))
	}
	return list
}

func GetPlayerAppearance(userId uint32) *proto.PlayerAppearance {
	basic, ok := db.GetGameBasic(userId)
	if !ok {
		log.Game.Errorf("userId:%v func GetPlayerAppearance 玩家不存在", userId)
		return nil
	}
	return &proto.PlayerAppearance{
		AvatarFrame: basic.AvatarFrame,
		Pendant:     basic.Pendant,
	}
}

func (s *Player) UpBasicByTeam() error {
	characterId := s.GetTeamModel().GetTeamInfo().Char1
	badgeId := s.GetCharacterModel().GetCharacterInfo(characterId).GetPbCharacterAppearance().GetBadge()
	err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.CharacterId = characterId
		basic.TeamLeaderBadge = badgeId
		return true
	})
	return err
}
