package model

import (
	"errors"
	"math/rand/v2"
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

const (
	characterSSR = iota + 1
	characterSR
	posterSSR
	posterSR
	fashion
	desire
	furniture
	weaponSSR
	weaponSR
	dyeStuff
)

var (
	DefaultGachaInfo = func(gachaId uint32) *GachaInfo {
		return &GachaInfo{
			GachaId: gachaId,
		}
	}
	ssrPoolID       = int32(400)   // ssr 角色
	srPoolID        = int32(300)   // sr 角色
	ssrPosterPoolID = int32(200)   // ssr 海报
	srPosterPoolID  = int32(100)   // sr 海报
	dyeStuffPoolID  = int32(40000) // 染色剂
	furniturePoolID = int32(40001) // 家具
	srWeaponPoolID  = int32(40002) // 武器紫
	ssrWeaponPoolID = int32(40003) // 武器金
	fashionPoolID   = int32(40004) // 时装
)

type GachaModel struct {
	GachaMap     map[uint32]*GachaInfo                 `json:"gachaMap,omitempty"`
	GachaTypeMap map[proto.EUIGachaType]*GachaTypeInfo `json:"gachaTypeMap,omitempty"`
}

type GachaInfo struct {
	GachaId    uint32 `json:"gachaId,omitempty"`    // 卡池id
	GachaTimes uint32 `json:"gachaTimes,omitempty"` // 已抽数
}

type GachaTypeInfo struct {
	GuaranteeSSR     int32 `json:"guaranteeSSR,omitempty"`     // 距离上次ssr角色保底已抽数量
	GuaranteeSR      int32 `json:"guaranteeSR,omitempty"`      // 距离上次sr角色保底已抽数量
	GuaranteeFashion int32 `json:"guaranteeFashion,omitempty"` // 距离上次服装保底已抽数量
	GuaranteeDesire  int32 `json:"guaranteeDesire,omitempty"`  // 距离上次愿望保底已抽数量
}

func DefaultGachaModel() *GachaModel {
	return &GachaModel{}
}

func (s *Player) GetGachaModel() *GachaModel {
	if s.Gacha == nil {
		s.Gacha = DefaultGachaModel()
	}
	return s.Gacha
}

func (g *GachaModel) GetGachaMap() map[uint32]*GachaInfo {
	if g.GachaMap == nil {
		g.GachaMap = make(map[uint32]*GachaInfo)
	}
	return g.GachaMap
}

func (g *GachaModel) GetGachaTypeMap() map[proto.EUIGachaType]*GachaTypeInfo {
	if g.GachaTypeMap == nil {
		g.GachaTypeMap = make(map[proto.EUIGachaType]*GachaTypeInfo)
	}
	return g.GachaTypeMap
}

func (g *GachaModel) GetGachaInfo(id uint32) *GachaInfo {
	list := g.GetGachaMap()
	info, ok := list[id]
	if !ok {
		info = DefaultGachaInfo(id)
		list[id] = info
	}
	return info
}

func (g *GachaModel) GetGachaTypeInfo(t proto.EUIGachaType) *GachaTypeInfo {
	list := g.GetGachaTypeMap()
	info, ok := list[t]
	if !ok {
		info = &GachaTypeInfo{
			GuaranteeSSR: 0,
			GuaranteeSR:  0,
		}
		list[t] = info
	}
	return info
}

func (g *GachaInfo) GachaInfo() *proto.GachaInfo {
	conf := gdconf.GetGachaData(g.GachaId)
	return &proto.GachaInfo{
		GachaId:        g.GachaId,
		GachaTimes:     g.GachaTimes,
		HasFullPick:    false,
		IsFree:         conf.Conf.IsFree,
		OptionalUpItem: 0,
		OptionalValue:  0,
		Guarantee:      0,
	}
}

type GachaCtx struct {
	player      *Player
	req         *proto.GachaReq
	conf        *gdconf.GachaData
	probability *gdconf.GachaProbability
	gachaInfo   *GachaInfo
	typeInfo    *GachaTypeInfo
	gachaNum    int32
	records     []*db.OFGachaRecord
	ItemDetails []*proto.ItemDetail
}

// 创建一个抽卡上下文
func (s *Player) NewGachaCtx(req *proto.GachaReq) (*GachaCtx, error) {
	ctx := &GachaCtx{
		player:    s,
		req:       req,
		conf:      gdconf.GetGachaData(req.GachaId),
		gachaInfo: s.GetGachaModel().GetGachaInfo(req.GachaId),
	}
	if ctx.conf == nil {
		return nil, errors.New("gacha conf is nil")
	}
	ctx.probability = gdconf.GetGachaProbability(ctx.conf.Conf.NewUIGachaType)
	if ctx.probability == nil {
		return nil, errors.New("gacha probability is nil")
	}
	ctx.typeInfo = s.GetGachaModel().GetGachaTypeInfo(proto.EUIGachaType(ctx.conf.Conf.NewUIGachaType))
	// 新建事务
	itemTx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, err
	}
	// 抵扣物品
	if req.IsSingle {
		ctx.gachaNum = ctx.conf.Conf.Method1Count
		itemTx.DelBaseItem(
			uint32(ctx.conf.Conf.ConsumeItem1ID),
			int64(ctx.conf.Conf.ConsumeItem1Num),
		)
	} else {
		ctx.gachaNum = ctx.conf.Conf.Method2Count
		itemTx.DelBaseItem(
			uint32(ctx.conf.Conf.ConsumeItem2ID),
			int64(ctx.conf.Conf.ConsumeItem2Num),
		)
	}
	if itemTx.Commit().Error != nil {
		return nil, itemTx.Error
	}
	ctx.records = make([]*db.OFGachaRecord, ctx.gachaNum)

	return ctx, nil
}

