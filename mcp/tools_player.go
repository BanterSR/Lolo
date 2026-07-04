package mcp

import (
	"fmt"

	"github.com/bytedance/sonic"

	"gucooing/lolo/db"
	"gucooing/lolo/game/model"
	"gucooing/lolo/gdconf"
)

// registerPlayerTools 玩家查询:仅读取数据库中的离线数据,不触碰在线内存(避免并发竞争)
func (s *Server) registerPlayerTools() {
	s.registerTool(&Tool{
		Name:        "player_basic",
		Description: "查询玩家基础信息(昵称/等级/经验/头像/签名/在线状态)",
		InputSchema: schema(H{"userId": prop("integer", "玩家id")}, "userId"),
		Handler:     s.playerBasic,
	})
	s.registerTool(&Tool{
		Name:        "player_account",
		Description: "查询玩家账号信息(渠道/设备/封禁状态)",
		InputSchema: schema(H{"userId": prop("integer", "玩家id")}, "userId"),
		Handler:     s.playerAccount,
	})
	s.registerTool(&Tool{
		Name:        "player_live",
		Description: "查询在线玩家实时快照(仅在线可用):当前场景id/坐标位置/房间号/世界等级/当前队伍/角色与物品数量/最后活跃。经游戏主线程读取,避免并发竞争;离线请改用其它 player_* 离线查询",
		InputSchema: schema(H{"userId": prop("integer", "玩家id")}, "userId"),
		Handler:     s.playerLive,
	})
	s.registerTool(&Tool{
		Name:        "player_characters",
		Description: "查询玩家已拥有角色列表(id/名称/等级/星级)",
		InputSchema: schema(H{
			"userId": prop("integer", "玩家id"),
			"lang":   langProp(),
		}, "userId"),
		Handler: s.playerCharacters,
	})
	s.registerTool(&Tool{
		Name:        "player_items",
		Description: "查询玩家背包基础物品(可按背包分类过滤),返回物品id/名称/数量",
		InputSchema: schema(H{
			"userId": prop("integer", "玩家id"),
			"bagTag": prop("integer", "背包分类过滤,可选;不传则返回全部"),
			"limit":  prop("integer", "返回数量上限,缺省50,最大200"),
			"lang":   langProp(),
		}, "userId"),
		Handler: s.playerItems,
	})
	s.registerTool(&Tool{
		Name:        "player_team",
		Description: "查询玩家当前队伍(队长及三名角色)",
		InputSchema: schema(H{
			"userId": prop("integer", "玩家id"),
			"lang":   langProp(),
		}, "userId"),
		Handler: s.playerTeam,
	})
	s.registerTool(&Tool{
		Name:        "player_search",
		Description: "按昵称模糊搜索玩家,返回玩家id/昵称/等级",
		InputSchema: schema(H{
			"keyword": prop("string", "昵称关键字"),
			"limit":   prop("integer", "返回数量上限,缺省20,最大100"),
		}, "keyword"),
		Handler: s.playerSearch,
	})
	s.registerTool(&Tool{
		Name:        "player_gacha_records",
		Description: "查询玩家某卡池的抽卡记录(分页,每页5条),返回物品id/名称/时间",
		InputSchema: schema(H{
			"userId":  prop("integer", "玩家id"),
			"gachaId": prop("integer", "卡池id(可由 game_gacha_pools 获取)"),
			"page":    prop("integer", "页码,从1开始,缺省1"),
			"lang":    langProp(),
		}, "userId", "gachaId"),
		Handler: s.playerGachaRecords,
	})
	s.registerTool(&Tool{
		Name:        "player_friends",
		Description: "查询玩家好友关系,type 可选 friend(好友)/apply_received(收到申请)/apply_sent(发出申请)/black(黑名单)",
		InputSchema: schema(H{
			"userId": prop("integer", "玩家id"),
			"type":   prop("string", "friend/apply_received/apply_sent/black,缺省 friend"),
		}, "userId"),
		Handler: s.playerFriends,
	})
}

