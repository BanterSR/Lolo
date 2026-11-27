package main

import (
	"context"
	"sync"
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

var (
	logLootErrOnce sync.Once
	logLootOKOnce  sync.Once
)

func (b *Bot) scenarioLoot(ctx context.Context) {
	if b.send(&proto.PlayerMainDataReq{}) != nil {
		return
	}
	if _, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.PlayerMainDataRsp)
		return ok
	}); err != nil {
		logOnce(&logLootErrOnce, "拾取链烟测获取主数据失败:%v", err)
		return
	}
	if _, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.SceneDataNotice)
		return ok
	}); err != nil {
		logOnce(&logLootErrOnce, "拾取链烟测等待场景失败:%v", err)
		return
	}

	if b.send(&proto.GatherReq{GatherItem: &proto.GatherItem{Index: 0, Reward: 0, IsLucky: true}}) != nil {
		return
	}
	gather, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.GatherRsp)
		return ok
	})
	if err != nil || gather.Body.(*proto.GatherRsp).GetStatus() == proto.StatusCode_StatusCode_OK {
		logOnce(&logLootErrOnce, "伪造采集点未被拒绝:%v", err)
		return
	}

	for attempt := 0; attempt < 2; attempt++ {
		if b.send(&proto.PickupReq{DropItemIndex: 0, PickIndex: 0}) != nil {
			return
		}
		pickup, err := b.waitHS(func(m *alg.GameMsg) bool {
			_, ok := m.Body.(*proto.PickupRsp)
			return ok
		})
		if err != nil || pickup.Body.(*proto.PickupRsp).GetStatus() == proto.StatusCode_StatusCode_OK {
			logOnce(&logLootErrOnce, "伪造拾取索引未被拒绝:%v", err)
			return
		}
	}

	if b.send(&proto.TreasureBoxOpenReq{
		LocType:          proto.TreasureBoxLocType_TreasureBoxLocType_DUNGEON,
		TreasureBoxIndex: 0,
	}) != nil {
		return
	}
	box, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.TreasureBoxOpenRsp)
		return ok
	})
	if err != nil || box.Body.(*proto.TreasureBoxOpenRsp).GetStatus() == proto.StatusCode_StatusCode_OK {
		logOnce(&logLootErrOnce, "伪造副本宝箱未被拒绝:%v", err)
		return
	}

	time.Sleep(100 * time.Millisecond)
	logOnce(&logLootOKOnce, "采集拾取链烟测通过:伪造采集/重复拾取/伪造宝箱均被拒绝")
}
