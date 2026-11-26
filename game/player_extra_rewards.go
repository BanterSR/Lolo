package game

import (
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

// 活动/收集类请求的持久化确认。没有权威奖励表的功能返回空奖励，
// 但会记录领取状态，避免客户端重复请求或重启后回档。

func (g *Game) BlessTree(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BlessTreeReq)
	rsp := &proto.BlessTreeRsp{
		Status: proto.StatusCode_StatusCode_OK,
		DefId:  req.DefId,
		Tress:  &proto.PlayerBlessTreeInfo{TreeIds: make([]uint32, 0)},
	}
	defer g.send(s, msg.PacketId, rsp)

	conf := gdconf.GetBlessingTreeConfigure(req.DefId)
	if conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	if gdconf.GetBlessingTreeInfoConfigure(uint32(conf.GetBlessingTreeInfoID())) == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	prefix := "bless:" + strconv.Itoa(int(req.DefId)) + ":"
	for key, value := range s.GetArchive().GetArchiveMap() {
		if value != "1" || !strings.HasPrefix(key, prefix) {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(key, prefix), 10, 32)
		if err == nil {
			rsp.Tress.TreeIds = append(rsp.Tress.TreeIds, uint32(id))
		}
	}
	sort.Slice(rsp.Tress.TreeIds, func(i, j int) bool { return rsp.Tress.TreeIds[i] < rsp.Tress.TreeIds[j] })
}

