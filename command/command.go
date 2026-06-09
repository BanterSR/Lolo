package command

import (
	"gucooing/lolo/game"
	"gucooing/lolo/game/model"

	"github.com/gin-gonic/gin"
)

func NewCommand(router *gin.Engine, gs *game.Game) {
	gpt := newGpt()
	gs.RegisterBot(gpt)
}

type AiBot struct {
	uuid string
	*game.BotInfo
}

func (a AiBot) UUID() string {
	return a.uuid
}

func (a AiBot) GetBotInfo() *game.BotInfo {
	return a.BotInfo
}

func (a AiBot) Handle(s *model.Player, text string) string {
	return text
}

func newGpt() *AiBot {
	gpt := &AiBot{
		uuid: "1",
		BotInfo: &game.BotInfo{
			Head:        41101,
			Badge:       5000,
			NickName:    "GPT-5.5",
			AvatarFrame: 0,
			Sing:        "我会稳稳的接住",
			GuildName:   "",
			Level:       50,
		},
	}
	return gpt
}
