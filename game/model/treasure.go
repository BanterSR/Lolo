package model

import (
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
	"math/rand/v2"
	"sync/atomic"
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
	info := &proto.DropItem{
		Index: d.Index,
		Items: make([]*proto.ItemDetail, len(d.Items)),
	}
	for i, item := range d.Items {
		info.Items[i] = item.ItemDetail()
	}
	return info
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

	d.TempAreaPack[index] = info
	return info
}

func (d *SceneTempPack) GetTempAreaPack(index uint32) *DropItem {
	return d.TempAreaPack[index]
}

func RandItemDetail(worldLevel uint32, tag proto.EBagItemTag) EBagItemTag {
	conf := gdconf.RandItemByNewBagItemTag(tag)
	switch tag {
	case proto.EBagItemTag_EBagItemTag_Material: // 材料
		return &ItemBaseInfo{
			ItemId:   uint32(conf.ID),
			Num:      rand.Int64N(2*(int64(worldLevel)+1)) + 1,
			ItemType: proto.EBagItemTag(conf.NewBagItemTag),
			PackType: proto.PackType_PackType_TempStorageArea,
		}
	case proto.EBagItemTag_EBagItemTag_Weapon: // 武器
		weapConf := gdconf.GetWeaponAllInfo(uint32(conf.ID))
		if weapConf == nil {
			log.Game.Warnf("[WeaponItemID:%v]未知的武器掉落物", conf.ID)
			return nil
		}
		return &ItemWeaponInfo{
			ItemBaseInfo: &ItemBaseInfo{
				ItemType: proto.EBagItemTag_EBagItemTag_Weapon,
				PackType: proto.PackType_PackType_Inventory,
				ItemId:   uint32(weapConf.WeaponInfo.GetItemID()),
			},
			WeaponId:         weapConf.WeaponId,
			InstanceId:       0,
			WeaponSystemType: proto.EWeaponSystemType(weapConf.WeaponInfo.NewWeaponSystemType),
			Attack:           1, // 攻击力
			DamageBalance:    1, // 伤害平衡
			CriticalRatio:    1, // 临界比率
			RandomProperty:   make([]*RandomProperty, 0),
			WearerId:         0,
			WearerIndex:      0,
			Level:            rand.Uint32N((worldLevel+1)*20-alg.MaxSlice(worldLevel*20, 1)+1) + alg.MaxSlice(worldLevel*20, 1),
			StrengthLevel:    0, // 强度等级
			StrengthExp:      0, // 强度经验
			Star:             1, // 星
			Inscription1:     0, //
			Durability:       0, // 磨损度
			PropertyIndex:    1, //
			IsLock:           false,
		}
	case proto.EBagItemTag_EBagItemTag_Armor: // 装备
	}

	return nil
}

func (d *SceneTempPack) Pickup(dropIndex uint32, pickIndex int32) []*proto.ItemDetail {
	temp, ok := d.TempAreaPack[dropIndex]
	if !ok {
		return nil
	}
	list := make([]*proto.ItemDetail, 0, len(temp.Items))
	for _, i := range temp.Items {
		i.SetPackType(proto.PackType_PackType_TempPack)
		alg.AddLists(&list, i.ItemDetail())
	}
	return list
}
