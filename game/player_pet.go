package game

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/excel"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) GetPetReq(s *model.Player, msg *alg.GameMsg) {
	// req := msg.Body.(*proto.GetPetReq)
	rsp := &proto.GetPetRsp{
		Status:   proto.StatusCode_StatusCode_Ok,
		Pets:     make([]*proto.PetInstance, 0),
		TotalNum: 0,
		EndIndex: 0,
	}
	defer g.send(s, msg.PacketId, rsp)
	petMap := s.Item.GetItemPetInfoMap()
	for _, pet := range petMap {
		alg.AddLists(&rsp.Pets, pet.GetPbPetInstance())
	}
	rsp.TotalNum = uint32(len(petMap))
	rsp.EndIndex = uint32(len(petMap))
}

func (g *Game) ChangePet(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.ChangePetReq)
	rsp := &proto.ChangePetRsp{
		Status:        proto.StatusCode_StatusCode_Ok,
		PetInstanceId: req.PetInstanceId,
	}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdataPet)
	}()
	_, ok := s.GetItemModel().GetItemPetInfoMap()[req.PetInstanceId]
	if !ok {
		rsp.Status = proto.StatusCode_StatusCode_PetNotFound
		return
	}
	s.GetSceneModel().CurPetInstanceId = req.PetInstanceId
}

var (
	coreSceneAreaID = func(sceneID uint32) (uint32, bool) {
		conf, err := sceneConfigProvider()
		if err != nil || conf == nil {
			return 0, false
		}
		for _, scene := range conf.GetScene().GetDatas() {
			if scene != nil && uint32(scene.ID) == sceneID {
				return uint32(scene.AreaID), true
			}
		}
		return 0, false
	}
	corePetCapacity = func() (int, error) {
		conf := gdconf.GetGlobalConfigConfigure("PetBagOriginal")
		if conf == nil || len(conf.Values) != 1 {
			return 0, fmt.Errorf("missing PetBagOriginal")
		}
		value, err := strconv.Atoi(conf.Values[0])
		if err != nil || value <= 0 {
			return 0, fmt.Errorf("invalid PetBagOriginal")
		}
		return value, nil
	}
	corePetNameMaxLength = func() (int, error) {
		// 当前没有独立的PetName配置，复用权威玩家名称上限。
		conf := gdconf.GetGlobalConfigConfigure("Player_Name_Max_Length")
		if conf == nil || len(conf.Values) != 1 {
			return 0, fmt.Errorf("missing Player_Name_Max_Length")
		}
		value, err := strconv.Atoi(conf.Values[0])
		if err != nil || value <= 0 {
			return 0, fmt.Errorf("invalid Player_Name_Max_Length")
		}
		return value, nil
	}
)

func currentCoreSceneID(g *Game, s *model.Player) uint32 {
	if g == nil || s == nil {
		return 0
	}
	scenePlayer := g.getWordInfo().getScenePlayer(s)
	if scenePlayer == nil || scenePlayer.CurScene == nil {
		return 0
	}
	return scenePlayer.CurScene.GetSceneId()
}

func findPetConfigure(conf *excel.AllPetDatas, monsterID uint32) *excel.PetConfigure {
	if conf == nil {
		return nil
	}
	for _, pet := range conf.GetPet().GetDatas() {
		if pet != nil && uint32(pet.MonsterID) == monsterID {
			return pet
		}
	}
	return nil
}

func findCatcherConfigure(conf *excel.AllExpressionDatas, catcherID uint32) *excel.CatcherConfigure {
	if conf == nil {
		return nil
	}
	for _, catcher := range conf.GetCatcher().GetDatas() {
		if catcher != nil && (uint32(catcher.ID) == catcherID || uint32(catcher.NeedItemID) == catcherID) {
			return catcher
		}
	}
	return nil
}

func captureAttemptKey(s *model.Player, sceneID uint32, req *proto.PetCaptureReq) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d:%d:%d:%d:%d:%d:%d", s.UserId, sceneID,
		req.MonsterId, req.Hp, req.MaxHp, req.CatcherId, req.CatchAreaId)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func validPetName(name string, maxLength int, pets map[uint32]*model.ItemPetInfo, self uint32) proto.StatusCode {
	if !utf8.ValidString(name) || name == "" || name != strings.TrimSpace(name) || utf8.RuneCountInString(name) > maxLength {
		return proto.StatusCode_StatusCode_IllegalLen
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return proto.StatusCode_StatusCode_IllegalChar
		}
	}
	lower := strings.ToLower(name)
	for _, forbidden := range []string{
		"http://", "https://", "www.", "<script", "</script", "javascript:",
		"傻逼", "操你", "fuck", "shit",
	} {
		if strings.Contains(lower, forbidden) {
			return proto.StatusCode_StatusCode_BadWord
		}
	}
	for id, pet := range pets {
		if id != self && pet != nil && strings.EqualFold(pet.Name, name) {
			return proto.StatusCode_StatusCode_DuplicatedName
		}
	}
	return proto.StatusCode_StatusCode_Ok
}

