package command

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
)

type item struct {
	base
	ItemId uint32 `form:"item_id"`
	Count  int64  `form:"count"`
	All    bool   `form:"all"`
}

func (i *item) Options() *Options {
	return &Options{}
}

func (i *item) Handle(ctx *Context) {
	if i.Count <= 0 {
		i.Count = 1
	}
	ctx.PlayerHandle(i, func(s *model.Player) {
		if i.All {
			for _, conf := range gdconf.GetAllItemConfigure() {
				s.AddAllTypeItem(uint32(conf.ID), i.Count)
			}
		} else {
			s.AddAllTypeItem(i.ItemId, i.Count)
		}
		ctx.Response().String("添加物品成功")
	})
}
