package shillgptbot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// openai "github.com/sashabaranov/go-openai"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/dialog"
	"github.com/go-telegram/ui/keyboard/inline"
)

var (
	telegramToken string
	lastMessages  map[int64]*lastMessage
	shilling      map[int64]shillState

	shillingMutex = &sync.RWMutex{}
)

type lastMessage struct {
	messageID int
	text      string
}

type shillGPTBot struct {
	bot    *bot.Bot
	logger *zap.Logger
	atom   *zap.AtomicLevel
	mongo  *storage.Mongo
	ready  bool
}

type shillState struct {
	inProgress bool
	user       models.User
	tweetLink  string
	tweetText  string
}

func NewShillGPTBot() *shillGPTBot {
	// see https://pkg.go.dev/go.uber.org/zap#AtomicLevel
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	atom.SetLevel(zap.ErrorLevel)
	atom.SetLevel(zap.InfoLevel)
	atom.SetLevel(zap.DebugLevel)

	lastMessages = make(map[int64]*lastMessage)
	shilling = make(map[int64]shillState)

	return &shillGPTBot{
		logger: logger,
		atom:   &atom,
		mongo:  storage.NewMongo(),
		// tclient: twitterOauth2Client(),
		ready: false,
	}
}

// Run
func (sb *shillGPTBot) Run() {
	telegramToken = viper.GetString("telegram.token")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(sb.defaultHandler),
		// bot.WithDebug(),
	}

	b, err := bot.New(telegramToken, opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	sb.bot = b

	sb.registerHandlers()

	b.Start(ctx)
}

// registerHandlers
func (sb *shillGPTBot) registerHandlers() {
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/shillx", bot.MatchTypeExact, sb.shillHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/shillx@", bot.MatchTypePrefix, sb.shillHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, sb.cancelHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/cancel@", bot.MatchTypePrefix, sb.cancelHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, sb.startHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, sb.helpHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/settings", bot.MatchTypeExact, sb.settingsHandler)
}

// shillHandler
func (sb *shillGPTBot) shillHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	shillingMutex.Lock()
	defer shillingMutex.Unlock()

	state := sb.chatShillState(chatID)
	canShill := !state.inProgress

	if !canShill {
		message := "shill in progress, click cancel or /cancel to start again"
		_, err := sb.sendMessage(ctx, b, chatID, message, &models.ReplyParameters{})
		if err != nil {
			sb.logger.Warn(
				"an issue occurred while to send already shilling message",
				zap.Error(err),
			)
		}
		sb.deleteMessage(ctx, chatID, update.Message.ID)
		return
	}

	state.inProgress = true
	state.user = *update.Message.From
	sb.updateChatShillState(chatID, state)
	sb.sendMessageWithCancel(ctx, b, chatID, "Please provide the tweet link")
}

// cancelHandler
func (sb *shillGPTBot) cancelHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	shillingMutex.Lock()
	defer shillingMutex.Unlock()

	state := sb.chatShillState(chatID)
	if !state.inProgress {
		return
	}

	sb.sendCancelMessage(ctx, b, chatID)
}

// sendCancelMessage
func (sb *shillGPTBot) sendCancelMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	sb.sendMessageAndReset(ctx, b, chatID, "cancelled")
}

// sendMessageAndReset
func (sb *shillGPTBot) sendMessageAndReset(ctx context.Context, b *bot.Bot, chatID int64, message string) {
	shilling[chatID] = shillState{}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   message,
	})
}

// sendMessageWithCancel
func (sb *shillGPTBot) sendMessageWithCancel(ctx context.Context, b *bot.Bot, chatID int64, message string) {
	kb := inline.New(b).
		Button("Cancel", []byte("cancel"), sb.onInlineKeyboardSelect)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: kb,
	})
}

// onInlineKeyboardSelect
func (sb *shillGPTBot) onInlineKeyboardSelect(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID

	buttonValue := string(data)
	if buttonValue == "cancel" {
		sb.sendCancelMessage(ctx, b, chatID)
	}
}

