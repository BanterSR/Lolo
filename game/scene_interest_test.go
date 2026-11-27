package game

import (
	"net"
	"testing"

	pb "google.golang.org/protobuf/proto"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

type sceneInterestConn struct {
	messages []pb.Message
}

func (c *sceneInterestConn) Read() (*alg.GameMsg, error)    { return nil, nil }
func (c *sceneInterestConn) SendBin(uint32, uint32, []byte) {}
func (c *sceneInterestConn) Send(_ uint32, message pb.Message) {
	c.messages = append(c.messages, message)
}
func (c *sceneInterestConn) SetUID(uint32)        {}
func (c *sceneInterestConn) GetSeqId() uint32     { return 0 }
func (c *sceneInterestConn) SetServerTag(string)  {}
func (c *sceneInterestConn) Close()               {}
func (c *sceneInterestConn) LocalAddr() net.Addr  { return nil }
func (c *sceneInterestConn) RemoteAddr() net.Addr { return nil }

func testScenePlayer(id uint32, x, y int32, decimalPlaces uint32) *ScenePlayer {
	player := &model.Player{UserId: id}
	return &ScenePlayer{
		Player: player,
		CurScene: &model.ScenePlayerInfo{
			Player:  player,
			SceneId: 1,
			Pos:     &proto.Vector3{X: x, Y: y, DecimalPlaces: decimalPlaces},
		},
	}
}

func TestNearbyScenePlayersFiltersDistanceAndSelf(t *testing.T) {
	origin := testScenePlayer(1, 0, 0, 0)
	near := testScenePlayer(2, 10, 0, 0)
	far := testScenePlayer(3, 1000, 0, 0)
	players := map[uint32]*ScenePlayer{
		origin.UserId: origin,
		near.UserId:   near,
		far.UserId:    far,
	}

	got := nearbyScenePlayers(origin, newSceneInterestGrid(players))
	if len(got) != 2 || got[0].UserId != 1 || got[1].UserId != 2 {
		t.Fatalf("nearby players = %+v, want self and nearby player", got)
	}
}

func TestNearbyScenePlayersUsesDecimalPlaces(t *testing.T) {
	origin := testScenePlayer(1, 0, 0, 2)
	near := testScenePlayer(2, 1000, 0, 2)
	far := testScenePlayer(3, 100000, 0, 2)
	players := map[uint32]*ScenePlayer{
		origin.UserId: origin,
		near.UserId:   near,
		far.UserId:    far,
	}

	got := nearbyScenePlayers(origin, newSceneInterestGrid(players))
	if len(got) != 2 || got[1].UserId != 2 {
		t.Fatalf("decimal-scaled nearby players = %+v, want self and player 10 units away", got)
	}
}

func TestNearbyScenePlayersCapsDenseInterestSet(t *testing.T) {
	origin := testScenePlayer(1, 0, 0, 0)
	players := map[uint32]*ScenePlayer{origin.UserId: origin}
	for i := 2; i <= 100; i++ {
		player := testScenePlayer(uint32(i), 0, 0, 0)
		players[player.UserId] = player
	}

	got := nearbyScenePlayers(origin, newSceneInterestGrid(players))
	if len(got) != sceneInterestMaxPlayers {
		t.Fatalf("dense interest set has %d players, want %d", len(got), sceneInterestMaxPlayers)
	}
	for i, player := range got {
		if player.UserId != uint32(i+1) {
			t.Fatalf("dense interest set is not deterministic at %d: %d", i, player.UserId)
		}
	}
}

func TestNearbyScenePlayersFallsBackWhenPositionMissing(t *testing.T) {
	origin := testScenePlayer(1, 0, 0, 0)
	origin.CurScene.SetPos(nil)
	other := testScenePlayer(2, 5000, 5000, 0)
	players := map[uint32]*ScenePlayer{origin.UserId: origin, other.UserId: other}

	got := nearbyScenePlayers(origin, newSceneInterestGrid(players))
	if len(got) != 1 || got[0].UserId != other.UserId {
		t.Fatalf("missing-position fallback returned %v, want the positioned player", got)
	}
}

func TestSendSceneSyncDataUsesLatestNearbyRecords(t *testing.T) {
	channel := &ChannelInfo{
		allPlayer:       make(map[uint32]*ScenePlayer),
		sceneSyncLatest: make(map[uint32]*proto.SceneSyncData),
	}
	for i := uint32(1); i <= sceneInterestChannelLimit+1; i++ {
		x := int32(10000)
		if i <= 2 {
			x = int32(i - 1)
		}
		player := testScenePlayer(i, x, 0, 0)
		conn := new(sceneInterestConn)
		player.Conn = conn
		channel.allPlayer[player.UserId] = player
	}
	channel.playerNum = int64(len(channel.allPlayer))
	channel.sceneSyncLatest[1] = &proto.SceneSyncData{PlayerId: 1}
	channel.sceneSyncLatest[2] = &proto.SceneSyncData{PlayerId: 2}
	channel.sceneSyncLatest[3] = &proto.SceneSyncData{PlayerId: 3}

	channel.sendSceneSyncData()

	near := channel.allPlayer[1].Conn.(*sceneInterestConn)
	if len(near.messages) != 1 {
		t.Fatalf("near player received %d messages, want 1", len(near.messages))
	}
	notice := near.messages[0].(*proto.PlayerSceneSyncDataNotice)
	if len(notice.Data) != 2 || notice.Data[0].PlayerId != 1 || notice.Data[1].PlayerId != 2 {
		t.Fatalf("near player sync data = %+v, want players 1 and 2", notice.Data)
	}
	far := channel.allPlayer[3].Conn.(*sceneInterestConn)
	if len(far.messages) != 1 {
		t.Fatalf("far player received %d messages, want 1", len(far.messages))
	}
	farNotice := far.messages[0].(*proto.PlayerSceneSyncDataNotice)
	if len(farNotice.Data) != 1 || farNotice.Data[0].PlayerId != 3 {
		t.Fatalf("far player sync data = %+v, want only player 3", farNotice.Data)
	}
}
