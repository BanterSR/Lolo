package model

import (
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
	"time"
)

func CopyVector3(rot *proto.Vector3) *proto.Vector3 {
	return &proto.Vector3{
		X: rot.X,
		Y: rot.Y,
		Z: rot.Z,
	}
}

type SceneModel struct {
	SceneMap map[uint32]*SceneInfo `json:"sceneMap,omitempty"`
}

func (s *Player) GetSceneModel() *SceneModel {
	if s.Scene == nil {
		s.Scene = new(SceneModel)
	}
	return s.Scene
}

func (sm *SceneModel) GetSceneMap() map[uint32]*SceneInfo {
	if sm.SceneMap == nil {
		sm.SceneMap = make(map[uint32]*SceneInfo)
	}
	return sm.SceneMap
}

func (sm *SceneModel) GetSceneInfo(sceneId uint32) *SceneInfo {
	list := sm.GetSceneMap()
	info, ok := list[sceneId]
	if !ok {
		info = &SceneInfo{
			SceneId:     sceneId,
			Collections: make(map[proto.ECollectionType]*CollectionInfo),
		}
		list[sceneId] = info
	}
	return info
}

type SceneInfo struct {
	SceneId      uint32                                    `json:"sceneId,omitempty"`
	Collections  map[proto.ECollectionType]*CollectionInfo `json:"collections,omitempty"`  // 收集
	AreaDatas    map[uint32]*AreaData                      `json:"areaDatas,omitempty"`    // 锚点
	GatherLimits map[uint32]*GatherLimit                   `json:"gatherLimits,omitempty"` // 资源点
}

func (si *SceneInfo) GetCollections() map[proto.ECollectionType]*CollectionInfo {
	if si.Collections == nil {
		si.Collections = make(map[proto.ECollectionType]*CollectionInfo)
	}
	return si.Collections
}

func (si *SceneInfo) GetAreaDatas() map[uint32]*AreaData {
	if si.AreaDatas == nil {
		si.AreaDatas = make(map[uint32]*AreaData)
	}
	return si.AreaDatas
}

func (si *SceneInfo) GetGatherLimits() map[uint32]*GatherLimit {
	if si.GatherLimits == nil {
		si.GatherLimits = make(map[uint32]*GatherLimit)
	}
	return si.GatherLimits
}

type CollectionInfo struct {
	Type             uint32                             `json:"type,omitempty"`
	ItemMap          map[uint32]*PBCollectionRewardData `json:"itemMap,omitempty"`
	Level            uint32                             `json:"level,omitempty"`
	Exp              uint32                             `json:"exp,omitempty"`
	LastRefreshTime  time.Time                          `json:"lastRefreshTime,omitempty"`  // 上次刷新时间
	CollectedMoonIds []uint32                           `json:"collectedMoonIds,omitempty"` // 收集的月亮
}

func (si *SceneInfo) GetCollectionInfo(t proto.ECollectionType) *CollectionInfo {
	list := si.GetCollections()
	info, ok := list[t]
	if !ok {
		info = &CollectionInfo{
			Type:             uint32(t),
			ItemMap:          make(map[uint32]*PBCollectionRewardData),
			Level:            0,
			Exp:              0,
			LastRefreshTime:  time.Now(),
			CollectedMoonIds: make([]uint32, 0),
		}
		list[t] = info
	}
	if time.Now().Add(-4 * time.Minute).After(info.LastRefreshTime) {
		switch t {
		case proto.ECollectionType_ECollectionType_CollectMoonPiece:
			info.LastRefreshTime = time.Now()
			info.CollectedMoonIds = make([]uint32, 0)
		}
	}

	return info
}

func (c *CollectionInfo) CollectionData() *proto.CollectionData {
	info := &proto.CollectionData{
		Type:    c.Type,
		ItemMap: make(map[uint32]*proto.PBCollectionRewardData),
		Level:   c.Level,
		Exp:     c.Exp,
	}
	for k, v := range c.ItemMap {
		info.ItemMap[k] = v.PBCollectionRewardData()
	}

	return info
}

type PBCollectionRewardData struct {
	ItemId uint32             `json:"itemId,omitempty"`
	Status proto.RewardStatus `json:"status,omitempty"`
}

func (p *PBCollectionRewardData) PBCollectionRewardData() *proto.PBCollectionRewardData {
	return &proto.PBCollectionRewardData{
		ItemId: p.ItemId,
		Status: p.Status,
	}
}

type AreaData struct {
	AreaId    uint32          `json:"areaId,omitempty"`
	AreaState proto.AreaState `json:"areaState,omitempty"`
	Level     uint32          `json:"level,omitempty"`
}

func (a *AreaData) AreaData() *proto.AreaData {
	return &proto.AreaData{
		AreaId:    a.AreaId,
		AreaState: a.AreaState,
		Level:     a.Level,
		Items:     make([]*proto.BaseItem, 0),
	}
}

type GatherLimit struct {
	GatherType          uint32 `json:"gatherType,omitempty"`
	GatherNum           uint32 `json:"gatherNum,omitempty"`
	GatherLimitNum      uint32 `json:"gatherLimitNum,omitempty"`
	LuckyGatherLimitNum uint32 `json:"luckyGatherLimitNum,omitempty"`
}

func (g *GatherLimit) GatherLimit() *proto.GatherLimit {
	return &proto.GatherLimit{
		GatherType:          g.GatherType,
		GatherNum:           g.GatherNum,
		GatherLimitNum:      g.GatherLimitNum,
		LuckyGatherLimitNum: g.LuckyGatherLimitNum,
	}
}

func (si *SceneInfo) SceneGatherLimit() *proto.SceneGatherLimit {
	info := &proto.SceneGatherLimit{
		SceneId:      si.SceneId,
		GatherLimits: make([]*proto.GatherLimit, 0, len(si.GatherLimits)),
	}
	for _, v := range si.GetGatherLimits() {
		alg.AddList(&info.GatherLimits, v.GatherLimit())
	}

	return info
}

func (si *SceneInfo) GetGatherLimit(t uint32) *GatherLimit {
	list := si.GetGatherLimits()
	info, ok := list[t]
	if !ok {
		info = &GatherLimit{
			GatherType:          t,
			GatherNum:           0,
			GatherLimitNum:      0,
			LuckyGatherLimitNum: 0,
		}
		list[t] = info
	}
	return info
}
