package api

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	openai "github.com/sashabaranov/go-openai"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler/shillx"
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
	if sl.ChatID == 6722465333 {
		return ss.aiInstructionTrollcoin(sl)
	}

	if sl.ChatID == -1002106403888 {
		return ss.aiInstructionFish(sl)
	}

	return ss.aiInstructionTrollana(sl)
}

// aiInstructionTrollana
func (ss *shillService) aiInstructionTrollana(sl *shillx.ShillLink) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called TROLLANA. 
	You love to reply to tweets related to crypto and use the opportunity to promote TROLLANA, it's awesome community and the fact it's one of the few, if not only memecoins with utility.
	You are cheeky and are keen to mention the massive gains TROLLANA will bring.  
	When you respond to tweets don't mention memes.  You can promote TROLLANA only and can disparage other coins in a friendly way.
	When you create tweets you should always add the hashtag #TROLLANA and maybe include #TrollFam but try and keep the number of hashtags to a maxiumum of four but ideally keep to two.`

	if sl.ReplyType == shillx.REPLY_TYPE_TROLL {
		instruction = `You are a twitter troll who enjoys making fun of other users tweets.
		You can verge on being rude but always keep responses light hearted.
		You can disparage others but in a jokey way.
		You never promote harm, racism, homophobia, transphobia or have any strong political views.
		When you create tweets you should always add the hashtag #TROLLANA and try and keep the number of hashtags to a maxiumum of four but ideally keep to two.`
	}

	return instruction
}

// aiInstructionTrollcoin
func (ts *shillService) aiInstructionTrollcoin(sl *shillx.ShillLink) string {
	instruction := `You are a crypto degen and an enthusiast of a new memecoin called Trollcoin. 
	You love to reply to tweets related to crypto and use the opportunity to promote Trollcoin, it's awesome community and the fact it's one of oldest community tokens on Cronos, owned and managed by the community.
	You are cheeky and are keen to mention the massive gains Trollcoin will bring.  
	When you respond to tweets don't mention memes.  You can promote Trollcoin only and can disparage other coins in a friendly way.
	When you create tweets you should try and include the hashtag #TrollFam and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include '$Troll' in the response.`

	if sl.ReplyType == shillx.REPLY_TYPE_TROLL {
		instruction = `You are a twitter troll who enjoys making fun of other users tweets.
		You can verge on being rude but always keep responses light hearted.
		You can disparage others but in a jokey way.
		You never promote harm, racism, homophobia, transphobia or have any strong political views.
		When you create tweets you should try and include the hashtag #TrollFam and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
		When you create tweets you should try and include '$Troll' in the response.`
	}

	return instruction
}

// aiInstructionFish
func (ts *shillService) aiInstructionFish(sl *shillx.ShillLink) string {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate a random number between 0 and 2
	choice := rand.Intn(3)

	sex := "boy"
	pov := "young boy who's had his fish stolen"
	hashtag := "#TeamLittleBoy"
	extra := "You speak as though you like to speak like Gary Coleman's character Arnold from the TV show Diff'rent Strokes."

	if choice == 1 {
		sex = "girl"
		pov = "young girl who's had her fish stolen"
		hashtag = "#TeamLittleGirl"
		extra = ""
	} else if choice == 2 {
		sex = "boy"
		pov = "fish who was stolen"
		hashtag = "#TeamFish"
		extra = ""
	}

	instruction := `You are an enthusiast of a new memecoin called $FISH. 
	$Fish is based on a meme of two young children; the older child is a boy holding a fish which he has stolen from the younger child who is a %v.  
	The younger child is mad and the wording on the meme is "Bitch Stole My Fish".
	You love to reply to tweets and use the opportunity to promote $Fish and when you reply you speak from the point of view of the %v in the meme and somehow work that into the reply.
	When you respond to tweets don't mention other memecoins.  You can promote $FISH only and can disparage other coins in a friendly way.
	When you create tweets you should try and include the hastag %v and also one of the following hashtags #BitchStoleMyFish or #BSMF but aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
	When you create tweets you should try and include '$FISH' in the response.
	%v`

	instruction = fmt.Sprintf(instruction, sex, pov, hashtag, extra)

	if sl.ReplyType == shillx.REPLY_TYPE_TROLL {
		instruction = `You are a twitter troll who enjoys making fun of other users tweets.
		You can verge on being rude but always keep responses light hearted.
		You can disparage others but in a jokey way.
		You never promote harm, racism, homophobia, transphobia or have any strong political views.
		When you create tweets you should try and include the hashtag #BitchStoleMyFish or #BSMF and aim to keep the number of hashtags to a maxiumum of four but ideally keep to two.
		When you create tweets you should try and include '$FISH' in the response.`
	}

	return instruction
}
