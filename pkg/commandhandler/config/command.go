package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/config"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/tghelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	COMMAND_SET_TOKEN_NAME     = "setTokenName"
	COMMAND_SET_HASH_TAGS      = "setHashtags"
	COMMAND_SET_CASH_TAGS      = "setCashtags"
	COMMAND_SET_COMMUNITY_DESC = "setCommunityDescription"
	COMMAND_SET_CANCEL         = "cancel"
	COMMAND_NONE               = "none"
)

var (
	state      = make(map[int64]configHandlerState)
	stateMutex = &sync.RWMutex{}
)

type configHandlerState struct {
	activeCommand string
	lastPrompts   []*models.Message
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
	stateMutex.Lock()
	defer stateMutex.Unlock()

	cch.tgh = tghelper.NewTGHelper(b, cch.logger)

	chatID := update.Message.Chat.ID

	chs, err := cch.state(chatID)
	if err != nil {
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
	case COMMAND_SET_COMMUNITY_DESC:
		cch.receiveCommunityDescription(state[chatID], ctx, b, update)
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

	c, err := cch.configByChatID(chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	message := `Current configuration:

<b>Token name:</b> %s
<b>Hashtag(s):</b> %s
<b>Cashtag(s):</b> %s
<b>Community:</b> %s`

	message = fmt.Sprintf(
		message,
		cch.displayConfigValue(c.Token),
		cch.displayConfigValue(c.Hashtags),
		cch.displayConfigValue(c.Cashtags),
		cch.displayConfigValue(c.Community),
	)

	sendMessageParams := &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: cch.configKeyboard(b),
	}

	chs.lastPrompts, _ = cch.tgh.DeleteAllMessages(ctx, chatID, chs.lastPrompts)

	prompt, err := b.SendMessage(ctx, sendMessageParams)
	if err != nil {
		cch.logger.Error(
			"failed to send config main menu",
			zap.Int64("chatID", chatID),
			zap.Error(err),
		)
	}

	chs.lastPrompts = append(chs.lastPrompts, prompt)
}

// displayConfigValue
func (cch *configCommandHandler) displayConfigValue(value string) string {
	if value == "" {
		value = "<i>Not set</i>"
	}

	return value
}

// Reset
func (cch *configCommandHandler) Reset(ctx context.Context, b *bot.Bot, chatID int64) {}

// Cancel
func (cch *configCommandHandler) Cancel(chatID int64) {
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	chs.activeCommand = COMMAND_NONE
	chs.done = false
	cch.updateState(chatID, chs)
}

// Done
func (cch *configCommandHandler) Done(chatID int64) bool {
	chs, err := cch.state(chatID)
	if err != nil {
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

// onCancel
func (cch *configCommandHandler) onCancel(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	chs.lastPrompts, _ = cch.tgh.DeleteAllMessages(ctx, chatID, chs.lastPrompts)
	cch.Cancel(chatID)
}

// onConfigDone
func (cch *configCommandHandler) onConfigDone(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	chs.lastPrompts, _ = cch.tgh.DeleteAllMessages(ctx, chatID, chs.lastPrompts)
}

// onClear
func (cch *configCommandHandler) onClear(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	c, err := cch.configByChatID(chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	switch chs.activeCommand {
	case COMMAND_SET_HASH_TAGS:
		c.Hashtags = ""
		if err = c.Update(&c); err != nil {
			cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
			cch.logger.Error(
				"an error occurred trying to clear the hashtags in the config",
				zap.String("command", COMMAND_SET_COMMUNITY_DESC),
				zap.Int64("chatID", chatID),
				zap.Error(err),
			)
			return
		}
	case COMMAND_SET_CASH_TAGS:
		c.Cashtags = ""
		if err = c.Update(&c); err != nil {
			cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
			cch.logger.Error(
				"an error occurred trying to clear the cashtags in the config",
				zap.String("command", COMMAND_SET_COMMUNITY_DESC),
				zap.Int64("chatID", chatID),
				zap.Error(err),
			)
			return
		}
	case COMMAND_SET_COMMUNITY_DESC:
		c.Community = ""
		if err = c.Update(&c); err != nil {
			cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
			cch.logger.Error(
				"an error occurred trying to clear the community description in the config",
				zap.String("command", COMMAND_SET_COMMUNITY_DESC),
				zap.Int64("chatID", chatID),
				zap.Error(err),
			)
			return
		}
	}

	chs.lastPrompts, _ = cch.tgh.DeleteLastMessage(ctx, chatID, chs.lastPrompts)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// onConfigSetTokenName
func (cch *configCommandHandler) onConfigSetTokenName(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	message := "What is the name of your token?"

	prompt, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelKeyboard(b),
	})

	if err != nil {
		cch.logger.Error(
			"failed to send \"set token name\" command prompt",
			zap.Int64("chatID", chatID),
			zap.Error(err),
		)
	}

	chs.activeCommand = COMMAND_SET_TOKEN_NAME
	chs.done = false
	chs.lastPrompts = append(chs.lastPrompts, prompt)
	cch.updateState(chatID, chs)
}

// onConfigSetHashtags
func (cch *configCommandHandler) onConfigSetHashtags(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	message := `What hashtag should I use when shilling your token? 
	
You can set multiple hashtags but I'll only use one or two at a time.

Multiple hashtags should be separated by spaces e.g.
#MyAwesomeToken #MYTOKENTOTHEMOON #MyTokenIsTheBest

Make sure your primary hashtag is set first.`

	prompt, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelClearKeyboard(b),
	})

	if err != nil {
		cch.logger.Error(
			"failed to send \"set hashtags\" command prompt",
			zap.Int64("chatID", chatID),
			zap.Error(err),
		)
	}

	chs.activeCommand = COMMAND_SET_HASH_TAGS
	chs.done = false
	chs.lastPrompts = append(chs.lastPrompts, prompt)
	cch.updateState(chatID, chs)
}

// onConfigSetHashtags
func (cch *configCommandHandler) onConfigSetCashtags(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	message := `What cashtag should I use when shilling your token?

e.g. $MYTOKEN
`

	prompt, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelClearKeyboard(b),
	})

	if err != nil {
		cch.logger.Error(
			"failed to send \"set cashtags\" command prompt",
			zap.Int64("chatID", chatID),
			zap.Error(err),
		)
	}

	chs.activeCommand = COMMAND_SET_CASH_TAGS
	chs.done = false
	chs.lastPrompts = append(chs.lastPrompts, prompt)
	cch.updateState(chatID, chs)
}

