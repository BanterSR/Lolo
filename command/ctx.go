package command

import (
	"context"
	"gucooing/lolo/game"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gateway"
	"time"
)

type Context struct {
	*Command
	context.Context
	response *Response
}

func (c *Command) NewContext(tc context.Context) *Context {
	ctx := &Context{
		Command:  c,
		Context:  tc,
		response: new(Response),
	}

	return ctx
}

func (ctx *Context) Gate() *gateway.Gateway {
	return ctx.Command.ga
}

func (ctx *Context) Game() *game.Game {
	return ctx.Command.gs
}

// 投递到game中执行
func (ctx *Context) GameHandle(f func(g *game.Game)) {
	f(ctx.gs)
}

// 投递到目标玩家上
func (ctx *Context) PlayerHandle(s CommandInterface, f func(s *model.Player)) {
	playerID := s.GetPlayerID()
	if playerID == 0 {
		ctx.Response().ErrorCode(ResponseEmptyPlayer, nil)
		return
	}
	taskErr := ctx.gs.InvokePlayerFunc(playerID, f, 5*time.Second)
	if taskErr != nil {
		ctx.Response().Error(taskErr)
		return
	}
}

func (ctx *Context) Response() *Response {
	return ctx.response
}

type Response struct {
	Code    string
	Message any
}

const (
	ResponseOK           = "OK"
	ResponseError        = "INTERNAL_ERROR"  // 内部错误
	ResponseUnknownCode  = "UNKNOWN_CODE"    // 未知Code
	ResponseEmptyPlayer  = "EMPTY_PLAYER_ID" // 玩家ID为空
	ResponseInvalidParam = "INVALID_PARAM"   // 参数错误
)

func (re *Response) Error(err error) {
	re.Code = ResponseError
	re.Message = err.Error()
}

func (re *Response) ErrorCode(code string, err error) {
	re.Code = code
	re.Message = err
}

func (re *Response) Json(f any) {
	re.Code = ResponseOK
	re.Message = f
}

func (re *Response) CodeJson(code string, f any) {
	re.Code = code
	re.Message = f
}

func (re *Response) String(s string) {
	re.Code = ResponseOK
	re.Message = s
}

func (re *Response) CodeString(code, s string) {
	re.Code = code
	re.Message = s
}
