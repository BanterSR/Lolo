package gdconf

type Constant struct {
	DefaultCharacter     []uint32 `json:"DefaultCharacter"`
	DefaultBadge         uint32   `json:"DefaultBadge"`
	DefaultUmbrellaId    uint32   `json:"DefaultUmbrellaId"`
	EquipmentPresetNum   uint32   `json:"EquipmentPresetNum"`
	OutfitPresetNum      uint32   `json:"OutfitPresetNum"`
	DefaultInstanceIndex uint32   `json:"DefaultInstanceIndex"`
	DefaultSceneId       uint32   `json:"DefaultSceneId"`
	DefaultChannelId     uint32   `json:"DefaultChannelId"`
	ChannelTick          int      `json:"ChannelTick"`
}

func (g *GameConfig) loadConstant() {
	g.Data.Constant = new(Constant)
	ReadJson(g.dataPath, "Constant.json", &g.Data.Constant)
}

func GetConstant() *Constant {
	return cc.Data.Constant
}
