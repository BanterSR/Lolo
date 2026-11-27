package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"sync/atomic"

	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

type TreasureBox struct {
	Index           uint32                 `json:"index,omitempty"`           // 序号
	BoxId           uint32                 `json:"boxId,omitempty"`           // id
	Type            proto.ETreasureBoxType `json:"type,omitempty"`            // 类型
	State           proto.TreasureBoxState `json:"state,omitempty"`           // 状态
	NextRefreshTime int64                  `json:"nextRefreshTime,omitempty"` // 下次更新时间
}

func (t *TreasureBox) TreasureBoxData() *proto.TreasureBoxData {
	info := &proto.TreasureBoxData{
		Index:           t.Index,
		BoxId:           t.BoxId,
		Type:            t.Type,
		State:           t.State,
		NextRefreshTime: t.NextRefreshTime,
		Rewards:         make([]*proto.ItemDetail, 0),
	}

	return info
}

// 当前场景的临时背包
type SceneTempPack struct {
	index        atomic.Uint32
	TempPack     []EBagItemTag        // 玩家临时背包-临时快照
	TempAreaPack map[uint32]*DropItem // 掉落物
}

type DropItem struct {
	Index uint32
	Items []EBagItemTag
}

func (d *DropItem) PbDropItem() *proto.DropItem {
	if d == nil {
		return nil
	}
	info := &proto.DropItem{
		Index: d.Index,
		Items: make([]*proto.ItemDetail, len(d.Items)),
	}
	for i, item := range d.Items {
		if nilBagItem(item) {
			info.Items[i] = nil
			continue
		}
		info.Items[i] = item.ItemDetail()
	}
	return info
}

func nilBagItem(item EBagItemTag) bool {
	if item == nil {
		return true
	}
	value := reflect.ValueOf(item)
	return value.Kind() == reflect.Ptr && value.IsNil()
}

const tempPackArchiveKey = "scene_temp_pack_v1"

type tempPackArchive struct {
	NextIndex uint32                `json:"nextIndex,omitempty"`
	Items     []tempPackArchiveItem `json:"items,omitempty"`
	Drops     []tempPackArchiveDrop `json:"drops,omitempty"`
}

type tempPackArchiveItem struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type tempPackArchiveDrop struct {
	Index uint32                `json:"index"`
	Items []tempPackArchiveItem `json:"items"`
}

func archiveTempPackItem(item EBagItemTag) (tempPackArchiveItem, error) {
	if nilBagItem(item) {
		return tempPackArchiveItem{}, errors.New("nil temporary item")
	}
	var name string
	switch item.(type) {
	case *ItemBaseInfo:
		name = "base"
	case *ItemWeaponInfo:
		name = "weapon"
	case *ItemArmorInfo:
		name = "armor"
	case *ItemPosterInfo:
		name = "poster"
	case *ItemFashionInfo:
		name = "fashion"
	case *ItemInscriptionInfo:
		name = "inscription"
	case *ItemPetInfo:
		name = "pet"
	default:
		return tempPackArchiveItem{}, fmt.Errorf("unsupported temporary item %T", item)
	}
	data, err := json.Marshal(item)
	return tempPackArchiveItem{Type: name, Data: data}, err
}

func restoreTempPackItem(item tempPackArchiveItem) (result EBagItemTag, err error) {
	defer func() {
		if recover() != nil {
			result, err = nil, errors.New("temporary item is invalid")
		}
	}()
	if len(item.Data) == 0 {
		return nil, errors.New("temporary item data is missing")
	}
	switch item.Type {
	case "base":
		result = new(ItemBaseInfo)
	case "weapon":
		result = new(ItemWeaponInfo)
	case "armor":
		result = new(ItemArmorInfo)
	case "poster":
		result = new(ItemPosterInfo)
	case "fashion":
		result = new(ItemFashionInfo)
	case "inscription":
		result = new(ItemInscriptionInfo)
	case "pet":
		result = new(ItemPetInfo)
	default:
		return nil, errors.New("temporary item type is invalid")
	}
	if err := json.Unmarshal(item.Data, result); err != nil {
		return nil, err
	}
	if nilBagItem(result) || result.ItemDetail() == nil || result.ItemDetail().MainItem == nil {
		return nil, errors.New("temporary item is invalid")
	}
	return result, nil
}

