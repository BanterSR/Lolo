package game

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/constant"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

// gdconf 尚未提供 Level/Head/Phone/Action 访问器，配置加载集中在游戏层。
// 任一请求所需配置缺失或非法时直接拒绝，不使用兜底数值。
var (
	profileLevelConfigProvider  = loadProfileLevelConfig
	profileHeadConfigProvider   = loadProfileHeadConfig
	profilePhoneConfigProvider  = loadProfilePhoneConfig
	profileActionConfigProvider = loadProfileActionConfig
	profileExpressionProvider   = loadExpressionConfig
	profileItemConfigure        = gdconf.GetItemConfigure
	profileGlobalConfigure      = gdconf.GetGlobalConfigConfigure
	profileAchieveConfigure     = gdconf.GetAchieveConfigure
	profileRewardPool           = gdconf.GetRewardPool
	profileRewardItemPool       = gdconf.GetRewardItemPool
	profilePlayerLevel          = defaultProfilePlayerLevel
	profileAccessor             = model.GetPlayerProfile
	profileSavePlayer           = func(s *model.Player) error { return s.SavePlayerNow() }
	profileRollInt64N           = rand.Int64N
	profileNow                  = time.Now
)

const (
	profileServerTimezoneOffset = int64(8 * 60 * 60)
	profileEnergyDaySeconds     = int64(24 * 60 * 60)
)

var (
	profileLevelOnce    sync.Once
	profileLevelConfig  *excel.AllLevelDatas
	profileLevelErr     error
	profileHeadOnce     sync.Once
	profileHeadConfig   *excel.AllHeadDatas
	profileHeadErr      error
	profilePhoneOnce    sync.Once
	profilePhoneConfig  *excel.AllPhoneDatas
	profilePhoneErr     error
	profileActionOnce   sync.Once
	profileActionConfig *excel.AllActionDatas
	profileActionErr    error
)

func loadProfileLevelConfig() (*excel.AllLevelDatas, error) {
	profileLevelOnce.Do(func() {
		profileLevelConfig, profileLevelErr = loadExcelConfig[excel.AllLevelDatas]("Level.json")
	})
	return profileLevelConfig, profileLevelErr
}

func loadProfileHeadConfig() (*excel.AllHeadDatas, error) {
	profileHeadOnce.Do(func() {
		profileHeadConfig, profileHeadErr = loadExcelConfig[excel.AllHeadDatas]("Head.json")
	})
	return profileHeadConfig, profileHeadErr
}

func loadProfilePhoneConfig() (*excel.AllPhoneDatas, error) {
	profilePhoneOnce.Do(func() {
		profilePhoneConfig, profilePhoneErr = loadExcelConfig[excel.AllPhoneDatas]("Phone.json")
	})
	return profilePhoneConfig, profilePhoneErr
}

func loadProfileActionConfig() (*excel.AllActionDatas, error) {
	profileActionOnce.Do(func() {
		profileActionConfig, profileActionErr = loadExcelConfig[excel.AllActionDatas]("Action.json")
	})
	return profileActionConfig, profileActionErr
}

func defaultProfilePlayerLevel(s *model.Player) (uint32, bool) {
	if s == nil {
		return 0, false
	}
	basic, ok := db.GetGameBasic(s.UserId)
	if !ok || basic == nil {
		return 0, false
	}
	return basic.Level, true
}

func requirePlayerProfile(s *model.Player) (*model.ProfileModel, proto.StatusCode) {
	profile := profileAccessor(s)
	if profile == nil {
		return nil, proto.StatusCode_StatusCode_PlayerNotFound
	}
	profile.EnsureMaps()
	return profile, proto.StatusCode_StatusCode_Ok
}

func saveProfileMutation(s *model.Player, status *proto.StatusCode) {
	if err := profileSavePlayer(s); err != nil {
		*status = proto.StatusCode_StatusCode_SysErr
		log.Game.Errorf("UserId:%v 保存玩家档案失败:%s", s.UserId, err.Error())
	}
}

