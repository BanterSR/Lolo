package model

import (
	"sort"

	"gucooing/lolo/protocol/proto"
)

// TestMails 是本地私服内置的邮件模板。附件使用游戏配置中真实存在的货币
// 物品（102 星石、110 限定券），领取时走正常背包发放流程。
var TestMails = map[uint32]*proto.MailBriefData{
	1: {
		MailId:      1,
		ModelId:     0,
		MailState:   proto.MailState_MailState_Unclaim,
		Sender:      "Lolo",
		SendTime:    1795000000,
		ContentType: proto.MailContentType_MailContentType_Text,
		Title:       "维护补偿",
		Content:     "亲爱的队长：\n\n感谢您在《开放空间》维护期间的耐心等待！\n本次维护补偿：星石*700、限定券*10。\n请在邮箱中领取。\n\n《开放空间》运营团队",
		OverdueDay:  3650,
		Reward:      make([]*proto.ItemDetail, 0),
		Items: []*proto.BaseItem{
			{ItemId: 102, Num: 700},
			{ItemId: 110, Num: 10},
		},
		IsQuestionnaire: false,
		CollectStatus:   false,
	},
	2: {
		MailId:          2,
		ModelId:         0,
		MailState:       proto.MailState_MailState_Unread,
		Sender:          "Lolo",
		SendTime:        1795000000,
		ContentType:     proto.MailContentType_MailContentType_Text,
		Title:           "私服使用说明",
		Content:         "欢迎来到本地私服。\n\n单机进度保存在本地数据库中；切换模式或更换电脑前，请先在启动器内备份存档。\n\n祝游戏愉快。",
		OverdueDay:      3650,
		Reward:          make([]*proto.ItemDetail, 0),
		Items:           make([]*proto.BaseItem, 0),
		IsQuestionnaire: false,
		CollectStatus:   false,
	},
}

// MailState 记录单个邮件的已读、已领取与收集状态。
type MailState struct {
	Read      bool `json:"read,omitempty"`
	Claimed   bool `json:"claimed,omitempty"`
	Collected bool `json:"collected,omitempty"`
}

// MailModel 是玩家邮件状态的持久化模型。
type MailModel struct {
	States  map[uint32]*MailState `json:"states,omitempty"`
	Deleted map[uint32]bool       `json:"deleted,omitempty"`
}

func (s *Player) GetMailModel() *MailModel {
	if s.Mail == nil {
		s.Mail = new(MailModel)
	}
	return s.Mail
}

func (m *MailModel) getStates() map[uint32]*MailState {
	if m.States == nil {
		m.States = make(map[uint32]*MailState)
	}
	return m.States
}

func (m *MailModel) getDeleted() map[uint32]bool {
	if m.Deleted == nil {
		m.Deleted = make(map[uint32]bool)
	}
	return m.Deleted
}

func (m *MailModel) GetState(mailId uint32) *MailState {
	states := m.getStates()
	state, ok := states[mailId]
	if !ok {
		state = new(MailState)
		states[mailId] = state
	}
	return state
}

func (m *MailModel) IsDeleted(mailId uint32) bool {
	return m.getDeleted()[mailId]
}

func (m *MailModel) Delete(mailId uint32) {
	m.getDeleted()[mailId] = true
}

// Briefs 返回按 MailId 升序排列的邮件列表。模板不会被修改，
// 每个玩家看到的已读/领取状态来自 MailModel。
func (m *MailModel) Briefs() []*proto.MailBriefData {
	ids := make([]uint32, 0, len(TestMails))
	for id := range TestMails {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	mails := make([]*proto.MailBriefData, 0, len(ids))
	for _, id := range ids {
		template := TestMails[id]
		if template == nil || m.IsDeleted(id) {
			continue
		}
		mails = append(mails, m.brief(template))
	}
	return mails
}

func (m *MailModel) brief(template *proto.MailBriefData) *proto.MailBriefData {
	mail := &proto.MailBriefData{
		MailId:          template.MailId,
		ModelId:         template.ModelId,
		MailState:       template.MailState,
		Sender:          template.Sender,
		SendTime:        template.SendTime,
		ContentType:     template.ContentType,
		Title:           template.Title,
		Content:         template.Content,
		OverdueDay:      template.OverdueDay,
		IsQuestionnaire: template.IsQuestionnaire,
		IsGemDuel:       template.IsGemDuel,
		CollectStatus:   template.CollectStatus,
	}
	mail.Reward = append([]*proto.ItemDetail(nil), template.Reward...)
	mail.Items = append([]*proto.BaseItem(nil), template.Items...)

	hasReward := len(mail.Reward) > 0 || len(mail.Items) > 0
	state := m.getStates()[template.MailId]
	if state == nil {
		if hasReward {
			mail.MailState = proto.MailState_MailState_Unclaim
		}
		return mail
	}
	mail.CollectStatus = mail.CollectStatus || state.Collected
	switch {
	case state.Claimed:
		mail.MailState = proto.MailState_MailState_Claimed
	case hasReward:
		mail.MailState = proto.MailState_MailState_Unclaim
	case state.Read:
		mail.MailState = proto.MailState_MailState_Readed
	default:
		mail.MailState = proto.MailState_MailState_Unread
	}
	return mail
}
