package gdconf

type GachaProbability struct {
	GuaranteeSR          int32 `json:"guaranteeSR"`          // sr 角色保底
	GuaranteeSSR         int32 `json:"guaranteeSSR"`         // ssr 角色保底
	ProbabilitySSRUP     int32 `json:"probabilitySSRUP"`     // ssrup 角色概率
	ProbabilitySSR       int32 `json:"probabilitySSR"`       // ssr角色概率
	ProbabilitySR        int32 `json:"probabilitySR"`        // sr角色概率
	ProbabilityPosterSSR int32 `json:"probabilityPosterSSR"` // ssr 映像概率
	ProbabilityPosterSR  int32 `json:"probabilityPosterSR"`  // sr 映像概率
	GuaranteeFashion     int32 `json:"guaranteeFashion"`     // 服装保底
	GuaranteeDesire      int32 `json:"guaranteeDesire"`      // 愿望保底
	ProbabilityFashion   int32 `json:"probabilityFashion"`   // 服装概率
	ProbabilityFurniture int32 `json:"probabilityFurniture"` // 家具概率
	ProbabilityWeaponSSR int32 `json:"probabilityWeaponSSR"` // ssr武器概率
	ProbabilityWeaponSR  int32 `json:"probabilityWeaponSR"`  // sr武器概率
	ProbabilityDyeStuff  int32 `json:"probabilityDyeStuff"`  // 染色剂概率
}

func (g *GameConfig) loadGachaProbability() {
	g.Data.GachaProbabilitys = make(map[int32]*GachaProbability)
	ReadJson(g.dataPath, "GachaProbability.json", &g.Data.GachaProbabilitys)
}

func GetGachaProbability(id int32) *GachaProbability {
	return cc.Data.GachaProbabilitys[id]
}