// onConfigSetCommunityDescription
func (cch *configCommandHandler) onConfigSetCommunityDescription(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	chs, err := cch.state(chatID)
	if err != nil {
		return
	}

	message := `Describe your community (max. 500 chars).
	
Your description will help me create better shill responses.`

	prompt, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: cch.backCancelClearKeyboard(b),
	})

	if err != nil {
		cch.logger.Error(
			"failed to send \"set community description\" command prompt",
			zap.Int64("chatID", chatID),
			zap.Error(err),
		)
	}

	chs.activeCommand = COMMAND_SET_COMMUNITY_DESC
	chs.done = false
	chs.lastPrompts = append(chs.lastPrompts, prompt)
	cch.updateState(chatID, chs)
}

// receiveTokenName
func (cch *configCommandHandler) receiveTokenName(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	tokenName := strings.TrimSpace(update.Message.Text)

	if !cch.validateTokenName(tokenName) {
		cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
		cch.tgh.DeleteLastMessage(ctx, chatID, chs.lastPrompts)
		prompt, err := cch.tgh.SendMessage(ctx, b, chatID, "Invalid token name, please try again", &models.ReplyParameters{})
		if err != nil {
			cch.DisplayMainMenu(ctx, b, chatID)
			return
		}

		chs.lastPrompts = append(chs.lastPrompts, prompt)
		cch.updateState(chatID, chs)
		return
	}

	c, err := cch.configByChatID(chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Token = strings.ToUpper(tokenName)
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

	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	chs.lastPrompts, _ = cch.tgh.DeleteLastMessage(ctx, chatID, chs.lastPrompts)

	cch.DisplayMainMenu(ctx, b, chatID)
}

// validateTokenName
func (cch *configCommandHandler) validateTokenName(tokenName string) bool {
	re := regexp.MustCompile(`^[\p{L}\p{N}]+$`)
	return re.MatchString(tokenName)
}

// receiveHashTags
func (cch *configCommandHandler) receiveHashTags(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	hashtags := strings.TrimSpace(update.Message.Text)

	if !cch.validateHashtags(hashtags) {
		cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
		if len(chs.lastPrompts) > 1 {
			chs.lastPrompts, _ = cch.tgh.DeleteLastMessage(ctx, chatID, chs.lastPrompts)
		}
		prompt, err := cch.tgh.SendMessage(ctx, b, chatID, "Invalid hashtags, please try again", &models.ReplyParameters{})
		if err != nil {
			cch.DisplayMainMenu(ctx, b, chatID)
			return
		}

		chs.lastPrompts = append(chs.lastPrompts, prompt)
		cch.updateState(chatID, chs)
		return
	}

	c, err := cch.configByChatID(chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Hashtags = hashtags
	if err = c.Update(&c); err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		cch.logger.Error(
			"an error occurred trying to update the hashtags in the config",
			zap.String("command", COMMAND_SET_HASH_TAGS),
			zap.Int64("chatID", chatID),
			zap.String("hashtags", hashtags),
			zap.Error(err),
		)
		return
	}

	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	chs.lastPrompts, _ = cch.tgh.DeleteAllMessages(ctx, chatID, chs.lastPrompts)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// validateHashtags
func (cch *configCommandHandler) validateHashtags(hashtags string) bool {
	// Regular expression for validating hashtags, including Unicode characters
	re := regexp.MustCompile(`^(#[\p{L}\p{N}_]+)( #[\p{L}\p{N}_]+)*$`)
	return re.MatchString(hashtags)
}

// receiveCashTags
func (cch *configCommandHandler) receiveCashTags(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	cashtags := strings.TrimSpace(update.Message.Text)

	if !cch.validateCashtags(cashtags) {
		cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
		if len(chs.lastPrompts) > 1 {
			chs.lastPrompts, _ = cch.tgh.DeleteLastMessage(ctx, chatID, chs.lastPrompts)
		}
		prompt, err := cch.tgh.SendMessage(ctx, b, chatID, "Invalid cashtags, please try again", &models.ReplyParameters{})
		if err != nil {
			cch.DisplayMainMenu(ctx, b, chatID)
			return
		}

		chs.lastPrompts = append(chs.lastPrompts, prompt)
		cch.updateState(chatID, chs)
		return
	}

	c, err := cch.configByChatID(chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Cashtags = cashtags
	if err = c.Update(&c); err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		cch.logger.Error(
			"an error occurred trying to update the cashtags in the config",
			zap.String("command", COMMAND_SET_CASH_TAGS),
			zap.Int64("chatID", chatID),
			zap.String("cashtags", cashtags),
			zap.Error(err),
		)
		return
	}

	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	chs.lastPrompts, _ = cch.tgh.DeleteAllMessages(ctx, chatID, chs.lastPrompts)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// validateCashtags
func (cch *configCommandHandler) validateCashtags(hashtags string) bool {
	re := regexp.MustCompile(`^(\$[A-Za-z0-9]+)( \$[A-Za-z0-9]+)*$`)
	return re.MatchString(hashtags)
}

// receiveCommunityDescription
func (cch *configCommandHandler) receiveCommunityDescription(chs configHandlerState, ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	community := strings.TrimSpace(update.Message.Text)

	if !cch.validateCommnunityDescription(community) {
		cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
		if len(chs.lastPrompts) > 1 {
			chs.lastPrompts, _ = cch.tgh.DeleteLastMessage(ctx, chatID, chs.lastPrompts)
		}
		prompt, err := cch.tgh.SendMessage(ctx, b, chatID, "Invalid description, please try again", &models.ReplyParameters{})
		if err != nil {
			cch.DisplayMainMenu(ctx, b, chatID)
			return
		}

		chs.lastPrompts = append(chs.lastPrompts, prompt)
		cch.updateState(chatID, chs)
		return
	}

	c, err := cch.configByChatID(chatID)
	if err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		return
	}

	c.Community = community
	if err = c.Update(&c); err != nil {
		cch.tgh.SendErrorTryAgainMessage(ctx, b, chatID)
		cch.logger.Error(
			"an error occurred trying to update the community description in the config",
			zap.String("command", COMMAND_SET_COMMUNITY_DESC),
			zap.Int64("chatID", chatID),
			zap.String("community", community),
			zap.Error(err),
		)
		return
	}

	cch.tgh.DeleteMessage(ctx, chatID, update.Message.ID)
	chs.lastPrompts, _ = cch.tgh.DeleteAllMessages(ctx, chatID, chs.lastPrompts)
	cch.DisplayMainMenu(ctx, b, chatID)
}

// validateCommnunityDescription
func (cch *configCommandHandler) validateCommnunityDescription(community string) bool {
	return len(community) <= 500
}

// configByChatID
func (cch *configCommandHandler) configByChatID(chatID int64) (config.Config, error) {
	c, found, err := config.ConfigByChatID(cch.mongo, chatID)
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
		c = config.NewConfig(cch.mongo)
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

// state
func (cch *configCommandHandler) state(chatID int64) (configHandlerState, error) {
	chs, ok := state[chatID]
	if !ok {
		return configHandlerState{}, errors.New("failed to fetch config state")
	}

	return chs, nil
}
