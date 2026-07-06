package main

import (
	"context"
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

// scenarioScene 场景同步：先拉主数据加入房间，抓取服务器下发的默认出生点作为锚点，
// 随后以 1 秒为间隔做「左移两坐标→右移两坐标」的 4 秒保活循环 + 偶发动作。
func (b *Bot) scenarioScene(ctx context.Context) {
	// PlayerMainData 触发服务端 loginGame -> joinSceneChannel，加入房间后服务端下发 SceneDataNotice。
	if b.send(&proto.PlayerMainDataReq{}) != nil {
		return
	}
	// 等待 SceneDataNotice：其中携带本玩家在场景中的默认出生点（服务端随机出生点）。
	// 超时（通常意味着未成功加入房间或服务端过载）则退回原点，保证保活循环仍能进行。
	anchor := sceneAnchor{charId: 101001, dp: 2, rdp: 2}
	if notice, err := b.waitHS(func(m *alg.GameMsg) bool {
		_, ok := m.Body.(*proto.SceneDataNotice)
		return ok
	}); err != nil {
		logOnce(&logSceneAnchorOnce, "未收到 SceneDataNotice，保活将从原点开始: %v", err)
	} else {
		anchor = b.sceneAnchorFrom(notice, anchor)
	}

	// 保活位移固定 1 秒一步、4 秒一循环（遵循固定节奏，与 cfg.Action 无关）。
	move := time.NewTicker(time.Second)
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
			if b.send(anchor.moveRecord(tick)) != nil {
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

// sceneAnchor 是保活循环的锚点：服务器下发的默认出生点（坐标 + 朝向 + 首角色 id）。
type sceneAnchor struct {
	charId     uint32
	x, y, z    int32  // 出生坐标
	dp         uint32 // 出生坐标小数位
	rx, ry, rz int32  // 出生朝向
	rdp        uint32 // 出生朝向小数位
}

// keepAliveOffsets 是相对锚点在 X 轴上的偏移（单位：坐标），每秒一步、四步一循环：
// 0 → 左1 → 左2 → 右回1（下一循环回到 0），即「左移两个坐标然后右移两个坐标」。
// 想改变摆动幅度/形状，直接改这张表即可。
var keepAliveOffsets = []int32{0, -1, -2, -1}

// sceneAnchorFrom 从 SceneDataNotice 中取出本玩家（按 userId 匹配）首角色的默认出生点。
func (b *Bot) sceneAnchorFrom(m *alg.GameMsg, def sceneAnchor) sceneAnchor {
	sd, ok := m.Body.(*proto.SceneDataNotice)
	if !ok || sd.GetData() == nil {
		return def
	}
	for _, p := range sd.GetData().GetPlayers() {
		if p.GetPlayerId() != b.userId {
			continue
		}
		ch := p.GetTeam().GetChar1()
		if ch == nil {
			break
		}
		a := def
		a.charId = ch.GetCharId()
		if pos := ch.GetPos(); pos != nil {
			a.x, a.y, a.z, a.dp = pos.GetX(), pos.GetY(), pos.GetZ(), pos.GetDecimalPlaces()
		}
		if rot := ch.GetRot(); rot != nil {
			a.rx, a.ry, a.rz, a.rdp = rot.GetX(), rot.GetY(), rot.GetZ(), rot.GetDecimalPlaces()
		}
		return a
	}
	return def
}

// moveRecord 构造第 tick 步的位置上报：以出生点为锚，仅 X 轴按 keepAliveOffsets 摆动。
func (a sceneAnchor) moveRecord(tick int) *proto.PlayerSceneRecordReq {
	off := keepAliveOffsets[tick%len(keepAliveOffsets)]
	x := a.x + off*pow10(a.dp) // 1 个「坐标」= 10^dp 个原始整数单位
	return &proto.PlayerSceneRecordReq{
		Data: &proto.PlayerRecorderData{
			Ping: 30,
			CharRecorderDataLst: []*proto.CharacterRecorderData{
				{
					CharId: a.charId,
					Pos:    &proto.Vector3{X: x, Y: a.y, Z: a.z, DecimalPlaces: a.dp},
					Rot:    &proto.Vector3{X: a.rx, Y: a.ry, Z: a.rz, DecimalPlaces: a.rdp},
				},
			},
		},
	}
}

// pow10 返回 10^n，用于把「坐标」换算成带小数位的原始整数值。
func pow10(n uint32) int32 {
	p := int32(1)
	for i := uint32(0); i < n; i++ {
		p *= 10
	}
	return p
}
