package command

import (
	"gucooing/lolo/game/model"
)

type hi struct {
	baseCommand
}

func (h *hi) Options() *CommandOptions {
	return &CommandOptions{
		IsPlayer: false,
	}
}

func (h *hi) Handle(s *model.Player) (string, error) {
	return "hi!", nil
}
