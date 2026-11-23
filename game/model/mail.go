package model

import "gucooing/lolo/protocol/proto"

var (
	TestMails = map[uint32]*proto.MailBriefData{
		1: {
			MailId:      1,
			ModelId:     0,
			MailState:   0,
			Sender:      "Lolo",
			SendTime:    1767834987,
			ContentType: proto.MailContentType_MailContentType_TEXT,
			Title:       "测试邮件标题",
			Content:     "亲爱的队长：\n\n感谢您在《开放空间》维护期间的耐心等待！\n这是本次维护更新的补偿【星石*700、自选招募券*1、限定券*10、映像[早上好]*1、映像[小小诗歌]*1、小型体力药剂*5、小型精力药水*5】，请及时领取。祝队长游戏愉快~\n\n【领取范围】4月20日23:59(UTC+8)前注册并登陆的队长可在邮箱领取。\n\n《开放空间》运营团队\n2025年4月17日\n",
			OverdueDay:  30,
			Reward:      make([]*proto.ItemDetail, 0),
			Items:       make([]*proto.BaseItem, 0),
			//Items: []*proto.BaseItem{
			//	{
			//		ItemId: 101001,
			//		Num:    1,
			//	},
			//},
			IsQuestionnaire: false,
			CollectStatus:   false,
		},
	}
)
