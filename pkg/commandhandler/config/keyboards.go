package config

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/go-telegram/ui/keyboard/inline"
)

func (cch *configCommandHandler) configKeyboard(b *bot.Bot) *inline.Keyboard {
	return inline.New(b, inline.WithPrefix("config")).
		Row().
		Button("Set Token Name", []byte("setToken"), cch.onConfigSetTokenName).
		Button("Shill Instructions", []byte("setShillInstructions"), cch.onInlineKeyboardSelect).
		Row().
		Button("Set Hashtag(s)", []byte("setHashtags"), cch.onConfigSetHashTags).
		Button("Set Cashtag(s)", []byte("setCashtags"), cch.onConfigSetCashTags).
		Row().
		Button("Cancel", []byte("cancel"), cch.onInlineKeyboardSelect)
}

func (cch *configCommandHandler) backCancelKeyboard(b *bot.Bot) *inline.Keyboard {
	return inline.New(b, inline.WithPrefix("configBackCancel")).
		Row().
		Button("Back", []byte("back"), cch.onBack).
		Button("Cancel", []byte("cancel"), cch.onInlineKeyboardSelect)
}

// onInlineKeyboardSelect
func (cch *configCommandHandler) onInlineKeyboardSelect(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
	buttonValue := string(data)
	fmt.Printf("button presses %v\n", buttonValue)
}
