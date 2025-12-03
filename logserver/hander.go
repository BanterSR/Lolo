package logserver

import (
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

func (g *LogServer) logMainLoop() {
	log.ClientLog.Info("log server 主线程启动")
	defer log.ClientLog.Info("log server 主线程退出")
	for {
		select {
		case <-g.doneChan:
			return
		case msg := <-g.logMagChan:
			g.routerHandler(msg.conn, msg.msg)
		}
	}
}

func (g *LogServer) PlayerPing(conn ofnet.Conn, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerPingReq)
	conn.Send(cmd.PlayerPingRsp, msg.PacketId, &proto.PlayerPingRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		ClientTimeMs: req.ClientTimeMs,
		ServerTimeMs: req.ClientTimeMs,
	})
}
