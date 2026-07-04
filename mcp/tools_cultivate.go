package mcp

import (
	"fmt"
	"sort"

	"gucooing/lolo/gdconf"
)

// registerCultivateTools 养成推荐:制作配方 / 武器成长 / 护甲 / 铭文
func (s *Server) registerCultivateTools() {
	s.registerTool(&Tool{
		Name:        "make_recipe",
		Description: "查询制作配方(按配方id),返回产物/所需材料与数量/制作等级/耗时/成功率",
		InputSchema: schema(H{
			"id":   prop("integer", "制作配方id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.makeRecipe,
	})
	s.registerTool(&Tool{
		Name:        "weapon_info",
		Description: "查询武器信息(按武器id),返回名称/类型/基础评分/耐久/被动技能与各级攻击·暴击成长",
		InputSchema: schema(H{
			"id":   prop("integer", "武器id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.weaponInfo,
	})
	s.registerTool(&Tool{
		Name:        "armor_info",
		Description: "查询护甲信息(按护甲id),返回名称/部位/基础评分/套装id/被动技能",
		InputSchema: schema(H{
			"id":   prop("integer", "护甲id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.armorInfo,
	})
	s.registerTool(&Tool{
		Name:        "inscription_info",
		Description: "查询铭文信息(按铭文id),返回名称与各等级的被动效果/金币/升级材料",
		InputSchema: schema(H{
			"id":   prop("integer", "铭文id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.inscriptionInfo,
	})
	s.registerTool(&Tool{
		Name:        "make_list",
		Description: "列出全部制作配方的最小快照(id+产物名),用于枚举所有可制作物",
		InputSchema: schema(listProps()),
		Handler:     s.makeList,
	})
	s.registerTool(&Tool{
		Name:        "weapon_list",
		Description: "列出全部武器的最小快照(id+名称)",
		InputSchema: schema(listProps()),
		Handler:     s.weaponList,
	})
	s.registerTool(&Tool{
		Name:        "armor_list",
		Description: "列出全部护甲的最小快照(id+名称)",
		InputSchema: schema(listProps()),
		Handler:     s.armorList,
	})
	s.registerTool(&Tool{
		Name:        "inscription_list",
		Description: "列出全部铭文的最小快照(id+名称)",
		InputSchema: schema(listProps()),
		Handler:     s.inscriptionList,
	})
}

func (s *Server) makeList(args Args) (any, error) {
	lang := args.Lang()
	all := gdconf.GetAllMakeItemConfigure()
	items := make([]idName, 0, len(all))
	for _, c := range all {
		items = append(items, idName{uint32(c.GetID()), itemName(lang, uint32(c.GetGetItemID()))})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) weaponList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetWeaponAllMap()))
	for id, all := range gdconf.GetWeaponAllMap() {
		if all == nil || all.WeaponInfo == nil {
			continue
		}
		items = append(items, idName{id, itemName(lang, uint32(all.WeaponInfo.GetItemID()))})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) armorList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetArmorAllMap()))
	for id, all := range gdconf.GetArmorAllMap() {
		if all == nil || all.ArmorInfo == nil {
			continue
		}
		items = append(items, idName{id, itemName(lang, uint32(all.ArmorInfo.GetItemID()))})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) inscriptionList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetInscriptionAllMap()))
	for id, all := range gdconf.GetInscriptionAllMap() {
		if all == nil || all.InscriptionInfo == nil {
			continue
		}
		items = append(items, idName{id, itemName(lang, uint32(all.InscriptionInfo.GetItemID()))})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) makeRecipe(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	c := gdconf.GetMakeItemConfigure(id)
	if c == nil {
		return nil, fmt.Errorf("配方不存在: %d", id)
	}
	mats := make([]H, 0, len(c.GetMakeItems()))
	for _, it := range c.GetMakeItems() {
		mats = append(mats, itemBrief(lang, uint32(it.GetItemID()), it.GetItemCount()))
	}
	return H{
		"id":          c.GetID(),
		"output":      itemBrief(lang, uint32(c.GetGetItemID()), 1),
		"makeLevel":   c.GetMakeLevel(),
		"needTime":    c.GetNeedTime(),
		"probability": c.GetSuccessfulProbability(),
		"materials":   mats,
	}, nil
}

func (s *Server) weaponInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetWeaponAllInfo(id)
	if all == nil || all.WeaponInfo == nil {
		return nil, fmt.Errorf("武器不存在: %d", id)
	}
	w := all.WeaponInfo
	idxs := make([]uint32, 0, len(all.PropertyGroup))
	for k := range all.PropertyGroup {
		idxs = append(idxs, k)
	}
	sort.Slice(idxs, func(i, j int) bool { return idxs[i] < idxs[j] })
	growth := make([]H, 0, len(idxs))
	for _, k := range idxs {
		g := all.PropertyGroup[k]
		growth = append(growth, H{
			"level":     g.GetWeaponLevel(),
			"minAttack": g.GetMinAttack(),
			"maxAttack": g.GetMaxAttack(),
			"minCrit":   g.GetMinCriticalRatio(),
			"maxCrit":   g.GetMaxCriticalRatio(),
		})
	}
	return H{
		"id":              w.GetID(),
		"name":            weaponName(lang, id),
		"weaponType":      w.GetNewWeaponType(),
		"systemType":      w.GetNewWeaponSystemType(),
		"baseScore":       w.GetBaseScore(),
		"durability":      w.GetDurability(),
		"passiveSpellIds": w.GetPassiveSpellIDs(),
		"growth":          growth,
	}, nil
}

func (s *Server) armorInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetArmorAllInfo(id)
	if all == nil || all.ArmorInfo == nil {
		return nil, fmt.Errorf("护甲不存在: %d", id)
	}
	a := all.ArmorInfo
	return H{
		"id":              a.GetID(),
		"name":            itemName(lang, uint32(a.GetItemID())),
		"equipType":       a.GetNewEquipType(),
		"baseScore":       a.GetBaseScore(),
		"suit":            []int32{a.GetSuitIndex(), a.GetSuitID2(), a.GetSuitID3()},
		"passiveSpellIds": a.GetPassiveSpellIDs(),
	}, nil
}

func (s *Server) inscriptionInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	all := gdconf.GetInscriptionAllInfo(id)
	if all == nil || all.InscriptionInfo == nil {
		return nil, fmt.Errorf("铭文不存在: %d", id)
	}
	ins := all.InscriptionInfo
	levels := make([]H, 0, len(ins.GetInscriptionGroupInfo()))
	for _, g := range ins.GetInscriptionGroupInfo() {
		mats := make([]H, 0, len(g.GetInscriptionNeedItem()))
		for _, it := range g.GetInscriptionNeedItem() {
			mats = append(mats, itemBrief(lang, uint32(it.GetNeedItemID()), it.GetItemCount()))
		}
		levels = append(levels, H{
			"level":          g.GetLevel(),
			"passiveSpellId": g.GetPassiveSpellID(),
			"gold":           g.GetNeedGoldCount(),
			"materials":      mats,
		})
	}
	return H{
		"id":         ins.GetID(),
		"name":       itemName(lang, uint32(ins.GetItemID())),
		"systemType": ins.GetNewWeaponSystemType(),
		"levels":     levels,
	}, nil
}
