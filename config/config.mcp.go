package config

// Mcp AI mcp/tools 服务器配置
type Mcp struct {
	Enable bool   `json:"Enable"` // 是否启用 mcp 服务
	Path   string `json:"Path"`   // mcp http 路由前缀,默认 /mcp
	Token  string `json:"Token"`  // 访问令牌,为空则不校验(建议对外暴露时设置)
}

var defaultMcp = &Mcp{
	Enable: false,
	Path:   "/v1/mcp",
	Token:  "",
}

func GetMcp() *Mcp {
	conf := GetConfig()
	if conf.Mcp == nil {
		conf.Mcp = defaultMcp
	}
	return conf.Mcp
}

func (x *Mcp) GetEnable() bool {
	return x.Enable
}

func (x *Mcp) GetPath() string {
	if x.Path == "" {
		return defaultMcp.Path
	}
	return x.Path
}

func (x *Mcp) GetToken() string {
	return x.Token
}
