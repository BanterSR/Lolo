package mcp

import (
	"fmt"

	"gucooing/lolo/gdconf"
)

// registerWorldTools 世界信息:场景/任务/剧情/成就/商店(详情 + 最小快照列表)
func (s *Server) registerWorldTools() {
	s.registerTool(&Tool{
		Name:        "scene_info",
		Description: "查询场景信息(按场景id),返回名称/所属区域/出生点·采集点·宝箱·副本·怪物数量",
		InputSchema: schema(H{"id": prop("integer", "场景id"), "lang": langProp()}, "id"),
		Handler:     s.sceneInfo,
	})
	s.registerTool(&Tool{
		Name:        "scene_list",
		Description: "列出全部场景的最小快照(id+名称),用于枚举世界地图/区域",
		InputSchema: schema(listProps()),
		Handler:     s.sceneList,
	})
	s.registerTool(&Tool{
		Name:        "quest_info",
		Description: "查询任务信息(按任务id),返回名称/任务组/类型/奖励id/条件组id",
		InputSchema: schema(H{"id": prop("integer", "任务id"), "lang": langProp()}, "id"),
		Handler:     s.questInfo,
	})
	s.registerTool(&Tool{
		Name:        "quest_list",
		Description: "列出全部任务的最小快照(id+名称)。任务较多,可用 limit/offset 分页",
		InputSchema: schema(listProps()),
		Handler:     s.questList,
	})
	s.registerTool(&Tool{
		Name:        "story_info",
		Description: "查询剧情章节(按章节id),返回名称/前置章节/解锁等级/包含的剧情id列表",
		InputSchema: schema(H{"id": prop("integer", "剧情章节id"), "lang": langProp()}, "id"),
		Handler:     s.storyInfo,
	})
	s.registerTool(&Tool{
		Name:        "story_list",
		Description: "列出全部剧情章节的最小快照(id+名称)",
		InputSchema: schema(listProps()),
		Handler:     s.storyList,
	})
	s.registerTool(&Tool{
		Name:        "achievement_info",
		Description: "查询成就(按成就id),返回名称/描述/达成条件类型/计数",
		InputSchema: schema(H{"id": prop("integer", "成就id"), "lang": langProp()}, "id"),
		Handler:     s.achievementInfo,
	})
	s.registerTool(&Tool{
		Name:        "achievement_list",
		Description: "列出全部成就的最小快照(id+名称)。成就较多,可用 limit/offset 分页",
		InputSchema: schema(listProps()),
		Handler:     s.achievementList,
	})
	s.registerTool(&Tool{
		Name:        "shop_info",
		Description: "查询商店(按商店id),返回名称/类型/开放时间/在售物品id与名称",
		InputSchema: schema(H{"id": prop("integer", "商店id"), "lang": langProp()}, "id"),
		Handler:     s.shopInfo,
	})
	s.registerTool(&Tool{
		Name:        "shop_list",
		Description: "列出全部商店的最小快照(id+名称)",
		InputSchema: schema(listProps()),
		Handler:     s.shopList,
	})
}

func (s *Server) sceneInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	sc := gdconf.GetSceneInfo(id)
	if sc == nil || sc.Info == nil {
		return nil, fmt.Errorf("场景不存在: %d", id)
	}
	info := sc.Info
	return H{
		"id":            id,
		"name":          stringText(gdconf.GetStringScene(lang, int32(id)), 0),
		"region":        stringText(gdconf.GetStringScene(lang, int32(id)), 1),
		"bornCount":     len(info.GetBorn()),
		"gatherCount":   len(info.GetGatherPointSetInfo()),
		"treasureCount": len(info.GetCollectionTreasureInfos()),
		"dungeonCount":  len(info.GetDungeonInfos()),
		"monsterCount":  len(info.GetMonsterInfos()),
	}, nil
}

