package mcp

import (
	"fmt"
	"strings"

	"gucooing/lolo/gdconf"
)

// registerGameTools 游戏咨询:查询游戏资源(gdconf),按 id 精确命中,返回精简高价值数据
func (s *Server) registerGameTools() {
	s.registerTool(&Tool{
		Name:        "game_item",
		Description: "查询物品/货币配置(按物品id),返回名称/描述/品质/背包分类",
		InputSchema: schema(H{
			"id":   prop("integer", "物品id"),
			"lang": langProp(),
		}, "id"),
		Handler: s.gameItem,
	})
	s.registerTool(&Tool{
		Name:        "item_list",
		Description: "列出全部物品/货币的最小快照(id+名称)。物品很多,建议配合 limit/offset 分页",
		InputSchema: schema(listProps()),
		Handler:     s.itemList,
	})
	s.registerTool(&Tool{
		Name:        "game_gacha_pools",
		Description: "查询当前开放的卡池列表(含开放时间与up角色)",
		InputSchema: schema(H{"lang": langProp()}),
		Handler:     s.gameGachaPools,
	})
	s.registerTool(&Tool{
		Name:        "game_search",
		Description: "按名称关键字搜索角色/物品,返回匹配的 id 与名称(用于把名称换成id再精确查询;要枚举全部请用各模块的 *_list)",
		InputSchema: schema(H{
			"keyword": prop("string", "名称关键字"),
			"type":    prop("string", "搜索范围: character/item/all,缺省 all"),
			"lang":    langProp(),
			"limit":   prop("integer", "返回数量上限,缺省20,最大50"),
		}, "keyword"),
		Handler: s.gameSearch,
	})
}

func (s *Server) itemList(args Args) (any, error) {
	lang := args.Lang()
	all := gdconf.GetAllItemConfigure()
	items := make([]idName, 0, len(all))
	for _, c := range all {
		items = append(items, idName{uint32(c.GetID()), itemText(lang, c.GetTextID(), 0)})
	}
	return snapshotList(items, args.Int("limit"), args.Int("offset")), nil
}

func (s *Server) gameItem(args Args) (any, error) {
	lang := args.Lang()
	id := args.Uint32("id")
	c := gdconf.GetItemConfigure(id)
	if c == nil {
		return nil, fmt.Errorf("物品不存在: %d", id)
	}
	return H{
		"id":         c.GetID(),
		"name":       itemText(lang, c.GetTextID(), 0),
		"desc":       itemText(lang, c.GetTextID(), 1),
		"quality":    c.GetQuality(),
		"stackCount": c.GetStackCount(),
		"bagTag":     c.GetNewBagItemTag(),
	}, nil
}

func (s *Server) gameGachaPools(args Args) (any, error) {
	lang := args.Lang()
	pools := gdconf.GetOpenGachas()
	list := make([]H, 0, len(pools))
	for _, p := range pools {
		if p == nil || p.Conf == nil {
			continue
		}
		conf := p.Conf
		ups := make([]H, 0, 3)
		for _, cid := range []int32{conf.GetCharacterID1(), conf.GetCharacterID2(), conf.GetCharacterID3()} {
			if cid == 0 {
				continue
			}
			ups = append(ups, H{"id": cid, "name": s.characterName(lang, uint32(cid))})
		}
		list = append(list, H{
			"id":           conf.GetID(),
			"name":         itemText(lang, conf.GetTextID(), 0),
			"openingTime":  conf.GetOpeningTime(),
			"closingTime":  conf.GetClosingTime(),
			"upCharacters": ups,
		})
	}
	return H{"count": len(list), "pools": list}, nil
}

func (s *Server) gameSearch(args Args) (any, error) {
	lang := args.Lang()
	keyword := strings.TrimSpace(args.String("keyword"))
	if keyword == "" {
		return nil, fmt.Errorf("keyword 不能为空")
	}
	scope := args.String("type")
	if scope == "" {
		scope = "all"
	}
	limit := args.Int("limit")
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	list := make([]H, 0, limit)
	if scope == "all" || scope == "character" {
		for id, all := range gdconf.GetCharacterAllMap() {
			if all == nil || all.CharacterInfo == nil {
				continue
			}
			name := charText(lang, all.CharacterInfo.GetNameID())
			if name != "" && strings.Contains(name, keyword) {
				list = append(list, H{"type": "character", "id": id, "name": name})
				if len(list) >= limit {
					return H{"count": len(list), "results": list}, nil
				}
			}
		}
	}
	if scope == "all" || scope == "item" {
		for _, c := range gdconf.GetAllItemConfigure() {
			name := itemText(lang, c.GetTextID(), 0)
			if name != "" && strings.Contains(name, keyword) {
				list = append(list, H{"type": "item", "id": c.GetID(), "name": name})
				if len(list) >= limit {
					break
				}
			}
		}
	}
	return H{"count": len(list), "results": list}, nil
}

// characterName 角色id -> 角色名
func (s *Server) characterName(lang gdconf.Lang, id uint32) string {
	all := gdconf.GetCharacterAll(id)
	if all == nil || all.CharacterInfo == nil {
		return ""
	}
	return charText(lang, all.CharacterInfo.GetNameID())
}
