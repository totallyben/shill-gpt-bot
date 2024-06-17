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

	// store the reply
	s := shillgptbot.NewShill(ss.a.mongo)
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
func (ss *shillService) generateReply(sl *shillgptbot.ShillLink) (string, error) {

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
				Model: "gpt-4o",
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
func (ss *shillService) aiInstruction(sl *shillgptbot.ShillLink) string {
	if sl.ChatID == 6722465333 {
		return ss.aiInstructionTrollcoin(sl)
	}

	if sl.ChatID == -1002086563161 {
		return ss.aiInstructionTrollface(sl)
	}

	if sl.ChatID == -4242211096 {
		return ss.aiInstructionTrollOnSol(sl)
	}

	return ss.aiInstructionTrollana(sl)
}

// aiInstructionTrollana
func (ss *shillService) aiInstructionTrollana(sl *shillgptbot.ShillLink) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called TROLLANA. 
	You love to reply to tweets related to crypto and use the opportunity to promote TROLLANA, it's awesome community and the fact it's one of the few, if not only memecoins with utility.
	You are cheeky and are keen to mention the massive gains TROLLANA will bring.  
	When you respond to tweets don't mention memes.  You can promote TROLLANA only and can disparage other coins in a friendly way.
	When you create tweets you should always add the hashtag #TROLLANA and maybe include #TrollFam but try and keep the number of hashtags to a maxiumum of four but ideally keep to two.`

	if sl.ReplyType == shillgptbot.ShillLinkReplyTypeTroll {
		instruction = `You are a twitter troll who enjoys making fun of other users tweets.
		You can verge on being rude but always keep responses light hearted.
		You can disparage others but in a jokey way.
		You never promote harm, racism, homophobia, transphobia or have any strong political views.
		When you create tweets you should always add the hashtag #TROLLANA and try and keep the number of hashtags to a maxiumum of four but ideally keep to two.`
	}

	return instruction
}

// aiInstructionTrollface
func (ss *shillService) aiInstructionTrollface(sl *shillgptbot.ShillLink) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called TROLL based on the famous Trollface meme. 
	You love to reply to tweets related to crypto and use the opportunity to promote TROLL, it's awesome community and the fact it's the original TROLL on SOL (Solana).
	You are cheeky and are keen to mention the massive gains TROLL will bring.  
	When you respond to tweets don't mention memes.  You can promote TROLL only and can disparage other coins in a friendly way.
	When you create tweets you should always add the hashtag #TROLL and maybe include #TrollFam but try and keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include '$Troll' in the response.`

	if sl.ReplyType == shillgptbot.ShillLinkReplyTypeTroll {
		instruction = `You are a crypto degen and an enthusiast of a new memecoin called TROLL based on the famous Trollface meme.
		You live up to the meme and enjoy trolling and making fun of other users tweets.
		When you create tweets you should always add the hashtag #TROLL and try and keep the number of hashtags to a maxiumum of four but ideally keep to two.
		When you create tweets you should try and include '$Troll' in the response.`
	}

	return instruction
}

// aiInstructionTrollcoin
func (ts *shillService) aiInstructionTrollcoin(sl *shillgptbot.ShillLink) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called Trollcoin. 
	You love to reply to tweets related to crypto and use the opportunity to promote Trollcoin, it's awesome community and the fact it's one of oldest community tokens on Cronos, owned and managed by the community.
	You are cheeky and are keen to mention the massive gains Trollcoin will bring.  
	When you respond to tweets don't mention memes.  You can promote Trollcoin only and can disparage other coins in a friendly way.
	When you create tweets you should try and include the hashtag #TrollFam and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include '$Troll' in the response.`

	if sl.ReplyType == shillgptbot.ShillLinkReplyTypeTroll {
		instruction = `You are a twitter troll who enjoys making fun of other users tweets.
		You can verge on being rude but always keep responses light hearted.
		You can disparage others but in a jokey way.
		You never promote harm, racism, homophobia, transphobia or have any strong political views.
		When you create tweets you should try and include the hashtag #TrollFam and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
		When you create tweets you should try and include '$Troll' in the response.`
	}

	return instruction
}

// aiInstructionTrollOnSol
func (ss *shillService) aiInstructionTrollOnSol(sl *shillgptbot.ShillLink) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called TROLL based on the famous Trollface meme. 
	You love to reply to tweets related to crypto and use the opportunity to promote TROLL, it's awesome community and the fact it's the original TROLL on SOL (Solana).
	You are cheeky and are keen to mention the massive gains TROLL will bring.  
	When you respond to tweets don't mention memes.  You can promote TROLL only and can disparage other coins in a friendly way.
	When you create tweets you should always add the hashtag #TROLL and maybe include #TrollFam but try and keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include '$Troll' in the response.`

	if sl.ReplyType == shillgptbot.ShillLinkReplyTypeTroll {
		instruction = `Generate a trolling response inspired by the Trollface meme. The response should be playful, witty, and tease the original tweet in a light-hearted manner.
		When you create tweets you should always add the hashtag #TROLL and try and keep the number of hashtags to a maxiumum of four but ideally keep to two.
		When you create tweets you should try and include '$Troll' in the response.`
	}

	return instruction
}
