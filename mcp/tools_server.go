package mcp

import (
	"gucooing/lolo/config"
	"gucooing/lolo/db"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/pkg"
)

// registerServerTools 服务器咨询:运行状态与内容规模
func (s *Server) registerServerTools() {
	s.registerTool(&Tool{
		Name:        "server_status",
		Description: "查询服务器运行状态:版本/模式/在线与注册玩家数",
		InputSchema: schema(H{}),
		Handler:     s.serverStatus,
	})
	s.registerTool(&Tool{
		Name:        "server_content",
		Description: "查询服务器内容规模:角色/物品/武器/开放卡池数量",
		InputSchema: schema(H{}),
		Handler:     s.serverContent,
	})
}

func (s *Server) serverStatus(args Args) (any, error) {
	online := -1
	if s.hub != nil {
		online = s.hub.OnlinePlayerCount()
	}
	total, err := db.CountGameBasic()
	if err != nil {
		return nil, err
	}
	return H{
		"app":           pkg.AppName,
		"serverVersion": pkg.ServerVersion,
		"clientVersion": pkg.ClientVersion,
		"commit":        pkg.Commit,
		"mode":          config.GetMode(),
		"onlinePlayers": online,
		"totalPlayers":  total,
	}, nil
}

func (s *Server) serverContent(args Args) (any, error) {
	return H{
		"characters": len(gdconf.GetCharacterAllMap()),
		"items":      len(gdconf.GetAllItemConfigure()),
		"weapons":    len(gdconf.GetWeaponAllMap()),
		"openGachas": len(gdconf.GetOpenGachas()),
	}, nil
}
