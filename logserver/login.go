package logserver

import (
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

func (g *LogServer) login(conn ofnet.Conn, msg *alg.GameMsg) {
	switch req := msg.Body.(type) {
	case *proto.ClientLogAuthReq:
		if req.LogServerToken != "114514" {
			return
		}
		rsp := &proto.ClientLogAuthRsp{
			Status: proto.StatusCode_StatusCode_OK,
		}
		conn.Send(cmd.ClientLogAuthRsp, msg.PacketId, rsp)
		// new log session
		go g.receive(conn)
	default:
		// 异常消息
		return
	}
}