func (g *Game) BlessTreeUnlock(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.BlessTreeUnlockReq)
	rsp := &proto.BlessTreeUnlockRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		DefId:   req.DefId,
		TreeId:  req.TreeId,
		Rewards: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	conf := gdconf.GetBlessingTreeConfigure(req.DefId)
	if conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	info := gdconf.GetBlessingTreeInfoConfigure(uint32(conf.GetBlessingTreeInfoID()))
	if info == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	var node = func() *excelBlessingTreeNode {
		for _, item := range info.GetBlessingTreeInfoGroupInfo() {
			if item != nil && uint32(item.GetBoxID()) == req.TreeId {
				return &excelBlessingTreeNode{
					boxID:        uint32(item.GetBoxID()),
					relateBoxID:  uint32(item.GetRelateBoxID()),
					itemID:       uint32(item.GetItemID()),
					itemCount:    int64(item.GetItemCount()),
					rewardItemID: uint32(item.GetRewardItemID()),
					rewardCount:  int64(item.GetRewardItemCount()),
				}
			}
		}
		return nil
	}()
	if node == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	key := "bless:" + strconv.Itoa(int(req.DefId)) + ":" + strconv.Itoa(int(req.TreeId))
	if s.GetArchive().GetArchiveValue(key) == "1" {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	if node.relateBoxID != 0 {
		parentKey := "bless:" + strconv.Itoa(int(req.DefId)) + ":" + strconv.Itoa(int(node.relateBoxID))
		if s.GetArchive().GetArchiveValue(parentKey) != "1" {
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			return
		}
	}
	if node.itemID != 0 && node.itemCount > 0 {
		tx, err := s.GetItemModel().Begin()
		if err != nil {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
		defer tx.Rollback()
		tx.DelBaseItem(node.itemID, node.itemCount)
		if tx.Error != nil {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
		tx.Commit()
	}
	if node.rewardItemID != 0 && node.rewardCount > 0 {
		if ctx := s.AddAllTypeItem(node.rewardItemID, node.rewardCount); ctx != nil {
			if detail := ctx.AddItemDetail(); detail != nil {
				rsp.Rewards = append(rsp.Rewards, detail)
			}
		}
	}
	s.GetArchive().SetArchiveMap(key, "1")
	if len(rsp.Rewards) > 0 {
		g.PackNoticeByItems(s, rsp.Rewards)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

type excelBlessingTreeNode struct {
	boxID        uint32
	relateBoxID  uint32
	itemID       uint32
	itemCount    int64
	rewardItemID uint32
	rewardCount  int64
}

func (g *Game) ActivityUploadWishTxt(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActivityUploadWishTxtReq)
	s.GetArchive().SetArchiveMap("wishTxt:"+strconv.Itoa(int(req.ActivityId)), req.Txt)
	g.send(s, msg.PacketId, &proto.ActivityUploadWishTxtRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		Txt:        req.Txt,
		ActivityId: req.ActivityId,
	})
}

func (g *Game) ActivityInviteRewardClaim(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActivityInviteRewardClaimReq)
	rsp := &proto.ActivityInviteRewardClaimRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		ActivityId:  req.ActivityId,
		ConditionId: req.ConditionId,
		Rewards:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	activity := s.GetActivityModel()
	rewarded := activity.GetRewardedAchievements()
	if rewarded[req.ConditionId] {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	entries := gdconf.GetActivityInviteQuestReward(req.ConditionId)
	if len(entries) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	for _, entry := range entries {
		if entry == nil || entry.GetItemID() <= 0 || entry.GetItemNum() <= 0 {
			continue
		}
		if gdconf.GetItemConfigure(uint32(entry.GetItemID())) == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
	}
	for _, entry := range entries {
		if entry == nil || entry.GetItemID() <= 0 || entry.GetItemNum() <= 0 {
			continue
		}
		ctx := s.AddAllTypeItem(uint32(entry.GetItemID()), int64(entry.GetItemNum()))
		if ctx == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
		if detail := ctx.AddItemDetail(); detail != nil {
			rsp.Rewards = append(rsp.Rewards, detail)
		}
	}
	rewarded[req.ConditionId] = true
	if len(rsp.Rewards) > 0 {
		g.PackNoticeByItems(s, rsp.Rewards)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

func (g *Game) ActivityRecallRewardClaim(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActivityRecallRewardClaimReq)
	key := "recall:" + strconv.Itoa(int(req.ActivityId)) + ":" + strconv.Itoa(int(req.ConditionId))
	rsp := &proto.ActivityRecallRewardClaimRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		ActivityId:  req.ActivityId,
		ConditionId: req.ConditionId,
		Rewards:     make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if s.GetArchive().GetArchiveValue(key) != "" {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	entries := gdconf.GetRecallActivityReward(req.ConditionId)
	if len(entries) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	for _, entry := range entries {
		if entry == nil || entry.GetItemID() <= 0 || entry.GetItemNum() <= 0 {
			continue
		}
		if gdconf.GetItemConfigure(uint32(entry.GetItemID())) == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
	}
	for _, entry := range entries {
		if entry == nil || entry.GetItemID() <= 0 || entry.GetItemNum() <= 0 {
			continue
		}
		ctx := s.AddAllTypeItem(uint32(entry.GetItemID()), int64(entry.GetItemNum()))
		if ctx == nil {
			rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
			return
		}
		if detail := ctx.AddItemDetail(); detail != nil {
			rsp.Rewards = append(rsp.Rewards, detail)
		}
	}
	s.GetArchive().SetArchiveMap(key, "1")
	if len(rsp.Rewards) > 0 {
		g.PackNoticeByItems(s, rsp.Rewards)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

func (g *Game) ActivityKoiUseItem(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActivityKoiUseItemReq)
	rsp := &proto.ActivityKoiUseItemRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		Num:        req.Num,
		Codes:      make([]string, 0),
		ActivityId: req.ActivityId,
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) ActivitySandbagHit(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActivitySandbagHitReq)
	rsp := &proto.ActivitySandbagHitRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		ActivityId: req.ActivityId,
	}
	defer g.send(s, msg.PacketId, rsp)
	if req.ActivityId == 0 || req.NpcId == 0 || req.HitCount == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	key := "sandbagHit:" + strconv.Itoa(int(req.ActivityId)) + ":" + strconv.Itoa(int(req.NpcId))
	archive := s.GetArchive()
	count, _ := strconv.ParseUint(archive.GetArchiveValue(key), 10, 64)
	count += uint64(req.HitCount)
	archive.SetArchiveMap(key, strconv.FormatUint(count, 10))
}

func (g *Game) ActivitySandbagReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ActivitySandbagRewardReq)
	rsp := &proto.ActivitySandbagRewardRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		ActivityId: req.ActivityId,
		SeqList:    make([]uint32, 0),
		Rewards:    make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	conf := gdconf.GetActivityInteractionConfigure(req.ActivityId)
	if conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	level := defaultActivityService().playerLevel(s)
	for _, seq := range req.SeqList {
		key := "sandbag:" + strconv.Itoa(int(req.ActivityId)) + ":" +
			strconv.Itoa(int(req.RewardType)) + ":" + strconv.Itoa(int(seq))
		if s.GetArchive().GetArchiveValue(key) != "" {
			continue
		}
		var poolID uint32
		for _, item := range conf.GetActivityInteractionItem() {
			if item != nil && uint32(item.GetIndex()) == seq {
				poolID = uint32(item.GetRewardPoolGroupID())
				break
			}
		}
		if poolID == 0 {
			continue
		}
		items, status := grantRewardItemPool(s, poolID, level)
		if status != proto.StatusCode_StatusCode_OK {
			rsp.Status = status
			return
		}
		s.GetArchive().SetArchiveMap(key, "1")
		rsp.SeqList = append(rsp.SeqList, seq)
		rsp.Rewards = append(rsp.Rewards, items...)
	}
	if len(rsp.SeqList) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	if len(rsp.Rewards) > 0 {
		g.PackNoticeByItems(s, rsp.Rewards)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

// grantRewardItemPool 从 RewardItemPool 直接发放一组奖励，用于交互活动等
// 只引用物品池而非完整 RewardPool 的配置。
func grantRewardItemPool(s *model.Player, poolID uint32, level uint32) ([]*proto.ItemDetail, proto.StatusCode) {
	pool := gdconf.GetRewardItemPool(poolID)
	if pool == nil || len(pool.GetRewardItemPoolGroup()) == 0 {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	items := make([]*proto.ItemDetail, 0)
	for _, entry := range pool.GetRewardItemPoolGroup() {
		if entry == nil || entry.GetItemID() <= 0 {
			continue
		}
		if entry.GetMinLevelLimit() > 0 && level < uint32(entry.GetMinLevelLimit()) {
			continue
		}
		if entry.GetMaxLevelLimit() > 0 && level > uint32(entry.GetMaxLevelLimit()) {
			continue
		}
		minCount := int64(entry.GetItemMinCount())
		maxCount := int64(entry.GetItemMaxCount())
		if minCount <= 0 {
			minCount = 1
		}
		if maxCount < minCount {
			maxCount = minCount
		}
		count := minCount
		if maxCount > minCount {
			count += int64(rand.IntN(int(maxCount-minCount+1)))
		}
		if gdconf.GetItemConfigure(uint32(entry.GetItemID())) == nil {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		ctx := s.AddAllTypeItem(uint32(entry.GetItemID()), count)
		if ctx == nil {
			return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
		}
		if detail := ctx.AddItemDetail(); detail != nil {
			items = append(items, detail)
		}
	}
	if len(items) == 0 {
		return nil, proto.StatusCode_StatusCode_CFG_NOT_FOUND
	}
	return items, proto.StatusCode_StatusCode_OK
}

func (g *Game) ManualFlagUnlock(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ManualFlagUnlockReq)
	s.GetArchive().SetArchiveMap("manual:"+strconv.Itoa(int(req.FlagId)), "1")
	g.send(s, msg.PacketId, &proto.ManualFlagUnlockRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Flag: &proto.ManualFlag{
			FlagId: req.FlagId,
			Status: proto.RewardStatus_Reward,
		},
	})
}

func (g *Game) ManualFlagReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ManualFlagRewardReq)
	rsp := &proto.ManualFlagRewardRsp{
		Status: proto.StatusCode_StatusCode_OK,
		Flag:   &proto.ManualFlag{FlagId: req.FlagId, Status: proto.RewardStatus_Reward},
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	conf := gdconf.GetManualItemByFlagId(req.FlagId)
	if conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	key := "manualReward:" + strconv.Itoa(int(req.FlagId))
	if s.GetArchive().GetArchiveValue(key) != "" {
		rsp.Status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
		return
	}
	rewardID := uint32(conf.GetRewardID())
	if rewardID == 0 {
		rewardID = uint32(conf.GetUnLockRewardID())
	}
	rewards, status := authoritativeDungeons.rewards(rewardID)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	items, status := authoritativeDungeons.grantRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_OK {
		rsp.Status = status
		return
	}
	s.GetArchive().SetArchiveMap(key, "1")
	rsp.Items = items
	if len(items) > 0 {
		g.PackNoticeByItems(s, items)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

func (g *Game) PlayerAbilityLevelUp(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerAbilityLevelUpReq)
	rsp := &proto.PlayerAbilityLevelUpRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		Items:    make([]*proto.ItemDetail, 0),
		Ability:  &proto.PlayerAbility{AbilityId: req.AbilityId},
	}
	defer g.send(s, msg.PacketId, rsp)

	conf := gdconf.GetAbilityConfigure(req.AbilityId)
	if conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	current := modelAbilityLevel(s, req.AbilityId)
	var nextLevel int32
	var costItem uint32
	var costCount int64
	var rewardItem uint32
	for _, level := range conf.GetLevelInfo() {
		if level == nil || uint32(level.GetLevel()) != current+1 {
			continue
		}
		nextLevel = level.GetLevel()
		costItem = uint32(level.GetCostItemID())
		costCount = int64(level.GetCostCount())
		rewardItem = uint32(level.GetRewardItemID())
		break
	}
	if nextLevel == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		rsp.Ability.Level = current
		return
	}
	if costItem != 0 && costCount > 0 {
		tx, err := s.GetItemModel().Begin()
		if err != nil {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
		defer tx.Rollback()
		tx.DelBaseItem(costItem, costCount)
		if tx.Error != nil {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
		tx.Commit()
	}
	if rewardItem != 0 {
		if ctx := s.AddAllTypeItem(rewardItem, 1); ctx != nil {
			if detail := ctx.AddItemDetail(); detail != nil {
				alg.AddList(&rsp.Items, detail)
				g.PackNoticeByItems(s, rsp.Items)
			}
		}
	}
	s.GetArchive().SetArchiveMap(
		"ability:"+strconv.Itoa(int(req.AbilityId)),
		strconv.Itoa(int(nextLevel)),
	)
	rsp.Ability.Level = uint32(nextLevel)
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

func (g *Game) AbilityBadgeAchieveReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AbilityBadgeAchieveRewardReq)
	rewarded := s.GetActivityModel().GetRewardedAbilityBadges()
	granted := make([]uint32, 0, len(req.AchieveIds))
	for _, id := range req.AchieveIds {
		if id == 0 || rewarded[id] {
			continue
		}
		rewarded[id] = true
		granted = append(granted, id)
	}
	g.send(s, msg.PacketId, &proto.AbilityBadgeAchieveRewardRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		AchieveIds: granted,
		Items:      make([]*proto.ItemDetail, 0),
	})
}

func (g *Game) AbilityBadgePageBoxActive(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AbilityBadgePageBoxActiveReq)
	key := "abilityBadgeBox:" + strconv.Itoa(int(req.Page)) + ":" + strconv.Itoa(int(req.BoxId))
	status := proto.StatusCode_StatusCode_OK
	if s.GetArchive().GetArchiveValue(key) != "" {
		status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	} else {
		s.GetArchive().SetArchiveMap(key, "1")
	}
	g.send(s, msg.PacketId, &proto.AbilityBadgePageBoxActiveRsp{
		Status: status,
		Page:   req.Page,
		BoxId:  req.BoxId,
		Items:  make([]*proto.ItemDetail, 0),
	})
}

