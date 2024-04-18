package tghelper

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/keyboard/inline"
	"go.uber.org/zap"
)

type TGHelper struct {
	bot    *bot.Bot
	logger *zap.Logger
}

// NewAction
func NewTGHelper(bot *bot.Bot, logger *zap.Logger) TGHelper {
	return TGHelper{
		bot:    bot,
		logger: logger,
	}
}

// SendMessage
func (tgh *TGHelper) SendMessage(ctx context.Context, b *bot.Bot, chatID int64, message string, replyParams *models.ReplyParameters) (*models.Message, error) {
	params := &bot.SendMessageParams{
		ChatID:          chatID,
		Text:            message,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: replyParams,
	}

	// fmt.Printf("quote params %+v\n", params.ReplyParameters)

	sentMessage, err := b.SendMessage(ctx, params)
	if err != nil {
		tgh.logger.Error(
			"an error occurred trying to send a message",
			zap.Error(err),
		)
	}

	return sentMessage, err
}

// SendMessageWithCancel
func (tgh *TGHelper) SendMessageWithCancel(ctx context.Context, b *bot.Bot, chatID int64, message string) (*models.Message, error) {
	kb := inline.New(b).
		Button("Cancel", []byte("cancel"), tgh.onKeyboardCancel)

	return b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: kb,
	})
}

// SendMessageWithBackOrCancel
func (tgh *TGHelper) SendMessageWithBackOrCancel(ctx context.Context, b *bot.Bot, chatID int64, message string) (*models.Message, error) {
	kb := inline.New(b).
		Row().
		Button("Back", []byte("back"), tgh.onKeyboardBack).
		Button("Cancel", []byte("cancel"), tgh.onKeyboardCancel)

	return b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        message,
		ReplyMarkup: kb,
	})
}

// DeleteMessage
func (tgh *TGHelper) DeleteMessage(ctx context.Context, chatID int64, messageID int) (bool, error) {
	return tgh.bot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
}

// onKeyboardBackl
func (tgh *TGHelper) onKeyboardBack(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	// to-do
}

// onKeyboardCancel
func (tgh *TGHelper) onKeyboardCancel(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	chatID := mes.Message.Chat.ID
	tgh.SendCancelledMessage(ctx, b, chatID)
}

// SendCancelledMessage
func (tgh *TGHelper) SendCancelledMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "cancelled",
	})
}

// SendErrorTryAgainMessage
func (tgh *TGHelper) SendErrorTryAgainMessage(ctx context.Context, b *bot.Bot, chatID int64) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Oops, looks like we're having trouble, please try again.",
	})
}

// EscapeChars
func EscapeChars(input string) string {
	output := strings.ReplaceAll(input, ".", "\\.")
	output = strings.ReplaceAll(output, "!", "\\!")
	output = strings.ReplaceAll(output, "_", "\\_")
	output = strings.ReplaceAll(output, "<", "\\<")
	output = strings.ReplaceAll(output, ">", "\\>")
	output = strings.ReplaceAll(output, "=", "\\=")
	output = strings.ReplaceAll(output, "-", "\\-")

	return output
}
