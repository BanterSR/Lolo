package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"

	"gucooing/lolo/pkg"
)

const (
	protocolVersion = "2025-06-18"
	serverName      = "lolo-game-mcp"
)

// JSON-RPC 2.0 报文
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// tools/call 入参与出参
type callParams struct {
	Name      string `json:"name"`
	Arguments Args   `json:"arguments"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// handleGet mcp 使用 Streamable HTTP,不提供服务端主动推送流
func (s *Server) handleGet(c *gin.Context) {
	c.Status(http.StatusMethodNotAllowed)
}

// handleRPC 处理一次 JSON-RPC 请求
func (s *Server) handleRPC(c *gin.Context) {
	req := new(rpcRequest)
	if err := sonic.ConfigDefault.NewDecoder(c.Request.Body).Decode(req); err != nil {
		s.writeError(c, nil, -32700, "parse error")
		return
	}
	// 通知(无 id)不需要响应
	notification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		s.writeResult(c, req.ID, s.initializeResult(req.Params))
	case "notifications/initialized", "notifications/cancelled":
		c.Status(http.StatusAccepted)
	case "ping":
		s.writeResult(c, req.ID, gin.H{})
	case "tools/list":
		s.writeResult(c, req.ID, gin.H{"tools": s.listTools()})
	case "tools/call":
		s.handleCall(c, req)
	default:
		if notification {
			c.Status(http.StatusAccepted)
			return
		}
		s.writeError(c, req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) initializeResult(params json.RawMessage) gin.H {
	ver := protocolVersion
	if len(params) > 0 {
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if sonic.Unmarshal(params, &p) == nil && p.ProtocolVersion != "" {
			ver = p.ProtocolVersion
		}
	}
	return gin.H{
		"protocolVersion": ver,
		"capabilities":    gin.H{"tools": gin.H{"listChanged": false}},
		"serverInfo":      gin.H{"name": serverName, "version": pkg.ServerVersion},
	}
}

// listTools 输出工具列表(按注册顺序)
func (s *Server) listTools() []gin.H {
	list := make([]gin.H, 0, len(s.toolNames))
	for _, name := range s.toolNames {
		t := s.toolMap[name]
		list = append(list, gin.H{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	return list
}

func (s *Server) handleCall(c *gin.Context, req *rpcRequest) {
	p := new(callParams)
	if len(req.Params) > 0 {
		if err := sonic.Unmarshal(req.Params, p); err != nil {
			s.writeError(c, req.ID, -32602, "invalid params")
			return
		}
	}
	t, ok := s.toolMap[p.Name]
	if !ok {
		s.writeError(c, req.ID, -32602, "unknown tool: "+p.Name)
		return
	}
	res, err := t.Handler(p.Arguments)
	if err != nil {
		s.writeResult(c, req.ID, &callResult{
			Content: []toolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
		return
	}
	s.writeResult(c, req.ID, &callResult{
		Content: []toolContent{{Type: "text", Text: toText(res)}},
	})
}

func (s *Server) writeResult(c *gin.Context, id json.RawMessage, result any) {
	s.write(c, &rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) writeError(c *gin.Context, id json.RawMessage, code int, msg string) {
	s.write(c, &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *Server) write(c *gin.Context, rsp *rpcResponse) {
	bin, err := sonic.Marshal(rsp)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", bin)
}