type profileItemReward struct {
	itemID uint32
	count  int64
}

type profileEnergyRules struct {
	maxEnergy int64
	interval  int64
}

func profileEnergyDay(now int64) int64 {
	return (now + profileServerTimezoneOffset) / profileEnergyDaySeconds
}

func normalizeProfileEnergyPurchases(profile *model.ProfileModel, now int64) bool {
	day := profileEnergyDay(now)
	if profile.EnergyBuyDay == day {
		return false
	}
	profile.EnergyBuyDay = day
	profile.EnergyBuyNum = 0
	return true
}

func configuredProfilePositiveInt(key string) (int64, bool) {
	conf := profileGlobalConfigure(key)
	if conf == nil || len(conf.Values) != 1 {
		return 0, false
	}
	value, err := strconv.ParseInt(conf.Values[0], 10, 64)
	return value, err == nil && value > 0
}

func loadProfileEnergyRules(playerLevel uint32) (profileEnergyRules, bool) {
	conf, err := profileLevelConfigProvider()
	if err != nil || conf == nil || conf.GetPlayerLevel() == nil {
		return profileEnergyRules{}, false
	}
	maxEnergy := int64(0)
	for _, row := range conf.GetPlayerLevel().GetDatas() {
		if row != nil && row.ID > 0 && uint32(row.ID) == playerLevel {
			maxEnergy = int64(row.PlayerEnergy)
			break
		}
	}
	interval, ok := configuredProfilePositiveInt("Player_Recover_Vitality_Second")
	if !ok || maxEnergy <= 0 || maxEnergy > math.MaxUint32 {
		return profileEnergyRules{}, false
	}
	itemConf := profileItemConfigure(constant.CurrencyVitality)
	if itemConf == nil || proto.EBagItemTag(itemConf.GetNewBagItemTag()) != proto.EBagItemTag_EBagItemTag_Energy {
		return profileEnergyRules{}, false
	}
	return profileEnergyRules{maxEnergy: maxEnergy, interval: interval}, true
}

func ensureProfileEnergyItem(s *model.Player) *model.ItemBaseInfo {
	const itemID = constant.CurrencyVitality
	items := s.GetItemModel()
	if info := items.GetItemBaseInfo(itemID); info != nil {
		return info
	}
	conf := profileItemConfigure(itemID)
	if conf == nil || proto.EBagItemTag(conf.GetNewBagItemTag()) != proto.EBagItemTag_EBagItemTag_Energy {
		return nil
	}
	info := &model.ItemBaseInfo{
		ItemId:   itemID,
		Num:      0,
		ItemType: proto.EBagItemTag_EBagItemTag_Energy,
		PackType: proto.PackType_PackType_Inventory,
	}
	items.GetItemBaseMap()[itemID] = info
	return info
}

func recoverProfileEnergy(s *model.Player, profile *model.ProfileModel, rules profileEnergyRules, now int64) (*model.ItemBaseInfo, bool, proto.StatusCode) {
	info := ensureProfileEnergyItem(s)
	if info == nil || info.Num < 0 {
		return nil, false, proto.StatusCode_StatusCode_CfgNotFound
	}
	dirty := false
	if info.Num > rules.maxEnergy {
		info.Num = rules.maxEnergy
		dirty = true
	}
	if profile.EnergyLastRecoverAt <= 0 || profile.EnergyLastRecoverAt > now {
		profile.EnergyLastRecoverAt = now
		return info, true, proto.StatusCode_StatusCode_Ok
	}
	if info.Num >= rules.maxEnergy {
		if profile.EnergyLastRecoverAt != now {
			profile.EnergyLastRecoverAt = now
			dirty = true
		}
		return info, dirty, proto.StatusCode_StatusCode_Ok
	}
	elapsed := now - profile.EnergyLastRecoverAt
	steps := elapsed / rules.interval
	if steps <= 0 {
		return info, dirty, proto.StatusCode_StatusCode_Ok
	}
	missing := rules.maxEnergy - info.Num
	recovered := steps
	if recovered > missing {
		recovered = missing
	}
	info.Num += recovered
	profile.EnergyLastRecoverAt += steps * rules.interval
	if info.Num >= rules.maxEnergy {
		profile.EnergyLastRecoverAt = now
	}
	return info, true, proto.StatusCode_StatusCode_Ok
}

