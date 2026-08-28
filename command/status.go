package command

type status struct {
	base
}

func (t *status) Options() *Options {
	return &Options{}
}

type StatusResponse struct {
	ConnNum   int64 `json:"connNum"`   // 网关连接数
	TaskNum   int   `json:"taskNum"`   // game中排队的任务数量
	PlayerNum int64 `json:"PlayerNum"` // 在线玩家数量
}

func (t *status) Handle(ctx *Context) {
	ga := ctx.Gate()
	gs := ctx.Game()

	s := &StatusResponse{
		ConnNum:   ga.ConnNum(),
		TaskNum:   gs.TaskNum(),
		PlayerNum: gs.LoadPlayerNum(),
	}
	ctx.Response().Json(s)
}
