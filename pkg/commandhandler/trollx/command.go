package trollx

import (
	"context"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/commandhandler/shillx"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type trollCommandHandler struct {
	commandhandler.Command
	sch shillx.ShillCommandHandler
}

// NewTrollCommandHandler
func NewTrollCommandHandler() commandhandler.CommandHandler {
	atom := zap.NewAtomicLevel()
	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	))
	defer logger.Sync()

	sch := shillx.ShillCommandHandler{}
	sch.SetLogger(logger)
	sch.SetMongo(storage.NewMongo())

	return &trollCommandHandler{
		sch: sch,
	}
}

// Handle
func (tch *trollCommandHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID

	shs, ok := tch.sch.State(chatID)
	if !ok {
		shs = shillx.ShillHandlerState{
			ChatID: chatID,
		}
	}
	shs.ReplyType = shillx.REPLY_TYPE_TROLL
	tch.sch.UpdateState(chatID, shs)
	tch.sch.Handle(ctx, b, update)
}

// Cancelled
func (tch *trollCommandHandler) Cancelled(chatID int64) bool {
	return tch.sch.Cancelled(chatID)
}

// PostCancel
func (tch *trollCommandHandler) Cancel(chatID int64) {
	tch.sch.Cancel(chatID)
}

// Done
func (tch *trollCommandHandler) Done(chatID int64) bool {
	return tch.sch.Done(chatID)
}