type profileEnergyTier struct {
	currencyID uint32
	price      int64
	amount     int64
}

func profileEnergyTierAt(index uint32) (profileEnergyTier, bool) {
	conf := profileGlobalConfigure("Player_Energy_Buy_Times")
	if conf == nil || index >= uint32(len(conf.Values)) {
		return profileEnergyTier{}, false
	}
	parts := strings.Split(conf.Values[index], ":")
	if len(parts) != 3 {
		return profileEnergyTier{}, false
	}
	currency, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil || currency == 0 {
		return profileEnergyTier{}, false
	}
	price, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || price <= 0 {
		return profileEnergyTier{}, false
	}
	amount, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || amount <= 0 {
		return profileEnergyTier{}, false
	}
	itemConf := profileItemConfigure(uint32(currency))
	if itemConf == nil || proto.EBagItemTag(itemConf.GetNewBagItemTag()) != proto.EBagItemTag_EBagItemTag_Currency {
		return profileEnergyTier{}, false
	}
	return profileEnergyTier{currencyID: uint32(currency), price: price, amount: amount}, true
}

func (g *Game) PlayerEnergyInfo(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PlayerEnergyInfoRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	playerLevel, ok := profilePlayerLevel(s)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	rules, ok := loadProfileEnergyRules(playerLevel)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	now := profileNow().Unix()
	normalizeProfileEnergyPurchases(profile, now)
	info, dirty, status := recoverProfileEnergy(s, profile, rules, now)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	rsp.EnergyBuyNum = profile.EnergyBuyNum
	rsp.EnergyItem = info.ItemDetail()
	if dirty {
		saveProfileMutation(s, &rsp.Status)
	}
}

func (g *Game) PlayerEnergyBuy(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.PlayerEnergyBuyRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Items:  make([]*proto.ItemDetail, 0, 1),
	}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	playerLevel, ok := profilePlayerLevel(s)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	rules, ok := loadProfileEnergyRules(playerLevel)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	now := profileNow().Unix()
	normalizeProfileEnergyPurchases(profile, now)
	rsp.EnergyBuyNum = profile.EnergyBuyNum
	tier, ok := profileEnergyTierAt(profile.EnergyBuyNum)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_EnergyBuyTimesLimit
		return
	}
	energy, _, status := recoverProfileEnergy(s, profile, rules, now)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	if energy.Num >= rules.maxEnergy {
		rsp.Status = proto.StatusCode_StatusCode_Fail
		return
	}
	currency := s.GetItemModel().GetItemBaseInfo(tier.currencyID)
	oldCurrency := int64(0)
	if currency != nil {
		oldCurrency = currency.Num
	}
	oldEnergy := energy.Num
	oldProfile := *profile
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	defer tx.Rollback()
	tx.DelBaseItem(tier.currencyID, tier.price)
	if tx.Error != nil {
		rsp.Status = proto.StatusCode_StatusCode_ItemNotEnough
		return
	}
	amount := tier.amount
	if amount > rules.maxEnergy-energy.Num {
		amount = rules.maxEnergy - energy.Num
	}
	items, err := grantProfileBaseRewards(s, []profileItemReward{{itemID: constant.CurrencyVitality, count: amount}})
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	tx.Commit()
	profile.EnergyBuyNum++
	rsp.EnergyBuyNum = profile.EnergyBuyNum
	rsp.Items = append(rsp.Items, items...)
	if err := profileSavePlayer(s); err != nil {
		*profile = oldProfile
		energy.Num = oldEnergy
		if currency != nil {
			currency.Num = oldCurrency
		}
		rsp.Status = proto.StatusCode_StatusCode_SysErr
		log.Game.Errorf("UserId:%v 精力购买落库失败:%s", s.UserId, err.Error())
		return
	}
	g.send(s, 0, tx.PackNotice)
}

