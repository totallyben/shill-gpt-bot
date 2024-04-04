package api

import (
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

func openaiClient() *openai.Client {
	openaiToken := viper.GetString("openai.token")
	return openai.NewClient(openaiToken)
}
