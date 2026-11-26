package game

import (
	"math/rand/v2"
	"sync"
	"time"

	"gucooing/lolo/config"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

var lifeConfigCache struct {
	sync.Once
	cfg *model.LifeConfig
	err error
}

func getLifeConfig() (*model.LifeConfig, error) {
	lifeConfigCache.Do(func() {
		lifeConfigCache.cfg, lifeConfigCache.err = model.LoadLifeConfig(config.GetResources().GetResourcePath())
	})
	return lifeConfigCache.cfg, lifeConfigCache.err
}

func lifeRequestToken(s *model.Player, msg *alg.GameMsg, operation string, id uint32) string {
	if msg == nil || msg.PacketHead == nil {
		return ""
	}
	return s.GetLifeModel().RequestToken(s.LoginUUID, msg.SeqId, msg.MsgId, id, operation)
}

func saveLifePlayer(s *model.Player) error {
	if err := s.SavePlayerNow(); err != nil {
		log.Game.Errorf("UserId:%v 生活技能落库失败:%s", s.UserId, err.Error())
		return err
	}
	return nil
}

func prepareLifeRead(s *model.Player, lifeType proto.LifeType, now int64) (*model.LifeModel, *model.LifeConfig, proto.StatusCode, bool) {
	if !model.ValidLifeType(lifeType) {
		return nil, nil, proto.StatusCode_StatusCode_BadReq, false
	}
	cfg, err := getLifeConfig()
	if err != nil {
		log.Game.Errorf("UserId:%v 生活技能配置加载失败:%s", s.UserId, err.Error())
		return nil, nil, proto.StatusCode_StatusCode_CfgNotFound, false
	}
	life := s.GetLifeModel()
	changed, err := life.RefreshSkillsAndSave(lifeType, cfg, now, func() error { return saveLifePlayer(s) })
	if err != nil {
		return life, cfg, proto.StatusCode_StatusCode_SysErr, false
	}
	return life, cfg, proto.StatusCode_StatusCode_Ok, changed
}

func (g *Game) GetLifeInfo(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetLifeInfoReq)
	rsp := &proto.GetLifeInfoRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	life, cfg, status, changed := prepareLifeRead(s, req.LifeType, time.Now().Unix())
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	rsp.LifeBaseInfo = life.BaseInfo(req.LifeType)
	rsp.LifeSkill = &proto.LifeSkill{LifeType: req.LifeType, LifeSkillInfo: life.SkillInfo(req.LifeType, cfg)}
	rsp.LifeAchieve = life.AchieveInfo(req.LifeType, cfg)
	if changed {
		g.send(s, 0, &proto.LifeSkillLevelUpNotice{
			Status: proto.StatusCode_StatusCode_Ok,
			LifeSkills: []*proto.LifeSkill{{
				LifeType: req.LifeType, LifeSkillInfo: rsp.LifeSkill.LifeSkillInfo,
			}},
		})
	}
}

func (g *Game) GetLifeSkill(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetLifeSkillReq)
	rsp := &proto.GetLifeSkillRsp{Status: proto.StatusCode_StatusCode_Ok, LifeType: req.LifeType}
	defer g.send(s, msg.PacketId, rsp)
	life, cfg, status, _ := prepareLifeRead(s, req.LifeType, time.Now().Unix())
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	rsp.SkillInfo = life.SkillInfo(req.LifeType, cfg)
}

func (g *Game) GetLifeAchieve(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetLifeAchieveReq)
	rsp := &proto.GetLifeAchieveRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	life, cfg, status, _ := prepareLifeRead(s, req.LifeType, time.Now().Unix())
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	rsp.LifeAchieve = life.AchieveInfo(req.LifeType, cfg)
}

func (g *Game) produceLifeRecipe(s *model.Player, msg *alg.GameMsg, lifeType proto.LifeType, recipeID uint32, hasCampFire bool) model.LifeReceipt {
	cfg, err := getLifeConfig()
	if err != nil {
		return model.LifeReceipt{Status: proto.StatusCode_StatusCode_CfgNotFound, LifeType: lifeType}
	}
	receipt := s.GetLifeModel().ProduceRecipe(s.GetItemModel(), cfg, lifeType, recipeID, model.LifeProductionOptions{
		Token: lifeRequestToken(s, msg, "produce", recipeID), Now: time.Now().Unix(), HasCampFire: hasCampFire,
		Persist: func() error { return saveLifePlayer(s) },
	})
	if receipt.Status == proto.StatusCode_StatusCode_Ok && receipt.Success {
		g.sendLifeProductionNotices(s, receipt)
	}
	return receipt
}

