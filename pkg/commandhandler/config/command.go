package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/tghelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	COMMAND_SET_TOKEN_NAME     = "setTokenName"
	COMMAND_SET_HASH_TAGS      = "setHashtags"
	COMMAND_SET_CASH_TAGS      = "setCashtags"
	COMMAND_SHILL_INSTRUCTIONS = "setShillIntructions"
	COMMAND_NONE               = "none"
)

var (
	state      = make(map[int64]configHandlerState)
	stateMutex = &sync.RWMutex{}
)

type configHandlerState struct {
	activeCommand string
	lastPrompt    *models.Message
	done          bool
}

type configCommandHandler struct {
	commandhandler.Command
	tgh    tghelper.TGHelper
	logger *zap.Logger
	mongo  *storage.Mongo
}

// NewConfigCommandHandler
func NewConfigCommandHandler() commandhandler.CommandHandler {
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	return &configCommandHandler{
		logger: logger,
		mongo:  storage.NewMongo(),
	}
}

// Handle
func (cch *configCommandHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	fmt.Println("config handler")
	stateMutex.Lock()
	defer stateMutex.Unlock()

	cch.tgh = tghelper.NewTGHelper(b, cch.logger)

	chatID := update.Message.Chat.ID

	chs, ok := state[chatID]
	if !ok {
		cch.DisplayMainMenu(ctx, b, chatID)
		return
	}

	switch chs.activeCommand {
	case COMMAND_SET_TOKEN_NAME:
		cch.receiveTokenName(state[chatID], ctx, b, update)
	case COMMAND_SET_HASH_TAGS:
		cch.receiveHashTags(state[chatID], ctx, b, update)
	case COMMAND_SET_CASH_TAGS:
		cch.receiveCashTags(state[chatID], ctx, b, update)
	default:
		cch.DisplayMainMenu(ctx, b, chatID)
	}

}

// DisplayMainMenu
func (cch *configCommandHandler) DisplayMainMenu(ctx context.Context, b *bot.Bot, chatID int64) {
	chs := configHandlerState{
		activeCommand: COMMAND_NONE,
	}
	cch.updateState(chatID, chs)

	c, err := cch.configByChatID(ctx, b, chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	message := `CONFIG MENU
		
Current configuration

Token name: %s
Hashtag(s): %s
Cashtag(s): %s
Shill instructions: %s`

	message = fmt.Sprintf(
		message,
		c.Token,
		c.Hashtags,
		c.Cashtags,
		c.Community,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.configKeyboard(b),
	})
}

// Cancel
func (cch *configCommandHandler) Cancel(chatID int64) {
	chs, ok := state[chatID]
	if !ok {
		return
	}

	chs.activeCommand = COMMAND_NONE
	chs.done = false
	cch.updateState(chatID, chs)

	fmt.Printf("Cancel %+v\n", state[chatID])
}

// Done
func (cch *configCommandHandler) Done(chatID int64) bool {
	chs, ok := state[chatID]
	if !ok {
		return false
	}

	return chs.done
}

// updateState
func (cch *configCommandHandler) updateState(chatID int64, chs configHandlerState) {
	state[chatID] = chs
}

// onBack
func (cch *configCommandHandler) onBack(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	cch.DisplayMainMenu(ctx, b, chatID)
}

// onConfigSetTokenName
func (cch *configCommandHandler) onConfigSetTokenName(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, ok := state[chatID]
	if !ok {
		fmt.Printf("state not found \n")
		return
	}

	message := "What is the name of your token?"

	// e(ctx, &bot.SendMessageParams{
	// 	ChatID:      chatID,
	// 	Text:        message,
	// 	ReplyMarkup: cch.configKeyboard(b),
	// })

	// cch.tgh.SendMessageWithCancel(ctx, b, chatID, message)

	prompt, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelKeyboard(b),
	})

	chs.activeCommand = COMMAND_SET_TOKEN_NAME
	chs.lastPrompt = prompt
	cch.updateState(chatID, chs)
}

