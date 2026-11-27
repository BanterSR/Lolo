package game

import (
	"io"
	"net"
	"testing"

	"github.com/gookit/slog"
	pb "google.golang.org/protobuf/proto"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/cmd"
	"gucooing/lolo/protocol/proto"
)

type fuzzConn struct{}

func (fuzzConn) Read() (*alg.GameMsg, error)    { return nil, io.EOF }
func (fuzzConn) SendBin(uint32, uint32, []byte) {}
func (fuzzConn) Send(uint32, pb.Message)        {}
func (fuzzConn) SetUID(uint32)                  {}
func (fuzzConn) GetSeqId() uint32               { return 0 }
func (fuzzConn) SetServerTag(string)            {}
func (fuzzConn) Close()                         {}
func (fuzzConn) LocalAddr() net.Addr            { return nil }
func (fuzzConn) RemoteAddr() net.Addr           { return nil }

func fuzzPlayerItemCount(player *model.Player) int64 {
	if player == nil || player.Item == nil {
		return 0
	}
	var count int64
	for _, item := range player.Item.ItemBaseInfo {
		if item != nil && item.Num > 0 {
			count += item.Num
		}
	}
	count += int64(len(player.Item.ItemWeaponMap))
	count += int64(len(player.Item.ItemArmorMap))
	count += int64(len(player.Item.ItemPosterMap))
	count += int64(len(player.Item.ItemFashionMap))
	count += int64(len(player.Item.ItemInscriptionMap))
	count += int64(len(player.Item.ItemPetInfoMap))
	return count
}

func TestEveryRoutedRequestContainsZeroValuePacket(t *testing.T) {
	previous := log.Game
	log.Game = slog.NewStdLogger()
	defer func() { log.Game = previous }()

	game := &Game{userMap: make(map[uint32]*model.Player)}
	game.newRouter()
	for msgID := range game.handlerFuncRouteMap {
		body := cmd.Get().GetProtoObjByCmdId(msgID)
		if body == nil {
			t.Fatalf("route %d has no registered protobuf request type", msgID)
		}
		player := &model.Player{
			UserId:    1,
			Online:    true,
			LoginUUID: "fuzz",
			Conn:      fuzzConn{},
			Item:      model.DefaultItemModel(),
			Character: model.DefaultCharacterModel(),
		}
		game.userMap[1] = player
		before := fuzzPlayerItemCount(player)
		game.routeHandle(nil, 1, "fuzz", &alg.GameMsg{
			PacketHead: &proto.PacketHead{MsgId: msgID},
			Body:       body,
		})
		if after := fuzzPlayerItemCount(player); after != before {
			t.Fatalf("zero-value request %s granted items: before=%d after=%d",
				cmd.Get().GetCmdNameByCmdId(msgID), before, after)
		}
		delete(game.userMap, 1)
	}
	if game.chatInfo != nil {
		for _, channel := range game.chatInfo.allChatChannel {
			channel.Close()
		}
	}
}

func FuzzEveryRoutedRequestRandomBody(f *testing.F) {
	previous := log.Game
	log.Game = slog.NewStdLogger()
	defer func() { log.Game = previous }()

	router := &Game{}
	router.newRouter()
	for msgID := range router.handlerFuncRouteMap {
		f.Add(msgID, []byte{})
	}
	f.Add(uint32(cmd.PlayerMainDataReq), []byte{0xff, 0x00, 0x80})
	f.Add(uint32(cmd.GachaReq), []byte{0x08, 0x01, 0x10, 0xff, 0xff, 0xff, 0xff, 0x0f})

	f.Fuzz(func(t *testing.T, msgID uint32, payload []byte) {
		if len(payload) > alg.PacketMaxLen {
			t.Skip()
		}
		handler, ok := router.handlerFuncRouteMap[msgID]
		if !ok {
			t.Skip()
		}
		body := cmd.Get().GetProtoObjByCmdId(msgID)
		if body == nil || pb.Unmarshal(payload, body) != nil {
			t.Skip()
		}
		player := &model.Player{
			UserId:    1,
			Online:    true,
			LoginUUID: "fuzz",
			Conn:      fuzzConn{},
			Item:      model.DefaultItemModel(),
			Character: model.DefaultCharacterModel(),
		}
		game := &Game{
			userMap:            map[uint32]*model.Player{1: player},
			handlerFuncRouteMap: map[uint32]HandlerFunc{msgID: handler},
		}
		before := fuzzPlayerItemCount(player)
		game.routeHandle(nil, 1, "fuzz", &alg.GameMsg{
			PacketHead: &proto.PacketHead{MsgId: msgID},
			Body:       body,
		})
		if after := fuzzPlayerItemCount(player); after != before {
			t.Fatalf("request %s granted items from random body: before=%d after=%d",
				cmd.Get().GetCmdNameByCmdId(msgID), before, after)
		}
		if game.chatInfo != nil {
			for _, channel := range game.chatInfo.allChatChannel {
				channel.Close()
			}
		}
	})
}