func (s *Server) playerBasic(args Args) (any, error) {
	userId := args.Uint32("userId")
	basic, ok := db.GetGameBasic(userId)
	if !ok {
		return nil, fmt.Errorf("玩家不存在: %d", userId)
	}
	return H{
		"userId":        userId,
		"nickName":      basic.NickName,
		"level":         basic.Level,
		"exp":           basic.Exp,
		"head":          basic.Head,
		"sign":          basic.Sign,
		"sex":           basic.Sex,
		"characterId":   basic.CharacterId,
		"birthday":      basic.Birthday,
		"lastLoginTime": basic.LastLoginTime,
		"createTime":    basic.CreatedAt.Unix(),
		"online":        s.isOnline(userId),
	}, nil
}

func (s *Server) playerAccount(args Args) (any, error) {
	userId := args.Uint32("userId")
	user, err := db.GetOFUserByUserId(userId)
	if err != nil {
		return nil, fmt.Errorf("账号不存在: %d", userId)
	}
	return H{
		"userId":    user.UserId,
		"sdkUid":    user.SdkUid,
		"channelId": user.ChannelId,
		"deviceId":  user.DeviceId,
		"ban":       user.Ban,
		"banTime":   user.BanTime.Unix(),
		"banText":   user.BanText,
	}, nil
}

func (s *Server) playerLive(args Args) (any, error) {
	userId := args.Uint32("userId")
	if s.hub == nil {
		return nil, fmt.Errorf("实时查询不可用(未接入 game)")
	}
	info, ok := s.hub.LivePlayerInfo(userId)
	if !ok {
		return nil, fmt.Errorf("玩家不在线或查询超时: %d", userId)
	}
	return info, nil
}

func (s *Server) playerCharacters(args Args) (any, error) {
	lang := args.Lang()
	userId := args.Uint32("userId")
	player, err := s.loadPlayer(userId)
	if err != nil {
		return nil, err
	}
	list := make([]H, 0)
	if player.Character != nil {
		for id, c := range player.Character.CharacterMap {
			list = append(list, H{
				"id":         id,
				"name":       s.characterName(lang, id),
				"level":      c.Level,
				"star":       c.Star,
				"breakLevel": c.BreakLevel,
			})
		}
	}
	return H{"userId": userId, "count": len(list), "characters": list}, nil
}

func (s *Server) playerItems(args Args) (any, error) {
	lang := args.Lang()
	userId := args.Uint32("userId")
	limit := args.Int("limit")
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	filterTag := args.Int("bagTag")
	player, err := s.loadPlayer(userId)
	if err != nil {
		return nil, err
	}
	list := make([]H, 0, limit)
	truncated := false
	if player.Item != nil {
		for _, item := range player.Item.ItemBaseInfo {
			if filterTag != 0 && int(item.ItemType) != filterTag {
				continue
			}
			if len(list) >= limit {
				truncated = true
				break
			}
			name := ""
			if conf := gdconf.GetItemConfigure(item.ItemId); conf != nil {
				name = itemText(lang, conf.GetTextID(), 0)
			}
			list = append(list, H{
				"itemId": item.ItemId,
				"name":   name,
				"num":    item.Num,
				"tag":    item.ItemType,
			})
		}
	}
	return H{"userId": userId, "count": len(list), "truncated": truncated, "items": list}, nil
}

func (s *Server) playerTeam(args Args) (any, error) {
	lang := args.Lang()
	userId := args.Uint32("userId")
	player, err := s.loadPlayer(userId)
	if err != nil {
		return nil, err
	}
	if player.Team == nil || player.Team.TeamInfo == nil {
		return H{"userId": userId, "team": nil}, nil
	}
	t := player.Team.TeamInfo
	member := func(id uint32) H {
		return H{"id": id, "name": s.characterName(lang, id)}
	}
	return H{
		"userId": userId,
		"team":   []H{member(t.Char1), member(t.Char2), member(t.Char3)},
	}, nil
}

