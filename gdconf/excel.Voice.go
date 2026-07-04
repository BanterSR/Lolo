package gdconf

import "gucooing/lolo/protocol/excel"

// 语言 -> Voice 资源文件名(与 String 一样按语言分文件)
var voiceFileMap = map[Lang]string{
	LangSimplified:  "Voice_Simplified.json",
	LangTraditional: "Voice_Traditional.json",
	LangEnglish:     "Voice_English.json",
	LangJapanese:    "Voice_Japanese.json",
	LangKorea:       "Voice_Korea.json",
}

type Voice struct {
	VoiceCharacter map[Lang]map[int32]*excel.VoiceCharacterConfigure
}

func (g *GameConfig) loadVoice() {
	info := &Voice{
		VoiceCharacter: make(map[Lang]map[int32]*excel.VoiceCharacterConfigure),
	}
	g.Excel.Voice = info
	for lang, name := range voiceFileMap {
		all := new(excel.AllVoiceDatas)
		ReadJson(g.excelPath, name, &all)
		character := make(map[int32]*excel.VoiceCharacterConfigure)
		for _, v := range all.GetVoiceCharacter().GetDatas() {
			character[v.GetID()] = v
		}
		info.VoiceCharacter[lang] = character
	}
}

// GetVoiceCharacter 角色语音(按角色的 VoiceID):台词条目一览(标签/解锁等级/分类/音频路径)
func GetVoiceCharacter(lang Lang, id int32) *excel.VoiceCharacterConfigure {
	return cc.Excel.Voice.VoiceCharacter[lang][id]
}
