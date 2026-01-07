package game

import (
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
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
	// 给商品
	bagItem := s.AddAllTypeItem(uint32(conf.ItemID), int64(conf.ItemNum*int32(req.BuyTimes)))
	alg.AddList(&rsp.Items, bagItem.AddItemDetail())
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
