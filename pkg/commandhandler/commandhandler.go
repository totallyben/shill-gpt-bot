package commandhandler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type CommandHandler interface {
	Handle(ctx context.Context, b *bot.Bot, update *models.Update)
	Cancelled(chatID int64) bool
	Cancel(chatID int64)
	Reset(ctx context.Context, b *bot.Bot, chatID int64)
	Done(chatID int64) bool
}

type Command struct{}

func (c *Command) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {}
func (c *Command) Cancelled(chatId int64) bool {
	return false
}
func (c *Command) Cancel(chatId int64)                                 {}
func (c *Command) Reset(ctx context.Context, b *bot.Bot, chatID int64) {}
func (c *Command) Done(chatId int64) bool {
	return true
}
