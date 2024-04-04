package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	openai "github.com/sashabaranov/go-openai"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/shillgptbot"
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
	sl, found, err := shillgptbot.ShillLinkByID(ss.a.mongo, shillID)
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

	redirectUrl := fmt.Sprintf("https://twitter.com/intent/tweet?in_reply_to=%s&text=%s", sl.TweetID, url.QueryEscape(reply))

	return c.Redirect(http.StatusFound, redirectUrl)
}

// generateReply
func (ts *shillService) generateReply(sl *shillgptbot.ShillLink) (string, error) {

	maxChars := 280
	charLimit := 260

	instruction := `You are a twitter shill who is crypto degen and an enthusiast of a new cryptocurrency called TROLLANA. 
	You love to shill tweets and use the opportunity to promote TROLLANA and it's community of fellow shills. 
	You can verge on being rude and keen to mention the massive gains TROLLANA will bring.  
	When you respond to tweets don't mention memes.  You can promote TROLLANA only but can disparage other coins.
	When you create tweets you should always add the hashtag #TROLLANA and try and keep the number of hashtags to a maxiumum of four but ideally keep to two.
	Respond to the following tweet in your unique style and keep the response to a maximum of %d characters: '%v'
	%v`

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
						Content: fmt.Sprintf(instruction, charLimit, sl.TweetText),
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
