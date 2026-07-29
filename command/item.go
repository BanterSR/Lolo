package command

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
)

type item struct {
	baseCommand
	ItemId uint32 `form:"item_id"`
	Count  int64  `form:"count"`
	All    bool   `form:"all"`
}

func (i *item) Options() *CommandOptions {
	return &CommandOptions{
		IsPlayer: true,
	}
}

func (i *item) Handle(s *model.Player) (string, error) {
	if i.Count < 0 {
		i.Count = 1
	}
	if i.All {
		for _, conf := range gdconf.GetAllItemConfigure() {
			s.AddAllTypeItem(uint32(conf.ID), i.Count)
		}
	} else {
		s.AddAllTypeItem(i.ItemId, i.Count)
	}
	return "物品获取完成", nil
}
