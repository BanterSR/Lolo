package gdconf

import (
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

/*
	big ssr 角色 1.00%
	400 ssr 角色 1.00%
	300 sr  角色 6.00%
	200 ssr 映像 0.8%
	100 sr  映像 92.2%
*/

type Gacha struct {
	all           *excel.AllGachaDatas
	GachaDatas    map[uint32]*GachaData
	OpenList      []*GachaData
	poolMap       map[int32]*excel.GachaPoolConfigure
	rewardPoolMap map[int32]*excel.GachaRewardPoolConfigure
}

type GachaData struct {
	Conf    *excel.GachaInfoConfigure
	Pools   map[int32]*excel.GachaRewardPoolConfigure
	BigPool *excel.GachaRewardPoolConfigure
}

func (g *GameConfig) loadGacha() {
	info := &Gacha{
		all:           new(excel.AllGachaDatas),
		GachaDatas:    make(map[uint32]*GachaData),
		OpenList:      make([]*GachaData, 0),
		poolMap:       make(map[int32]*excel.GachaPoolConfigure),
		rewardPoolMap: make(map[int32]*excel.GachaRewardPoolConfigure),
	}
	g.Excel.Gacha = info
	name := "Gacha.json"
	ReadJson(g.excelPath, name, &info.all)
	getGacha := func(gachaId uint32) *GachaData {
		data, ok := info.GachaDatas[gachaId]
		if !ok {
			data = &GachaData{
				Conf:    nil,
				Pools:   make(map[int32]*excel.GachaRewardPoolConfigure),
				BigPool: nil,
			}
			info.GachaDatas[gachaId] = data
		}
		return data
	}

	for _, conf := range info.all.GetPool().GetDatas() {
		info.poolMap[conf.ID] = conf
	}
	for _, conf := range info.all.GetRewardPool().GetDatas() {
		info.rewardPoolMap[conf.ID] = conf
	}

	for _, conf := range info.all.GetInfo().GetDatas() {
		data := getGacha(uint32(conf.ID))
		data.Conf = conf
		if isOpenGacha(conf) {
			alg.AddList(&info.OpenList, data)
		}
		data.BigPool = info.rewardPoolMap[conf.BigGuaranteePoolID]
		method, ok := info.poolMap[conf.Method1ID]
		if !ok {
			log.App.Errorf("请检查%s文件配置,卡池:%v缺少角色池:%v",
				name, conf.ID, conf.Method1ID)
			continue
		}
		for _, pool := range method.Items {
			poolInfo, okk := info.rewardPoolMap[pool.FreeGachaPoolID]
			if !okk {
				log.App.Warnf("请检查卡池Pool:%v,缺少奖励配置:%v",
					method.ID, pool.FreeGachaPoolID)
				continue
			}
			data.Pools[pool.FreeGachaPoolID] = poolInfo
		}
	}
}

func GetAllGacha() *excel.AllGachaDatas {
	return cc.Excel.Gacha.all
}

func GetOpenGachas() []*GachaData {
	return cc.Excel.Gacha.OpenList
}

func isOpenGacha(g *excel.GachaInfoConfigure) bool {
	if proto.EUIGachaType(g.NewUIGachaType) != proto.EUIGachaType_EUIGachaType_Limit {
		return true
	}
	openTime, err := time.Parse("2006-01-02 15:04:05", g.OpeningTime)
	// closTime, err := time.Parse("2006-01-02 15:04:05", g.ClosingTime)
	if err != nil {
		return false
	}
	curTime := time.Now().In(time.FixedZone("UTC+8", 8*60*60))
	// return curTime.After(openTime) && curTime.Before(closTime)
	return curTime.After(openTime)
}

func GetGachaData(gachaId uint32) *GachaData {
	return cc.Excel.Gacha.GachaDatas[gachaId]
}