// SaveTempPackArchive stores scene-owned drops in the player's existing
// archive so a reconnect or ordinary scene transition cannot lose them.
func (s *Player) SaveTempPackArchive(pack *SceneTempPack) error {
	if s == nil {
		return errors.New("player is missing")
	}
	archive := tempPackArchive{}
	if pack != nil {
		for _, item := range pack.TempPack {
			encoded, err := archiveTempPackItem(item)
			if err != nil {
				return err
			}
			archive.Items = append(archive.Items, encoded)
		}
		for _, drop := range pack.TempAreaPack {
			if drop == nil || drop.Index == 0 {
				return errors.New("temporary drop is invalid")
			}
			encoded := tempPackArchiveDrop{Index: drop.Index}
			for _, item := range drop.Items {
				itemData, err := archiveTempPackItem(item)
				if err != nil {
					return err
				}
				encoded.Items = append(encoded.Items, itemData)
			}
			archive.Drops = append(archive.Drops, encoded)
		}
		archive.NextIndex = pack.index.Load()
	}
	data, err := json.Marshal(archive)
	if err != nil {
		return err
	}
	s.GetArchive().SetArchiveMap(tempPackArchiveKey, string(data))
	return nil
}

func (s *Player) LoadTempPackArchive() (*SceneTempPack, error) {
	if s == nil {
		return nil, errors.New("player is missing")
	}
	raw := s.GetArchive().GetArchiveValue(tempPackArchiveKey)
	pack := newSceneTempPack()
	if raw == "" {
		return pack, nil
	}
	var archive tempPackArchive
	if err := json.Unmarshal([]byte(raw), &archive); err != nil {
		return nil, err
	}
	seen := make(map[EBagItemTag]bool)
	for _, encoded := range archive.Items {
		item, err := restoreTempPackItem(encoded)
		if err != nil {
			return nil, err
		}
		if seen[item] {
			return nil, errors.New("duplicate temporary item")
		}
		seen[item] = true
		item.SetPackType(proto.ItemDetail_PackType_TempPack)
		pack.TempPack = append(pack.TempPack, item)
	}
	for _, encoded := range archive.Drops {
		if encoded.Index == 0 || pack.TempAreaPack[encoded.Index] != nil {
			return nil, errors.New("duplicate temporary drop index")
		}
		drop := &DropItem{Index: encoded.Index}
		for _, itemData := range encoded.Items {
			item, err := restoreTempPackItem(itemData)
			if err != nil {
				return nil, err
			}
			if seen[item] {
				return nil, errors.New("duplicate temporary drop object")
			}
			seen[item] = true
			item.SetPackType(proto.ItemDetail_PackType_TempStorageArea)
			drop.Items = append(drop.Items, item)
		}
		pack.TempAreaPack[encoded.Index] = drop
		if encoded.Index > archive.NextIndex {
			archive.NextIndex = encoded.Index
		}
	}
	pack.index.Store(archive.NextIndex)
	return pack, nil
}

func newSceneTempPack() *SceneTempPack {
	return &SceneTempPack{
		TempPack:     make([]EBagItemTag, 0),
		TempAreaPack: make(map[uint32]*DropItem),
	}
}

