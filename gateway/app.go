package gateway

import (
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pb "google.golang.org/protobuf/proto"
	"resty.dev/v3"

	"gucooing/lolo/config"
	"gucooing/lolo/game"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
	"gucooing/lolo/protocol/quick"
)

type Gateway struct {
	cfg          *config.GateWay
	net          ofnet.Net
	router       *gin.Engine
	loginChan    chan *LoginInfo
	delLoginChan chan string
	doneChan     chan struct{}
	client       *resty.Client
	game         *game.Game
}

func NewGateway(router *gin.Engine) *Gateway {
	log.NewGate()
	log.NewPacket()

	g := &Gateway{
		cfg:          config.GetGateWay(),
		router:       router,
		loginChan:    make(chan *LoginInfo, 1000),
		delLoginChan: make(chan string, 1000),
		doneChan:     make(chan struct{}),
		client:       DefaultClient(),
		game:         game.NewGame(router),
	}

	var err error
	g.net, err = ofnet.NewNet("tcp", g.cfg.GetOuterAddr(), log.Gate)
	if err != nil {
		panic(err)
	}

	g.net.SetFileLog(log.Packet)
	g.net.SetBlackPackId(func() map[uint32]struct{} {
		list := make(map[uint32]struct{})
		for _, packString := range g.cfg.GetBlackCmd() {
			id := cmd.Get().GetCmdIdByCmdName(packString)
			list[id] = struct{}{}
		}
		return list
	}())
	g.net.StartStatsLoop()

	go g.loginSessionManagement()
	return g
}

func (g *Gateway) RunGateway() error {
	for {
		select {
		case <-g.doneChan:
			log.Gate.Infof("gateway main loop stopped")
			return nil
		default:
		}

		conn, err := g.net.Accept()
		if err != nil {
			return err
		}

		conn.SetServerTag("GateWay")
		log.Gate.Infof("Gateway accepted new connection: %s", conn.RemoteAddr())
		go g.NewSession(conn)
	}
}

func (g *Gateway) NewSession(conn ofnet.Conn) {
	var message pb.Message
	timer := time.NewTimer(10 * time.Second)

	for {
		select {
		case <-timer.C:
			log.Gate.Debug("login timeout")
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
				goto verified
			}

			conn.Close()
			timer.Stop()
			return
		}
	}

verified:
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
	loginUUID := uuid.NewString()
	for {
		select {
		case <-g.doneChan:
			return
		default:
			msg, err := conn.Read()
			if err == nil {
				g.game.GetGameMsgChan() <- &game.GameMsg{
					UserId:  userId,
					UUID:    loginUUID,
					Conn:    conn,
					GameMsg: msg,
				}
				continue
			}

			conn.Close()
			log.Gate.Infof("[UID:%v][UUID:%v] network connection closed", userId, loginUUID)
			g.game.DoPlayer() <- &game.DonePlayerCtx{
				UserId: userId,
				UUID:   loginUUID,
			}
			return
		}
	}
}

func (g *Gateway) Close() {
	_ = g.net.Close()
	close(g.doneChan)
	g.game.Close()
	log.Gate.Infof("gateway closed")
}

func DefaultClient() *resty.Client {
	return resty.New().
		SetRetryCount(10).
		SetRetryWaitTime(50 * time.Millisecond).
		SetRetryMaxWaitTime(2 * time.Second)
}

func (g *Gateway) GetToken(uid, token string) bool {
	if !g.cfg.GetCheckToken() {
		return true
	}

	resp, err := g.client.R().
		SetBody(&quick.CheckSdkTokenRequest{
			Token: token,
			UID:   uid,
		}).
		Post(g.cfg.GetCheckUrl())
	if err != nil {
		return false
	}

	rsp := new(quick.CheckSdkTokenResponse)
	if err = sonic.Unmarshal(resp.Bytes(), rsp); err != nil {
		return false
	}
	if rsp.Uid != uid || rsp.Code != 0 {
		return false
	}
	return true
}
