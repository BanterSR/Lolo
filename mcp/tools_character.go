package mcp

import (
	"fmt"
	"sort"

	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/excel"
)

// registerCharacterTools 角色攻略:角色信息 / 技能详情 / 养成(升级与突破)线
func (s *Server) registerCharacterTools() {
	s.registerTool(&Tool{
		Name:        "character_info",
		Description: "查询角色信息(按角色id),返回角色名/元素/阵营/武器与护甲类型/主动·被动·EX技能id/默认武器",
		InputSchema: schema(H{
			"id":   prop("integer", "角色id,例如 101001"),
			"lang": langProp(),
		}, "id"),
		Handler: s.characterInfo,
	})
	s.registerTool(&Tool{
		Name:        "character_skills",
		Description: "查询角色技能详情(按角色id),返回每个技能的名称/描述/类型/伤害类型/冷却/耗蓝(按技能等级,缺省1级)",
		InputSchema: schema(H{
			"id":    prop("integer", "角色id"),
			"level": prop("integer", "技能等级,缺省1"),
			"lang":  langProp(),
		}, "id"),
		Handler: s.characterSkills,
	})
	s.registerTool(&Tool{
		Name:        "character_voice",
		Description: "查询角色语音台词条目(按角色id):每条语音的分类/标签/解锁等级/音频路径。注:服务器数据不含台词全文与声优(CV),仅有条目",
		InputSchema: schema(H{
			"id":   prop("integer", "角色id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.characterVoice,
	})
	s.registerTool(&Tool{
		Name:        "character_growth",
		Description: "查询角色养成(按角色id),返回各阶属性上限与升阶材料、突破(星级)所需材料/金币/属性加成",
		InputSchema: schema(H{
			"id":   prop("integer", "角色id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.characterGrowth,
	})
	s.registerTool(&Tool{
		Name:        "character_list",
		Description: "列出全部角色的最小快照(id+名称),用于枚举所有角色再按 id 精查",
		InputSchema: schema(listProps()),
		Handler:     s.characterList,
	})
}

func (s *Server) characterList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetCharacterAllMap()))
	for id, all := range gdconf.GetCharacterAllMap() {
		if all == nil || all.CharacterInfo == nil {
			continue
		}
		items = append(items, idName{id, charText(lang, all.CharacterInfo.GetNameID())})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) characterInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetCharacterAll(id)
	if all == nil || all.CharacterInfo == nil {
		return nil, fmt.Errorf("角色不存在: %d", id)
	}
	c := all.CharacterInfo
	return H{
		"id":              c.GetID(),
		"name":            charText(lang, c.GetNameID()),
		"itemId":          c.GetItemID(),
		"color":           c.GetColor(),
		"element":         c.GetNewElementType(),
		"camp":            c.GetNewCampType(),
		"sex":             c.GetNewSexType(),
		"weaponType":      c.GetNewWeaponType(),
		"armorType":       c.GetNewArmorType(),
		"defaultWeapon":   c.GetDefaultWeaponID(),
		"spellIds":        c.GetSpellIDs(),
		"passiveSpellIds": c.GetPassiveSpellIDs(),
		"exSpellIds":      c.GetExSpellIDs(),
		"handSpellIds":    c.GetHandSpellID(),
		"posterIds":       c.GetPosterIDs(),
		"openingTime":     c.GetOpeningTime(),
		"closingTime":     c.GetClosingTime(),
		"isShow":          c.GetIsShow(),
	}, nil
}

func (s *Server) characterVoice(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetCharacterAll(id)
	if all == nil || all.CharacterInfo == nil {
		return nil, fmt.Errorf("角色不存在: %d", id)
	}
	voice := gdconf.GetVoiceCharacter(lang, all.CharacterInfo.GetVoiceID())
	if voice == nil {
		return H{"id": id, "name": charText(lang, all.CharacterInfo.GetNameID()), "count": 0, "voices": []H{}}, nil
	}
	list := make([]H, 0, len(voice.GetVoiceCharacterItem()))
	for _, it := range voice.GetVoiceCharacterItem() {
		v := H{
			"type":        it.GetNewVoiceType(),
			"label":       it.GetVoiceName(),
			"unlockLevel": it.GetLevel(),
			"soundPath":   it.GetSoundPath(),
		}
		if text := it.GetTextValue(); text != "" {
			v["text"] = text
		}
		list = append(list, v)
	}
	return H{
		"id":     id,
		"name":   charText(lang, all.CharacterInfo.GetNameID()),
		"count":  len(list),
		"voices": list,
	}, nil
}

func (s *Server) characterSkills(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetCharacterAll(id)
	if all == nil || all.CharacterInfo == nil {
		return nil, fmt.Errorf("角色不存在: %d", id)
	}
	level := args.Uint32("level")
	if level == 0 {
		level = 1
	}
	c := all.CharacterInfo
	skills := make([]H, 0)
	appendSkills := func(kind string, ids []int32) {
		for _, sid := range ids {
			skills = append(skills, skillDetail(lang, uint32(sid), level, kind))
		}
	}
	appendSkills("active", c.GetSpellIDs())
	appendSkills("passive", c.GetPassiveSpellIDs())
	appendSkills("ex", c.GetExSpellIDs())
	appendSkills("hand", c.GetHandSpellID())
	return H{"id": id, "name": charText(lang, c.GetNameID()), "count": len(skills), "skills": skills}, nil
}

func (s *Server) characterGrowth(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetCharacterAll(id)
	if all == nil || all.CharacterInfo == nil {
		return nil, fmt.Errorf("角色不存在: %d", id)
	}
	rules := make([]H, 0, len(all.LevelRules))
	for _, r := range all.LevelRules {
		mats := make([]H, 0, len(r.GetRuleNeedItem()))
		for _, it := range r.GetRuleNeedItem() {
			mats = append(mats, itemBrief(lang, uint32(it.GetNeedItemID()), it.GetNeedItemCount()))
		}
		rules = append(rules, H{
			"topLevel":  r.GetTopLevel(),
			"maxLevel":  r.GetTopMaxLevel(),
			"maxHP":     r.GetMaxHP(),
			"attack":    r.GetAttack(),
			"defense":   r.GetDefense(),
			"materials": mats,
		})
	}
	stars := make([]H, 0)
	for star, info := range gdconf.GetCharacterStarMap(id) {
		stars = append(stars, H{
			"star":     star,
			"material": itemBrief(lang, uint32(info.GetItemID()), info.GetItemNum()),
			"gold":     info.GetNeedGoldCount(),
			"para":     info.GetPara(),
		})
	}
	sort.Slice(stars, func(i, j int) bool { return stars[i]["star"].(uint32) < stars[j]["star"].(uint32) })
	res := H{
		"id":         id,
		"name":       charText(lang, all.CharacterInfo.GetNameID()),
		"levelRules": rules,
		"stars":      stars,
	}
	// 官方推荐练度(推荐武器/目标等级/练度副本)
	if p := gdconf.GetCharacterPractice(id); p != nil {
		res["recommend"] = H{
			"weaponId":        p.GetWeaponID(),
			"weaponName":      weaponName(lang, uint32(p.GetWeaponID())),
			"targetLevel":     p.GetCharacterLv(),
			"practiceDungeon": p.GetDungeonID(),
		}
	}
	return res, nil
}

// skillDetail 解析单个技能在指定等级的详情
func skillDetail(lang gdconf.Lang, skillId, level uint32, kind string) H {
	h := H{"skillId": skillId, "kind": kind}
	conf := spellAtLevel(skillId, level)
	if conf == nil {
		return h
	}
	h["level"] = conf.GetLevel()
	h["name"] = stringText(gdconf.GetStringSpell(lang, conf.GetTextID()), 0)
	h["desc"] = stringText(gdconf.GetStringSpell(lang, conf.GetTextID()), 1)
	h["spellType"] = conf.GetNewSpellType()
	h["damageType"] = conf.GetNewDamageType()
	h["cd"] = conf.GetCD()
	h["costMp"] = conf.GetCostMp()
	return h
}

// spellAtLevel 取技能指定等级配置,缺失时回退到最小可用等级
func spellAtLevel(skillId, level uint32) *excel.SpellConfigure {
	lvMap := gdconf.GetSpellLevelMap(skillId)
	if lvMap == nil {
		return nil
	}
	if conf, ok := lvMap[level]; ok {
		return conf
	}
	var minLv uint32
	var res *excel.SpellConfigure
	for lv, conf := range lvMap {
		if res == nil || lv < minLv {
			minLv, res = lv, conf
		}
	}
	return res
}
