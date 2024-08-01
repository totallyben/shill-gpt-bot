package config

import (
	"github.com/go-telegram/bot"
	"github.com/go-telegram/ui/keyboard/inline"
)

// configKeyboard
func (cch *configCommandHandler) configKeyboard(b *bot.Bot) *inline.Keyboard {
	return inline.New(b, inline.WithPrefix("config")).
		Row().
		Button("Set Token Name", []byte("setToken"), cch.onConfigSetTokenName).
		Button("Describe Community", []byte("setCommunityDescription"), cch.onConfigSetCommunityDescription).
		Row().
		Button("Set Hashtag(s)", []byte("setHashtags"), cch.onConfigSetHashtags).
		Button("Set Cashtag(s)", []byte("setCashtags"), cch.onConfigSetCashtags).
		Row().
		Button("Done", []byte("done"), cch.onConfigDone)
}

// backCancelKeyboard
func (cch *configCommandHandler) backCancelKeyboard(b *bot.Bot) *inline.Keyboard {
	return inline.New(b, inline.WithPrefix("configBackCancel")).
		Row().
		Button("Back", []byte("back"), cch.onBack).
		Button("Cancel", []byte("cancel"), cch.onCancel)
}

// backCancelClearKeyboard
func (cch *configCommandHandler) backCancelClearKeyboard(b *bot.Bot) *inline.Keyboard {
	return inline.New(b, inline.WithPrefix("configBackCancel")).
		Row().
		Button("Back", []byte("back"), cch.onBack).
		Button("Cancel", []byte("cancel"), cch.onCancel).
		Row().
		Button("Clear", []byte("back"), cch.onClear)
}
