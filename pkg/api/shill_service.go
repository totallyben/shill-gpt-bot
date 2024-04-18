package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	openai "github.com/sashabaranov/go-openai"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler/shillx"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/config"
	"go.uber.org/zap"
)

const shillServiceBasePath string = "/shill"

// shillService
type shillService struct {
	a *Api
}

// newShillService
func newShillService(a *Api) *shillService {
	return &shillService{
		a: a,
	}
}

// LoadRoutes
func (ss *shillService) LoadRoutes(parentGroup *echo.Group) {
	g := parentGroup.Group(shillServiceBasePath)

	g.GET("/:shillID", ss.createTwitterReply)
}

// createTwitterReply
func (ss *shillService) createTwitterReply(c echo.Context) error {
	shillID := c.Param("shillID")

	// check if we have a wallet
	sl, found, err := shillx.ShillLinkByID(ss.a.mongo, shillID)
	if err != nil {
		ss.a.logger.Error(
			"could not fetch shill link",
			zap.String("shillID", shillID),
			zap.Error(err),
		)
		return ReturnError(c, err)
	}

	if !found {
		ss.a.logger.Error(
			"shill not found",
			zap.String("shillID", shillID),
		)
		return ReturnError(c, ErrShillNotFound)
	}

	reply, err := ss.generateReply(sl)
	if err != nil {
		return ReturnError(c, err)
	}

	// store the reply
	s := shillx.NewShill(ss.a.mongo)
	s.ChatID = sl.ChatID
	s.TweetID = sl.TweetID
	s.Reply = reply
	if err := s.Insert(s); err != nil {
		ss.a.logger.Warn(
			"Unable to store generated shill reply",
			zap.String("shillID", shillID),
		)
	}

	redirectUrl := fmt.Sprintf("https://twitter.com/intent/tweet?in_reply_to=%s&text=%s", sl.TweetID, url.QueryEscape(reply))

	return c.Redirect(http.StatusFound, redirectUrl)
}

// generateReply
func (ss *shillService) generateReply(sl *shillx.ShillLink) (string, error) {

	maxChars := 280
	charLimit := 260

	instruction := fmt.Sprintf(
		"%s. Respond to the following tweet in your unique style and keep the response to a maximum of %d characters: '%v'",
		ss.aiInstruction(sl),
		charLimit,
		sl.TweetText,
	)

	client := openaiClient()

	attempt := 1
	maxAttempts := 3
	reply := ""
	var resp openai.ChatCompletionResponse
	var err error
	for {
		if attempt > maxAttempts {
			return "", ErrOpenAiReplyLength
		}

		resp, err = client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT4TurboPreview,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: instruction,
					},
				},
			},
		)

		if err != nil {
			return "", err
		}

		reply = strings.Trim(resp.Choices[0].Message.Content, `"`)

		// check character limit was
		if len(reply) > maxChars {
			attempt++
			continue
		}
		break
	}

	return reply, nil
}

// aiInstruction
func (ss *shillService) aiInstruction(sl *shillx.ShillLink) string {
	c, found, err := config.ConfigByChatID(ss.a.mongo, sl.ChatID)
	if err != nil {
		ss.a.logger.Error(
			"an error occurred trying to fetch config by chat ID",
			zap.Int64("chatID", sl.ChatID),
			zap.Error(err),
		)
		return ""
	}

	if !found {
		ss.a.logger.Error(
			"could not find config for chat ID",
			zap.Int64("chatID", sl.ChatID),
			zap.Error(err),
		)
		return ""
	}

	if sl.ReplyType == shillx.REPLY_TYPE_TROLL {
		return ss.aiInstructionTroll(c)
	}

	return ss.aiInstructionShill(c)
}

// aiInstructionShill
func (ss *shillService) aiInstructionShill(c config.Config) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called %s. 
	You love to reply to tweets related to crypto and use the opportunity to promote %s and it's awesome community.
	It's community as describes itself as %s.
	You are cheeky and are keen to mention the massive gains %s will bring.  
	When you respond to tweets don't mention memes.  You can promote %s only and can disparage other coins in a friendly way.
	When you create tweets you should try and include the hashtags "%s" and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include the cashtags "%s" in the response.`

	return fmt.Sprintf(
		instruction,
		c.Token,
		c.Token,
		c.Community,
		c.Token,
		c.Token,
		c.Hashtags,
		c.Cashtags,
	)
}

// aiInstructionTroll
func (ss *shillService) aiInstructionTroll(c config.Config) string {

	instruction := `You are a twitter troll who enjoys making fun of other users tweets.
	You can verge on being rude but always keep responses light hearted.
	You can disparage others but in a jokey way.
	You never promote harm, racism, homophobia, transphobia or have any strong political views.
	When you create tweets you should try and include the hashtags "%s" and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include the cashtags "%s" in the response.`

	return fmt.Sprintf(
		instruction,
		c.Hashtags,
		c.Cashtags,
	)
}
