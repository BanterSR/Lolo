package model

import "gucooing/lolo/protocol/proto"

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
	if sm.SceneMap[sceneId] == nil {
		sm.SceneMap[sceneId] = &SceneInfo{
			SceneId:     sceneId,
			Collections: make(map[uint32]*CollectionInfo),
		}
	}
	return sm.SceneMap[sceneId]
}

type SceneInfo struct {
	SceneId     uint32                     `json:"sceneId,omitempty"`
	Collections map[uint32]*CollectionInfo `json:"collections,omitempty"`
}

type CollectionInfo struct {
}
