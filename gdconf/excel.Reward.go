package gdconf

import (
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
)

type Reward struct {
	all       *excel.AllRewardDatas
	Pools     map[uint32]*excel.RewardPoolConfigure
	PoolItems map[uint32]*excel.RewardItemPoolConfigure
}

func (g *GameConfig) loadReward() {
	info := &Reward{
		all:       new(excel.AllRewardDatas),
		Pools:     make(map[uint32]*excel.RewardPoolConfigure),
		PoolItems: make(map[uint32]*excel.RewardItemPoolConfigure),
	}
	g.Excel.Reward = info
	name := "Reward.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetRewardPool().GetDatas() {
		info.Pools[uint32(v.ID)] = v
	}

	for _, v := range info.all.GetRewardItemPool().GetDatas() {
		info.PoolItems[uint32(v.ID)] = v
	}
}

func GetRewardPool(id uint32) *excel.RewardPoolConfigure {
	return cc.Excel.Reward.Pools[id]
}

func GetRewardItemPool(id uint32) *excel.RewardItemPoolConfigure {
	return cc.Excel.Reward.PoolItems[id]
}

func GetRewardItemPoolByRewardId(id uint32) []*excel.RewardItemPoolGroupInfo {
	pool := make([]*excel.RewardItemPoolGroupInfo, 0)
	for _, v := range GetRewardPool(id).GetRewardPoolGroup() {
		info := GetRewardItemPool(uint32(v.RewardPoolGroupID))
		alg.AddList(&pool, info.GetRewardItemPoolGroup()...)
	}
	return pool
}
