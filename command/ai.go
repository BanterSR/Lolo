package command

import (
	"context"
	"fmt"
	"gucooing/lolo/mcp"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

	"gucooing/lolo/config"
	"gucooing/lolo/game"
	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/cache"
)

const aiMaxToolRounds = 8

type AiBot struct {
	*Command
	*game.BotInfo
	session *cache.Cache[uint32, *aiMessage]
	openAi  *openai.Client
	cfg     *config.Bot
	uuid    string
}

type aiMessage struct {
	sync     sync.Mutex
	messages []*messageInfo
}

type messageInfo struct {
	userId  uint32
	time    time.Time
	message openai.ChatCompletionMessageParamUnion
}

func (m *messageInfo) GetUserId() uint32 {
	return m.userId
}

func (m *messageInfo) GetTime() time.Time {
	return m.time
}

func (m *messageInfo) GetText() string {
	if m.message.OfAssistant != nil {
		return m.message.OfAssistant.Content.OfString.String()
	} else if m.message.OfUser != nil {
		return m.message.OfUser.Content.OfString.String()
	}
	return ""
}

func (a *AiBot) UUID() string {
	return a.uuid
}

func (a *AiBot) GetBotInfo() *game.BotInfo {
	return a.BotInfo
}

func (a *AiBot) Handle(s *model.Player, text string) {
	go func() {
		ctx := context.Background()
		sessionInfo, ok := a.session.Get(s.UserId)
		if !ok {
			sessionInfo = &aiMessage{
				sync: sync.Mutex{},
				messages: []*messageInfo{
					{userId: 0, time: time.Now(),
						message: openai.SystemMessage(fmt.Sprintf("你现在对话的玩家名称是:%s,%s", s.NickName, a.cfg.System))},
				},
			}
			a.session.Set(s.UserId, sessionInfo)
		}
		sessionInfo.sync.Lock()
		defer sessionInfo.sync.Unlock()
		if len(sessionInfo.messages) > 500 {
			sessionInfo.messages = append(sessionInfo.messages[:1], sessionInfo.messages[51:]...)
		}
		sessionInfo.messages = append(sessionInfo.messages,
			&messageInfo{userId: s.UserId, time: time.Now(), message: openai.UserMessage(text)})

		messages := make([]openai.ChatCompletionMessageParamUnion, len(sessionInfo.messages))
		for i, message := range sessionInfo.messages {
			messages[i] = message.message
		}

		params := openai.ChatCompletionNewParams{
			Model:    a.cfg.Model,
			Messages: messages,
			WebSearchOptions: openai.ChatCompletionNewParamsWebSearchOptions{
				SearchContextSize: "high",
			},
			ReasoningEffort:      shared.ReasoningEffortXhigh,
			PromptCacheKey:       openai.String(fmt.Sprintf("lolo-ai-chat-%s-%d", a.uuid, s.UserId)),
			PromptCacheRetention: openai.ChatCompletionNewParamsPromptCacheRetention24h,
		}
		if m := mcp.Default(); m != nil {
			params.Tools = m.OpenAITools()
		}
		params.SetExtraFields(map[string]interface{}{
			"thinking": map[string]string{
				"type": "enabled",
			},
			"web_search": map[string]bool{
				"enabled": true,
			},
			"tool_choice": "auto",
		})
		for round := 0; round <= aiMaxToolRounds; round++ {
			params.Messages = messages
			completion, err := a.openAi.Chat.Completions.New(ctx, params)
			if err != nil {
				a.Command.gs.ChatPrivateMsgNotice(s, a.GetUserChatMsgData(err.Error(), time.Now()))
				return
			}
			if len(completion.Choices) == 0 {
				a.Command.gs.ChatPrivateMsgNotice(s, a.GetUserChatMsgData("暂时没有返回内容，请稍后再试", time.Now()))
				return
			}

			choice := completion.Choices[0]
			if len(choice.Message.ToolCalls) == 0 {
				content := choice.Message.Content
				if content == "" {
					content = choice.Message.Refusal
				}
				if content == "" {
					a.Command.gs.ChatPrivateMsgNotice(s, a.GetUserChatMsgData("暂时没有返回内容，请稍后再试", time.Now()))
					return
				}
				now := time.Now()
				a.Command.gs.ChatPrivateMsgNotice(s, a.GetUserChatMsgData(content, now))
				sessionInfo.messages = append(sessionInfo.messages,
					&messageInfo{userId: 0, time: now, message: openai.AssistantMessage(content)})
				return
			}

			server := mcp.Default()
			if server == nil {
				a.Command.gs.ChatPrivateMsgNotice(s, a.GetUserChatMsgData("我现在无法查询游戏数据，请稍后再试", time.Now()))
				return
			}
			messages = append(messages, choice.Message.ToParam())
			for _, tc := range choice.Message.ToolCalls {
				out := server.CallTool(tc.Function.Name, tc.Function.Arguments)
				messages = append(messages, openai.ToolMessage(out, tc.ID))
			}
		}

		a.Command.gs.ChatPrivateMsgNotice(s, a.GetUserChatMsgData("这次查询步骤过多，请把问题拆得更具体一些", time.Now()))
	}()
}

func (a *AiBot) GetMsgRecords(userId uint32) []game.MsgRecordInterface {
	list := make([]game.MsgRecordInterface, 0)
	session, ok := a.session.Get(userId)
	if !ok {
		return list
	}
	session.sync.Lock()
	defer session.sync.Unlock()
	for _, message := range session.messages {
		list = append(list, message)
	}
	return list
}

func (c *Command) NewAiBot(cfg *config.Bot) *AiBot {
	gpt := &AiBot{
		Command: c,
		BotInfo: CfgToBotInfo(cfg),
		uuid:    uuid.NewString(),
		cfg:     cfg,
		session: cache.New[uint32, *aiMessage](24 * time.Hour),
	}
	gpt.NewClient()

	return gpt
}

func (a *AiBot) NewClient() {
	a.openAi = new(openai.NewClient(
		option.WithHTTPClient(&http.Client{}),
		option.WithBaseURL(a.cfg.BaseUrl),
		option.WithAPIKey(a.cfg.ApiKey),
		option.WithMaxRetries(3),
	))
}
