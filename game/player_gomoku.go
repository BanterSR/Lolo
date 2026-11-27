package game

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

const gomokuBoardSize = 15

type gomokuRoom struct {
	id       int64
	players  [2]uint32
	ready    [2]bool
	board    []int32
	next     uint32
	state    proto.EGomokuChessBoardStatus
	watchers map[uint32]bool
	deadline int64
}

var (
	gomokuMu    sync.Mutex
	gomokuRooms = map[int64]*gomokuRoom{}
)

func newGomokuRoom(id int64) *gomokuRoom {
	return &gomokuRoom{
		id:       id,
		board:    make([]int32, gomokuBoardSize*gomokuBoardSize),
		state:    proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_None,
		watchers: make(map[uint32]bool),
	}
}

func (r *gomokuRoom) seatOf(userId uint32) int {
	for i, id := range r.players {
		if id != 0 && id == userId {
			return i
		}
	}
	return -1
}

func (r *gomokuRoom) start() {
	r.state = proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_Playing
	r.next = r.players[0]
	r.deadline = time.Now().Add(30 * time.Second).Unix()
}

// gomokuCheckWin 以最后落子点为中心，检查横、竖、两条斜线是否形成五连。
func gomokuCheckWin(board []int32, row, col int, stone int32) bool {
	if stone == 0 || row < 0 || col < 0 || row >= gomokuBoardSize || col >= gomokuBoardSize {
		return false
	}
	index := func(r, c int) int { return r*gomokuBoardSize + c }
	if index(row, col) >= len(board) || board[index(row, col)] != stone {
		return false
	}
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range directions {
		count := 1
		for _, sign := range []int{1, -1} {
			r, c := row+d[0]*sign, col+d[1]*sign
			for r >= 0 && r < gomokuBoardSize && c >= 0 && c < gomokuBoardSize &&
				index(r, c) < len(board) && board[index(r, c)] == stone {
				count++
				r += d[0] * sign
				c += d[1] * sign
			}
		}
		if count >= 5 {
			return true
		}
	}
	return false
}

func (g *Game) GomokuOnlinePlayerReady(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GomokuOnlinePlayerReadyReq)
	rsp := &proto.GomokuOnlinePlayerReadyRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)

	gomokuMu.Lock()
	defer gomokuMu.Unlock()
	room := gomokuRooms[req.Id]
	if room == nil {
		room = newGomokuRoom(req.Id)
		gomokuRooms[req.Id] = room
	}
	slot := room.seatOf(s.UserId)
	if slot < 0 {
		switch {
		case room.players[0] == 0:
			slot = 0
		case room.players[1] == 0:
			slot = 1
		default:
			rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
			return
		}
		room.players[slot] = s.UserId
	}
	room.ready[slot] = true
	if room.players[0] != 0 && room.players[1] != 0 && room.ready[0] && room.ready[1] {
		room.start()
	}
}

func (g *Game) GomokuOnlinePlayerStartPlay(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GomokuOnlinePlayerStartPlayReq)
	rsp := &proto.GomokuOnlinePlayerStartPlayRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)

	gomokuMu.Lock()
	defer gomokuMu.Unlock()
	room := gomokuRooms[req.Id]
	if room == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	slot := room.seatOf(s.UserId)
	if slot < 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	room.ready[slot] = true
	if room.players[0] != 0 && room.players[1] != 0 {
		room.start()
	}
}

func (g *Game) GomokuOnlinePlayerPlaceStone(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GomokuOnlinePlayerPlaceStoneReq)
	rsp := &proto.GomokuOnlinePlayerPlaceStoneRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)

	gomokuMu.Lock()
	defer gomokuMu.Unlock()
	room := gomokuRooms[req.Id]
	if room == nil || room.state != proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_Playing {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	slot := room.seatOf(s.UserId)
	if slot < 0 || room.players[slot] != room.next {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	if req.Row < 0 || req.Row >= gomokuBoardSize || req.Col < 0 || req.Col >= gomokuBoardSize {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	idx := int(req.Row)*gomokuBoardSize + int(req.Col)
	if room.board[idx] != 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	stone := int32(slot + 1)
	room.board[idx] = stone
	rsp.IsWin = gomokuCheckWin(room.board, int(req.Row), int(req.Col), stone)
	if rsp.IsWin {
		room.state = proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_Over
		room.next = 0
		room.deadline = 0
		return
	}
	room.next = room.players[1-slot]
	room.deadline = time.Now().Add(30 * time.Second).Unix()
	rsp.NextPlayerId = room.next
	rsp.TurnDeadline = room.deadline
}

func (g *Game) GomokuOnlinePlayerWatch(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GomokuOnlinePlayerWatchReq)
	rsp := &proto.GomokuOnlinePlayerWatchRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)

	gomokuMu.Lock()
	defer gomokuMu.Unlock()
	room := gomokuRooms[req.Id]
	if room == nil || room.seatOf(s.UserId) >= 0 {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	room.watchers[s.UserId] = true
}

func (g *Game) GomokuOnlinePlayerLeave(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GomokuOnlinePlayerLeaveReq)
	rsp := &proto.GomokuOnlinePlayerLeaveRsp{Status: proto.StatusCode_StatusCode_OK}
	defer g.send(s, msg.PacketId, rsp)

	gomokuMu.Lock()
	defer gomokuMu.Unlock()
	room := gomokuRooms[req.Id]
	if room == nil {
		return
	}
	if slot := room.seatOf(s.UserId); slot >= 0 {
		room.players[slot] = 0
		room.ready[slot] = false
		if room.state == proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_Playing {
			room.state = proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_Over
			room.next = 0
			room.deadline = 0
		}
	}
	delete(room.watchers, s.UserId)
	if room.players[0] == 0 && room.players[1] == 0 && len(room.watchers) == 0 {
		delete(gomokuRooms, req.Id)
	}
}