func findProfilePlayerLevel(level uint32) (*excel.PlayerLevelConfigure, bool) {
	conf, err := profileLevelConfigProvider()
	if err != nil || conf == nil || conf.GetPlayerLevel() == nil {
		return nil, false
	}
	for _, row := range conf.GetPlayerLevel().GetDatas() {
		if row != nil && row.ID > 0 && uint32(row.ID) == level {
			return row, true
		}
	}
	return nil, true
}

func findProfileWorldLevel(level uint32) (*excel.WorldLevelConfigure, bool) {
	conf, err := profileLevelConfigProvider()
	if err != nil || conf == nil || conf.GetWorldLevel() == nil {
		return nil, false
	}
	for _, row := range conf.GetWorldLevel().GetDatas() {
		if row != nil && row.Level >= 0 && uint32(row.Level) == level {
			return row, true
		}
	}
	return nil, true
}

func resolveProfileLevelRewards(rewardID int32, level uint32) ([]profileItemReward, bool) {
	if rewardID <= 0 {
		return nil, false
	}
	pool := profileRewardPool(uint32(rewardID))
	if pool == nil {
		return nil, false
	}
	rewards := make([]profileItemReward, 0)
	for _, group := range pool.GetRewardPoolGroup() {
		if group == nil || group.RewardPoolGroupID <= 0 {
			return nil, false
		}
		itemPool := profileRewardItemPool(uint32(group.RewardPoolGroupID))
		if itemPool == nil {
			return nil, false
		}
		for _, item := range itemPool.GetRewardItemPoolGroup() {
			if item == nil || item.ItemID <= 0 || item.ItemMinCount < 0 || item.ItemMaxCount < item.ItemMinCount {
				return nil, false
			}
			if item.MinLevelLimit > 0 && level < uint32(item.MinLevelLimit) {
				continue
			}
			if item.MaxLevelLimit > 0 && level > uint32(item.MaxLevelLimit) {
				continue
			}
			count, ok := collectionRewardCount(item.ItemMinCount, item.ItemMaxCount, profileRollInt64N)
			if !ok || count <= 0 {
				return nil, false
			}
			rewards = append(rewards, profileItemReward{itemID: uint32(item.ItemID), count: count})
		}
	}
	return rewards, len(rewards) > 0
}

func grantPlayerLevelReward(s *model.Player, profile *model.ProfileModel, level uint32) ([]*proto.ItemDetail, proto.StatusCode) {
	if profile.HasPlayerLevelReward(level) {
		return nil, proto.StatusCode_StatusCode_RepeatGetReward
	}
	row, configured := findProfilePlayerLevel(level)
	if !configured || row == nil {
		return nil, proto.StatusCode_StatusCode_CfgNotFound
	}
	playerLevel, ok := profilePlayerLevel(s)
	if !ok {
		return nil, proto.StatusCode_StatusCode_PlayerNotFound
	}
	if playerLevel < level {
		return nil, proto.StatusCode_StatusCode_PlayerLevelLimit
	}
	rewards, ok := resolveProfileLevelRewards(row.RewardID, level)
	if !ok {
		return nil, proto.StatusCode_StatusCode_CfgNotFound
	}
	tx, err := s.GetItemModel().Begin()
	if err != nil {
		return nil, proto.StatusCode_StatusCode_BadReq
	}
	defer tx.Rollback()
	oldCounts := make(map[uint32]int64, len(rewards))
	oldExists := make(map[uint32]bool, len(rewards))
	for _, reward := range rewards {
		info := s.GetItemModel().GetItemBaseInfo(reward.itemID)
		oldExists[reward.itemID] = info != nil
		if info != nil {
			oldCounts[reward.itemID] = info.Num
		}
	}
	items, err := grantProfileBaseRewards(s, rewards)
	if err != nil {
		return nil, proto.StatusCode_StatusCode_CfgNotFound
	}
	tx.Commit()
	profile.MarkPlayerLevelReward(level)
	if err := profileSavePlayer(s); err != nil {
		for _, reward := range rewards {
			info := s.GetItemModel().GetItemBaseInfo(reward.itemID)
			if !oldExists[reward.itemID] {
				delete(s.GetItemModel().GetItemBaseMap(), reward.itemID)
			} else if info != nil {
				info.Num = oldCounts[reward.itemID]
			}
		}
		delete(profile.PlayerLevelRewards, level)
		return nil, proto.StatusCode_StatusCode_SysErr
	}
	return items, proto.StatusCode_StatusCode_Ok
}