func (g *Game) AbilityBadgePageReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.AbilityBadgePageRewardReq)
	key := "abilityBadgePage:" + strconv.Itoa(int(req.Page))
	status := proto.StatusCode_StatusCode_OK
	if s.GetArchive().GetArchiveValue(key) != "" {
		status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	} else {
		s.GetArchive().SetArchiveMap(key, "1")
	}
	g.send(s, msg.PacketId, &proto.AbilityBadgePageRewardRsp{
		Status: status,
		Page:   req.Page,
		Items:  make([]*proto.ItemDetail, 0),
	})
}

func (g *Game) GetCharacterAchievementBadgeAward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCharacterAchievementBadgeAwardReq)
	key := "characterBadge:" + strconv.Itoa(int(req.CharacterId)) + ":" + strconv.Itoa(int(req.AchievementId))
	status := proto.StatusCode_StatusCode_OK
	rewarded := make([]uint32, 0)
	if s.GetArchive().GetArchiveValue(key) != "" {
		status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	} else {
		s.GetArchive().SetArchiveMap(key, "1")
		rewarded = append(rewarded, req.AchievementId)
	}
	g.send(s, msg.PacketId, &proto.GetCharacterAchievementBadgeAwardRsp{
		Status:                   status,
		UpdateItems:              make([]*proto.ItemDetail, 0),
		Achieves:                 make([]*proto.Achieve, 0),
		RewardedAchievementIdLst: rewarded,
	})
}

