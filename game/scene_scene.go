package game

import (
	"errors"
	"gucooing/lolo/pkg/alg"
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
	Team        *SceneTeamInfo // 队伍场景信息
	SceneId     uint32
	ChannelId   uint32
	channelInfo *ChannelInfo // 绑定的房间
	NetFreeze   bool         // 冻结收发包
	// 音乐
	MusicalItemId         uint32
	MusicalItemSource     proto.MusicalItemSource
	MusicalItemInstanceId int64
	PlayingMusicNote      *proto.PlayingMusicNote
	// 其他
	Dungeon *PlayerDungeon // 副本
}

// 队伍场景信息
type SceneTeamInfo struct {
	*model.Player // 玩家主体
	Pos           *proto.Vector3
	Rot           *proto.Vector3
	Team          *model.TeamInfo
}

func (t *SceneTeamInfo) GetPbSceneTeam() (info *proto.SceneTeam) {
	info = &proto.SceneTeam{
		Char1: t.GetPbSceneCharacter(t.Team.Char1),
		Char2: t.GetPbSceneCharacter(t.Team.Char2),
		Char3: t.GetPbSceneCharacter(t.Team.Char3),
	}
	return
}

func (t *SceneTeamInfo) GetPbSceneCharacter(characterId uint32) (info *proto.SceneCharacter) {
	characterInfo := t.GetCharacterModel().GetCharacterInfo(characterId)
	if characterInfo == nil {
		log.Game.Debugf("玩家:%v队伍角色:%v不存在", t.UserId, characterId)
		return nil
	}
	info = &proto.SceneCharacter{
		Pos:                 t.Pos,
		Rot:                 t.Rot,
		CharId:              characterInfo.CharacterId,
		CharLv:              0, // ok
		CharBreakLv:         0, // ok
		CharStar:            0, // ok
		CharacterAppearance: characterInfo.GetPbCharacterAppearance(),
		OutfitPreset:        t.GetPbSceneCharacterOutfitPreset(characterInfo),
		WeaponId:            0, // ok
		WeaponStar:          0, // ok
		Armors:              make([]*proto.BaseArmor, 0),
		Posters:             make([]*proto.BasePoster, 0),
		GatherWeapon:        0, // ok
		IsDead:              false,
		InscriptionId:       0,
		InscriptionLv:       0,
		MpGameWeapon:        0,
	}
	// 基础
	t.UpdateCharacterBasic(info)
	// 装备
	t.UpdateCharacterEquip(info)
	// 装备
	{
		equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
		if equipmentPreset == nil {
			log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
				t.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
		} else {
			// 盔甲
			for _, armor := range equipmentPreset.Armors {
				item := t.GetItemModel().GetItemArmorInfo(armor.InstanceId)
				alg.AddList(&info.Armors, item.BaseArmor())
			}
			// 海报
			for _, poster := range equipmentPreset.Posters {
				item := t.GetItemModel().GetItemPosterInfo(poster.InstanceId)
				alg.AddList(&info.Posters, item.BasePoster())
			}
		}

	}

	return
}

func (t *SceneTeamInfo) GetPbSceneCharacterOutfitPreset(characterInfo *model.CharacterInfo) *proto.SceneCharacterOutfitPreset {
	outfit := characterInfo.GetOutfitPreset(characterInfo.InUseOutfitPresetIndex)
	if outfit == nil {
		log.Game.Warnf("玩家:%v角色:%v外貌序号:%v缺少",
			t.UserId, characterInfo.CharacterId, characterInfo.InUseOutfitPresetIndex)
		return nil
	}
	getODS := func(id, index uint32) *proto.OutfitDyeScheme {
		fashionInfo := t.GetItemModel().GetItemFashionInfo(id)
		if fashionInfo == nil ||
			fashionInfo.GetDyeScheme(index) == nil {
			return &proto.OutfitDyeScheme{
				SchemeIndex: 0,
				Colors:      make([]*proto.PosColor, 0),
				IsUnLock:    false,
			}
		}
		return fashionInfo.GetDyeScheme(index).OutfitDyeScheme()
	}
	info := &proto.SceneCharacterOutfitPreset{
		Hat:                    outfit.Hat,
		HatDyeScheme:           getODS(outfit.Hat, outfit.HatDyeSchemeIndex),
		Hair:                   outfit.Hair,
		HairDyeScheme:          getODS(outfit.Hair, outfit.HairDyeSchemeIndex),
		Clothes:                outfit.Clothes,
		ClothDyeScheme:         getODS(outfit.Clothes, outfit.ClothesDyeSchemeIndex),
		Ornament:               outfit.Ornament,
		OrnDyeScheme:           getODS(outfit.Ornament, outfit.OrnamentDyeSchemeIndex),
		HideInfo:               outfit.OutfitHideInfo.OutfitHideInfo(),
		PendTop:                outfit.PendTop,
		PendTopDyeScheme:       getODS(outfit.PendTop, outfit.PendTopDyeSchemeIndex),
		PendChest:              outfit.PendChest,
		PendChestDyeScheme:     getODS(outfit.PendChest, outfit.PendChestDyeSchemeIndex),
		PendPelvis:             outfit.PendPelvis,
		PendPelvisDyeScheme:    getODS(outfit.PendPelvis, outfit.PendPelvisDyeSchemeIndex),
		PendUpFace:             outfit.PendUpFace,
		PendUpFaceDyeScheme:    getODS(outfit.PendUpFace, outfit.PendUpFaceDyeSchemeIndex),
		PendDownFace:           outfit.PendDownFace,
		PendDownFaceDyeScheme:  getODS(outfit.PendDownFace, outfit.PendDownFaceDyeSchemeIndex),
		PendLeftHand:           outfit.PendLeftHand,
		PendLeftHandDyeScheme:  getODS(outfit.PendLeftHand, outfit.PendLeftHandDyeSchemeIndex),
		PendRightHand:          outfit.PendRightHand,
		PendRightHandDyeScheme: getODS(outfit.PendRightHand, outfit.PendRightHandDyeSchemeIndex),
		PendLeftFoot:           outfit.PendLeftFoot,
		PendLeftFootDyeScheme:  getODS(outfit.PendLeftFoot, outfit.PendLeftFootDyeSchemeIndex),
		PendRightFoot:          outfit.PendRightFoot,
		PendRightFootDyeScheme: getODS(outfit.PendRightFoot, outfit.PendRightFootDyeSchemeIndex),
	}

	return info
}

