package game

import (
	"sort"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) WorldLevelAchieveList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.WorldLevelAchieveListRsp{
		Status:            proto.StatusCode_StatusCode_OK,
		Achieves:          make([]*proto.Achieve, 0),
		UnlockWorldLevels: make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	for level, unlocked := range profile.UnlockedWorldLevels {
		if unlocked {
			rsp.UnlockWorldLevels = append(rsp.UnlockWorldLevels, level)
		}
	}
	sort.Slice(rsp.UnlockWorldLevels, func(i, j int) bool {
		return rsp.UnlockWorldLevels[i] < rsp.UnlockWorldLevels[j]
	})
}