// GenMonsterDead 生成怪物掉落物
// tag 怪物类型
// cold 是否必出
func (d *SceneTempPack) GenMonsterDead(s *Player, tag proto.EMonsterTag, cold bool) *DropItem {
	if tag == proto.EMonsterTag_EMonsterTag_Normal && !cold {
		return nil
	}
	index := d.index.Add(1)

	info := &DropItem{
		Index: index,
		Items: make([]EBagItemTag, 0),
	}

	//switch tag {
	//case proto.EMonsterTag_EMonsterTag_Normal: // 普通怪
	//// 数量1 品质低
	//case proto.EMonsterTag_EMonsterTag_Elite: // 精英怪
	//// 数量2 品质中低
	//case proto.EMonsterTag_EMonsterTag_Boss: // boss怪
	//// 数量3 品质中高
	//case proto.EMonsterTag_EMonsterTag_Special: // 特殊怪
	//}
	// TODO 直接纯随机
	alg.AddLists(&info.Items, RandItemDetail(s.GetSceneModel().GetWorldLevel(), proto.EBagItemTag_EBagItemTag_Material))
	alg.AddLists(&info.Items, RandItemDetail(s.GetSceneModel().GetWorldLevel(), proto.EBagItemTag_EBagItemTag_Weapon))
	alg.AddLists(&info.Items, RandItemDetail(s.GetSceneModel().GetWorldLevel(), proto.EBagItemTag_EBagItemTag_Armor))
	items := info.Items[:0]
	for _, item := range info.Items {
		if !nilBagItem(item) {
			items = append(items, item)
		}
	}
	info.Items = items
	if len(info.Items) == 0 {
		return nil
	}
	if d.TempAreaPack == nil {
		d.TempAreaPack = make(map[uint32]*DropItem)
	}

	d.TempAreaPack[index] = info
	return info
}

func NewSceneTempPack() *SceneTempPack {
	return newSceneTempPack()
}

func (d *SceneTempPack) NewDrop(items []EBagItemTag) *DropItem {
	if d == nil || len(items) == 0 {
		return nil
	}
	filtered := make([]EBagItemTag, 0, len(items))
	for _, item := range items {
		if !nilBagItem(item) {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	index := d.index.Add(1)
	info := &DropItem{
		Index: index,
		Items: filtered,
	}
	if d.TempAreaPack == nil {
		d.TempAreaPack = make(map[uint32]*DropItem)
	}
	d.TempAreaPack[index] = info
	return info
}

func (d *SceneTempPack) GetTempAreaPack(index uint32) *DropItem {
	if d == nil {
		return nil
	}
	return d.TempAreaPack[index]
}

func RandItemDetail(worldLevel uint32, tag proto.EBagItemTag) EBagItemTag {
	conf := gdconf.RandItemByNewBagItemTag(tag)
	if conf == nil {
		return nil
	}
	switch tag {
	case proto.EBagItemTag_EBagItemTag_Material: // 材料
		return &ItemBaseInfo{
			ItemId:   uint32(conf.ID),
			Num:      rand.Int64N(2*(int64(worldLevel)+1)) + 1,
			ItemType: proto.EBagItemTag(conf.NewBagItemTag),
			PackType: proto.ItemDetail_PackType_TempStorageArea,
		}
	case proto.EBagItemTag_EBagItemTag_Weapon: // 武器
		info := GenItemWeaponInfo(uint32(conf.ID), alg.MinSlice(10, (worldLevel/5)+1))
		if info == nil {
			return nil
		}
		info.PackType = proto.ItemDetail_PackType_TempStorageArea

		return info
	case proto.EBagItemTag_EBagItemTag_Armor: // 装备
	}

	return nil
}

func (d *SceneTempPack) Pickup(dropIndex uint32, pickIndex int32) []*proto.ItemDetail {
	if d == nil {
		return nil
	}
	temp := d.GetTempAreaPack(dropIndex)
	if temp == nil || len(temp.Items) == 0 {
		return nil
	}
	items := temp.Items
	switch {
	case pickIndex == -1:
		items = append([]EBagItemTag(nil), items...)
		temp.Items = nil
		delete(d.TempAreaPack, dropIndex)
	case pickIndex >= 1 && int(pickIndex) <= len(items):
		position := int(pickIndex) - 1 // Protocol pickup indices are one-based.
		selected := items[position]
		if nilBagItem(selected) {
			return nil
		}
		items = []EBagItemTag{selected}
		copy(temp.Items[position:], temp.Items[position+1:])
		temp.Items[len(temp.Items)-1] = nil
		temp.Items = temp.Items[:len(temp.Items)-1]
		if len(temp.Items) == 0 {
			delete(d.TempAreaPack, dropIndex)
		}
	default:
		return nil
	}
	list := make([]*proto.ItemDetail, 0, len(items))
	for _, item := range items {
		if nilBagItem(item) {
			continue
		}
		item.SetPackType(proto.ItemDetail_PackType_TempPack)
		alg.AddLists(&list, item.ItemDetail())
	}
	return list
}
