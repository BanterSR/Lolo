package main

import (
	"context"
	"sync"
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

var (
	logLifeErrOnce sync.Once
	logLifeOKOnce  sync.Once
)

func (b *Bot) waitLifeHS(ctx context.Context, match func(*alg.GameMsg) bool) (*alg.GameMsg, error) {
	timer := time.NewTimer(handshakeTimeout)
	defer timer.Stop()
	for {
		select {
		case msg := <-b.hsCh:
			if match(msg) {
				return msg, nil
			}
		case <-b.dead:
			return nil, errDisconnected
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, errTimeout
		}
	}
}

func (b *Bot) paceLifeSmoke(ctx context.Context) bool {
	sleepCtx(ctx, orDur(b.cfg.Action, 2*time.Second))
	return ctx.Err() == nil
}

func (b *Bot) scenarioLife(ctx context.Context) {
	lifeTypes := []proto.LifeType{
		proto.LifeType_LIFE_TYPE_Cooking,
		proto.LifeType_LIFE_TYPE_Fishing,
		proto.LifeType_LIFE_TYPE_Making,
		proto.LifeType_LIFE_TYPE_Handicraft,
	}
	for _, lifeType := range lifeTypes {
		if ctx.Err() != nil || !b.paceLifeSmoke(ctx) {
			return
		}
		if b.send(&proto.GetLifeInfoReq{LifeType: lifeType}) != nil {
			return
		}
		msg, err := b.waitLifeHS(ctx, func(m *alg.GameMsg) bool {
			_, ok := m.Body.(*proto.GetLifeInfoRsp)
			return ok
		})
		if err != nil {
			logOnce(&logLifeErrOnce, "生活技能烟测获取信息失败 type=%d err=%v", lifeType, err)
			return
		}
		rsp := msg.Body.(*proto.GetLifeInfoRsp)
		if rsp.GetStatus() != proto.StatusCode_StatusCode_OK || rsp.GetLifeBaseInfo() == nil || rsp.GetLifeBaseInfo().GetLevel() < 1 || rsp.GetLifeSkill() == nil || rsp.GetLifeAchieve() == nil {
			logOnce(&logLifeErrOnce, "生活技能烟测信息不完整 type=%d status=%v", lifeType, rsp.GetStatus())
			return
		}
		if !b.paceLifeSmoke(ctx) {
			return
		}
		if b.send(&proto.GetLifeSkillReq{LifeType: lifeType}) != nil {
			return
		}
		skillMsg, err := b.waitLifeHS(ctx, func(m *alg.GameMsg) bool {
			_, ok := m.Body.(*proto.GetLifeSkillRsp)
			return ok
		})
		if err != nil {
			logOnce(&logLifeErrOnce, "生活技能烟测获取技能失败 type=%d err=%v", lifeType, err)
			return
		}
		if skillMsg.Body.(*proto.GetLifeSkillRsp).GetStatus() != proto.StatusCode_StatusCode_OK {
			logOnce(&logLifeErrOnce, "生活技能烟测获取技能被拒 type=%d status=%v", lifeType, skillMsg.Body.(*proto.GetLifeSkillRsp).GetStatus())
			return
		}
		if !b.paceLifeSmoke(ctx) {
			return
		}
		if b.send(&proto.GetLifeAchieveReq{LifeType: lifeType}) != nil {
			return
		}
		achieveMsg, err := b.waitLifeHS(ctx, func(m *alg.GameMsg) bool {
			_, ok := m.Body.(*proto.GetLifeAchieveRsp)
			return ok
		})
		if err != nil {
			logOnce(&logLifeErrOnce, "生活技能烟测获取成就失败 type=%d err=%v", lifeType, err)
			return
		}
		if achieveMsg.Body.(*proto.GetLifeAchieveRsp).GetStatus() != proto.StatusCode_StatusCode_OK || achieveMsg.Body.(*proto.GetLifeAchieveRsp).GetLifeAchieve() == nil {
			logOnce(&logLifeErrOnce, "生活技能烟测获取成就被拒 type=%d status=%v", lifeType, achieveMsg.Body.(*proto.GetLifeAchieveRsp).GetStatus())
			return
		}
	}
	logOnce(&logLifeOKOnce, "生活技能烟测通过 types=4")
}