func (g *Game) GetCharacterAchievementUnlockPayment(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetCharacterAchievementUnlockPaymentReq)
	key := "characterUnlockPayment:" + strconv.Itoa(int(req.CharacterId))
	unlock := s.GetArchive().GetArchiveValue(key) != ""
	if !unlock {
		s.GetArchive().SetArchiveMap(key, "1")
	}
	g.send(s, msg.PacketId, &proto.GetCharacterAchievementUnlockPaymentRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		UpdateItem:  make([]*proto.ItemDetail, 0),
		IsUnlock:    true,
		CharacterId: req.CharacterId,
	})
}

func (g *Game) GetOneTypeLifePictorialBookCount(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetOneTypeLifePictorialBookCountReq)
	rsp := &proto.GetOneTypeLifePictorialBookCountRsp{
		Status:              proto.StatusCode_StatusCode_OK,
		LifeAchieveCountLst: make([]*proto.LifeAchieveCount, 0),
	}
	if state := s.GetLifeModel().GetState(req.LifeType); state != nil {
		ids := make([]uint32, 0, len(state.Achievements))
		for id := range state.Achievements {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			achievement := state.Achievements[id]
			if achievement == nil {
				continue
			}
			rsp.LifeAchieveCountLst = append(rsp.LifeAchieveCountLst, &proto.LifeAchieveCount{
				Id:    id,
				Count: achievement.Count,
			})
		}
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) GetTreasureAchieveReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetTreasureAchieveRewardReq)
	key := "treasureAchieve:" + strconv.Itoa(int(req.AchieveId))
	status := proto.StatusCode_StatusCode_OK
	rewarded := make([]uint32, 0)
	if s.GetArchive().GetArchiveValue(key) != "" {
		status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	} else {
		s.GetArchive().SetArchiveMap(key, "1")
		rewarded = append(rewarded, req.AchieveId)
	}
	g.send(s, msg.PacketId, &proto.GetTreasureAchieveRewardRsp{
		Status:             status,
		Items:              make([]*proto.ItemDetail, 0),
		RewardedAchieveIds: rewarded,
	})
}

