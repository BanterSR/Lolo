package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Character struct {
	all              *excel.AllCharacterDatas
	CharacterAllMap  map[uint32]*CharacterAllInfo
	GrowthLevelMap   map[int32]map[uint32]*excel.CharacterLevelInfo
	CharacterStarMap map[uint32]map[uint32]*excel.CharacterStarInfo
	CharacterByShop  map[uint32]*excel.CharacterConfigure
	AchieveRewardMap map[uint32]map[uint32]*excel.CharacterAchieveRewardInfo
}

type CharacterAllInfo struct {
	CharacterId   uint32
	CharacterInfo *excel.CharacterConfigure
	LevelRules    []*excel.CharacterLevelRuleInfo
}

func (g *GameConfig) loadCharacter() {
	info := &Character{
		all:              new(excel.AllCharacterDatas),
		CharacterAllMap:  make(map[uint32]*CharacterAllInfo),
		GrowthLevelMap:   make(map[int32]map[uint32]*excel.CharacterLevelInfo),
		CharacterStarMap: make(map[uint32]map[uint32]*excel.CharacterStarInfo),
		CharacterByShop:  make(map[uint32]*excel.CharacterConfigure),
		AchieveRewardMap: make(map[uint32]map[uint32]*excel.CharacterAchieveRewardInfo),
	}
	g.Excel.Character = info
	name := "Character.json"
	ReadJson(g.excelPath, name, &info.all)

	getCharacterAllMap := func(id int32) *CharacterAllInfo {
		if info.CharacterAllMap[uint32(id)] == nil {
			info.CharacterAllMap[uint32(id)] = &CharacterAllInfo{
				CharacterId: uint32(id),
			}
		}
		return info.CharacterAllMap[uint32(id)]
	}
	getLevelMap := func(id int32) map[uint32]*excel.CharacterLevelInfo {
		if info.GrowthLevelMap[id] == nil {
			info.GrowthLevelMap[id] = make(map[uint32]*excel.CharacterLevelInfo)
		}
		return info.GrowthLevelMap[id]
	}
	getStarMap := func(id uint32) map[uint32]*excel.CharacterStarInfo {
		if info.CharacterStarMap[id] == nil {
			info.CharacterStarMap[id] = make(map[uint32]*excel.CharacterStarInfo)
		}
		return info.CharacterStarMap[id]
	}
	getAchieveRewardMap := func(id uint32) map[uint32]*excel.CharacterAchieveRewardInfo {
		if info.AchieveRewardMap[id] == nil {
			info.AchieveRewardMap[id] = make(map[uint32]*excel.CharacterAchieveRewardInfo)
		}
		return info.AchieveRewardMap[id]
	}

	for _, v := range info.all.GetCharacter().GetDatas() {
		getCharacterAllMap(v.ID).CharacterInfo = v
		info.CharacterByShop[uint32(v.ShopID)] = v
	}
	for _, v := range info.all.GetLevelRule().GetDatas() {
		getCharacterAllMap(v.ID).LevelRules = v.LevelRuleInfo
	}
	for _, v := range info.all.GetLevel().GetDatas() {
		levelMap := getLevelMap(v.ID)
		for _, v2 := range v.GetLevelInfo() {
			levelMap[uint32(v2.Level)] = v2
		}
	}
	for _, v := range info.all.GetCharacterStar().GetDatas() {
		levelMap := getStarMap(uint32(v.ID))
		for _, v2 := range v.GetStarInfo() {
			levelMap[uint32(v2.Star)] = v2
		}
	}
	for _, v := range info.all.GetCharacterAchieveReward().GetDatas() {
		rewardMap := getAchieveRewardMap(uint32(v.ID))
		for _, v2 := range v.GetAchieveRewardInfo() {
			rewardMap[uint32(v2.Num)] = v2
		}
	}
}

func GetCharacterAllMap() map[uint32]*CharacterAllInfo {
	return cc.Excel.Character.CharacterAllMap
}

func GetCharacterAll(id uint32) *CharacterAllInfo {
	return cc.Excel.Character.CharacterAllMap[id]
}

func GetCharacterLevelMap(id int32) map[uint32]*excel.CharacterLevelInfo {
	return cc.Excel.Character.GrowthLevelMap[id]
}

func AddCharacterExp(levelId, oldExp int32, oldLevel, maxLevel uint32) (newLevel, newExp uint32) {
	levelMap := GetCharacterLevelMap(levelId)
	for {
		if oldLevel >= maxLevel {
			return oldLevel, 0
		}
		conf, ok := levelMap[oldLevel]
		if !ok || oldExp < conf.NeedExp {
			return oldLevel, uint32(oldExp)
		}
		oldExp -= conf.NeedExp
		oldLevel++
	}
}

func GetCharacterStarMap(characterId uint32) map[uint32]*excel.CharacterStarInfo {
	return cc.Excel.Character.CharacterStarMap[characterId]
}

func GetCharacterStar(characterId, star uint32) *excel.CharacterStarInfo {
	list := cc.Excel.Character.CharacterStarMap[characterId]
	if list == nil {
		return nil
	}
	return list[star]
}

func GetCharacterInfoByShop(shopId uint32) *excel.CharacterConfigure {
	return cc.Excel.Character.CharacterByShop[shopId]
}

func GetAchieveRewardInfo(characterId, rewardIndex uint32) *excel.CharacterAchieveRewardInfo {
	list := cc.Excel.Character.AchieveRewardMap[characterId]
	if list == nil {
		return nil
	}
	return list[rewardIndex]
}
