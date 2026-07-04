package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Lang string

const (
	LangSimplified  Lang = "zh-Hans" // 简体中文
	LangTraditional Lang = "zh-Hant" // 繁体中文
	LangEnglish     Lang = "en"      // 英文
	LangJapanese    Lang = "ja"      // 日文
	LangKorea       Lang = "ko"      // 韩文
)

// 语言 -> String 资源文件名
var langFileMap = map[Lang]string{
	LangSimplified:  "String_Simplified.json",
	LangTraditional: "String_Traditional.json",
	LangEnglish:     "String_English.json",
	LangJapanese:    "String_Japanese.json",
	LangKorea:       "String_Korea.json",
}

type String struct {
	StringItemText      map[Lang]map[int32]*excel.StringItemTextConfigure
	StringCharacterName map[Lang]map[int32]*excel.StringCharacterNameConfigure
	StringSpell         map[Lang]map[int32]*excel.StringSpellConfigure
	StringDungeon       map[Lang]map[int32]*excel.StringDungeonConfigure
	StringQuest         map[Lang]map[int32]*excel.StringQuestConfigure
	StringScene         map[Lang]map[int32]*excel.StringSceneConfigure
	StringAchieve       map[Lang]map[int32]*excel.StringAchieveConfigure
	StringShop          map[Lang]map[int32]*excel.StringShopConfigure
	StringStory         map[Lang]map[int32]*excel.StringStoryConfigure
	StringMake          map[Lang]map[int32]*excel.StringMakeConfigure
}

func (g *GameConfig) loadString() {
	info := &String{
		StringItemText:      make(map[Lang]map[int32]*excel.StringItemTextConfigure),
		StringCharacterName: make(map[Lang]map[int32]*excel.StringCharacterNameConfigure),
		StringSpell:         make(map[Lang]map[int32]*excel.StringSpellConfigure),
		StringDungeon:       make(map[Lang]map[int32]*excel.StringDungeonConfigure),
		StringQuest:         make(map[Lang]map[int32]*excel.StringQuestConfigure),
		StringScene:         make(map[Lang]map[int32]*excel.StringSceneConfigure),
		StringAchieve:       make(map[Lang]map[int32]*excel.StringAchieveConfigure),
		StringShop:          make(map[Lang]map[int32]*excel.StringShopConfigure),
		StringStory:         make(map[Lang]map[int32]*excel.StringStoryConfigure),
		StringMake:          make(map[Lang]map[int32]*excel.StringMakeConfigure),
	}
	g.Excel.String = info
	for lang, name := range langFileMap {
		all := new(excel.AllStringDatas)
		ReadJson(g.excelPath, name, &all)
		itemText := make(map[int32]*excel.StringItemTextConfigure)
		for _, v := range all.GetStringItemText().GetDatas() {
			itemText[v.GetID()] = v
		}
		info.StringItemText[lang] = itemText
		charName := make(map[int32]*excel.StringCharacterNameConfigure)
		for _, v := range all.GetStringCharacterName().GetDatas() {
			charName[v.GetID()] = v
		}
		info.StringCharacterName[lang] = charName
		spell := make(map[int32]*excel.StringSpellConfigure)
		for _, v := range all.GetStringSpell().GetDatas() {
			spell[v.GetID()] = v
		}
		info.StringSpell[lang] = spell
		dungeon := make(map[int32]*excel.StringDungeonConfigure)
		for _, v := range all.GetStringDungeon().GetDatas() {
			dungeon[v.GetID()] = v
		}
		info.StringDungeon[lang] = dungeon
		quest := make(map[int32]*excel.StringQuestConfigure)
		for _, v := range all.GetStringQuestText().GetDatas() {
			quest[v.GetID()] = v
		}
		info.StringQuest[lang] = quest
		scene := make(map[int32]*excel.StringSceneConfigure)
		for _, v := range all.GetStringScene().GetDatas() {
			scene[v.GetID()] = v
		}
		info.StringScene[lang] = scene
		achieve := make(map[int32]*excel.StringAchieveConfigure)
		for _, v := range all.GetStringAchieve().GetDatas() {
			achieve[v.GetID()] = v
		}
		info.StringAchieve[lang] = achieve
		shop := make(map[int32]*excel.StringShopConfigure)
		for _, v := range all.GetStringShop().GetDatas() {
			shop[v.GetID()] = v
		}
		info.StringShop[lang] = shop
		story := make(map[int32]*excel.StringStoryConfigure)
		for _, v := range all.GetStringStory().GetDatas() {
			story[v.GetID()] = v
		}
		info.StringStory[lang] = story
		makeText := make(map[int32]*excel.StringMakeConfigure)
		for _, v := range all.GetStringMakeText().GetDatas() {
			makeText[v.GetID()] = v
		}
		info.StringMake[lang] = makeText
	}
}

func GetStringItemText(lang Lang, id int32) *excel.StringItemTextConfigure {
	return cc.Excel.String.StringItemText[lang][id]
}

func GetStringCharacterName(lang Lang, id int32) *excel.StringCharacterNameConfigure {
	return cc.Excel.String.StringCharacterName[lang][id]
}

func GetStringSpell(lang Lang, id int32) *excel.StringSpellConfigure {
	return cc.Excel.String.StringSpell[lang][id]
}

func GetStringDungeon(lang Lang, id int32) *excel.StringDungeonConfigure {
	return cc.Excel.String.StringDungeon[lang][id]
}

func GetStringQuest(lang Lang, id int32) *excel.StringQuestConfigure {
	return cc.Excel.String.StringQuest[lang][id]
}

func GetStringScene(lang Lang, id int32) *excel.StringSceneConfigure {
	return cc.Excel.String.StringScene[lang][id]
}

func GetStringAchieve(lang Lang, id int32) *excel.StringAchieveConfigure {
	return cc.Excel.String.StringAchieve[lang][id]
}

func GetStringShop(lang Lang, id int32) *excel.StringShopConfigure {
	return cc.Excel.String.StringShop[lang][id]
}

func GetStringStory(lang Lang, id int32) *excel.StringStoryConfigure {
	return cc.Excel.String.StringStory[lang][id]
}

func GetStringMake(lang Lang, id int32) *excel.StringMakeConfigure {
	return cc.Excel.String.StringMake[lang][id]
}