func (g *Game) TreasureIllustrationReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TreasureIllustrationRewardReq)
	key := "treasureIllustration:" + strconv.Itoa(int(req.ArchaeologyType)) + ":" + strconv.Itoa(int(req.Id))
	status := proto.StatusCode_StatusCode_OK
	if s.GetArchive().GetArchiveValue(key) != "" {
		status = proto.StatusCode_StatusCode_REPEAT_GET_REWARD
	} else {
		s.GetArchive().SetArchiveMap(key, "1")
	}
	g.send(s, msg.PacketId, &proto.TreasureIllustrationRewardRsp{
		Status:          status,
		ArchaeologyType: req.ArchaeologyType,
		Id:              req.Id,
		Rewards:         make([]*proto.ItemDetail, 0),
	})
}

func (g *Game) TreasureDig(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.TreasureDigReq)
	g.send(s, msg.PacketId, &proto.TreasureDigRsp{
		Status:     proto.StatusCode_StatusCode_OK,
		TreasureId: req.TreasureId,
		Rewards:    make([]*proto.ItemDetail, 0),
		Success:    req.TreasureId > 0,
	})
}

func (g *Game) Sewing(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.SewingReq)
	receipt := g.produceLifeRecipe(s, msg, proto.LifeType_LIFE_TYPE_Sewing, req.SewingId, false)
	rsp := &proto.SewingRsp{Status: receipt.Status}
	if receipt.Status == proto.StatusCode_StatusCode_OK {
		rsp.LifeCommonResult = lifeCommonResult(receipt)
	}
	g.send(s, msg.PacketId, rsp)
}

