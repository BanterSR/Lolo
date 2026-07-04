package mcp

import (
	"sort"

	"github.com/bytedance/sonic"

	"gucooing/lolo/gdconf"
)

// Tool 一个可被 mcp / openai 调用的工具
type Tool struct {
	Name        string                       // 工具名
	Description string                       // 工具描述,供 AI 判断何时调用
	InputSchema map[string]any               // 入参 JSON Schema (object)
	Handler     func(args Args) (any, error) // 执行函数,返回结果(会被序列化为 JSON 文本)
}

// H 工具返回结果的通用键值结构
type H = map[string]any

// Args 工具入参,由 json 解析而来
type Args map[string]any

func (a Args) String(key string) string {
	if v, ok := a[key].(string); ok {
		return v
	}
	return ""
}

func (a Args) Uint32(key string) uint32 {
	switch v := a[key].(type) {
	case float64:
		return uint32(v)
	case int:
		return uint32(v)
	case string:
		return uint32(parseInt(v))
	}
	return 0
}

func (a Args) Int(key string) int {
	switch v := a[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		return int(parseInt(v))
	}
	return 0
}

func (a Args) Bool(key string) bool {
	if v, ok := a[key].(bool); ok {
		return v
	}
	return false
}

// Lang 取语言参数并归一化为受支持的语言,缺省简体中文
func (a Args) Lang() gdconf.Lang {
	switch a.String("lang") {
	case string(gdconf.LangTraditional), "zh-tw", "cht", "繁体", "繁體":
		return gdconf.LangTraditional
	case string(gdconf.LangEnglish), "en-us", "english", "英文":
		return gdconf.LangEnglish
	case string(gdconf.LangJapanese), "jp", "japanese", "日文":
		return gdconf.LangJapanese
	case string(gdconf.LangKorea), "kr", "korean", "韩文":
		return gdconf.LangKorea
	default:
		return gdconf.LangSimplified
	}
}

func parseInt(s string) int64 {
	var n int64
	neg := false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		return -n
	}
	return n
}

// schema 构造 object 类型入参 schema
func schema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

// prop 构造一个属性定义
func prop(typ, desc string) map[string]any {
	return map[string]any{"type": typ, "description": desc}
}

// langProp 语言参数(所有查询资源文本的工具通用)
func langProp() map[string]any {
	return map[string]any{
		"type":        "string",
		"description": "语言,可选 zh-Hans(简体) zh-Hant(繁体) en ja ko,缺省 zh-Hans",
		"enum":        []string{"zh-Hans", "zh-Hant", "en", "ja", "ko"},
	}
}

// textAt 安全取文本数组指定下标
func textAt(list []string, i int) string {
	if i >= 0 && i < len(list) {
		return list[i]
	}
	return ""
}

// itemText 物品文本 id -> 名称/描述(index 0 名称,1 描述)
func itemText(lang gdconf.Lang, textID int32, index int) string {
	return textAt(gdconf.GetStringItemText(lang, textID).GetText(), index)
}

// charText 角色名文本 id -> 角色名
func charText(lang gdconf.Lang, nameID int32) string {
	return textAt(gdconf.GetStringCharacterName(lang, nameID).GetText(), 0)
}

// stringText 取任意 String 文本表配置的指定下标(0 名称,1 描述);
// proto getter 对 nil 接收者安全,传入未命中的 nil 配置返回空串
func stringText(c interface{ GetText() []string }, index int) string {
	return textAt(c.GetText(), index)
}

// itemName 物品id -> 名称
func itemName(lang gdconf.Lang, itemId uint32) string {
	if conf := gdconf.GetItemConfigure(itemId); conf != nil {
		return itemText(lang, conf.GetTextID(), 0)
	}
	return ""
}

// itemBrief 物品id+数量 -> {itemId,name,num},养成/掉落/配方等处解析材料通用
func itemBrief(lang gdconf.Lang, itemId uint32, num int32) H {
	return H{"itemId": itemId, "name": itemName(lang, itemId), "num": num}
}

// rewardDrops 奖励池id -> 掉落物品列表 {itemId,name,min,max},副本/怪物掉落通用
func rewardDrops(lang gdconf.Lang, rewardId uint32) []H {
	if rewardId == 0 {
		return []H{}
	}
	groups := gdconf.GetRewardItemPoolByRewardId(rewardId)
	list := make([]H, 0, len(groups))
	for _, g := range groups {
		itemId := uint32(g.GetItemID())
		if itemId == 0 {
			itemId = uint32(g.GetShowItemID())
		}
		list = append(list, H{"itemId": itemId, "name": itemName(lang, itemId), "min": g.GetItemMinCount(), "max": g.GetItemMaxCount()})
	}
	return list
}

// weaponName 武器id -> 名称(经武器配置的 ItemID 再取物品文本)
func weaponName(lang gdconf.Lang, weaponId uint32) string {
	all := gdconf.GetWeaponAllInfo(weaponId)
	if all == nil || all.WeaponInfo == nil {
		return ""
	}
	if item := gdconf.GetItemConfigure(uint32(all.WeaponInfo.GetItemID())); item != nil {
		return itemText(lang, item.GetTextID(), 0)
	}
	return ""
}

// idName 最小快照的一项:仅 id 与名称
type idName struct {
	Id   uint32
	Name string
}

// snapshotList 各模块 *_list 工具通用的最小快照:按 id 升序,只含 id+名称,最省 token。
// limit<=0 返回全部;offset 用于分页(超大表如物品/任务可按需翻页)。
func snapshotList(items []idName, limit, offset int) H {
	sort.Slice(items, func(i, j int) bool { return items[i].Id < items[j].Id })
	total := len(items)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	list := make([]H, 0, end-offset)
	for _, it := range items[offset:end] {
		list = append(list, H{"id": it.Id, "name": it.Name})
	}
	return H{"total": total, "count": len(list), "list": list}
}

// listProps *_list 工具通用入参(语言+分页)
func listProps() map[string]any {
	return H{
		"lang":   langProp(),
		"limit":  prop("integer", "返回数量上限,缺省全部;超大表(物品/任务)可用它+offset分页"),
		"offset": prop("integer", "偏移量,分页用,缺省0"),
	}
}

// toText 把工具结果序列化为紧凑 JSON 文本
func toText(v any) string {
	if v == nil {
		return "null"
	}
	if s, ok := v.(string); ok {
		return s
	}
	bin, err := sonic.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(bin)
}