func (g *Game) PlayerLevelReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerLevelRewardReq)
	rsp := &proto.PlayerLevelRewardRsp{
		Status: proto.StatusCode_StatusCode_Ok,
		Level:  req.Level,
		Items:  make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	items, status := grantPlayerLevelReward(s, profile, req.Level)
	rsp.Status = status
	rsp.Items = append(rsp.Items, items...)
}

func configuredProfileHead(headID uint32) (*excel.HeadConfigure, bool) {
	conf, err := profileHeadConfigProvider()
	if err != nil || conf == nil || conf.GetHead() == nil {
		return nil, false
	}
	for _, row := range conf.GetHead().GetDatas() {
		if row != nil && row.HeadID > 0 && uint32(row.HeadID) == headID {
			return row, true
		}
	}
	return nil, true
}

func validOwnedProfileHead(s *model.Player, headID uint32) bool {
	conf, configured := configuredProfileHead(headID)
	if !configured || conf == nil || !conf.IsShow || uint32(conf.HeadID) != headID {
		return false
	}
	itemConf := profileItemConfigure(headID)
	if itemConf == nil || proto.EBagItemTag(itemConf.GetNewBagItemTag()) != proto.EBagItemTag_EBagItemTag_Head {
		return false
	}
	// 直接读持久化 map，不能调用会补默认头像的 GetItemHeadMap。
	info := s.GetItemModel().ItemHeadMap[headID]
	return info != nil && info.ItemBaseInfo != nil && info.HeadId == headID &&
		info.ItemId == headID && info.ItemType == proto.EBagItemTag_EBagItemTag_Head
}

func (g *Game) ChangeHead(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeHeadReq)
	rsp := &proto.ChangeHeadRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	if !validOwnedProfileHead(s, req.Head) {
		rsp.Status = proto.StatusCode_StatusCode_HeadNotUnlock
		return
	}
	if err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.Head = req.Head
		return true
	}); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	saveProfileMutation(s, &rsp.Status)
	if rsp.Status == proto.StatusCode_StatusCode_Ok {
		rsp.Head = req.Head
	}
}

func (g *Game) ChangePlayerSex(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangePlayerSexReq)
	rsp := &proto.ChangePlayerSexRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	if req.Sex != proto.ESexType_ESexType_Male && req.Sex != proto.ESexType_ESexType_Female {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	if err := db.UpGameBasic(s.UserId, func(basic *db.OFGameBasic) bool {
		basic.Sex = req.Sex
		return true
	}); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	saveProfileMutation(s, &rsp.Status)
	if rsp.Status == proto.StatusCode_StatusCode_Ok {
		rsp.Sex = req.Sex
	}
}

func validOwnedProfilePhoneCase(s *model.Player, phoneCase uint32) bool {
	conf, err := profilePhoneConfigProvider()
	if err != nil || conf == nil || conf.GetPhone() == nil {
		return false
	}
	configured := false
	for _, row := range conf.GetPhone().GetDatas() {
		if row != nil && row.ID > 0 && uint32(row.ID) == phoneCase && row.AssetPath != "" {
			configured = true
			break
		}
	}
	if !configured {
		return false
	}
	itemConf := profileItemConfigure(phoneCase)
	if itemConf == nil || proto.EBagItemTag(itemConf.GetNewBagItemTag()) != proto.EBagItemTag_EBagItemTag_PhoneCase {
		return false
	}
	info := s.GetItemModel().GetItemBaseInfo(phoneCase)
	return info != nil && info.ItemId == phoneCase && info.Num > 0 &&
		info.ItemType == proto.EBagItemTag_EBagItemTag_PhoneCase
}

