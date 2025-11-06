package logserver

import (
	"github.com/gin-gonic/gin"

	"gucooing/lolo/config"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type LogServer struct {
	cfg    *config.LogServer
	net    ofnet.Net   // 传输层
	router *gin.Engine // http 服务器
}

func NewLogServer(router *gin.Engine) *LogServer {
	var err error
	l := &LogServer{
		cfg:    config.GetLogServer(),
		router: router,
	}

	l.net, err = ofnet.NewNet("tcp", l.cfg.GetOuterAddr())
	if err != nil {
		panic(err)
	}

	return l
}

func (g *LogServer) RunLogServer() error {
	for {
		conn, err := g.net.Accept()
		if err != nil {
			return err
		}
		conn.SetServerTag("LogServer")
		conn.SetLogMsg(true)
		log.Game.Debugf("LogServer 接受了新的连接请求:%s", conn.RemoteAddr())
		go g.NewSession(conn)
	}
}

func (g *LogServer) NewSession(conn ofnet.Conn) {
	for {
		msg, err := conn.Read()
		if err != nil {
			conn.Close()
			log.Game.Error(err.Error())
			return
		}
		if msg.MsgId == cmd.ClientLogAuthReq {
			conn.Send(cmd.ClientLogAuthRsp, msg.PacketId, &proto.ClientLogAuthRsp{
				Status: proto.StatusCode_StatusCode_OK,
			})
		}
		if msg.MsgId == cmd.PlayerPingReq {
			conn.Send(cmd.PlayerPingRsp, msg.PacketId, &proto.PlayerPingRsp{
				Status:       proto.StatusCode_StatusCode_OK,
				ClientTimeMs: 0,
				ServerTimeMs: 0,
			})
		}
		log.Game.Debugf("msg:%s", msg.Body)
	}
}

func (g *LogServer) Close() {}