func (s *Server) playerSearch(args Args) (any, error) {
	keyword := args.String("keyword")
	if keyword == "" {
		return nil, fmt.Errorf("keyword 不能为空")
	}
	limit := args.Int("limit")
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	basics, err := db.SearchGameBasicByNickName(keyword, limit)
	if err != nil {
		return nil, err
	}
	list := make([]H, 0, len(basics))
	for _, b := range basics {
		list = append(list, H{"userId": b.UserId, "nickName": b.NickName, "level": b.Level})
	}
	return H{"count": len(list), "players": list}, nil
}

func (s *Server) playerGachaRecords(args Args) (any, error) {
	lang := args.Lang()
	userId := args.Uint32("userId")
	gachaId := args.Uint32("gachaId")
	page := args.Uint32("page")
	if page == 0 {
		page = 1
	}
	records, totalPage, err := db.GetGachaRecords(userId, gachaId, page)
	if err != nil {
		return nil, err
	}
	list := make([]H, 0, len(records))
	for _, r := range records {
		name := ""
		if conf := gdconf.GetItemConfigure(r.ItemId); conf != nil {
			name = itemText(lang, conf.GetTextID(), 0)
		}
		list = append(list, H{
			"itemId":    r.ItemId,
			"name":      name,
			"gachaTime": r.GachaTime,
		})
	}
	return H{
		"userId":    userId,
		"gachaId":   gachaId,
		"page":      page,
		"totalPage": totalPage,
		"count":     len(list),
		"records":   list,
	}, nil
}

func (s *Server) playerFriends(args Args) (any, error) {
	userId := args.Uint32("userId")
	typ := args.String("type")
	if typ == "" {
		typ = "friend"
	}
	if typ == "black" {
		blacks, err := db.GetAllFriendBlack(userId)
		if err != nil {
			return nil, err
		}
		list := make([]H, 0, len(blacks))
		for _, b := range blacks {
			list = append(list, H{"blackId": b.BlackId, "nickName": s.nickName(b.BlackId)})
		}
		return H{"userId": userId, "type": typ, "count": len(list), "friends": list}, nil
	}
	var friends []*db.OFFriend
	var err error
	switch typ {
	case "friend":
		friends, err = db.GetAllFiend(userId)
	case "apply_received":
		friends, err = db.GetAllFriendApply(userId)
	case "apply_sent":
		friends, err = db.GetAllFriendSenderApply(userId)
	default:
		return nil, fmt.Errorf("未知的 type: %s", typ)
	}
	if err != nil {
		return nil, err
	}
	list := make([]H, 0, len(friends))
	for _, f := range friends {
		list = append(list, H{
			"friendId": f.FriendId,
			"nickName": s.nickName(f.FriendId),
			"alias":    f.Alias,
			"intimacy": f.FriendIntimacy,
			"tag":      f.FriendTag,
		})
	}
	return H{"userId": userId, "type": typ, "count": len(list), "friends": list}, nil
}

// nickName 玩家id -> 昵称
func (s *Server) nickName(userId uint32) string {
	if basic, ok := db.GetGameBasic(userId); ok {
		return basic.NickName
	}
	return ""
}

// loadPlayer 从数据库读取玩家离线快照(不触碰在线内存)
func (s *Server) loadPlayer(userId uint32) (*model.Player, error) {
	game, err := db.GetOFGameByUserId(userId)
	if err != nil {
		return nil, fmt.Errorf("玩家不存在: %d", userId)
	}
	player := &model.Player{UserId: userId}
	if len(game.BinData) != 0 {
		if err = sonic.Unmarshal(game.BinData, player); err != nil {
			return nil, fmt.Errorf("玩家数据解析失败: %s", err.Error())
		}
	}
	return player, nil
}

func (s *Server) isOnline(userId uint32) bool {
	return s.hub != nil && s.hub.IsPlayerOnline(userId)
}
