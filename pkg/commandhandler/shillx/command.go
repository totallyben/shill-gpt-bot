package shillx

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/dialog"
	"github.com/spf13/viper"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/tghelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	REPLY_TYPE_SHILL = "shill"
	REPLY_TYPE_TROLL = "troll"
)

var (
	state      = make(map[int64]ShillHandlerState)
	stateMutex = &sync.RWMutex{}
)

type ShillHandlerState struct {
	ChatID     int64
	inProgress bool
	done       bool
	tweetLink  string
	tweetText  string
	ReplyType  string
	lastPrompt *models.Message
	user       models.User
}

type ShillCommandHandler struct {
	commandhandler.Command
	tgh    tghelper.TGHelper
	logger *zap.Logger
	mongo  *storage.Mongo
}

// NewShillCommandHandler
func NewShillCommandHandler() commandhandler.CommandHandler {
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	return &ShillCommandHandler{
		logger: logger,
		mongo:  storage.NewMongo(),
	}
}

// Handle
func (sch *ShillCommandHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	chatID := update.Message.Chat.ID

	shs, ok := state[chatID]
	if !ok {
		shs = ShillHandlerState{
			ChatID:    chatID,
			ReplyType: REPLY_TYPE_SHILL,
		}
	}

	sch.tgh = tghelper.NewTGHelper(b, sch.logger)

	// canShill := !shs.inProgress || shs.done

	// if !canShill {
	// 	message := fmt.Sprintf("%s in progress, click cancel or /cancel to start again", shs.ReplyType)
	// 	_, err := sch.tgh.SendMessage(ctx, b, chatID, message, &models.ReplyParameters{})
	// 	if err != nil {
	// 		sch.logger.Warn(
	// 			"an issue occurred while to send already shilling message",
	// 			zap.Error(err),
	// 		)
	// 	}

	// 	sch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	// 	return
	// }

	if !shs.inProgress {
		sch.requestTweetLink(shs, ctx, b, update)
		return
	}

	// if shs.user.ID != update.Message.From.ID {
	// 	message := fmt.Sprintf("%s in progress, wait your turn", shs.ReplyType)
	// 	_, err := sch.tgh.SendMessage(ctx, b, chatID, message, &models.ReplyParameters{})
	// 	if err != nil {
	// 		sch.logger.Warn(
	// 			"an issue occurred while to send already shilling message",
	// 			zap.Error(err),
	// 		)
	// 	}
	// 	sch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	// 	return
	// }

	if shs.tweetLink == "" {
		if err := sch.receiveTweetLink(shs, ctx, b, update); err != nil {
			return
		}

		sch.requestTweetText(state[chatID], ctx, b)
		return
	}

	sch.generateShill(shs, ctx, b, update)
}

// Reset
func (sch *ShillCommandHandler) Reset(ctx context.Context, b *bot.Bot, chatID int64) {}

// Cancel
func (sch *ShillCommandHandler) Cancelled(chatID int64) bool {
	return false
}

// Cancel
func (sch *ShillCommandHandler) Cancel(chatID int64) {
	shs, ok := state[chatID]
	if !ok {
		return
	}

	shs.inProgress = false
	shs.done = true
	sch.UpdateState(shs.ChatID, shs)
}

// Done
func (sch *ShillCommandHandler) Done(chatID int64) bool {
	shs, ok := state[chatID]
	if !ok {
		return false
	}

	return shs.done
}

// requestTweetLink
func (sch *ShillCommandHandler) requestTweetLink(shs ShillHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) error {
	shs.inProgress = true
	shs.done = false
	shs.user = *update.Message.From
	sch.UpdateState(shs.ChatID, shs)
	prompt, err := sch.tgh.SendMessageWithCancel(ctx, b, shs.ChatID, "Please provide the tweet link")
	if err != nil {
		sch.SendMessageAndFinish(shs, ctx, b, shs.ChatID, "sorry an error occurred, please try again 1")
		sch.logger.Error(
			"an error occurred trying to sendMessageWithCancel",
			zap.Error(err),
		)
		return err
	}
	shs.lastPrompt = prompt
	sch.UpdateState(shs.ChatID, shs)

	return nil
}

func (sch *ShillCommandHandler) SendMessageAndFinish(shs ShillHandlerState, ctx context.Context, b *bot.Bot, chatID int64, message string) {
	sch.tgh.SendMessage(ctx, b, chatID, message, &models.ReplyParameters{})
	shs.inProgress = false
	shs.done = true
	sch.UpdateState(shs.ChatID, shs)
}

// receiveTweetLink
func (sch *ShillCommandHandler) receiveTweetLink(shs ShillHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) error {

	if !sch.isTweetURL(update.Message.Text) {
		errorMessage := "not a valid tweet url, please start again"
		sch.SendMessageAndFinish(shs, ctx, b, shs.ChatID, errorMessage)
		return errors.New(errorMessage)
	}

	tweetUrl := update.Message.Text
	parsedUrl, err := url.Parse(tweetUrl)
	if err != nil {
		errorMessage := "sorry an error occurred, please try again 3"
		sch.SendMessageAndFinish(shs, ctx, b, shs.ChatID, errorMessage)
		return errors.New(errorMessage)
	}
	parsedUrl.RawQuery = ""
	tweetUrl = parsedUrl.String()

	shs.tweetLink = tweetUrl
	sch.UpdateState(shs.ChatID, shs)

	sch.tgh.DeleteMessage(ctx, shs.ChatID, update.Message.ID)
	sch.tgh.DeleteMessage(ctx, shs.ChatID, shs.lastPrompt.ID)
	return nil
}

