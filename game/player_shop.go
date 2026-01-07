package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) ShopInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ShopInfoReq)
	rsp := &proto.ShopInfoRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		ShopId: req.ShopId,
		Grids:  make([]*proto.ShopGrid, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	grids := gdconf.GetGridsByShopId(req.ShopId)
	for _, grid := range grids {
		pool := gdconf.GetPools(uint32(grid.ShopPoolID))
		for _, poolItem := range pool {
			// 验证时间和有效性
			info := s.GetShopModel().
				GetShopInfo(req.ShopId).
				GetGridInfo(uint32(grid.GridID)).
				ShopGrid()

			alg.AddList(&rsp.Grids, info)

			info.PoolIndex = uint32(poolItem.Index)
		}
	}
}

func (g *Game) ShopBuy(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ShopBuyReq)
	rsp := &proto.ShopBuyRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		ShopId: req.ShopId,
		Grids:  new(proto.ShopGrid),
		Items:  make([]*proto.ItemDetail, 0), // 结果
	}
	defer func() {
		rsp.Grids = s.GetShopModel().GetShopInfo(req.ShopId).GetGridInfo(req.GridId).ShopGrid()
		g.send(s, msg.PacketId, rsp)
	}()
	conf := gdconf.GetPoolByGrid(req.ShopId, req.GridId, req.ItemId)
	if conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	// 扣钱
	ctx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	defer func() {
		if ctx.Error != nil {
			ctx.Rollback()
		}
	}()
	for _, item := range conf.ShopCurrencyItem {
		ctx.DelBaseItem(uint32(item.CurrencyID), int64(item.Price*int32(req.BuyTimes)))
	}
	if ctx.Error != nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	ctx.Commit()
	g.send(s, 0, ctx.PackNotice)
	// 根据商店类型给予商品
	switch proto.EShopType(gdconf.GetShopInfo(req.ShopId).NewShopType) {
	case proto.EShopType_EShopType_BattlePass: // 通行证

	case proto.EShopType_EShopType_CharacterBp: // 角色商店
		// 解锁角色购买
		characterInfo := s.GetCharacterModel().GetCharacterInfo(uint32(gdconf.GetCharacterInfoByShop(req.ShopId).ID))
		if characterInfo == nil {
			rsp.Status = proto.StatusCode_StatusCode_CharacterPlaced
			log.Game.Warnf("获取角色商店失败,商店%v不存在", req.ShopId)
			return
		}
		characterInfo.IsUnlockPayment = true

		alg.AddList(&rsp.Items, characterInfo.ItemDetail())
	default:
		bagItem := s.AddAllTypeItem(uint32(conf.ItemID), int64(conf.ItemNum*int32(req.BuyTimes)))
		alg.AddList(&rsp.Items, bagItem.AddItemDetail())
	}
}

func (g *Game) CreatePayOrder(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.CreatePayOrderReq)
	rsp := &proto.CreatePayOrderRsp{
		Status:    proto.StatusCode_StatusCode_Ok,
		OrderId:   "", // 订单号
		ResultStr: "", // ？
	}
	defer g.send(s, msg.PacketId, rsp)
}