func (g *Game) ChangePhoneCase(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangePhoneCaseReq)
	rsp := &proto.ChangePhoneCaseRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	if !validOwnedProfilePhoneCase(s, req.PhoneCase) {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	oldPhoneCase := profile.PhoneCase
	profile.PhoneCase = req.PhoneCase
	saveProfileMutation(s, &rsp.Status)
	if rsp.Status != proto.StatusCode_StatusCode_Ok {
		profile.PhoneCase = oldPhoneCase
		return
	}
	rsp.PhoneCase = req.PhoneCase
	g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdatePhoneCase)
}

func profileWorldLevelAvailable(profile *model.ProfileModel, level uint32) bool {
	return level == 0 || profile.IsWorldLevelUnlocked(level)
}

func validateProfileWorldPrerequisites(s *model.Player, profile *model.ProfileModel, conf *excel.WorldLevelConfigure, playerLevel uint32) proto.StatusCode {
	if conf == nil || conf.Level < 0 {
		return proto.StatusCode_StatusCode_CfgNotFound
	}
	if playerLevel < uint32(conf.Level) {
		return proto.StatusCode_StatusCode_PlayerLevelLimit
	}
	for _, info := range conf.GetWorldLevelInfo() {
		if info == nil {
			return proto.StatusCode_StatusCode_CfgNotFound
		}
		if info.AchieveID > 0 && !profile.CompletedAchievements[uint32(info.AchieveID)] {
			achieveID := uint32(info.AchieveID)
			achieve := profileAchieveConfigure(achieveID)
			if achieve == nil || achieve.CountParam <= 0 {
				return proto.StatusCode_StatusCode_CfgNotFound
			}
			if s.GetActivityModel().AchievementProgress(achieveID) < uint64(achieve.CountParam) {
				return proto.StatusCode_StatusCode_WorldLevelAchieveNotFinish
			}
		}
		if info.FlagID > 0 && !profile.CompletedFlags[uint32(info.FlagID)] {
			return proto.StatusCode_StatusCode_WorldLevelAchieveNotFinish
		}
	}
	return proto.StatusCode_StatusCode_Ok
}

func unlockProfileWorldLevel(s *model.Player, profile *model.ProfileModel, level, playerLevel uint32) proto.StatusCode {
	conf, configured := findProfileWorldLevel(level)
	if !configured {
		return proto.StatusCode_StatusCode_CfgNotFound
	}
	if conf == nil {
		return proto.StatusCode_StatusCode_BadReq
	}
	if profileWorldLevelAvailable(profile, level) {
		return proto.StatusCode_StatusCode_WorldLevelAlreadyUnlock
	}
	if status := validateProfileWorldPrerequisites(s, profile, conf, playerLevel); status != proto.StatusCode_StatusCode_Ok {
		return status
	}
	if level > 1 && !profile.IsWorldLevelUnlocked(level-1) {
		return proto.StatusCode_StatusCode_WorldLevelLocked
	}
	profile.MarkWorldLevelUnlocked(level)
	return proto.StatusCode_StatusCode_Ok
}

func changeProfileWorldLevel(s *model.Player, profile *model.ProfileModel, level, playerLevel uint32) proto.StatusCode {
	conf, configured := findProfileWorldLevel(level)
	if !configured {
		return proto.StatusCode_StatusCode_CfgNotFound
	}
	if conf == nil {
		return proto.StatusCode_StatusCode_BadReq
	}
	if !profileWorldLevelAvailable(profile, level) {
		return proto.StatusCode_StatusCode_WorldLevelLocked
	}
	if playerLevel < uint32(conf.Level) {
		return proto.StatusCode_StatusCode_PlayerLevelLimit
	}
	if s.GetSceneModel().GetWorldLevel() == level {
		return proto.StatusCode_StatusCode_WorldLevelNowIn
	}
	s.GetSceneModel().WorldLevel = level
	return proto.StatusCode_StatusCode_Ok
}

