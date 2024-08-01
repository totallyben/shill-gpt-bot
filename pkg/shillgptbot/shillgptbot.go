package shillgptbot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler/config"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler/shillx"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler/trollx"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/tghelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// openai "github.com/sashabaranov/go-openai"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	COMMAND_NONE   = "none"
	COMMAND_SHILL  = "shill"
	COMMAND_TROLL  = "troll"
	COMMAND_CONFIG = "config"
)

var (
	telegramToken string
	lastMessages  map[int64]*lastMessage
	bState        map[int64]*botState

	stateMutex = &sync.RWMutex{}
)

type botState struct {
	activeCommand  string
	user           models.User
	commandHandler commandhandler.CommandHandler
}

type lastMessage struct {
	messageID int
	text      string
}

type ShillGPTBot struct {
	bot    *bot.Bot
	logger *zap.Logger
	atom   *zap.AtomicLevel
	mongo  *storage.Mongo
	tgh    tghelper.TGHelper
	ready  bool
}

// NewShillGPTBot
func NewShillGPTBot() *ShillGPTBot {
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
	bState = make(map[int64]*botState)

	return &ShillGPTBot{
		logger: logger,
		atom:   &atom,
		mongo:  storage.NewMongo(),
		// tclient: twitterOauth2Client(),
		ready: false,
	}
}

// Run
func (sb *ShillGPTBot) Run() {
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
	sb.tgh = tghelper.NewTGHelper(b, sb.logger)

	sb.registerHandlers()

	b.Start(ctx)
}

// registerHandlers
func (sb *ShillGPTBot) registerHandlers() {
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/shillx", bot.MatchTypeExact, sb.shillHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/shillx@", bot.MatchTypePrefix, sb.shillHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/trollx", bot.MatchTypeExact, sb.trollHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/trollx@", bot.MatchTypePrefix, sb.trollHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, sb.cancelHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/cancel@", bot.MatchTypePrefix, sb.cancelHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, sb.startHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, sb.helpHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/config", bot.MatchTypeExact, sb.configHandler)
	sb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/config@", bot.MatchTypePrefix, sb.configHandler)
}

// shillHandler
func (sb *ShillGPTBot) shillHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	commandHandler := shillx.NewShillCommandHandler()
	chatID := update.Message.Chat.ID

	bs, ok := sb.botState(chatID)
	if !ok || bs.activeCommand != COMMAND_SHILL {
		bs = &botState{
			activeCommand:  COMMAND_SHILL,
			user:           *update.Message.From,
			commandHandler: commandHandler,
		}
		bState[chatID] = bs
	}

	bs.commandHandler.Handle(ctx, b, update)
}

// trollHandler
func (sb *ShillGPTBot) trollHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	commandHandler := trollx.NewTrollCommandHandler()
	chatID := update.Message.Chat.ID

	bs, ok := sb.botState(chatID)
	if !ok || bs.activeCommand != COMMAND_TROLL {
		bs = &botState{
			activeCommand:  COMMAND_TROLL,
			user:           *update.Message.From,
			commandHandler: commandHandler,
		}
		bState[chatID] = bs
	}

	bs.commandHandler.Handle(ctx, b, update)
}

// cancelHandler
func (sb *ShillGPTBot) cancelHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	sb.cancel(ctx, b, update)
}

// cancel
func (sb *ShillGPTBot) cancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	fmt.Printf("cancel!!")
	chatID := update.Message.Chat.ID

	bs, ok := sb.botState(chatID)
	if ok {
		bs.activeCommand = COMMAND_NONE
		bs.commandHandler.Cancel(chatID)
	}

	sb.tgh.SendCancelledMessage(ctx, b, chatID)
}

// defaultHandler
func (sb *ShillGPTBot) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	message := bot.EscapeMarkdown(update.Message.Text)
	if message == "cancelled" {
		sb.cancel(ctx, b, update)
		return
	}

	chatID := update.Message.Chat.ID

	// if _, ok := lastMessages[chatID]; !ok {
	// 	lastMessages[chatID] = &lastMessage{}
	// }

	// lastMessages[chatID].messageID = update.Message.ID
	// lastMessages[chatID].text = update.Message.Text

	bs, ok := sb.botState(chatID)
	if !ok {
		return
	}

	if bs.activeCommand == COMMAND_NONE {
		return
	}

	if bs.user.ID != update.Message.From.ID {
		return
	}

	if bs.commandHandler.Done(chatID) {
		bs.activeCommand = COMMAND_NONE
	}

	bs.commandHandler.Handle(ctx, b, update)
}

// startHandler
func (sb *ShillGPTBot) startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	sb.tgh.SendMessage(ctx, b, update.Message.Chat.ID, "Let's get shilling!", &models.ReplyParameters{})
}

// helpHandler
func (sb *ShillGPTBot) helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	sb.tgh.SendMessage(ctx, b, update.Message.Chat.ID, "Coming soon...", &models.ReplyParameters{})
}

// configHandler
func (sb *ShillGPTBot) configHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	commandHandler := config.NewConfigCommandHandler()
	chatID := update.Message.Chat.ID

	bs, ok := sb.botState(chatID)
	if !ok || bs.activeCommand != COMMAND_CONFIG {
		bs = &botState{
			activeCommand:  COMMAND_CONFIG,
			user:           *update.Message.From,
			commandHandler: commandHandler,
		}
		bState[chatID] = bs
	}

	sb.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	bs.commandHandler.Handle(ctx, b, update)
}

// botState
func (sb *ShillGPTBot) botState(chatID int64) (*botState, bool) {
	bs, ok := bState[chatID]
	return bs, ok
}

// Logger
func (sb *ShillGPTBot) Logger() *zap.Logger {
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
func (sb *ShillGPTBot) AtomicLevel() *zap.AtomicLevel {
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
