package mcp

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gucooing/lolo/config"
	"gucooing/lolo/pkg/log"
)

// Hub game 侧只读状态接口,用于查询在线状态与实时快照
type Hub interface {
	OnlinePlayerCount() int                              // 当前内存中加载的玩家数
	IsPlayerOnline(userId uint32) bool                   // 玩家是否在线(数据可能比数据库快照更新)
	LivePlayerInfo(userId uint32) (map[string]any, bool) // 在线玩家实时快照(经主线程读取,避免竞争)
}

// defaultServer 最近一次创建的 mcp 服务器,供 command 等包获取工具(无需改动其构造函数)
var defaultServer *Server

// Default 返回 mcp 服务器单例,未启用时为 nil
func Default() *Server {
	return defaultServer
}

// Server AI mcp / tools 服务器
type Server struct {
	hub       Hub
	token     string
	toolMap   map[string]*Tool
	toolNames []string // 保持注册顺序
}

// New 创建并挂载 mcp 服务器到 gin 路由
func New(router *gin.Engine, hub Hub) *Server {
	cfg := config.GetMcp()
	if !cfg.GetEnable() {
		log.App.Info("mcp:未启用")
		return nil
	}
	s := &Server{
		hub:     hub,
		token:   cfg.GetToken(),
		toolMap: make(map[string]*Tool),
	}
	if s.token == "" {
		s.token = uuid.NewString()
		log.App.Warnf("mcp:未配置 Token,已自动生成临时令牌(重启会变化,建议在配置 Mcp.Token 中固定): %s", s.token)
	}
	s.registerTools()
	defaultServer = s

	path := cfg.GetPath()
	group := router.Group(path, s.auth)
	group.POST("", s.handleRPC)
	group.POST("/", s.handleRPC)
	group.GET("", s.handleGet)
	group.GET("/", s.handleGet)

	log.App.Infof("mcp:已挂载 %s 已注册工具 %d 个", path, len(s.toolNames))
	return s
}

func (s *Server) Close() {}

// registerTool 注册一个工具
func (s *Server) registerTool(t *Tool) {
	if _, ok := s.toolMap[t.Name]; ok {
		log.App.Warnf("mcp:重复注册工具 %s", t.Name)
		return
	}
	s.toolMap[t.Name] = t
	s.toolNames = append(s.toolNames, t.Name)
}

// registerTools 注册全部工具
func (s *Server) registerTools() {
	s.registerGameTools()      // 游戏咨询(资源)
	s.registerCharacterTools() // 角色攻略(信息/技能/养成/语音)
	s.registerDungeonTools()   // 副本攻略(副本/怪物)
	s.registerCultivateTools() // 养成推荐(制作/装备)
	s.registerWorldTools()     // 世界信息(场景/任务/剧情/成就/商店)
	s.registerServerTools()    // 服务器咨询
	s.registerPlayerTools()    // 玩家查询(离线数据)
}

// auth 令牌鉴权,支持 Authorization: Bearer / X-Api-Key / ?token=
func (s *Server) auth(c *gin.Context) {
	if s.token == "" {
		c.Next()
		return
	}
	token := ""
	if h := c.GetHeader("Authorization"); h != "" {
		token = strings.TrimSpace(strings.TrimPrefix(h, "Bearer"))
	}
	if token == "" {
		token = c.GetHeader("X-Api-Key")
	}
	if token == "" {
		token = c.Query("token")
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Next()
}
