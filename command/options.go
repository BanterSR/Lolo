package command

type Options struct {
}

type base struct {
	PlayerID uint32 `form:"player_id"`
}

func (b *base) GetPlayerID() uint32 {
	return b.PlayerID
}
