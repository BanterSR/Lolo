package model

import (
	"time"

	"github.com/bytedance/sonic"

	"gucooing/lolo/db"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/pkg/ofnet"
)

type Player struct {
	Conn          ofnet.Conn      `json:"-"`
	Online        bool            `json:"-"`
	NetFreeze     bool            `json:"-"`
	Created       time.Time       `json:"-"`                       // 创建时间
	ActiveTime    time.Time       `json:"-"`                       // 上一次活跃时间
	LastSaveTime  time.Time       `json:"-"`                       // 上一次数据保存时间
	UserId        uint32          `json:"-"`                       // 玩家id
	NickName      string          `json:"-"`                       // 玩家昵称
	InstanceIndex uint32          `json:"instanceIndex,omitempty"` // 唯一索引生成
	Item          *ItemModel      `json:"item,omitempty"`          // 背包
	Character     *CharacterModel `json:"character,omitempty"`     // 角色
	Team          *TeamModel      `json:"team,omitempty"`          // 队伍
	Archive       *ArchiveModel   `json:"archive,omitempty"`       // 信息记录
	Chat          *ChatModel      `json:"chat,omitempty"`          // 聊天
	Gacha         *GachaModel     `json:"gacha,omitempty"`         // 卡池
	Garden        *GardenModel    `json:"garden,omitempty"`        // 花园
	Shop          *ShopModel      `json:"shop,omitempty"`          // 商店
	Scene         *SceneModel     `json:"scene,omitempty"`         // 场景信息
}

// 将玩家状态重置成在线
func (s *Player) Init(conn ofnet.Conn) {
	s.Conn = conn
	s.Online = true
	s.NetFreeze = false
}

func (s *Player) GetSeqId() uint32 {
	if s.Conn == nil {
		return 0
	}
	return s.Conn.GetSeqId()
}

func (s *Player) SetActiveTime() {
	s.ActiveTime = time.Now()
}

func (s *Player) SetLastSaveTime() {
	s.LastSaveTime = time.Now()
}

// 是否保存玩家数据

var (
	playerSaveIntervalTime = 5 * time.Minute
	playerCacheTime        = 5 * time.Minute
)

func (s *Player) IsSave() bool {
	if s.ActiveTime.Before(s.LastSaveTime) {
		return false
	}
	if s.LastSaveTime.Add(playerSaveIntervalTime).After(time.Now()) {
		return false
	}
	return true
}

func (s *Player) IsOffline() bool {
	if s.ActiveTime.Add(playerCacheTime).Before(time.Now()) && s.Online {
		return true
	}
	return false
}

func (s *Player) SavePlayer() error {
	s.SetLastSaveTime()
	var laseErr error
	if err := db.SaveOFGame(s.UserId, func(user *db.OFGame) bool {
		bin, err := sonic.Marshal(s)
		if err != nil {
			log.Game.Errorf("玩家:%v序列化失败err:%s",
				s.UserId, err.Error())
			return false
		}
		user.BinData = bin
		return true
	}); err != nil {
		laseErr = err
	}

	return laseErr
}
