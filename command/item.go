package command

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
)

type CommandItem struct {
	ItemId uint32
	Count  int64
	All    bool
}

func (c *CommandItem) Name() string {
	return "item"
}

func (c *CommandItem) Handle(s *model.Player) {
	if c.Count < 0 {
		c.Count = 1
	}
	if c.All {
		for _, conf := range gdconf.GetAllItemConfigure() {
			s.AddAllTypeItem(uint32(conf.ID), c.Count)
		}
	} else {
		s.AddAllTypeItem(c.ItemId, c.Count)
	}
}
