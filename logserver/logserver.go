package logserver

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gucooing/lolo/pkg/alg"
	"io"

	"gucooing/lolo/config"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
)

type LogServer struct {
	cfg                 *config.LogServer
	net                 ofnet.Net             // 传输层
	httpRouter          *gin.Engine           // http 服务器
	handlerFuncRouteMap map[uint32]logHandler // 路由
	doneChan            chan struct{}
	logMagChan          chan *logMessage
}

func NewLogServer(router *gin.Engine) *LogServer {
	var err error
	l := &LogServer{
		cfg:        config.GetLogServer(),
		httpRouter: router,
		doneChan:   make(chan struct{}),
		logMagChan: make(chan *logMessage, 200),
	}
	log.NewClientLog()

	l.net, err = ofnet.NewNet("tcp", l.cfg.GetOuterAddr(), log.ClientLog)
	if err != nil {
		panic(err)
	}
	l.net.SetLogMsg(l.cfg.GetIsLogMsg())
	l.routerInit() // 注册路由
	go l.logMainLoop()

	return l
}

func (g *LogServer) RunLogServer() error {
	for {
		conn, err := g.net.Accept()
		if err != nil {
			return err
		}
		conn.SetServerTag("LogServer")
		log.ClientLog.Debugf("LogServer 接受了新的连接请求:%s", conn.RemoteAddr())
		go g.NewSession(conn)
	}
}

func (g *LogServer) NewSession(conn ofnet.Conn) {
	for {
		msg, err := conn.Read()
		if err != nil {
			conn.Close()
			log.ClientLog.Error(err.Error())
			return
		}
		go g.login(conn, msg)
		log.ClientLog.Debugf("msg:%s", msg.Body)
	}
}

func (g *LogServer) Close() {
	close(g.doneChan)
}

type logMessage struct {
	conn ofnet.Conn
	msg  *alg.GameMsg
}

func (g *LogServer) receive(conn ofnet.Conn) {
	for {
		select {
		case <-g.doneChan:
			return
		default:
			msg, err := conn.Read()
			switch {
			case err == nil:
				g.logMagChan <- &logMessage{
					conn: conn,
					msg:  msg,
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
