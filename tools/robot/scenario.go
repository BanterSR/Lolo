package main

import (
	"context"
	"math/rand"
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func orDur(v Duration, def time.Duration) time.Duration {
	if v.D() > 0 {
		return v.D()
	}
	return def
}

// runScenario 根据配置分发到具体场景行为。
func (b *Bot) runScenario(ctx context.Context) {
	switch b.cfg.Scenario {
	case "login":
		// 登录风暴：完成登录即返回，Pool 会以同一账号重登，形成持续登录/登出流。
		return
	case "scene":
		b.scenarioScene(ctx)
	default:
		b.scenarioSteady(ctx)
	}
}

// scenarioSteady 定常运转：周期性 ping 保活 + 间歇性拉取主数据。
func (b *Bot) scenarioSteady(ctx context.Context) {
	ping := time.NewTicker(orDur(b.cfg.Ping, 15*time.Second))
	defer ping.Stop()
	data := time.NewTicker(30 * time.Second)
	defer data.Stop()
	sweep := time.NewTicker(5 * time.Second)
	defer sweep.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.dead:
			return
		case <-ping.C:
			if b.send(&proto.PlayerPingReq{ClientTimeMs: time.Now().UnixMilli()}) != nil {
				return
			}
		case <-data.C:
			if b.send(&proto.PlayerMainDataReq{}) != nil {
				return
			}
		case <-sweep.C:
			b.sweepInflight()
		}
	}
}

// scenarioScene 场景同步：先拉主数据加入房间，随后循环上报角色移动 + 偶发动作。
func (b *Bot) scenarioScene(ctx context.Context) {
	// PlayerMainData 触发服务端 loginGame -> joinSceneChannel，之后场景消息才生效
	if b.send(&proto.PlayerMainDataReq{}) != nil {
		return
	}
	if _, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.PlayerMainDataRsp)
		return ok
	}); err != nil {
		return
	}

	move := time.NewTicker(orDur(b.cfg.Action, 2*time.Second))
	defer move.Stop()
	ping := time.NewTicker(orDur(b.cfg.Ping, 15*time.Second))
	defer ping.Stop()
	sweep := time.NewTicker(5 * time.Second)
	defer sweep.Stop()

	var tick int
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.dead:
			return
		case <-ping.C:
			if b.send(&proto.PlayerPingReq{ClientTimeMs: time.Now().UnixMilli()}) != nil {
				return
			}
		case <-move.C:
			if b.send(b.moveRecord()) != nil {
				return
			}
			tick++
			if tick%5 == 0 {
				if b.send(&proto.SendActionReq{ActionId: 1}) != nil {
					return
				}
			}
		case <-sweep.C:
			b.sweepInflight()
		}
	}
}

// moveRecord 构造一次随机游走的角色位置上报。
func (b *Bot) moveRecord() *proto.PlayerSceneRecordReq {
	return &proto.PlayerSceneRecordReq{
		Data: &proto.PlayerRecorderData{
			Ping: 30,
			CharRecorderDataLst: []*proto.CharacterRecorderData{
				{
					CharId: 101001,
					Pos:    &proto.Vector3{X: int32(rand.Intn(20000) - 10000), Y: 0, Z: int32(rand.Intn(20000) - 10000), DecimalPlaces: 2},
					Rot:    &proto.Vector3{X: 0, Y: int32(rand.Intn(36000)), Z: 0, DecimalPlaces: 2},
				},
			},
		},
	}
}