func (c *GachaCtx) Run() {
	for i := int32(1); i <= c.gachaNum; i++ {
		c.gachaInfo.GachaTimes++
		itemId, itemType := c.getPool()
		itemConf := gdconf.GetItemConfigure(uint32(itemId))
		if itemConf == nil {
			continue
		}
		c.records[i-1] = &db.OFGachaRecord{
			GachaId:   c.gachaInfo.GachaId,
			UserID:    c.player.UserId,
			ItemId:    uint32(itemId),
			GachaTime: time.Now().Unix(),
		}
		// 构造回复
		itemInfo := c.player.AddAllTypeItem(uint32(itemId), 1)
		if itemInfo == nil {
			continue
		}
		itemDetail := itemInfo.ItemDetail()

		itemDetail.ExtraQuality = 3

		switch itemType {
		case characterSSR:
			alg.AddList(&itemDetail.Extras,
				c.player.AddAllTypeItem(107, 1500).ItemDetail().MainItem)
		case characterSR:
			alg.AddList(&itemDetail.Extras,
				c.player.AddAllTypeItem(107, 500).ItemDetail().MainItem)
		case posterSSR:
			alg.AddList(&itemDetail.Extras,
				c.player.AddAllTypeItem(107, 500).ItemDetail().MainItem)
		case posterSR:
			alg.AddList(&itemDetail.Extras,
				c.player.AddAllTypeItem(107, 50).ItemDetail().MainItem)
		}
		alg.AddList(&c.ItemDetails, itemDetail)
	}
	// 写入db
	err := db.CreateGachaRecords(c.records)
	if err != nil {
		log.Game.Errorf("UserId:%v func db.CreateGachaRecords err:%v", c.player.UserId, err)
	}
}

