package model

import (
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

type TeamModel struct {
	TeamInfo *TeamInfo
}

type TeamInfo struct {
	Char1 uint32
	Char2 uint32
	Char3 uint32
}

func (s *Player) GetTeamModel() *TeamModel {
	if s.Team == nil {
		s.Team = new(TeamModel)
	}
	return s.Team
}

func newTeamInfo() *TeamInfo {
	if len(gdconf.GetConstant().DefaultCharacter) < 1 {
		log.Game.Warnf("默认角色数量不能小于1个")
		return nil
	}
	info := &TeamInfo{
		Char1: 101001,
		Char2: 0,
		Char3: 0,
	}
	return info
}

func (t *TeamModel) GetTeamInfo() *TeamInfo {
	if t.TeamInfo == nil {
		t.TeamInfo = newTeamInfo()
	}
	return t.TeamInfo
}

func (t *TeamInfo) Team() *proto.Team {
	info := &proto.Team{
		Char1: t.Char1,
		Char2: t.Char2,
		Char3: t.Char3,
	}
	return info
}
