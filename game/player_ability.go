package game

import (
	"strconv"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) AbilityBadgeList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.AbilityBadgeListRsp{
		Status:                       proto.StatusCode_StatusCode_Ok,
		AbilityBadgePages:            make([]*proto.AbilityBadgePageInfo, 0),
		AbilityBadgeAchieves:         make([]*proto.Achieve, 0),
		AbilityBadgeRewardAchieveIds: make([]uint32, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	// 徽章页面/宝箱没有独立的本地配置表，先按已解锁能力返回成就列表。
	rewarded := s.GetActivityModel().GetRewardedAbilityBadges()
	for abilityID := range gdconf.GetAbilityMap() {
		level := modelAbilityLevel(s, abilityID)
		if level == 0 {
			continue
		}
		alg.AddList(&rsp.AbilityBadgeAchieves, &proto.Achieve{
			AchieveId: abilityID,
			Count:     uint64(level),
		})
		if rewarded[abilityID] {
			rsp.AbilityBadgeRewardAchieveIds = append(rsp.AbilityBadgeRewardAchieveIds, abilityID)
		}
	}
}

func (g *Game) PlayerAbilityList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PlayerAbilityListRsp{
		Status:      proto.StatusCode_StatusCode_Ok,
		AbilityList: make([]*proto.PlayerAbility, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	for abilityID := range gdconf.GetAbilityMap() {
		alg.AddList(&rsp.AbilityList, &proto.PlayerAbility{
			AbilityId: abilityID,
			Level:     modelAbilityLevel(s, abilityID),
		})
	}
}

func modelAbilityLevel(s *model.Player, abilityID uint32) uint32 {
	value, _ := strconv.ParseUint(
		s.GetArchive().GetArchiveValue("ability:"+strconv.Itoa(int(abilityID))), 10, 32)
	return uint32(value)
}
