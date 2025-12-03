package gateway

import (
	"errors"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/config"
	"gucooing/lolo/game"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type Gateway struct {
	cfg          *config.GateWay
	net          ofnet.Net       // 传输层
	router       *gin.Engine     // http 服务器
	loginChan    chan *LoginInfo // 登录通道
	delLoginChan chan string     // 撤销登录通道 sdk uid ->
	doneChan     chan struct{}   // 停止
	game         *game.Game
}

func NewGateway(router *gin.Engine) *Gateway {
	log.NewGate()
	var err error
	g := &Gateway{
		cfg:          config.GetGateWay(),
		router:       router,
		loginChan:    make(chan *LoginInfo, 1000),
		delLoginChan: make(chan string, 1000),
		doneChan:     make(chan struct{}),
		game:         game.NewGame(),
	}
	g.net, err = ofnet.NewNet("tcp", g.cfg.GetOuterAddr(), log.Gate)
	if err != nil {
		panic(err)
	}
	g.net.SetBlackPackId(func() map[uint32]struct{} {
		list := make(map[uint32]struct{})
		for _, packString := range g.cfg.GetBlackCmd() {
			id := cmd.Get().GetCmdIdByCmdName(packString)
			list[id] = struct{}{}
		}
		return list
	}())
	g.net.SetLogMsg(g.cfg.GetIsLogMsgPlayer())

	go g.loginSessionManagement()
	return g
}

func (g *Gateway) RunGateway() error {
	for {
		select {
		case <-g.doneChan:
			log.Gate.Infof("gate主线程停止")
			return nil
		default:
		}
		conn, err := g.net.Accept()
		if err != nil {
			return err
		}
		conn.SetServerTag("GateWay")
		log.Gate.Infof("Gateway 接受了新的连接请求:%s", conn.RemoteAddr())
		go g.NewSession(conn)
	}
}

func (g *Gateway) NewSession(conn ofnet.Conn) {
	var message pb.Message
	timer := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-timer.C:
			log.Gate.Debug("登录超时")
			conn.Close()
			timer.Stop()
			return
		default:
			msg, err := conn.Read()
			if err != nil {
				conn.Close()
				timer.Stop()
				log.Gate.Error(err.Error())
				return
			}
			if msg.MsgId == cmd.VerifyLoginTokenReq {
				message = msg.Body
				goto ty
			} else {
				conn.Close()
				timer.Stop()
				return
			}
		}
	}
ty:
	timer.Stop()
	req := message.(*proto.VerifyLoginTokenReq)
	if req == nil {
		conn.Close()
		return
	}
	g.loginChan <- &LoginInfo{
		VerifyLoginTokenReq: req,
		conn:                conn,
	}
}

func (g *Gateway) receive(conn ofnet.Conn, userId uint32) {
	defer func() {
		g.game.GetKillUserChan() <- userId
	}()
	for {
		select {
		case <-g.doneChan:
			return
		default:
			msg, err := conn.Read()
			switch {
			case err == nil:
				g.game.GetGameMsgChan() <- &game.GameMsg{
					UserId:  userId,
					Conn:    conn,
					GameMsg: msg,
				}
			case errors.Is(err, io.EOF),
				errors.Is(err, io.ErrClosedPipe):
				return
			default:
				log.Gate.Errorf("%s", err)
				return
			}
		}
	}
}

func (g *Gateway) Close() {
	close(g.doneChan)
	g.game.Close()
	log.Gate.Infof("gate退出完成")
}