func (g *Game) CookingFood(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.CookingFoodReq)
	receipt := g.produceLifeRecipe(s, msg, proto.LifeType_LifeType_Cooking, req.FoodId, req.HasCampFire)
	rsp := &proto.CookingFoodRsp{Status: receipt.Status}
	if receipt.Status == proto.StatusCode_StatusCode_Ok {
		rsp.LifeCommonResult = lifeCommonResult(receipt)
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) MakeProp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.MakePropReq)
	receipt := g.produceLifeRecipe(s, msg, proto.LifeType_LifeType_Making, req.MakeId, false)
	rsp := &proto.MakePropRsp{Status: receipt.Status}
	if receipt.Status == proto.StatusCode_StatusCode_Ok {
		rsp.LifeCommonResult = lifeCommonResult(receipt)
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) Handicraft(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.HandicraftReq)
	receipt := g.produceLifeRecipe(s, msg, proto.LifeType_LifeType_Handicraft, req.HandicraftId, false)
	rsp := &proto.HandicraftRsp{Status: receipt.Status}
	if receipt.Status == proto.StatusCode_StatusCode_Ok {
		rsp.LifeCommonResult = lifeCommonResult(receipt)
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) Fishing(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.FishingReq)
	rsp := &proto.FishingRsp{Status: proto.StatusCode_StatusCode_Ok, IsCancel: req.IsCancel}
	defer g.send(s, msg.PacketId, rsp)
	if req.IsCancel {
		return
	}
	cfg, err := getLifeConfig()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	receipt := s.GetLifeModel().Fish(s.GetItemModel(), cfg, model.FishingOptions{
		Token: lifeRequestToken(s, msg, "fish", req.FishingRodId), Now: time.Now().Unix(),
		PondID: req.FishingPondId, BoxID: req.FishingBoxId, RodInstanceID: req.FishingRodId,
		Persist: func() error { return saveLifePlayer(s) },
	})
	if receipt.Status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = receipt.Status
		return
	}
	if receipt.Success {
		g.sendLifeProductionNotices(s, receipt)
		g.send(s, 0, &proto.FishingResultNotice{Status: proto.StatusCode_StatusCode_Ok, Items: lifeItemDetails(receipt.ResultItems)})
	}
	rsp.FishingEventResult = &proto.FishingEventResult{
		CostEndurance: receipt.CostEndurance,
		CostTime:      receipt.CostTime,
		FishInfos:     lifeFishData(receipt.Fish),
	}
}

func (g *Game) LifeLeveUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.LifeLeveUpReq)
	cfg, err := getLifeConfig()
	rsp := &proto.LifeLeveUpRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	receipt := s.GetLifeModel().LevelUp(s.GetItemModel(), cfg, req.LifeType, lifeRequestToken(s, msg, "level-up", uint32(req.LifeType)),
		func() error { return saveLifePlayer(s) })
	if receipt.Status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = receipt.Status
		return
	}
	rsp.LifeBaseInfo = &proto.LifeBaseInfo{LifeType: req.LifeType, Level: receipt.Level, ProficiencyValue: receipt.Proficiency}
	g.sendLifePack(s, receipt.Items)
}

func (g *Game) LifeSkillLevelUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.LifeSkillLevelUpReq)
	cfg, err := getLifeConfig()
	rsp := &proto.LifeSkillLevelUpRsp{Status: proto.StatusCode_StatusCode_Ok, IsForceStudy: req.IsForceStudy}
	defer g.send(s, msg.PacketId, rsp)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	receipt := s.GetLifeModel().StudySkill(s.GetItemModel(), cfg, req.LifeType, req.SkillId, req.IsForceStudy,
		lifeRequestToken(s, msg, "skill", req.SkillId), time.Now().Unix(), func() error { return saveLifePlayer(s) })
	if receipt.Status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = receipt.Status
		return
	}
	skills := lifeSkillProto(receipt.Skills)
	rsp.LifeSkills = &proto.LifeSkill{LifeType: req.LifeType, LifeSkillInfo: skills}
	rsp.Items = lifeItemDetails(receipt.Items)
	g.sendLifePack(s, receipt.Items)
	g.send(s, 0, &proto.LifeSkillLevelUpNotice{
		Status: proto.StatusCode_StatusCode_Ok, LifeSkills: []*proto.LifeSkill{rsp.LifeSkills}, Items: rsp.Items,
	})
}

