package gateway

import (
	"gucooing/lolo/pkg/cache"
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
	sessionMap   *cache.Cache[uint32, *session] // userid
}

type session struct {
	userId uint32
	uuid   string
	conn   ofnet.Conn
	done   chan struct{}
}

func NewGateway(router *gin.Engine, gs *game.Game) *Gateway {
	log.NewGate()
	log.NewPacket()

	g := &Gateway{
		cfg:          config.GetGateWay(),
		router:       router,
		loginChan:    make(chan *LoginInfo, 1000),
		delLoginChan: make(chan string, 1000),
		doneChan:     make(chan struct{}),
		client:       DefaultClient(),
		game:         gs,
		sessionMap:   cache.New[uint32, *session](0),
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
		uuid:                uuid.NewString(),
	}
}

func (g *Gateway) receive(se *session) {
	defer func() {
		log.Gate.Debugf("[UID:%v][UUID:%v] network connection closed", se.userId, se.uuid)
		g.game.GetGateTask() <- &game.KillPlayer{
			UserId:     se.userId,
			UUID:       se.uuid,
			Reason:     proto.PlayerOfflineReason_PlayerOfflineReason_None,
			KillPlayer: false,
		}
		ose, ok := g.sessionMap.Get(se.userId)
		if ok && ose.uuid == se.uuid {
			g.sessionMap.Del(se.userId)
		}
		se.conn.Close()
	}()
	for {
		select {
		case <-g.doneChan:
			return
		case <-se.done:
			return
		default:
			msg, err := se.conn.Read()
			if err != nil {
				return
			}
			g.game.GetGateTask() <- &game.PlayerMsg{
				UserId:  se.userId,
				UUID:    se.uuid,
				Conn:    se.conn,
				GameMsg: msg,
			}
			continue
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
