package gdconf

import (
	"math/rand"

	"gucooing/lolo/protocol/config"
	"gucooing/lolo/protocol/proto"
)

type SceneConfig struct {
	all      *config.SceneConfig
	SceneMap map[int32]*SceneInfo
}

type SceneInfo struct {
	Info             *config.SceneInfo
	TreasureInfos    map[uint32]*config.CollectionTreasureInfo
	GatherPointIndex map[uint32]*config.GatherPointInfo
}

func (g *GameConfig) loadSceneConfig() {
	info := &SceneConfig{
		all:      new(config.SceneConfig),
		SceneMap: make(map[int32]*SceneInfo),
	}
	g.Config.SceneConfig = info
	name := "ScenesConfigAsset.json"
	ReadJson(g.configPath, name, &info.all)

	for _, scene := range info.all.GetScenes() {
		sceneInfo := &SceneInfo{
			Info:             scene,
			TreasureInfos:    make(map[uint32]*config.CollectionTreasureInfo),
			GatherPointIndex: make(map[uint32]*config.GatherPointInfo),
		}
		info.SceneMap[scene.ID] = sceneInfo
		// 宝箱信息
		//for _, v := range scene.GetCollectionTreasureInfos() {
		//
		//}
		// 资源聚集信息
		for _, v := range scene.GetGatherPointSetInfo() {
			for _, set := range v.GetGatherPointSets() {
				for _, point := range set.GetLifeGatherPointers() {
					index := uint32(point.Index)
					if _, ok := sceneInfo.GatherPointIndex[index]; ok {
						panic("序号重复")
					}
					sceneInfo.GatherPointIndex[index] = point
				}
			}
		}
	}
}

func GetSceneInfo(sceneId uint32) *SceneInfo {
	info := cc.Config.SceneConfig.SceneMap[int32(sceneId)]
	if info == nil {
		return nil
	}
	return info
}

func (s *SceneInfo) GetSceneInfoRandomBorn() (*config.Vector3, *config.Vector4) {
	n := len(s.Info.GetBorn())
	if n == 0 {
		return nil, nil
	}
	bornInfo := s.Info.GetBorn()[rand.Intn(n)]
	return bornInfo.Position, bornInfo.Rotation
}

func (s *SceneInfo) GatherPointInfo(index uint32) *config.GatherPointInfo {
	return s.GatherPointIndex[index]
}

func ConfigVector3ToProtoVector3(s *config.Vector3) *proto.Vector3 {
	return &proto.Vector3{
		X:             int32(s.GetX() * 100),
		Y:             int32(s.GetY() * 100),
		Z:             int32(s.GetZ() * 100),
		DecimalPlaces: 0,
	}
}

func ConfigVector4ToProtoVector3(s *config.Vector4) *proto.Vector3 {
	return &proto.Vector3{
		X:             int32(s.GetX() * 100),
		Y:             int32(s.GetY() * 100),
		Z:             int32(s.GetZ() * 100),
		DecimalPlaces: 0,
	}
}