func (t *SceneTeamInfo) UpdateCharacterBasic(info *proto.SceneCharacter) {
	characterInfo := t.GetCharacterModel().GetCharacterInfo(info.GetCharId())
	if characterInfo == nil || info == nil {
		return
	}
	info.CharLv = characterInfo.Level
	info.CharBreakLv = characterInfo.BreakLevel
	info.CharStar = characterInfo.Star
	info.CharLv = characterInfo.Level
}

func (t *SceneTeamInfo) UpdateCharacterEquip(info *proto.SceneCharacter) {
	characterInfo := t.GetCharacterModel().GetCharacterInfo(info.GetCharId())
	if characterInfo == nil || info == nil {
		return
	}
	info.GatherWeapon = characterInfo.GatherWeapon
	equipmentPreset := characterInfo.GetEquipmentPreset(characterInfo.InUseEquipmentPresetIndex)
	if equipmentPreset == nil {
		log.Game.Warnf("玩家:%v角色:%v装备序号:%v缺少",
			t.UserId, characterInfo.CharacterId, characterInfo.InUseEquipmentPresetIndex)
	} else {
		// 武器
		weaponInfo := t.GetItemModel().GetItemWeaponInfo(equipmentPreset.WeaponInstanceId)
		if weaponInfo == nil {
			log.Game.Warnf("玩家:%v角色:%v装备-武器:%v缺少",
				t.UserId, characterInfo.CharacterId, equipmentPreset.WeaponInstanceId)
		} else {
			info.WeaponStar = weaponInfo.Star
			info.WeaponId = weaponInfo.WeaponId
		}
	}
}

// 副本信息
type PlayerDungeon struct {
	DungeonId      uint32
	InDungeon      bool
	StartTimeMs    int64
	ConsumeTime    uint64
	Data           *proto.DungeonData
	TaskDataLst    []*proto.Achieve
	TreasureBoxes  map[uint32]*proto.TreasureBoxData
	PickedBoxes    map[uint32]bool
	FinalBoxOpened bool
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
		sp := value.(*ScenePlayer)
		sp.NetFreeze = false
		return sp
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
	pos, rot := gdconf.GetSceneInfoRandomBorn(sceneInfo.cfg.Info.GetBorn())

	info := &ScenePlayer{
		Player: player,
		Team: &SceneTeamInfo{
			Player: player,
			Pos:    gdconf.ConfigVector3ToProtoVector3(pos),
			Rot:    gdconf.ConfigVector4ToProtoVector3(rot),
			Team:   player.GetTeamModel().GetTeamInfo(),
		},
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
	if scenePlayer.channelInfo == nil {
		channelInfo, err := sceneInfo.getSceneChannel(scenePlayer.ChannelId)
		if err != nil {
			log.Game.Errorf("场景:%v,房间:%v不存在！err:%s",
				scenePlayer.SceneId, scenePlayer.ChannelId, err.Error())
			return
		}
		scenePlayer.channelInfo = channelInfo
	}
	scenePlayer.channelInfo.addScenePlayerChan <- scenePlayer
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