func (s *Server) sceneList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetSceneMap()))
	for sceneId := range gdconf.GetSceneMap() {
		items = append(items, idName{sceneId, stringText(gdconf.GetStringScene(lang, int32(sceneId)), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) questInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	qi := gdconf.GetQuestInfos()[id]
	if qi == nil || qi.Config == nil {
		return nil, fmt.Errorf("任务不存在: %d", id)
	}
	q := qi.Config
	return H{
		"id":           q.GetID(),
		"name":         stringText(gdconf.GetStringQuest(lang, q.GetName()), 0),
		"questGroup":   q.GetQuestGroup(),
		"type":         q.GetNewType(),
		"reward":       q.GetReward(),
		"firstReward":  q.GetFirstReward(),
		"conditionSet": q.GetConditionSetGroupID(),
	}, nil
}

func (s *Server) questList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetQuestInfos()))
	for id, qi := range gdconf.GetQuestInfos() {
		if qi == nil || qi.Config == nil {
			continue
		}
		items = append(items, idName{id, stringText(gdconf.GetStringQuest(lang, qi.Config.GetName()), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) storyInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	ch := gdconf.GetStoryChapters()[id]
	if ch == nil || ch.Config == nil {
		return nil, fmt.Errorf("剧情章节不存在: %d", id)
	}
	c := ch.Config
	storyIds := make([]int32, 0, len(ch.StoryList))
	for sid := range ch.StoryList {
		storyIds = append(storyIds, int32(sid))
	}
	return H{
		"id":               c.GetID(),
		"name":             stringText(gdconf.GetStringStory(lang, c.GetName()), 0),
		"preChapterUnlock": c.GetPreChapterUnlock(),
		"lvUnlock":         c.GetLvUnlock(),
		"storyIds":         storyIds,
	}, nil
}

func (s *Server) storyList(args Args) (any, error) {
	lang := args.Lang()
	items := make([]idName, 0, len(gdconf.GetStoryChapters()))
	for id, ch := range gdconf.GetStoryChapters() {
		if ch == nil || ch.Config == nil {
			continue
		}
		items = append(items, idName{id, stringText(gdconf.GetStringStory(lang, ch.Config.GetName()), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) achievementInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	a := gdconf.GetAchieveConfigure(id)
	if a == nil {
		return nil, fmt.Errorf("成就不存在: %d", id)
	}
	return H{
		"id":            a.GetID(),
		"name":          stringText(gdconf.GetStringAchieve(lang, a.GetUnlockConditionContentID()), 0),
		"desc":          stringText(gdconf.GetStringAchieve(lang, a.GetUnlockConditionContentID()), 1),
		"conditionType": a.GetNewAchieveCondition(),
		"countParam":    a.GetCountParam(),
	}, nil
}

func (s *Server) achievementList(args Args) (any, error) {
	lang := args.Lang()
	all := gdconf.GetAllAchieveConfigure()
	items := make([]idName, 0, len(all))
	for _, a := range all {
		items = append(items, idName{uint32(a.GetID()), stringText(gdconf.GetStringAchieve(lang, a.GetUnlockConditionContentID()), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) shopInfo(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	shop := gdconf.GetShopInfo(id)
	if shop == nil {
		return nil, fmt.Errorf("商店不存在: %d", id)
	}
	items := make([]H, 0, len(shop.GetItemIDs()))
	for _, iid := range shop.GetItemIDs() {
		items = append(items, H{"id": iid, "name": itemName(lang, uint32(iid))})
	}
	return H{
		"id":          shop.GetID(),
		"name":        stringText(gdconf.GetStringShop(lang, shop.GetTextID()), 0),
		"type":        shop.GetNewShopType(),
		"openingTime": shop.GetOpeningTime(),
		"closingTime": shop.GetClosingTime(),
		"items":       items,
	}, nil
}

func (s *Server) shopList(args Args) (any, error) {
	lang := args.Lang()
	all := gdconf.GetAllShopInfo()
	items := make([]idName, 0, len(all))
	for _, shop := range all {
		items = append(items, idName{uint32(shop.GetID()), stringText(gdconf.GetStringShop(lang, shop.GetTextID()), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}