func (g *Game) GetLifeAchievementReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetLifeAchievementRewardReq)
	cfg, err := getLifeConfig()
	rsp := &proto.GetLifeAchievementRewardRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	receipt := s.GetLifeModel().ClaimAchievement(s.GetItemModel(), cfg, req.LifeType, req.AchieveId,
		lifeRequestToken(s, msg, "achievement", req.AchieveId), rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(s.UserId))),
		func() error { return saveLifePlayer(s) })
	if receipt.Status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = receipt.Status
		return
	}
	rsp.Items = lifeItemDetails(receipt.ResultItems)
	rsp.RewardedAchieveIds = receipt.RewardedAchieveIDs
	g.sendLifePack(s, receipt.Items)
	g.send(s, 0, &proto.LifeAchieveNotice{
		Status:       proto.StatusCode_StatusCode_Ok,
		LifeAchieves: []*proto.LifeAchieve{s.GetLifeModel().AchieveInfo(req.LifeType, cfg)},
	})
}

func lifeCommonResult(receipt model.LifeReceipt) *proto.LifeCommonResult {
	returns := make([]*proto.BaseItem, 0, len(receipt.SkillReturnItems))
	for _, item := range receipt.SkillReturnItems {
		returns = append(returns, &proto.BaseItem{ItemId: item.ItemID, Num: item.Count})
	}
	return &proto.LifeCommonResult{
		ResultItems:      lifeItemDetails(receipt.ResultItems),
		IsSuccess:        receipt.Success,
		LifeType:         receipt.LifeType,
		SkillReturnItems: returns,
		IsLuckyReturn:    receipt.IsLuckyReturn,
	}
}

func lifeItemDetails(items []model.LifeItem) []*proto.ItemDetail {
	result := make([]*proto.ItemDetail, 0, len(items))
	for _, item := range items {
		if detail := item.ItemDetail(); detail != nil {
			result = append(result, detail)
		}
	}
	return result
}

func lifeFishData(fish []model.LifeFishResult) []*proto.FishData {
	result := make([]*proto.FishData, 0, len(fish))
	for _, item := range fish {
		result = append(result, &proto.FishData{ID: item.ID, Length: item.Length})
	}
	return result
}

func lifeSkillProto(skills []model.LifeSkillSnapshot) []*proto.SingleLifeSkillInfo {
	result := make([]*proto.SingleLifeSkillInfo, 0, len(skills))
	for _, skill := range skills {
		result = append(result, &proto.SingleLifeSkillInfo{SkillId: skill.SkillId, Lv: skill.Lv, StartStudyTime: skill.StartStudyTime})
	}
	return result
}

func (g *Game) sendLifePack(s *model.Player, items []model.LifeItem) {
	if len(items) == 0 {
		return
	}
	g.send(s, 0, &proto.PackNotice{Status: proto.StatusCode_StatusCode_Ok, Items: lifeItemDetails(items)})
}

func (g *Game) sendLifeProductionNotices(s *model.Player, receipt model.LifeReceipt) {
	g.sendLifePack(s, receipt.Items)
	life := s.GetLifeModel()
	g.send(s, 0, &proto.LifeProficiencyNotice{
		Status: proto.StatusCode_StatusCode_Ok,
		LifeProficiency: &proto.LifeProficiency{
			LifeType: receipt.LifeType, ProficiencyValue: receipt.Proficiency,
		},
	})
	if len(receipt.AchieveIDs) > 0 {
		cfg, err := getLifeConfig()
		if err == nil {
			g.send(s, 0, &proto.LifeAchieveNotice{
				Status:       proto.StatusCode_StatusCode_Ok,
				LifeAchieves: []*proto.LifeAchieve{life.AchieveInfo(receipt.LifeType, cfg)},
			})
		}
	}
}
