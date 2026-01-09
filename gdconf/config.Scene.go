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
	Info          *config.SceneInfo
	TreasureInfos map[uint32]*config.CollectionTreasureInfo
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
			Info:          scene,
			TreasureInfos: make(map[uint32]*config.CollectionTreasureInfo),
		}
		info.SceneMap[scene.ID] = sceneInfo
		for _, v := range scene.GetCollectionTreasureInfos() {

		}
	}
}

func GetSceneInfo(sceneId uint32) *config.SceneInfo {
	info := cc.Config.SceneConfig.SceneMap[int32(sceneId)]
	if info == nil {
		return nil
	}
	return info.Info
}

func GetSceneInfoRandomBorn(info *config.SceneInfo) (*config.Vector3, *config.Vector4) {
	n := len(info.GetBorn())
	if n == 0 {
		return nil, nil
	}
	bornInfo := info.GetBorn()[rand.Intn(n)]
	return bornInfo.Position, bornInfo.Rotation
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