func (g *Game) ItemSweep(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ItemSweepReq)
	rsp := &proto.ItemSweepRsp{
		Status:  proto.StatusCode_StatusCode_OK,
		Rewards: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if req.ItemId == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	count := int64(req.Count)
	if count <= 0 {
		count = 1
	}
	conf := gdconf.GetQuickObtainingConfigure(req.ItemId)
	if conf == nil || gdconf.GetItemConfigure(req.ItemId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	level := defaultActivityService().playerLevel(s)
	var selected *excelQuickObtainingItem
	for _, item := range conf.GetQuickObtainingItem() {
		if item == nil || item.GetNum() <= 0 || uint32(item.GetPlayerLevel()) > level {
			continue
		}
		if selected == nil || item.GetPlayerLevel() > selected.playerLevel {
			selected = &excelQuickObtainingItem{
				playerLevel:         item.GetPlayerLevel(),
				num:                 item.GetNum(),
				needVitality:        item.GetNeedVitality(),
				needSpecialCurrency: item.GetNeedSpecialcurrency(),
			}
		}
	}
	if selected == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	needVitality := int64(selected.needVitality) * count
	needSpecial := int64(selected.needSpecialCurrency) * count

	var vitalityItemID uint32
	if needVitality > 0 {
		for id, info := range s.GetItemModel().GetItemBaseMap() {
			if info != nil && info.ItemType == proto.EBagItemTag_EBagItemTag_Vitality && info.Num >= needVitality {
				vitalityItemID = id
				break
			}
		}
		if vitalityItemID == 0 {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
	}
	if needSpecial > 0 {
		special := s.GetItemModel().GetItemBaseInfo(103)
		if special == nil || special.Num < needSpecial {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
	}
	if needVitality > 0 || needSpecial > 0 {
		tx, err := s.GetItemModel().Begin()
		if err != nil {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
		defer tx.Rollback()
		if needVitality > 0 {
			tx.DelBaseItem(vitalityItemID, needVitality)
		}
		if needSpecial > 0 {
			tx.DelBaseItem(103, needSpecial)
		}
		if tx.Error != nil {
			rsp.Status = proto.StatusCode_StatusCode_ITEM_NOT_ENOUGH
			return
		}
		tx.Commit()
	}
	amount := int64(selected.num) * count
	ctx := s.AddAllTypeItem(req.ItemId, amount)
	if ctx == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	if detail := ctx.AddItemDetail(); detail != nil {
		rsp.Rewards = append(rsp.Rewards, detail)
		g.PackNoticeByItems(s, rsp.Rewards)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PLAYER_NOT_FOUND
	}
}

type excelQuickObtainingItem struct {
	playerLevel         int32
	num                 int32
	needVitality        int32
	needSpecialCurrency int32
}

func (g *Game) OutfitRandomColor(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitRandomColorReq)
	rsp := &proto.OutfitRandomColorRsp{
		Status:   proto.StatusCode_StatusCode_OK,
		OutfitId: req.OutfitId,
	}
	defer g.send(s, msg.PacketId, rsp)
	if s.GetItemModel().GetItemFashionInfo(req.OutfitId) == nil {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	dyeData := gdconf.GetDyeDataList()
	if len(dyeData) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_CFG_NOT_FOUND
		return
	}
	picked := dyeData[rand.IntN(len(dyeData))]
	clamp := func(value int32) uint32 {
		if value < 0 {
			return 0
		}
		if value > 255 {
			return 255
		}
		return uint32(value)
	}
	rsp.PosColor = &proto.PosColor{
		Red:   clamp(picked.GetColorR()),
		Green: clamp(picked.GetColorG()),
		Blue:  clamp(picked.GetColorB()),
	}
}

func (g *Game) OutfitDyeUnlockIndex(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitDyeUnlockIndexReq)
	fashion := s.GetItemModel().GetItemFashionInfo(req.OutfitId)
	if fashion == nil {
		g.send(s, msg.PacketId, &proto.OutfitDyeUnlockIndexRsp{Status: proto.StatusCode_StatusCode_CFG_NOT_FOUND})
		return
	}
	if fashion.GetDyeScheme(req.Index) == nil {
		fashion.DyeSchemes[req.Index] = &model.OutfitDyeScheme{
			SchemeIndex: req.Index,
			IsUnLock:    true,
			Colors:      make(model.Colors, 0),
		}
	}
	g.send(s, msg.PacketId, &proto.OutfitDyeUnlockIndexRsp{Status: proto.StatusCode_StatusCode_OK})
}