func (g *Game) UnlockWorldLevel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.UnlockWorldLevelReq)
	rsp := &proto.UnlockWorldLevelRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	playerLevel, ok := profilePlayerLevel(s)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	rsp.Status = unlockProfileWorldLevel(s, profile, req.WorldLevel, playerLevel)
	if rsp.Status == proto.StatusCode_StatusCode_Ok {
		rsp.WorldLevel = req.WorldLevel
		saveProfileMutation(s, &rsp.Status)
		if rsp.Status != proto.StatusCode_StatusCode_Ok {
			delete(profile.UnlockedWorldLevels, req.WorldLevel)
		}
	}
}

func (g *Game) ChangeWorldLevel(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangeWorldLevelReq)
	rsp := &proto.ChangeWorldLevelRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	profile, status := requirePlayerProfile(s)
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	playerLevel, ok := profilePlayerLevel(s)
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		return
	}
	oldWorldLevel := s.GetSceneModel().GetWorldLevel()
	rsp.Status = changeProfileWorldLevel(s, profile, req.WorldLevel, playerLevel)
	if rsp.Status == proto.StatusCode_StatusCode_Ok {
		rsp.WorldLevel = req.WorldLevel
		saveProfileMutation(s, &rsp.Status)
		if rsp.Status != proto.StatusCode_StatusCode_Ok {
			s.GetSceneModel().WorldLevel = oldWorldLevel
		}
	}
}

func applyProfileOutfitHide(s *model.Player, charID uint32, hide *proto.OutfitHideInfo) proto.StatusCode {
	if hide == nil {
		return proto.StatusCode_StatusCode_BadReq
	}
	character := s.GetCharacterModel().GetCharacterInfo(charID)
	if character == nil {
		return proto.StatusCode_StatusCode_BadReq
	}
	preset := character.GetOutfitPreset(character.InUseOutfitPresetIndex)
	if preset == nil {
		return proto.StatusCode_StatusCode_BadReq
	}
	if preset.OutfitHideInfo == nil {
		preset.OutfitHideInfo = &model.OutfitHideInfo{}
	}
	// 协议只开放这两个布尔字段，不整体替换或透传未知字段。
	preset.OutfitHideInfo.HideOrn = hide.HideOrn
	preset.OutfitHideInfo.HideBraid = hide.HideBraid
	return proto.StatusCode_StatusCode_Ok
}

func (g *Game) OutfitHideSwitch(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.OutfitHideSwitchReq)
	rsp := &proto.OutfitHideSwitchRsp{Status: proto.StatusCode_StatusCode_Ok, CharId: req.CharId}
	defer g.send(s, msg.PacketId, rsp)
	character := s.GetCharacterModel().GetCharacterInfo(req.CharId)
	var oldHide *model.OutfitHideInfo
	if character != nil {
		preset := character.GetOutfitPreset(character.InUseOutfitPresetIndex)
		if preset.OutfitHideInfo != nil {
			oldHide = &model.OutfitHideInfo{HideOrn: preset.OutfitHideInfo.HideOrn, HideBraid: preset.OutfitHideInfo.HideBraid}
		}
	}
	rsp.Status = applyProfileOutfitHide(s, req.CharId, req.HideInfo)
	if rsp.Status != proto.StatusCode_StatusCode_Ok {
		return
	}
	rsp.HideInfo = character.GetOutfitPreset(character.InUseOutfitPresetIndex).OutfitHideInfo.OutfitHideInfo()
	saveProfileMutation(s, &rsp.Status)
	if rsp.Status != proto.StatusCode_StatusCode_Ok {
		character.GetOutfitPreset(character.InUseOutfitPresetIndex).OutfitHideInfo = oldHide
		return
	}
	g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdateFashion, req.CharId)
}