func (g *Game) PetCapture(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PetCaptureReq)
	rsp := &proto.PetCaptureRsp{Status: proto.StatusCode_StatusCode_Ok, Items: make([]*proto.ItemDetail, 0)}
	defer g.send(s, msg.PacketId, rsp)
	if req.MonsterId == 0 || req.CatcherId == 0 || req.CatchAreaId == 0 || req.MaxHp == 0 || req.Hp > req.MaxHp {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	petConf, err := petConfigProvider()
	if err != nil || petConf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	expressionConf, err := expressionConfigProvider()
	if err != nil || expressionConf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	configuredPet := findPetConfigure(petConf, req.MonsterId)
	catcher := findCatcherConfigure(expressionConf, req.CatcherId)
	if configuredPet == nil || !configuredPet.DecisionCapture || catcher == nil || catcher.NeedItemID <= 0 || catcher.NeedItemCount <= 0 {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	sceneID := currentCoreSceneID(g, s)
	areaID, ok := coreSceneAreaID(sceneID)
	if sceneID == 0 || !ok || areaID != req.CatchAreaId {
		rsp.Status = proto.StatusCode_StatusCode_SceneNotUnlock
		return
	}
	area := s.GetSceneModel().GetSceneInfo(sceneID).GetAreaDatas()[req.CatchAreaId]
	if area == nil || area.AreaState != proto.AreaState_AreaState_Unlock {
		rsp.Status = proto.StatusCode_StatusCode_SceneNotUnlock
		return
	}
	key := captureAttemptKey(s, sceneID, req)
	if attempt, exists := s.GetPetModel().Attempts[key]; exists {
		rsp.Success = attempt.Success
		if attempt.Success {
			if pet := s.GetItemModel().GetItemPetInfoMap()[attempt.PetInstanceID]; pet != nil {
				rsp.Items = append(rsp.Items, pet.ItemDetail())
			}
		}
		return
	}
	capacity, err := corePetCapacity()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	if len(s.GetItemModel().GetItemPetInfoMap()) >= capacity {
		rsp.Status = proto.StatusCode_StatusCode_PetGridFull
		return
	}
	catcherItem := coreItemConfigure(uint32(catcher.NeedItemID))
	if catcherItem == nil || proto.EBagItemTag(catcherItem.NewBagItemTag) != proto.EBagItemTag_EBagItemTag_Catcher {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	petItem := coreItemConfigure(uint32(configuredPet.ID))
	if petItem == nil || proto.EBagItemTag(petItem.NewBagItemTag) != proto.EBagItemTag_EBagItemTag_Pet {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	// Parma[0]为权威表的万分数捕获率；缺失或越界时拒绝，不回退到猜测值。
	if len(configuredPet.Parma) == 0 || configuredPet.Parma[0] < 0 || configuredPet.Parma[0] > 10000 {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	rate := int(configuredPet.Parma[0])
	tx, status := spendCoreItems(s, map[uint32]int64{uint32(catcher.NeedItemID): int64(catcher.NeedItemCount)})
	if status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	if tx != nil {
		rsp.Items = append(rsp.Items, tx.PackNotice.Items...)
		g.send(s, 0, tx.PackNotice)
	}
	rsp.Success = coreRollIntN(10000) < rate
	attempt := model.PetCaptureAttempt{Success: rsp.Success}
	if rsp.Success {
		ctx := coreGrantItem(s, uint32(configuredPet.ID), 1)
		if ctx == nil || ctx.EBagItemTag == nil {
			_ = coreGrantItem(s, uint32(catcher.NeedItemID), int64(catcher.NeedItemCount))
			rsp.Status = proto.StatusCode_StatusCode_ItemGetFail
			return
		}
		pet, ok := ctx.EBagItemTag.(*model.ItemPetInfo)
		if !ok || pet == nil {
			_ = coreGrantItem(s, uint32(catcher.NeedItemID), int64(catcher.NeedItemCount))
			rsp.Status = proto.StatusCode_StatusCode_ItemGetFail
			return
		}
		// AddItemPet保留旧默认值；捕获实例只写协议和权威表能确定的字段。
		pet.Name = ""
		pet.Level = 1
		pet.NatureId = 0
		pet.SpecialColor = false
		pet.IntegrityId = 0
		pet.Integrity = make([]*model.PetIntegrityData, 0)
		pet.Star = 0
		pet.CatchSceneId = sceneID
		pet.CatchAreaId = req.CatchAreaId
		pet.CatchTime = coreNow().Unix()
		pet.Size = 0
		pet.Weight = 0
		pet.BadgeId = 0
		pet.BadgeIds = make([]uint32, 0)
		attempt.PetInstanceID = pet.InstanceId
		rsp.Items = append(rsp.Items, pet.ItemDetail())
	}
	s.GetPetModel().Attempts[key] = attempt
	saveCoreMutation(s, &rsp.Status)
}

func (g *Game) PetRename(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PetRenameReq)
	rsp := &proto.PetRenameRsp{Status: proto.StatusCode_StatusCode_Ok, PetInstanceId: req.PetInstanceId}
	defer g.send(s, msg.PacketId, rsp)
	pet := s.GetItemModel().GetItemPetInfoMap()[req.PetInstanceId]
	if pet == nil {
		rsp.Status = proto.StatusCode_StatusCode_PetNotFound
		return
	}
	maxLength, err := corePetNameMaxLength()
	if err != nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	if status := validPetName(req.Name, maxLength, s.GetItemModel().GetItemPetInfoMap(), req.PetInstanceId); status != proto.StatusCode_StatusCode_Ok {
		rsp.Status = status
		return
	}
	pet.Name = req.Name
	rsp.Name = req.Name
	saveCoreMutation(s, &rsp.Status)
}

func (g *Game) PetDecompose(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PetDecomposeReq)
	rsp := &proto.PetDecomposeRsp{Status: proto.StatusCode_StatusCode_Ok, Items: make([]*proto.ItemDetail, 0)}
	defer func() {
		g.send(s, msg.PacketId, rsp)
		g.SceneActionCharacterUpdate(s, proto.SceneActionType_SceneActionType_UpdataPet)
	}()
	if len(req.PetInstanceIds) == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	conf, err := petConfigProvider()
	if err != nil || conf == nil {
		rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
		return
	}
	petMap := s.GetItemModel().GetItemPetInfoMap()
	seen := make(map[uint32]struct{}, len(req.PetInstanceIds))
	ids := make([]uint32, 0, len(req.PetInstanceIds))
	configuredPets := make(map[uint32]*excel.PetConfigure, len(req.PetInstanceIds))
	for _, id := range req.PetInstanceIds {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		pet := petMap[id]
		if pet == nil {
			rsp.Status = proto.StatusCode_StatusCode_PetNotFound
			return
		}
		petConf := findPetConfigureByID(conf, pet.ItemId)
		if petConf == nil || len(petConf.ReleaseReward) == 0 {
			rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
			return
		}
		ids = append(ids, id)
		configuredPets[id] = petConf
	}
	rewardGroups := make(map[int32]*excel.ReleaseRewardConfigure)
	for _, group := range conf.GetReleaseReward().GetDatas() {
		if group != nil {
			rewardGroups[group.ID] = group
		}
	}
	rewards := make(map[uint32]int64)
	const maxCoreInt64 = int64(^uint64(0) >> 1)
	for _, id := range ids {
		for _, rewardID := range configuredPets[id].ReleaseReward {
			group := rewardGroups[rewardID]
			if group == nil || len(group.ReleaseRewardItem) == 0 {
				rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
				return
			}
			choice := group.ReleaseRewardItem[coreRollIntN(len(group.ReleaseRewardItem))]
			if choice == nil || choice.ReleaseRewardID <= 0 || choice.ReleaseRewardCount <= 0 {
				rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
				return
			}
			itemID := uint32(choice.ReleaseRewardID)
			count := int64(choice.ReleaseRewardCount)
			if count > maxCoreInt64-rewards[itemID] {
				rsp.Status = proto.StatusCode_StatusCode_CfgNotFound
				return
			}
			rewards[itemID] += count
		}
	}
	// 所有实例和产出校验通过后才卸下/删除，避免部分成功。
	carried := s.GetSceneModel().CurPetInstanceId
	unloaded := false
	for _, id := range ids {
		if carried == id {
			s.GetSceneModel().CurPetInstanceId = 0
			unloaded = true
			break
		}
	}
	items, status := grantCoreRewards(s, rewards)
	if status != proto.StatusCode_StatusCode_Ok {
		if unloaded {
			s.GetSceneModel().CurPetInstanceId = carried
		}
		rsp.Status = status
		return
	}
	for _, id := range ids {
		delete(petMap, id)
	}
	rsp.PetInstanceIds = ids
	rsp.Items = append(rsp.Items, items...)
	saveCoreMutation(s, &rsp.Status)
}

func findPetConfigureByID(conf *excel.AllPetDatas, petID uint32) *excel.PetConfigure {
	if conf == nil {
		return nil
	}
	for _, pet := range conf.GetPet().GetDatas() {
		if pet != nil && uint32(pet.ID) == petID {
			return pet
		}
	}
	return nil
}
