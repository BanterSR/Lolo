package model

import (
	"gucooing/lolo/protocol/proto"
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
	SceneId     uint32                                    `json:"sceneId,omitempty"`
	Collections map[proto.ECollectionType]*CollectionInfo `json:"collections,omitempty"`
}

func (si *SceneInfo) GetCollections() map[proto.ECollectionType]*CollectionInfo {
	if si.Collections == nil {
		si.Collections = make(map[proto.ECollectionType]*CollectionInfo)
	}
	return si.Collections
}

func (si *SceneInfo) GetCollectionInfo(t proto.ECollectionType) *CollectionInfo {
	list := si.GetCollections()
	info, ok := list[t]
	if !ok {
		info = &CollectionInfo{
			Type:    uint32(t),
			ItemMap: make(map[uint32]*PBCollectionRewardData),
			Level:   0,
			Exp:     0,
		}
		list[t] = info
	}
	return info
}

type CollectionInfo struct {
	Type    uint32                             `json:"type,omitempty"`
	ItemMap map[uint32]*PBCollectionRewardData `json:"itemMap,omitempty"`
	Level   uint32                             `json:"level,omitempty"`
	Exp     uint32                             `json:"exp,omitempty"`
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
	ItemId uint32 `json:"itemId,omitempty"`
}

func (p *PBCollectionRewardData) PBCollectionRewardData() *proto.PBCollectionRewardData {
	return &proto.PBCollectionRewardData{
		ItemId: p.ItemId,
	}
}
