package gdconf

import (
	"gucooing/lolo/protocol/config"
	"gucooing/lolo/protocol/proto"
)

type SceneConfig struct {
	all             *config.SceneConfig
	SceneMap        map[uint32]*SceneInfo        // 场景id
	DungeonSceneMap map[uint32]*DungeonSceneInfo // 副本id
}

type SceneInfo struct {
	Info             *config.SceneInfo
	TreasureInfos    map[uint32]*config.CollectionTreasureInfo
	GatherPointIndex map[uint32]*config.GatherPointInfo
}

// 副本场景信息
type DungeonSceneInfo struct {
	Info *config.SceneInfo
}

func (g *GameConfig) loadSceneConfig() {
	info := &SceneConfig{
		all:             new(config.SceneConfig),
		SceneMap:        make(map[uint32]*SceneInfo),
		DungeonSceneMap: make(map[uint32]*DungeonSceneInfo),
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
		info.SceneMap[uint32(scene.ID)] = sceneInfo
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
	// 副本信息
	for _, scene := range info.all.GetDungeons() {
		dungeonInfo := &DungeonSceneInfo{
			Info: scene,
		}
		info.DungeonSceneMap[uint32(scene.ID)] = dungeonInfo
	}
}

func GetSceneInfo(sceneId uint32) *SceneInfo {
	info := cc.Config.SceneConfig.SceneMap[sceneId]
	if info == nil {
		return nil
	}
	return info
}

func GetDungeonSceneInfo(dungeonId uint32) *DungeonSceneInfo {
	info := cc.Config.SceneConfig.DungeonSceneMap[dungeonId]
	if info == nil {
		return nil
	}
	return info
}

func GetSceneInfoRandomBorn(borns []*config.BornInfo) (*config.Vector3, *config.Vector4) {
	n := len(borns)
	if n == 0 {
		return nil, nil
	}
	// rn := rand.Intn(n)
	rn := 0
	bornInfo := borns[rn]
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
