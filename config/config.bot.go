package config

type BotType string

const (
	BotTypeChat    BotType = "chat"
	BotTypeCommand BotType = "command"
)

type Bot struct {
	Type        BotType `json:"type"`
	NickName    string  `json:"nickname"`
	Sing        string  `json:"sing"`
	GuildName   string  `json:"guild_name"`
	Level       uint32  `json:"level"`
	Head        uint32  `json:"head"`
	Badge       uint32  `json:"badge"`
	AvatarFrame uint32  `json:"avatar_frame"`
	BaseUrl     string  `json:"base_url"`
	ApiKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	System      string  `json:"system"` // 系统提示词
}
