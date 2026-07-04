package mcp

import (
	"fmt"

	"gucooing/lolo/gdconf"
	"gucooing/lolo/protocol/excel"
)

// registerDungeonTools 副本攻略:副本信息(推荐等级/体力/限时/星级/怪物/掉落) 与 怪物属性
func (s *Server) registerDungeonTools() {
	s.registerTool(&Tool{
		Name:        "dungeon_info",
		Description: "查询副本信息(按副本id),返回名称/推荐等级/怪物等级/体力消耗/限时/三星时间/怪物列表/掉落物品",
		InputSchema: schema(H{
			"id":   prop("integer", "副本id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.dungeonInfo,
	})
	s.registerTool(&Tool{
		Name:        "monster_info",
		Description: "查询怪物信息(按怪物id),返回名称/阵营/元素/技能id/指定等级的属性(生命/攻防/暴击)与掉落",
		InputSchema: schema(H{
			"id":    prop("integer", "怪物id"),
			"level": prop("integer", "属性等级,缺省取最低可用等级"),
			"lang":  langProp(),
		}, "id"),
		Handler: s.monsterInfo,
	})
	s.registerTool(&Tool{
		Name:        "dungeon_list",
		Description: "列出全部副本的最小快照(id+名称),用于枚举所有副本",
		InputSchema: schema(listProps()),
		Handler:     s.dungeonList,
	})
	s.registerTool(&Tool{
		Name:        "monster_list",
		Description: "列出全部怪物的最小快照(id+名称),用于枚举所有怪物。数量较多可用 limit/offset 分页",
		InputSchema: schema(listProps()),
		Handler:     s.monsterList,
	})
}

func (s *Server) dungeonList(args Args) (any, error) {
	lang := args.Lang()
	all := gdconf.GetAllDungeonConfigure()
	items := make([]idName, 0, len(all))
	for _, d := range all {
		items = append(items, idName{uint32(d.GetID()), stringText(gdconf.GetStringDungeon(lang, d.GetTitleID()), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) monsterList(args Args) (any, error) {
	lang := args.Lang()
	all := gdconf.GetAllMonsterCharacterConfigure()
	items := make([]idName, 0, len(all))
	for _, m := range all {
		items = append(items, idName{uint32(m.GetID()), charText(lang, m.GetNameID())})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) dungeonInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	d := gdconf.GetDungeonConfigure(id)
	if d == nil {
		return nil, fmt.Errorf("副本不存在: %d", id)
	}
	monsters := make([]H, 0, len(d.GetMonsterIDs()))
	for _, mid := range d.GetMonsterIDs() {
		h := H{"id": mid}
		if m := gdconf.GetMonsterCharacterConfigure(uint32(mid)); m != nil {
			h["name"] = charText(lang, m.GetNameID())
		}
		monsters = append(monsters, h)
	}
	return H{
		"id":          d.GetID(),
		"name":        stringText(gdconf.GetStringDungeon(lang, d.GetTitleID()), 0),
		"recommendLv": d.GetRecommendLv(),
		"monsterLv":   d.GetMonsterLv(),
		"needLevel":   d.GetNeedPlayerLevel(),
		"difficulty":  d.GetNewDifficultyLevel(),
		"element":     d.GetNewElementType(),
		"maxParty":    d.GetUseCharacterMaxNumber(),
		"fightTime":   d.GetFightTime(),
		"starTimers":  []int32{d.GetStarTimer1(), d.GetStarTimer2(), d.GetStarTimer3()},
		"stamina":     itemBrief(lang, uint32(d.GetItemID()), d.GetItemNum()),
		"monsters":    monsters,
		"drops":       rewardDrops(lang, uint32(d.GetRewardID())),
	}, nil
}

func (s *Server) monsterInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	m := gdconf.GetMonsterCharacterConfigure(id)
	if m == nil {
		return nil, fmt.Errorf("怪物不存在: %d", id)
	}
	res := H{
		"id":              m.GetID(),
		"name":            charText(lang, m.GetNameID()),
		"camp":            m.GetNewCampType(),
		"element":         m.GetNewElementType(),
		"normalSpellIds":  m.GetNormalSpellIDs(),
		"specialSpellIds": m.GetSpecialSpellIDs(),
		"drops":           rewardDrops(lang, uint32(m.GetRewardPoolID())),
	}
	if st := monsterStats(uint32(m.GetGrowthModelID()), args.Uint32("level")); st != nil {
		res["stats"] = H{
			"level":         st.GetLevel(),
			"maxHP":         st.GetMaxHP(),
			"maxMP":         st.GetMaxMP(),
			"attack":        st.GetAttack(),
			"defense":       st.GetDefense(),
			"criticalRatio": st.GetCriticalRatio(),
		}
	}
	return res, nil
}

// monsterStats 取怪物成长模型指定等级属性,level 为 0 或缺失时取最低可用等级
func monsterStats(modelId, level uint32) *excel.MonsterCharacterGrowthModelLevelInfo {
	model := gdconf.GetMonsterGrowthModelConfigure(modelId)
	if model == nil {
		return nil
	}
	group := model.GetLevelGroupInfo()
	if len(group) == 0 {
		return nil
	}
	if level != 0 {
		for _, lv := range group {
			if uint32(lv.GetLevel()) == level {
				return lv
			}
		}
	}
	return group[0]
}
