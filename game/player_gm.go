package game

import (
	"fmt"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

// gm 指令接口
type CommandInterface interface {
	Name() string
	Handle(s *model.Player)
}

// RunPlayerCommand 供外部(GM 等非主循环 goroutine)对在线玩家执行一条 GM 指令。
// 指令通过 game 主循环串行执行,外部无需加锁即可直接操作 Player。
// 返回玩家是否在线(指令是否执行)。
func (g *Game) RunPlayerCommand(userId uint32, command CommandInterface) (any, error) {
	return g.InvokePlayerFunc(userId, func(s *model.Player) (any, error) {
		command.Handle(s)
		return nil, nil
	}, 3*time.Second)
}

var gmCodeParamMap = map[string]func(){
	"111": func() {},
	// add_gm_count_item 添加新号道具
}

func (g *Game) GmCode(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GmCodeReq)
	rsp := &proto.GmCodeRsp{
		Status:        proto.StatusCode_StatusCode_Ok,
		Result:        "",                             // 返回的消息
		OnlinePlayers: make([]uint32, 0),              // 在线的玩家
		RewardInfos:   make([]*proto.GmRewardInfo, 0), // 奖励内容
	}
	defer g.send(s, msg.PacketId, rsp)
	handler, ok := gmCodeParamMap[req.Param]
	if !ok {
		log.Game.Warnf("暂未实现的Gm Code \"%s\"", req.Param)
		rsp.Result = fmt.Sprintf("暂未实现的Gm Code %s", req.Param)
		return
	}
	handler()
}
