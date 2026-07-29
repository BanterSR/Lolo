package command

import (
	"github.com/gin-gonic/gin"
	"gucooing/lolo/config"
	"gucooing/lolo/game"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/limiter"
	"gucooing/lolo/pkg/log"
	"net/http"
	"reflect"
	"time"
)

type CommandInterface interface {
	Options() *CommandOptions
	GetPlayerID() uint32
	Handle(s *model.Player) (string, error)
}

type Command struct {
	apiKey     string
	gs         *game.Game
	commandMap map[string]reflect.Type
}

func NewCommand(router *gin.Engine, gs *game.Game) {
	apiKey := config.GetGame().GetApiKey()
	if apiKey == "" {
		apiKey = alg.RandHex(20)
		log.Game.Warnf("api key为空，生成临时ApiKey:%s | 下次重启失效", apiKey)
	}
	c := &Command{
		apiKey:     apiKey,
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

type CommandOptions struct {
	IsPlayer bool // 是否作用于玩家
}

type baseCommand struct {
	PlayerID uint32 `form:"player_id"`
}

func (b *baseCommand) GetPlayerID() uint32 {
	return b.PlayerID
}

func (c *Command) registerAllCommand() {
	c.regCommand(new(hi))
	c.regCommand(new(item))
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

type CommandResponse struct {
	Code    string
	Message any
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

func (c *Command) HttpCommandRun(ctx *gin.Context) {
	rsp := &CommandResponse{Code: "ok"}
	defer func() {
		ctx.JSON(http.StatusOK, rsp)
	}()
	code := c.GetCommand(ctx.Query("code"))
	if code == nil {
		rsp.Code = "未知的指令"
		return
	}
	err := ctx.ShouldBindQuery(code)
	if err != nil {
		rsp.Code = err.Error()
		return
	}
	option := code.Options()
	if !option.IsPlayer {
		re, er := code.Handle(nil)
		if er != nil {
			rsp.Code = er.Error()
			return
		}
		rsp.Message = re
		return
	}
	re, er := c.gs.RunPlayerCommand(code.GetPlayerID(), code.Handle)
	if er != nil {
		rsp.Code = er.Error()
		return
	}
	rsp.Message = re
	return
}
