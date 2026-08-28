package command

import (
	"github.com/gin-gonic/gin"
	"gucooing/lolo/config"
	"gucooing/lolo/game"
	"gucooing/lolo/gateway"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/limiter"
	"gucooing/lolo/pkg/log"
	"net/http"
	"reflect"
	"time"
)

type CommandInterface interface {
	Options() *Options
	Handle(ctx *Context)
	// 基础方法值
	GetPlayerID() uint32
}

type Command struct {
	apiKey     string
	ga         *gateway.Gateway
	gs         *game.Game
	commandMap map[string]reflect.Type
}

func NewCommand(router *gin.Engine, ga *gateway.Gateway, gs *game.Game) {
	apiKey := config.GetGame().GetApiKey()
	if apiKey == "" && config.GetMode() != config.ModeDev {
		apiKey = alg.RandHex(20)
		log.Game.Warnf("api key为空，生成临时ApiKey:%s | 下次重启失效", apiKey)
	}
	c := &Command{
		apiKey:     apiKey,
		ga:         ga,
		gs:         gs,
		commandMap: make(map[string]reflect.Type),
	}
	api := router.Group("/api/v1", limiter.NewGinLimiter(1*time.Minute, 20), c.CommandAuto())
	{
		api.GET("/cmd", c.HttpCommandRun)
	}

	c.registerAllCommand()
	for _, cfg := range config.GetGame().GetBotList() {
		switch cfg.Type {
		//case config.BotTypeCommand:
		case config.BotTypeChat:
			ai := c.NewAiBot(cfg)
			c.gs.RegisterBot(ai)
		}
	}

}

func CfgToBotInfo(cfg *config.Bot) *game.BotInfo {
	return &game.BotInfo{
		Head:        cfg.Head,
		Badge:       cfg.Badge,
		NickName:    cfg.NickName,
		AvatarFrame: cfg.AvatarFrame,
		Sing:        cfg.Sing,
		GuildName:   cfg.GuildName,
		Level:       cfg.Level,
	}
}

func (c *Command) registerAllCommand() {
	c.regCommand(new(hi))
	c.regCommand(new(item))
	c.regCommand(new(status))
}

func (c *Command) regCommand(cmd CommandInterface) {
	refType := reflect.TypeOf(cmd)
	name := refType.Elem().Name()
	if _, ok := c.commandMap[name]; ok {
		log.Game.Warnf("command %s already exists", name)
		return
	}
	c.commandMap[name] = refType
	log.Game.Infof("register command %s", name)
}

func (c *Command) GetCommand(code string) CommandInterface {
	refType, exist := c.commandMap[code]
	if !exist {
		return nil
	}
	protoObjInst := reflect.New(refType.Elem())
	protoObj := protoObjInst.Interface().(CommandInterface)
	return protoObj
}

func (c *Command) CommandAuto() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if c.apiKey != "" && c.apiKey != ctx.GetHeader("X-Api-Key") {
			ctx.Status(http.StatusNotFound)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func (c *Command) HttpCommandRun(ginC *gin.Context) {
	ctx := c.NewContext(ginC)
	defer func() {
		ginC.JSON(http.StatusOK, ctx.response)
	}()
	code := c.GetCommand(ginC.Query("code"))
	if code == nil {
		ctx.Response().ErrorCode(ResponseUnknownCode, nil)
		return
	}
	err := ginC.ShouldBindQuery(code)
	if err != nil {
		ctx.Response().ErrorCode(ResponseInvalidParam, err)
		return
	}
	code.Handle(ctx)
	return
}
