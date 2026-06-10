package command

import (
	"github.com/gin-gonic/gin"
	"gucooing/lolo/config"
	"gucooing/lolo/game"
)

type Command struct {
	gs *game.Game
}

func NewCommand(router *gin.Engine, gs *game.Game) {
	c := &Command{
		gs: gs,
	}
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
