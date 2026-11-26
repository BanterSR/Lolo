package game

import (
	"strconv"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) NpcTalk(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.NpcTalkReq)
	rsp := &proto.NpcTalkRsp{
		Status: proto.StatusCode_StatusCode_Ok,
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil || req.Id == 0 {
		rsp.Status = proto.StatusCode_StatusCode_BadReq
		return
	}
	s.GetArchive().SetArchiveMap(
		"npcTalk:"+strconv.Itoa(int(req.Id)),
		strconv.FormatInt(time.Now().Unix(), 10),
	)
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v NPC对话落库失败:%v", s.UserId, err)
	}
}

func (g *Game) Tutorial(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.TutorialRsp{Status: proto.StatusCode_StatusCode_Ok}
	defer g.send(s, msg.PacketId, rsp)
	for questId := range s.GetQuestModel().GetQuestMap() {
		s.GetQuestModel().Complete(questId)
	}
	if err := s.SavePlayerNow(); err != nil {
		rsp.Status = proto.StatusCode_StatusCode_PlayerNotFound
		log.Game.Errorf("UserId:%v 教程进度落库失败:%s", s.UserId, err.Error())
	}
}
