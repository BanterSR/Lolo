package game

import (
	"github.com/bytedance/sonic"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

// 面向单个玩家
func (g *Game) SceneDataNotice(s *model.Player, channelInfo *ChannelInfo) {
	notice := &proto.SceneDataNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Data:   nil,
	}
	defer g.send(s, cmd.SceneDataNotice, 0, notice)
	data := channelInfo.GetPbSceneData()
	if data == nil {
		str, _ := sonic.MarshalString(channelInfo)
		log.Game.Errorf("玩家场景信息异常:%s", str)
		notice.Status = proto.StatusCode_StatusCode_CANT_JOIN_PLAYER_CURRENT_SCENE_CHANNEL
		return
	}
	notice.Data = data
}

// 面向整个玩家群体
func (g *Game) ServerSceneSyncDataNotice(channelInfo *ChannelInfo, scenePlayer *ScenePlayer) {
	notice := &proto.ServerSceneSyncDataNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Data:   make([]*proto.ServerSceneSyncData, 0),
	}
	defer channelInfo.sendAllPlayer(cmd.ServerSceneSyncDataNotice, 0, notice)
	notice.Data = append(notice.Data, &proto.ServerSceneSyncData{
		PlayerId: scenePlayer.UserId,
		ServerData: []*proto.SceneServerData{
			{
				ActionType: proto.SceneActionType_SceneActionType_ENTER,
				Player:     channelInfo.GetPbScenePlayer(scenePlayer),
				TodTime:    0,
			},
		},
	})
}

func (g *Game) PlayerSceneRecord(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.PlayerSceneRecordReq)
	rsp := &proto.PlayerSceneRecordRsp{
		Status: proto.StatusCode_StatusCode_OK,
	}
	defer g.send(s, cmd.PlayerSceneRecordRsp, msg.PacketId, rsp)

	g.send(s, cmd.PlayerSceneSyncDataNotice, 0, &proto.PlayerSceneSyncDataNotice{
		Status: proto.StatusCode_StatusCode_OK,
		Data: []*proto.SceneSyncData{
			{
				PlayerId: s.UserId,
				Data:     []*proto.PlayerRecorderData{req.Data},
			},
		},
	})
}

func (g *Game) SceneProcessList(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.SceneProcessListRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		SceneProcessList: make([]*proto.SceneProcess, 0),
	}
	defer g.send(s, cmd.SceneProcessListRsp, msg.PacketId, rsp)
	rsp.SceneProcessList = append(rsp.SceneProcessList, &proto.SceneProcess{
		SceneId: 9999,
		Process: 1,
	})
	rsp.SceneProcessList = append(rsp.SceneProcessList, &proto.SceneProcess{
		SceneId: 1,
		Process: 1,
	})
	rsp.SceneProcessList = append(rsp.SceneProcessList, &proto.SceneProcess{
		SceneId: 100,
		Process: 1,
	})
}
