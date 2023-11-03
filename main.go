package main

import (
	"context"
	_ "embed"
	"github.com/go-telegram/bot"
)

//go:embed manifest.plist
var plistTemplate string

//go:embed install.html
var installPageTemplate string
var handlerGroup *HandlerGroup
var config *Config

func main() {
	config = NewConfig()
	SetupClient()
	SetupS3()
	tgBot, err := bot.New(config.Token, bot.WithDebug(),
		bot.WithDefaultHandler(WrapHandlerGroupFunc(defaultHandler)),
		bot.WithServerURL(config.Server))
	if err != nil {
		panic(err)
	}
	handlerGroup = NewHandlerGroup()
	tgBot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix,
		WrapHandlerGroupFunc(handlerGroup.Start))
	ctx := context.Background()
	tgBot.Start(ctx)
}
func defaultHandler(update *Update) {
	if update.Message == nil {
		return
	}
	session := NewSession(update.Message.Chat.ID)
	if session.State != SessionStateDefault {
		return
	}
	if update.Message.Document == nil {
		update.MustSendReplyMessage("Please upload a .ipa file")
		return
	}
	handlerGroup.UploadIPA(update)
}