// requestTweetText
func (sch *ShillCommandHandler) requestTweetText(shs ShillHandlerState, ctx context.Context, b *bot.Bot) error {
	prompt, err := sch.tgh.SendMessageWithCancel(ctx, b, shs.ChatID, "Please provide the original tweet text")
	if err != nil {
		sch.SendMessageAndFinish(shs, ctx, b, shs.ChatID, "sorry an error occurred, please try again 4")
		sch.logger.Error(
			"an error occurred trying to SendMessageWithCancel",
			zap.Error(err),
		)
		return err
	}
	shs.lastPrompt = prompt
	sch.UpdateState(shs.ChatID, shs)

	return nil
}

// generateShill
func (sch *ShillCommandHandler) generateShill(shs ShillHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	shs.tweetText = update.Message.Text
	sch.UpdateState(shs.ChatID, shs)

	sch.tgh.DeleteMessage(ctx, shs.ChatID, update.Message.ID)
	sch.tgh.DeleteMessage(ctx, shs.ChatID, shs.lastPrompt.ID)

	link, err := sch.generateShillLink(shs.ChatID)
	if err != nil {
		sch.SendMessageAndFinish(shs, ctx, b, shs.ChatID, "sorry an error occurred, please try again 5")
		sch.logger.Error(
			"an error occurred trying to generateShillLink",
			zap.Error(err),
		)
		return
	}

	buttonLabel := "SHILL NOW!!"
	adjective := "shilling"
	action := "SHILL"
	if shs.ReplyType == REPLY_TYPE_TROLL {
		buttonLabel = "TROLL NOW!!"
		adjective = "trolling"
		action = "TROLL"
	}

	poweredBy := "\n\nPowered by $TROLLANA - https://t.me/TROLLANAOfficial"
	if shs.ChatID == -1002038440299 {
		poweredBy = ""
	}

	message := `%v

Let's go %v baby!!
	
Just click on the %v NOW button to generate your own, AI %v reply.%v
	`
	message = fmt.Sprintf(message, shs.tweetLink, adjective, action, action, poweredBy)
	message = tghelper.EscapeChars(message)

	dialogNodes := []dialog.Node{
		{ID: "shill", Text: message, Keyboard: [][]dialog.Button{{{Text: buttonLabel, URL: link}}}},
	}
	p := dialog.New(dialogNodes, dialog.WithPrefix("config"))
	_, err = p.Show(ctx, b, update.Message.Chat.ID, "shill")
	if err != nil {
		sch.SendMessageAndFinish(shs, ctx, b, shs.ChatID, "sorry an error occurred, please try again 6")
		log.Fatal(err)
		return
	}

	sch.UpdateState(shs.ChatID, ShillHandlerState{})
}

// UpdateState
func (sch *ShillCommandHandler) UpdateState(chatID int64, shs ShillHandlerState) {
	state[chatID] = shs
}

// isTweetURL
func (sch *ShillCommandHandler) isTweetURL(rawURL string) bool {
	trimmedURL := strings.TrimSpace(rawURL)

	u, err := url.Parse(trimmedURL)
	if err != nil {
		return false
	}

	if u.Hostname() != "twitter.com" && u.Hostname() != "www.twitter.com" && u.Hostname() != "x.com" && u.Hostname() != "www.x.com" {
		return false
	}

	re := regexp.MustCompile(`^/[^/]+/status/\d+$`)
	return re.MatchString(u.Path)
}

// extractTweetID
func (sch *ShillCommandHandler) extractTweetID(tweetURL string) string {
	parts := strings.Split(tweetURL, "/")
	return parts[len(parts)-1]
}

// generateShillLink
func (sch *ShillCommandHandler) generateShillLink(chatID int64) (string, error) {
	shs, ok := state[chatID]
	if !ok {
		shs = ShillHandlerState{}
	}

	sl := NewShillLink(sch.mongo)
	sl.ChatID = chatID
	sl.TweetID = sch.extractTweetID(shs.tweetLink)
	sl.TweetLink = shs.tweetLink
	sl.TweetText = shs.tweetText
	sl.ReplyType = shs.ReplyType

	if err := sl.Insert(sl); err != nil {
		return "", err
	}

	apiUrl := viper.GetString("apiUrl")
	return fmt.Sprintf("%v/shill/%v", apiUrl, sl.ID.Hex()), nil
}

// setLogger
func (sch *ShillCommandHandler) SetLogger(logger *zap.Logger) {
	sch.logger = logger
}

// setMongo
func (sch *ShillCommandHandler) SetMongo(mongo *storage.Mongo) {
	sch.mongo = mongo
}

// State
func (sch *ShillCommandHandler) State(chatID int64) (ShillHandlerState, bool) {
	shs, ok := state[chatID]
	return shs, ok
}