func (g *Game) GetGomokuOnlineChessboard(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GetGomokuOnlineChessboardReq)
	rsp := &proto.GetGomokuOnlineChessboardRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		BoardState:   proto.EGomokuChessBoardStatus_EGomokuChessBoardStatus_None,
		ChestPlayers: make([]*proto.PlayerBriefInfo, 0, 2),
		WatchPlayers: make([]*proto.PlayerBriefInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)

	gomokuMu.Lock()
	room := gomokuRooms[req.Id]
	if room == nil {
		gomokuMu.Unlock()
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.BoardState = room.state
	rsp.Board = append([]int32(nil), room.board...)
	rsp.NextPlayerId = room.next
	rsp.TurnDeadline = room.deadline
	players := room.players
	watchers := make([]uint32, 0, len(room.watchers))
	for id := range room.watchers {
		watchers = append(watchers, id)
	}
	gomokuMu.Unlock()

	for _, id := range players {
		if id == 0 {
			continue
		}
		if player := g.GetUser(id); player != nil {
			rsp.ChestPlayers = append(rsp.ChestPlayers, g.playerBriefInfo(player))
		}
	}
	for _, id := range watchers {
		if player := g.GetUser(id); player != nil {
			rsp.WatchPlayers = append(rsp.WatchPlayers, g.playerBriefInfo(player))
		}
	}
}

// GomokuSinglePlayerReward 只做棋盘胜负判定。本地资源中没有五子棋奖励表，
// 因此不发放虚构奖励，返回空 Items 由客户端自行结算表现。
func (g *Game) GomokuSinglePlayerReward(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GomokuSinglePlayerRewardReq)
	rsp := &proto.GomokuSinglePlayerRewardRsp{
		Status:       proto.StatusCode_StatusCode_OK,
		Items:        make([]*proto.ItemDetail, 0),
		ExtraRewards: make([]*proto.ItemDetail, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if len(req.Board) < gomokuBoardSize*gomokuBoardSize {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}
	rsp.IsSelfWin = gomokuCheckWin(req.Board, int(req.LastRow), int(req.LastCol), req.Self)
}

func (g *Game) CollapseFail(s *model.Player, msg *alg.GameMsg) {
	rsp := &proto.CollapseFailRsp{Status: proto.StatusCode_StatusCode_OK}
	g.send(s, msg.PacketId, rsp)
}

// GmCode 仅对本地管理员 UID 开放，支持 item <物品ID> <数量> 与 help 两条指令。
func (g *Game) GmCode(s *model.Player, msg *alg.GameMsg) {
	req := msg.Body.(*proto.GmCodeReq)
	rsp := &proto.GmCodeRsp{
		Status:      proto.StatusCode_StatusCode_OK,
		Result:      "",
		RewardInfos: make([]*proto.GmRewardInfo, 0),
	}
	defer g.send(s, msg.PacketId, rsp)
	if s.UserId != 1000000 {
		rsp.Status = proto.StatusCode_StatusCode_ACTION_NOT_UNLOCK
		rsp.Result = "gm code is only available for local uid 1000000"
		return
	}
	fields := strings.Fields(req.Param)
	if len(fields) == 0 {
		rsp.Result = "usage: item <id> <num>"
		return
	}
	switch strings.ToLower(fields[0]) {
	case "item":
		if len(fields) != 3 {
			rsp.Result = "usage: item <id> <num>"
			return
		}
		id, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			rsp.Result = "invalid item id"
			return
		}
		num, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || num <= 0 {
			rsp.Result = "invalid item num"
			return
		}
		ctx := s.AddAllTypeItem(uint32(id), num)
		if ctx == nil {
			rsp.Result = "item not found"
			return
		}
		rsp.RewardInfos = append(rsp.RewardInfos, &proto.GmRewardInfo{
			ItemId: uint32(id),
			Num:    num,
		})
		if detail := ctx.AddItemDetail(); detail != nil {
			g.send(s, 0, &proto.PackNotice{
				Status: proto.StatusCode_StatusCode_OK,
				Items:  []*proto.ItemDetail{detail},
			})
		}
		rsp.Result = "ok"
	default:
		rsp.Result = "unknown command: " + fields[0]
	}
}
