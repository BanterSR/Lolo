package config

import (
	"github.com/gookit/slog"
)

type Game struct {
	Log         *Log   `json:"Log"`
	MsgChanSize int    `json:"MsgChanSize"`
	BotList     []*Bot `json:"BotList"`
}

var defaultGame = &Game{
	Log: &Log{
		Level:   slog.InfoLevel,
		LogFile: false,
		AppName: "Game",
	},
	MsgChanSize: 100,
	BotList:     make([]*Bot, 0),
}

func GetGame() *Game {
	if GetConfig().Game == nil {
		GetConfig().Game = defaultGame
	}
	return GetConfig().Game
}

func (x *Game) GetLog() *Log {
	return x.Log
}

func (x *Game) GetMsgChanSize() int {
	return x.MsgChanSize
}

func (x *Game) GetBotList() []*Bot {
	return x.BotList
}