// onConfigSetHashTags
func (cch *configCommandHandler) onConfigSetHashTags(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, ok := state[chatID]
	if !ok {
		fmt.Printf("state not found \n")
		return
	}

	message := `What hashtag should I use when shilling your token? 
	
You can set multiple hashtags but I'll only use one or two at a time.

Multiple hashtags should be separated by spaces e.g.
MyAwesomeToken MYTOKENTOTHEMOON MyTokenIsTheBest

Make sure your primary hashtag is set first.`

	prompt, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelKeyboard(b),
	})

	chs.activeCommand = COMMAND_SET_HASH_TAGS
	chs.lastPrompt = prompt
	cch.updateState(chatID, chs)
}

// onConfigSetHashTags
func (cch *configCommandHandler) onConfigSetCashTags(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, ok := state[chatID]
	if !ok {
		fmt.Printf("state not found \n")
		return
	}

	message := `What cashtag should I use when shilling your token?

e.g. $MYTOKEN
`

	prompt, _ := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelKeyboard(b),
	})

	chs.activeCommand = COMMAND_SET_CASH_TAGS
	chs.lastPrompt = prompt
	cch.updateState(chatID, chs)
}

// receiveTokenName
func (cch *configCommandHandler) receiveTokenName(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	c, err := cch.configByChatID(ctx, b, chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Token = update.Message.Text
	if err = c.Update(&c); err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		cch.logger.Error(
			"an error occurred trying to update the token name in the config",
			zap.String("command", COMMAND_SET_TOKEN_NAME),
			zap.Int64("chatID", chatID),
			zap.String("token name", c.Token),
			zap.Error(err),
		)
		return
	}

	fmt.Printf("receive token name %v\n", update.Message.Text)
	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	cch.tgh.DeleteMessage(ctx, chatID, chs.lastPrompt.ID)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// receiveHashTags
func (cch *configCommandHandler) receiveHashTags(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	c, err := cch.configByChatID(ctx, b, chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Hashtags = update.Message.Text
	if err = c.Update(&c); err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		cch.logger.Error(
			"an error occurred trying to update the hashtags in the config",
			zap.String("command", COMMAND_SET_HASH_TAGS),
			zap.Int64("chatID", chatID),
			zap.String("hashtags", c.Hashtags),
			zap.Error(err),
		)
		return
	}

	fmt.Printf("receive hashtags %v\n", update.Message.Text)
	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	cch.tgh.DeleteMessage(ctx, chatID, chs.lastPrompt.ID)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// receiveCashTags
func (cch *configCommandHandler) receiveCashTags(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	c, err := cch.configByChatID(ctx, b, chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Cashtags = update.Message.Text
	if err = c.Update(&c); err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		cch.logger.Error(
			"an error occurred trying to update the cashtags in the config",
			zap.String("command", COMMAND_SET_CASH_TAGS),
			zap.Int64("chatID", chatID),
			zap.String("cashtags", c.Cashtags),
			zap.Error(err),
		)
		return
	}

	fmt.Printf("receive cashtags %v\n", update.Message.Text)
	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	cch.tgh.DeleteMessage(ctx, chatID, chs.lastPrompt.ID)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// configByChatID
func (cch *configCommandHandler) configByChatID(ctx context.Context, b *bot.Bot, chatID int64) (Config, error) {
	c, found, err := ConfigByChatID(cch.mongo, chatID)
	if err != nil {
		cch.logger.Error(
			"an error occurred trying to fetch config by chat ID",
			zap.String("command", COMMAND_SET_TOKEN_NAME),
			zap.Int64("chatID", chatID),
			zap.Error(err),
		)
		return c, err
	}

	if !found {
		c = NewConfig(cch.mongo)
		c.ChatID = chatID
		if err = c.Insert(&c); err != nil {
			cch.logger.Error(
				"an error occurred trying to create  the token name in the config",
				zap.String("command", COMMAND_SET_TOKEN_NAME),
				zap.Int64("chatID", chatID),
				zap.String("token name", c.Token),
				zap.Error(err),
			)
			return c, err
		}
	}

	return c, nil
}
