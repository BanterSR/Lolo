package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"

	"gucooing/lolo/config"
	"gucooing/lolo/pkg/log"
)

func newTestServer(t *testing.T) (*gin.Engine, *Server) {
	t.Helper()
	log.NewApp()
	mcpCfg := config.GetMcp()
	mcpCfg.Enable = true // 缺省配置关闭 mcp(安全),测试内显式开启
	mcpCfg.Path = "/mcp" // 与 do() 使用的请求路径保持一致
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	s := New(engine, nil)
	if s == nil {
		t.Fatal("mcp.New 返回 nil")
	}
	return engine, s
}

// do 发送一次 JSON-RPC 请求并返回解析后的响应
func do(t *testing.T, engine *gin.Engine, token, body string) (int, *rpcResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		return w.Code, nil
	}
	rsp := new(rpcResponse)
	if err := sonic.Unmarshal(w.Body.Bytes(), rsp); err != nil {
		t.Fatalf("响应解析失败: %s body=%s", err, w.Body.String())
	}
	return w.Code, rsp
}

func TestAuthRequired(t *testing.T) {
	engine, _ := newTestServer(t)
	code, _ := do(t, engine, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	if code != http.StatusUnauthorized {
		t.Fatalf("无令牌应返回 401,实际 %d", code)
	}
}

func TestInitialize(t *testing.T) {
	engine, s := newTestServer(t)
	_, rsp := do(t, engine, s.token, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	if rsp == nil || rsp.Error != nil {
		t.Fatalf("initialize 失败: %+v", rsp)
	}
	res, _ := sonic.Marshal(rsp.Result)
	if !strings.Contains(string(res), serverName) {
		t.Fatalf("initialize 结果缺少 serverInfo: %s", res)
	}
}

func TestToolsList(t *testing.T) {
	engine, s := newTestServer(t)
	_, rsp := do(t, engine, s.token, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if rsp == nil || rsp.Error != nil {
		t.Fatalf("tools/list 失败: %+v", rsp)
	}
	res, _ := sonic.Marshal(rsp.Result)
	for _, name := range []string{
		"game_item", "server_status", "player_basic", "player_gacha_records", "player_friends",
		"character_info", "character_skills", "character_voice", "character_list",
		"dungeon_info", "monster_info", "dungeon_list",
		"make_recipe", "weapon_info", "inscription_info", "weapon_list",
		"scene_info", "quest_info", "achievement_info", "shop_list", "item_list",
		"player_live",
	} {
		if !strings.Contains(string(res), name) {
			t.Fatalf("tools/list 缺少工具 %s: %s", name, res)
		}
	}
}

func TestUnknownMethod(t *testing.T) {
	engine, s := newTestServer(t)
	_, rsp := do(t, engine, s.token, `{"jsonrpc":"2.0","id":3,"method":"nope"}`)
	if rsp == nil || rsp.Error == nil || rsp.Error.Code != -32601 {
		t.Fatalf("未知方法应返回 -32601: %+v", rsp)
	}
}
