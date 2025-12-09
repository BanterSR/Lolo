package model

import (
	"gucooing/lolo/gdconf"
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
