package game

import (
	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GachaList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.GachaListRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Gachas: make([]*proto.GachaInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	list := s.GetGachaModel().GetGachaMap()
	for _, v := range gdconf.GetOpenGachas() {
		info, ok := list[uint32(v.Conf.ID)]
		if !ok {
			alg.AddList(&rsp.Gachas,
				model.DefaultGachaInfo(uint32(v.Conf.ID)).GachaInfo())
		} else {
			alg.AddList(&rsp.Gachas, info.GachaInfo())
		}
	}
}

func (g *Game) GachaRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GachaRecordReq)
	rsp := &proto.GachaRecordRsp{
		Status:    proto.StatusCode_StatusCode_Ok,
		GachaId:   req.GachaId,
		Page:      req.Page, // 当前页
		TotalPage: 0,        // 总页
		Records:   nil,
	}
	defer g.send(s, msg.PacketId, rsp)
	list, totalPage, err := db.GetGachaRecords(s.UserId, req.GachaId, req.Page)
	if err != nil {
		log.Game.Errorf("UserId:%v func db.GetGachaRecords err:%v", s.UserId, err)
		return
	}
	rsp.TotalPage = totalPage
	rsp.Records = make([]*proto.PlayerGachaRecord, len(list))
	for index, info := range list {
		rsp.Records[index] = info.PlayerGachaRecord()
	}
}

func (g *Game) Gacha(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GachaReq)
	rsp := &proto.GachaRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Items:  nil,
		Info:   nil,
	}
	defer func() {
		if gdconf.GetGachaData(req.GachaId) != nil {
			rsp.Info = s.GetGachaModel().GetGachaInfo(req.GachaId).GachaInfo()
		}
		g.send(s, msg.PacketId, rsp)
	}()
	if gdconf.GetGachaData(req.GachaId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_GachaTypeNotExist
		return
	}
	ctx, err := s.NewGachaCtx(req)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_GachaErr
		log.Game.Errorf("NewGachaCtx err:%v", err)
		return
	}
	ctx.Run()
	rsp.Items = ctx.ItemDetails
	if len(rsp.Items) > 0 {
		if err := RecordAchieveCondition(s, proto.EAchieveCondition_EAchieveCondition_Gacha, uint64(len(rsp.Items))); err != nil {
			log.Game.Warnf("UserId:%v 抽卡成就进度记录失败:%v", s.UserId, err)
		}
	}
	g.send(s, 0, ctx.ItemTx.PackNotice)
}

func gachaOptionalItemValid(gachaID, itemID uint32) bool {
	data := gdconf.GetGachaData(gachaID)
	if data == nil || data.Conf == nil || data.Conf.OptionalPoolID == 0 {
		return false
	}
	pool := gdconf.GetGachaRewardPool(data.Conf.OptionalPoolID)
	if pool == nil {
		return false
	}
	for _, item := range pool.Items {
		if item != nil && uint32(item.ItemID) == itemID && gdconf.GetItemConfigure(itemID) != nil {
			return true
		}
	}
	return false
}

func (g *Game) OptionalUpPoolItem(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OptionalUpPoolItemReq)
	rsp := &proto.OptionalUpPoolItemRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	if req.ItemId != 0 && !gachaOptionalItemValid(req.GachaId, req.ItemId) {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	info := s.GetGachaModel().GetGachaInfo(req.GachaId)
	info.OptionalItemId = req.ItemId
	rsp.Info = info.GachaInfo()
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_SysErr
	}
}

func (g *Game) ChooseGacha(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChooseGachaReq)
	rsp := &proto.ChooseGachaRsp{
		Status:  proto.StatusCode_StatusCode_Ok,
		GachaId: req.GachaId,
	}
	defer g.send(s, msg.PacketId, rsp)
	selected := gdconf.GetGachaData(req.ChooseGachaPool)
	if selected == nil || selected.Conf == nil || req.ChooseGachaPool == req.GachaId {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	info := s.GetGachaModel().GetGachaInfo(req.GachaId)
	info.ChoosePoolId = req.ChooseGachaPool
	rsp.ChooseGachaPool = info.ChoosePoolId
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_SysErr
	}
}

func (g *Game) GachaFullPick(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GachaFullPickReq)
	rsp := &proto.GachaFullPickRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	data := gdconf.GetGachaData(req.GachaId)
	info := s.GetGachaModel().GetGachaInfo(req.GachaId)
	if data == nil || data.Conf == nil || data.Conf.OptionalNum <= 0 ||
		info.GachaTimes < uint32(data.Conf.OptionalNum) || info.FullPickRewarded {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if !gachaOptionalItemValid(req.GachaId, req.ItemId) {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	item := s.AddAllTypeItem(req.ItemId, 1)
	if item == nil || item.EBagItemTag == nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	detail := item.AddItemDetail()
	if detail == nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	info.FullPickRewarded = true
	rsp.Item = []*proto.ItemDetail{detail}
	g.send(s, 0, &proto.PackNotice{Status: proto.StatusCode_StatusCode_Ok, Items: rsp.Item})
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_SysErr
	}
}
