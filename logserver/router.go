package logserver

import (
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
)

type logHandler = func(conn ofnet.Conn, msg *alg.GameMsg)

func (g *LogServer) routerInit() {
	g.handlerFuncRouteMap = map[uint32]logHandler{
		cmd.PlayerPingReq: g.PlayerPing,
	}
}

func (g *LogServer) routerHandler(conn ofnet.Conn, msg *alg.GameMsg) {
	handler, ok := g.handlerFuncRouteMap[msg.PacketId]
	if !ok {
		log.ClientLog.Errorf("no route for msg, cmdId: %v name:%s", msg.MsgId, cmd.Get().GetCmdNameByCmdId(msg.MsgId))
		return
	}
	handler(conn, msg)
}
