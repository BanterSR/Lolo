package game

import (
	"math"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) ShopInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ShopInfoReq)
	rsp := &proto.ShopInfoRsp{
		Status: proto.StatusCode_StatusCode_OK,
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

const maxShopBuyTimes uint32 = 100

type shopPurchaseCost struct {
	itemID uint32
	amount int64
}

func shopPurchaseCosts(conf *excel.ShopPoolConfigureItem, buyTimes uint32) ([]shopPurchaseCost, bool) {
	if conf == nil || buyTimes == 0 || buyTimes > maxShopBuyTimes {
		return nil, false
	}
	costs := make([]shopPurchaseCost, 0, len(conf.ShopCurrencyItem))
	for _, item := range conf.ShopCurrencyItem {
		if item == nil || item.CurrencyID <= 0 || item.Price < 0 {
			return nil, false
		}
		price := int64(item.Price)
		if price != 0 && int64(buyTimes) > math.MaxInt64/price {
			return nil, false
		}
		amount := price * int64(buyTimes)
		if amount != 0 {
			costs = append(costs, shopPurchaseCost{itemID: uint32(item.CurrencyID), amount: amount})
		}
	}
	return costs, true
}

func (g *Game) ShopBuy(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ShopBuyReq)
	rsp := &proto.ShopBuyRsp{
		Status: proto.StatusCode_StatusCode_OK,
		ShopId: req.ShopId,
		Grids:  new(proto.ShopGrid),
		Items:  make([]*proto.ItemDetail, 0),
	}
	shopConf := gdconf.GetShopInfo(req.ShopId)
	gridConf := gdconf.GetGrid(req.ShopId, req.GridId)
	gridInfo := s.GetShopModel().GetShopInfo(req.ShopId).GetGridInfo(req.GridId)
	defer func() {
		if gridInfo != nil {
			rsp.Grids = gridInfo.ShopGrid()
		}
		g.send(s, msg.PacketId, rsp)
	}()
	conf := gdconf.GetPoolByGrid(req.ShopId, req.GridId, req.ItemId)
	costs, valid := shopPurchaseCosts(conf, req.BuyTimes)
	if shopConf == nil || gridConf == nil || gridInfo == nil || !valid {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if gridConf.BuyMaxCount > 0 && (gridInfo.BuyTimes > uint32(gridConf.BuyMaxCount) ||
		req.BuyTimes > uint32(gridConf.BuyMaxCount)-gridInfo.BuyTimes) {
		rsp.Status = proto.StatusCode_StatusCode_SHOP_GRID_BUY_NUM_LIMIT
		return
	}

	ctx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	defer func() {
		if ctx.Error != nil {
			ctx.Rollback()
		}
	}()
	for _, cost := range costs {
		ctx.DelBaseItem(cost.itemID, cost.amount)
	}
	if ctx.Error != nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}

	var characterInfo *model.CharacterInfo
	var bagItem *model.AddItemCtx
	switch proto.EShopType(shopConf.NewShopType) {
	case proto.EShopType_EShopType_BattlePass:
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	case proto.EShopType_EShopType_CharacterBP:
		characterConf := gdconf.GetCharacterInfoByShop(req.ShopId)
		if characterConf == nil {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			return
		}
		characterInfo = s.GetCharacterModel().GetCharacterInfo(uint32(characterConf.ID))
		if characterInfo == nil || characterInfo.IsUnlockPayment {
			rsp.Status = proto.StatusCode_StatusCode_CHARACTER_PLACED
			return
		}
	default:
		if conf.ItemID <= 0 || conf.ItemNum <= 0 || int64(conf.ItemNum) > math.MaxInt64/int64(req.BuyTimes) {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			return
		}
		bagItem = s.AddAllTypeItem(uint32(conf.ItemID), int64(conf.ItemNum)*int64(req.BuyTimes))
		if bagItem == nil || bagItem.EBagItemTag == nil {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			return
		}
		alg.AddList(&rsp.Items, bagItem.AddItemDetail())
	}

	ctx.Commit()
	gridInfo.BuyTimes += req.BuyTimes
	g.send(s, 0, ctx.PackNotice)
	if characterInfo != nil {
		characterInfo.IsUnlockPayment = true
		g.send(s, 0, &proto.CharacterBpBuyNotice{
			Status:      proto.StatusCode_StatusCode_OK,
			CharacterId: characterInfo.CharacterId,
		})
	}
}

func (g *Game) CreatePayOrder(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CreatePayOrderReq)
	rsp := &proto.CreatePayOrderRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	defer g.send(s, msg.PacketId, rsp)
	// 本地私服没有支付渠道，不能伪造已支付订单；明确返回未支付，
	// 避免客户端拿到一个永远无法回调成功的假订单号。
	if req.ShopId == 0 || req.GridId == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.Status = proto.StatusCode_StatusCode_Not_Payment
	rsp.ResultStr = "local server does not provide payment"
}
