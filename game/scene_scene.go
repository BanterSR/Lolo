package game

import (
	"errors"
	"sync"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

var (
	minChannelId = uint32(1)
	maxChannelId = uint32(99999)
)

type WordInfo struct {
	game           *Game
	allScene       map[uint32]*SceneInfo // 整个服务上的全部场景
	allScenePlayer sync.Map              // 整个服务上的全部场景玩家对象
}

type SceneInfo struct {
	game       *Game
	cfg        *gdconf.SceneInfo       // 场景配置
	SceneId    uint32                  // 场景id
	allChannel map[uint32]*ChannelInfo // 全部房间
}

// 场景中玩家对象

type ScenePlayer struct {
	*model.Player
	// 基础信息
	Pos         *proto.Vector3
	Rot         *proto.Vector3
	SceneId     uint32
	ChannelId   uint32
	channelInfo *ChannelInfo // 绑定的房间
	// 音乐
	MusicalItemId         uint32
	MusicalItemSource     proto.MusicalItemSource
	MusicalItemInstanceId int64
	PlayingMusicNote      *proto.PlayingMusicNote
}

func (g *Game) getWordInfo() *WordInfo {
	if g.wordInfo == nil {
		g.wordInfo = &WordInfo{
			game: g,
		}
	}
	return g.wordInfo
}

func (w *WordInfo) getAllSceneInfo() map[uint32]*SceneInfo {
	if w.allScene == nil {
		w.allScene = make(map[uint32]*SceneInfo)
	}
	return w.allScene
}

func (w *WordInfo) getSceneInfo(sceneId uint32) (*SceneInfo, error) {
	list := w.getAllSceneInfo()
	if info, ok := list[sceneId]; ok {
		return info, nil
	}
	cfg := gdconf.GetSceneInfo(sceneId)
	if cfg == nil {
		return nil, errors.New("ScenesConfigAsset.json配置文件中没有该场景")
	}
	info := &SceneInfo{
		game:       w.game,
		cfg:        cfg,
		SceneId:    sceneId,
		allChannel: make(map[uint32]*ChannelInfo),
	}
	list[sceneId] = info
	return info, nil
}

func (w *WordInfo) getScenePlayer(player *model.Player) *ScenePlayer {
	value, ok := w.allScenePlayer.Load(player.UserId)
	if !ok {
		// 场景中没有该玩家
		return nil
	}
	return value.(*ScenePlayer)
}

func (w *WordInfo) getScenePlayerByUserId(userId uint32) *ScenePlayer {
	value, ok := w.allScenePlayer.Load(userId)
	if !ok {
		// 场景中没有该玩家
		return nil
	}
	return value.(*ScenePlayer)
}

func (w *WordInfo) getChannel(sceneId, channelId uint32) (*ChannelInfo, error) {
	sceneInfo, err := w.getSceneInfo(sceneId)
	if err != nil {
		return nil, err
	}
	return sceneInfo.getSceneChannel(channelId)
}

// 获取目标场景下的全部房间
func (s *SceneInfo) getAllSceneChannel() map[uint32]*ChannelInfo {
	if s.allChannel == nil {
		s.allChannel = make(map[uint32]*ChannelInfo)
	}
	return s.allChannel
}

func (s *SceneInfo) getSceneChannel(channelId uint32) (*ChannelInfo, error) {
	list := s.getAllSceneChannel()
	if info, ok := list[channelId]; ok {
		return info, nil
	}
	// add SceneChannel
	if channelId >= minChannelId && channelId <= maxChannelId {
		info := s.newChannelInfo(channelId, model.ChannelTypePublic)
		list[channelId] = info
		return info, nil
	}
	if channelId >= model.PrivateChannelStart &&
		db.IsUserExists(channelId) {
		info := s.newChannelInfo(channelId, model.ChannelTypePrivate)
		list[channelId] = info
		return info, nil
	}
	return nil, errors.New("没有该房间")
}

// 添加场景玩家对象
func (w *WordInfo) addScenePlayer(player *model.Player) *ScenePlayer {
	value, ok := w.allScenePlayer.Load(player.UserId)
	if ok {
		return value.(*ScenePlayer)
	}
	defaultSceneId := gdconf.GetConstant().DefaultSceneId
	defaultChannelId := gdconf.GetConstant().DefaultChannelId

	sceneInfo, err := w.getSceneInfo(defaultSceneId)
	if err != nil {
		log.Game.Errorf("默认场景不存在！请检查默认场景配置是否正确err:%s", err.Error())
		return nil
	}
	_, err = sceneInfo.getSceneChannel(defaultChannelId)
	if err != nil {
		log.Game.Error(err.Error())
		return nil
	}
	pos, rot := sceneInfo.cfg.GetSceneInfoRandomBorn()

	info := &ScenePlayer{
		Player:    player,
		Pos:       gdconf.ConfigVector3ToProtoVector3(pos),
		Rot:       gdconf.ConfigVector4ToProtoVector3(rot),
		SceneId:   defaultSceneId,   // 默认场景
		ChannelId: defaultChannelId, // 默认房间
	}
	w.allScenePlayer.Store(player.UserId, info)
	return info
}

// 加入房间
func (w *WordInfo) joinSceneChannel(s *model.Player) {
	scenePlayer := w.getScenePlayer(s)
	if scenePlayer == nil {
		log.Game.Warnf("玩家:%v没有准备好加入房间", s.UserId)
		return
	}
	sceneInfo, err := w.getSceneInfo(scenePlayer.SceneId)
	if sceneInfo == nil {
		log.Game.Errorf("场景:%v不存在！err:%s", scenePlayer.SceneId, err.Error())
		return
	}
	channelInfo, err := sceneInfo.getSceneChannel(scenePlayer.ChannelId)
	if err != nil {
		log.Game.Errorf("场景:%v,房间:%v不存在！err:%s",
			scenePlayer.SceneId, scenePlayer.ChannelId, err.Error())
		return
	}
	scenePlayer.channelInfo = channelInfo
	channelInfo.addScenePlayerChan <- scenePlayer
}

func (w *WordInfo) killScenePlayer(player *model.Player) {
	value, ok := w.allScenePlayer.LoadAndDelete(player.UserId)
	if !ok {
		return
	}
	scenePlayer := value.(*ScenePlayer)
	if scenePlayer.channelInfo != nil {
		scenePlayer.channelInfo.delScenePlayerChan <- scenePlayer // 退出房间
	}
}
