package command

import (
	"context"
	"fmt"
	"net/http"
	"sort"
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

type AiBot struct {
	*Command
	*game.BotInfo
	session *cache.Cache[uint32, *aiMessage]
	openAi  *openai.Client
	cfg     *config.Bot
	uuid    string
}

type aiMessage struct {
	messages *cache.Cache[int, *messageInfo]
}

type messageInfo struct {
	userId  uint32
	id      int
	time    time.Time
	message openai.ChatCompletionMessageParamUnion
}

func (m *aiMessage) MessageList() []*messageInfo {
	messageList := make([]*messageInfo, 0)
	m.messages.Range(func(id int, info *messageInfo) bool {
		messageList = append(messageList, info)
		return true
	})
	sort.Slice(messageList, func(i, j int) bool {
		return messageList[i].id < messageList[j].id
	})
	return messageList
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
				messages: cache.New[int, *messageInfo](0),
			}
			sessionInfo.messages.Set(0, &messageInfo{userId: 0, time: time.Now(),
				message: openai.SystemMessage(fmt.Sprintf("你现在对话的玩家名称是:%s,%s", s.NickName, a.cfg.System))})
			a.session.Set(s.UserId, sessionInfo)
		}
		messageList := sessionInfo.MessageList()
		if len(messageList) > 500 {
			delNum := 0
			for _, message := range messageList {
				if message.id == 0 {
					continue
				}
				sessionInfo.messages.Del(message.id)
				delNum++
				if delNum >= 50 {
					break
				}
			}
			messageList = sessionInfo.MessageList()
		}
		nextId := 1
		if len(messageList) > 0 {
			nextId = messageList[len(messageList)-1].id + 1
		}
		userMessage := &messageInfo{id: nextId, userId: s.UserId, time: time.Now(), message: openai.UserMessage(text)}
		sessionInfo.messages.Set(nextId, userMessage)
		messageList = append(messageList, userMessage)

		messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messageList))
		for _, message := range messageList {
			messages = append(messages, message.message)
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
		params.SetExtraFields(map[string]interface{}{
			"thinking": map[string]string{
				"type": "enabled",
			},
			"web_search": map[string]bool{
				"enabled": true,
			},
			"tool_choice": "auto",
		})
		completion, err := a.openAi.Chat.Completions.New(ctx, params)
		if err != nil {
			a.Command.gs.PostPlayerFunc(s.UserId, func(p *model.Player) {
				a.Command.gs.ChatPrivateMsgNotice(p, a.GetUserChatMsgData(err.Error(), time.Now()))
			})
			return
		}
		for _, choice := range completion.Choices {
			a.Command.gs.PostPlayerFunc(s.UserId, func(p *model.Player) {
				a.Command.gs.ChatPrivateMsgNotice(p, a.GetUserChatMsgData(choice.Message.Content, time.Now()))
			})
			nextId++
			sessionInfo.messages.Set(nextId,
				&messageInfo{userId: 0, time: time.Now(), message: openai.AssistantMessage(choice.Message.Content)})
		}
	}()
}

func (a *AiBot) GetMsgRecords(userId uint32) []game.MsgRecordInterface {
	list := make([]game.MsgRecordInterface, 0)
	session, ok := a.session.Get(userId)
	if !ok {
		return list
	}
	for _, message := range session.MessageList() {
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