func (c *GachaCtx) getPool() (int32, int) {
	switch proto.EUIGachaType(c.conf.Conf.NewUIGachaType) {
	case proto.EUIGachaType_EUIGachaType_New, // 新人
		proto.EUIGachaType_EUIGachaType_Default, // 普池
		proto.EUIGachaType_EUIGachaType_limit:   // 限时池
		c.typeInfo.GuaranteeSR++
		c.typeInfo.GuaranteeSSR++

		guaranteeSR := c.typeInfo.GuaranteeSR == c.probability.GuaranteeSR-1
		guaranteeSSR := c.typeInfo.GuaranteeSSR == c.probability.GuaranteeSSR
		if guaranteeSSR { // ssr
			return c.probabilitySSR(true), characterSSR
		}
		if guaranteeSR { // sr
			return c.probabilitySR(true), characterSR
		}
		randNum := rand.Int32N(10000) + 1
		if randNum -= c.probability.ProbabilitySSR; randNum <= 0 {
			return c.probabilitySSR(false), characterSSR
		}
		if randNum -= c.probability.ProbabilitySR; randNum <= 0 {
			return c.probabilitySR(false), characterSR
		}
		if randNum -= c.probability.ProbabilityPosterSSR; randNum <= 0 {
			return c.probabilityPosterSSR(), posterSSR
		}
		return c.probabilityPosterSR(), posterSR
	case proto.EUIGachaType_EUIGachaType_Fashion: // 服装池
		c.typeInfo.GuaranteeDesire++
		c.typeInfo.GuaranteeFashion++

		guaranteeFashion := c.typeInfo.GuaranteeFashion == c.probability.GuaranteeFashion-1
		if guaranteeFashion {
			return c.probabilityFashion(true), fashion
		}
		randNum := rand.Int32N(10000) + 1
		if randNum -= c.typeInfo.GuaranteeDesire * 50; randNum <= 0 {
			return c.probabilityDesire(), desire
		}
		if randNum -= c.probability.ProbabilityFashion; randNum <= 0 {
			return c.probabilityFashion(false), fashion
		}
		if randNum -= c.probability.ProbabilityFurniture; randNum <= 0 {
			return c.probabilityFurniture(), furniture
		}
		if randNum -= c.probability.ProbabilityWeaponSSR; randNum <= 0 {
			return c.probabilityWeaponSSR(), weaponSSR
		}
		if randNum -= c.probability.ProbabilityWeaponSR; randNum <= 0 {
			return c.probabilityWeaponSR(), weaponSR
		}
		return c.probabilityDyeStuff(), dyeStuff
	}
	log.Game.Warnf("未知的卡池类型:%s", proto.EUIGachaType(c.conf.Conf.NewUIGachaType).String())
	return 0, 0
}

// 概率落在ssr角色上
func (c *GachaCtx) probabilitySSR(up bool) int32 {
	if up {
		c.typeInfo.GuaranteeSSR = 0
	}

	if c.conf.BigPool != nil && (rand.Int32N(10000)+1 > 5000 || up) {
		return alg.RandUn(c.conf.BigPool.Items).GetItemID()
	} else {
		return alg.RandUn(c.conf.Pools[ssrPoolID].GetItems()).GetItemID()
	}
}

// 概率落在sr角色上
func (c *GachaCtx) probabilitySR(up bool) int32 {
	if up {
		c.typeInfo.GuaranteeSR = 0
	}
	return alg.RandUn(c.conf.Pools[srPoolID].GetItems()).GetItemID()
}

// 概率落在ssr海报上
func (c *GachaCtx) probabilityPosterSSR() int32 {
	return alg.RandUn(c.conf.Pools[ssrPosterPoolID].GetItems()).GetItemID()
}

// 概率落在sr海报上
func (c *GachaCtx) probabilityPosterSR() int32 {
	return alg.RandUn(c.conf.Pools[srPosterPoolID].GetItems()).GetItemID()
}

// 概率落在服装上
func (c *GachaCtx) probabilityFashion(up bool) int32 {
	if up {
		c.typeInfo.GuaranteeFashion = 0
	}
	return alg.RandUn(c.conf.Pools[fashionPoolID].GetItems()).GetItemID()
}

// 概率落在家具上
func (c *GachaCtx) probabilityFurniture() int32 {
	return alg.RandUn(c.conf.Pools[furniturePoolID].GetItems()).GetItemID()
}

// 概率落在ssr武器上
func (c *GachaCtx) probabilityWeaponSSR() int32 {
	return alg.RandUn(c.conf.Pools[ssrWeaponPoolID].GetItems()).GetItemID()
}

// 概率落在sr武器上
func (c *GachaCtx) probabilityWeaponSR() int32 {
	return alg.RandUn(c.conf.Pools[srWeaponPoolID].GetItems()).GetItemID()
}

// 概率落在染色剂上
func (c *GachaCtx) probabilityDyeStuff() int32 {
	return alg.RandUn(c.conf.Pools[dyeStuffPoolID].GetItems()).GetItemID()
}

// 概率落在愿望上
func (c *GachaCtx) probabilityDesire() int32 {
	c.typeInfo.GuaranteeDesire = 0
	return 0
}
