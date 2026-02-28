package log

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/gookit/slog/rotatefile"

	"gucooing/lolo/config"
)

var (
	App       *slog.SugaredLogger
	Gate      *slog.SugaredLogger
	Game      *slog.SugaredLogger
	ClientLog *slog.SugaredLogger
	Packet    *slog.SugaredLogger
)

func Close() {
	if App != nil {
		App.Close()
	}
	if Gate != nil {
		Gate.Close()
		Gate.Close()
	}
	if Game != nil {
		Game.Close()
	}
	if ClientLog != nil {
		ClientLog.Close()
		ClientLog.Close()
	}
	if Packet != nil {
		Packet.Close()
	}
}

func NewApp() {
	conf := config.GetLog()
	App = slog.NewStdLogger(func(sl *slog.SugaredLogger) {
		f := sl.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		sl.ChannelName = conf.AppName
		sl.Level = conf.Level
	})
	addHandler(App, conf)
}

func NewGate() {
	conf := config.GetGateWay().GetLog()
	Gate = slog.NewStdLogger(func(sl *slog.SugaredLogger) {
		f := sl.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		sl.ChannelName = conf.AppName
		sl.Level = conf.Level
	})
	addHandler(Gate, conf)
}

func NewGame() {
	conf := config.GetGame().GetLog()
	Game = slog.NewStdLogger(func(sl *slog.SugaredLogger) {
		f := sl.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		sl.ChannelName = conf.AppName
		sl.Level = conf.Level
	})
	addHandler(Game, conf)
}

func NewClientLog() {
	conf := config.GetLogServer().GetLog()
	ClientLog = slog.NewStdLogger(func(sl *slog.SugaredLogger) {
		f := sl.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		sl.ChannelName = conf.AppName
		sl.Level = conf.Level
	})
	addHandler(ClientLog, conf)
}

func NewPacket() {
	conf := &config.Log{
		Level:   slog.InfoLevel,
		LogFile: config.GetGateWay().GetIsLogMsgPlayer(),
		AppName: "Packet",
	}
	Packet = slog.NewStdLogger(func(sl *slog.SugaredLogger) {
		f := sl.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		sl.ChannelName = conf.AppName
		sl.Level = conf.Level
	})
	addHandler(Packet, conf)
}

func addHandler(l *slog.SugaredLogger, conf *config.Log) {
	if conf.LogFile {
		l.AddHandler(handler.NewBuilder().
			WithLogfile(fmt.Sprintf("./log/%s.log", conf.AppName)).
			WithLogLevels(slog.AllLevels).
			WithBuffSize(1024 * 10).
			WithRotateTime(func() rotatefile.RotateTime {
				if config.GetMode() == config.ModeDev {
					return rotatefile.Every15Min
				}
				return rotatefile.EveryDay
			}()).
			WithCompress(true).
			Build())
	}
}

type ResponseWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	context *gin.Context
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func GinLog(l *slog.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		var reqBody []byte
		blw := &ResponseWriter{ResponseWriter: c.Writer, body: bytes.NewBuffer([]byte{})}
		if config.GetMode() == config.ModeDev &&
			c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			c.Writer = blw
		}
		c.Next()
		if config.GetMode() == config.ModeDev {
			l.Debugf("HTTP [%s][Path:%s] req:%s resp:%s",
				c.Request.Method,
				path,
				string(reqBody),
				blw.body.String(),
			)
		}
		latency := time.Now().Sub(start)

		if raw != "" {
			path = path + "?" + raw
		}

		l.Debugf("HTTP [%s][Code:%3d][%11s][Ping:%7v][Path:%s]",
			c.Request.Method,
			c.Writer.Status(),
			c.ClientIP(),
			latency,
			path)
	}
}