// defaultHandler
func (sb *shillGPTBot) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}
	message := bot.EscapeMarkdown(update.Message.Text)
	if len(message) == 0 {
		return
	}

	chatID := update.Message.Chat.ID

	if _, ok := lastMessages[chatID]; !ok {
		lastMessages[chatID] = &lastMessage{}
	}

	lastMessages[chatID].messageID = update.Message.ID
	lastMessages[chatID].text = update.Message.Text

	state := sb.chatShillState(chatID)
	if !state.inProgress || state.user.ID != update.Message.From.ID {
		return
	}

	// we don't have tweet link yet
	if state.tweetLink == "" {
		if !sb.isTweetURL(update.Message.Text) {
			sb.sendMessageAndReset(ctx, b, chatID, "not a valid tweet url, please start again")
			return
		}

		state.tweetLink = update.Message.Text
		sb.updateChatShillState(chatID, state)
		sb.deleteMessage(ctx, chatID, update.Message.ID)
		sb.sendMessageWithCancel(ctx, b, chatID, "Please provide the original tweet text")
		return
	}

	// we don't have tweet text yet
	if state.tweetText == "" {
		state.tweetText = update.Message.Text
		sb.updateChatShillState(chatID, state)
		sb.deleteMessage(ctx, chatID, update.Message.ID)

		link, err := sb.generateShillLink(chatID)
		if err != nil {
			sb.sendMessageAndReset(ctx, b, chatID, "Sorry an error occurred, please try again")
			return
		}

		message := `%v

Let's go shilling baby!!
		
Just click on the SHILL NOW button to generate your own, AI SHILL reply.
		`
		message = fmt.Sprintf(message, state.tweetLink)
		message = sb.escapeChars(message)
		dialogNodes := []dialog.Node{
			{ID: "shill", Text: message, Keyboard: [][]dialog.Button{{{Text: "SHILL NOW!!", URL: link}}}},
		}
		p := dialog.New(dialogNodes, dialog.Inline())
		_, err = p.Show(ctx, b, update.Message.Chat.ID, "shill")
		if err != nil {
			sb.sendMessageAndReset(ctx, b, chatID, "Sorry an error occurred, please try again")
			log.Fatal(err)
			return
		}

		sb.updateChatShillState(chatID, shillState{})
		return
	}
}

// startHandler
func (sb *shillGPTBot) startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	sb.sendMessage(ctx, b, update.Message.Chat.ID, "Let's get Trolling!", &models.ReplyParameters{})
}

// helpHandler
func (sb *shillGPTBot) helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	sb.sendMessage(ctx, b, update.Message.Chat.ID, "Coming soon...", &models.ReplyParameters{})
}

// settingsHandler
func (sb *shillGPTBot) settingsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	sb.sendMessage(ctx, b, update.Message.Chat.ID, "Nothing to see here...", &models.ReplyParameters{})
}

// sendMessage
func (sb *shillGPTBot) sendMessage(ctx context.Context, b *bot.Bot, chatID int64, message string, replyParams *models.ReplyParameters) (*models.Message, error) {
	params := &bot.SendMessageParams{
		ChatID:          chatID,
		Text:            message,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: replyParams,
	}

	// fmt.Printf("quote params %+v\n", params.ReplyParameters)

	sentMessage, err := b.SendMessage(ctx, params)
	if err != nil {
		sb.logger.Error(
			"an error occurred trying to send a message",
			zap.Error(err),
		)
	}

	return sentMessage, err
}

// deleteMessage
func (sb *shillGPTBot) deleteMessage(ctx context.Context, chatID int64, messageID int) (bool, error) {
	return sb.bot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
}

// chatShillState
func (sb *shillGPTBot) chatShillState(chatID int64) shillState {
	state := shillState{}
	if _, ok := shilling[chatID]; ok {
		state = shilling[chatID]
	}

	return state
}

// generateShillLink
func (sb *shillGPTBot) generateShillLink(chatID int64) (string, error) {
	state := sb.chatShillState(chatID)
	sl := NewShillLink(sb.mongo)
	sl.ChatID = chatID
	sl.TweetID = sb.extractTweetID(state.tweetLink)
	sl.TweetLink = state.tweetLink
	sl.TweetText = state.tweetText

	if err := sl.Insert(sl); err != nil {
		return "", err
	}

	apiUrl := viper.GetString("apiUrl")
	return fmt.Sprintf("%v/shill/%v", apiUrl, sl.ID.Hex()), nil
}

// updateChatShillState
func (sb *shillGPTBot) updateChatShillState(chatID int64, state shillState) {
	shilling[chatID] = state
}

// isTweetURL
func (sb *shillGPTBot) isTweetURL(rawURL string) bool {
	trimmedURL := strings.TrimSpace(rawURL)

	u, err := url.Parse(trimmedURL)
	if err != nil {
		return false
	}

	if u.Hostname() != "twitter.com" && u.Hostname() != "www.twitter.com" {
		return false
	}

	re := regexp.MustCompile(`^/[^/]+/status/\d+$`)
	return re.MatchString(u.Path)
}

// extractTweetID
func (sb *shillGPTBot) extractTweetID(tweetURL string) string {
	parts := strings.Split(tweetURL, "/")
	return parts[len(parts)-1]
}

// escapeChars
func (sb *shillGPTBot) escapeChars(input string) string {
	output := strings.ReplaceAll(input, ".", "\\.")
	output = strings.ReplaceAll(output, "!", "\\!")

	return output
}

// Logger
func (sb *shillGPTBot) Logger() *zap.Logger {
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	return logger
}

// AtomicLevel
func (sb *shillGPTBot) AtomicLevel() *zap.AtomicLevel {
	atom := zap.NewAtomicLevel()
	return &atom
}

// InitConfig reads in config file and ENV variables if set.
// We do this here rather than in cmd/root.go so that we can
// call this in our tests
func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".newApp" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".shill-bot")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("No config file found at .$HOME/.shill-bot.yaml)")
	}
}
