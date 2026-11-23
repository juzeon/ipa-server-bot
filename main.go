package main

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gotd/botapi"
	"github.com/gotd/botapi/storage"
)

//go:embed manifest.plist
var plistTemplate string

//go:embed install.html
var installPageTemplate string

var handlerGroup *HandlerGroup
var config *Config

func main() {
	path, err := parseConfigPath(os.Args[1:])
	if err != nil {
		slog.Error("config path", "err", err)
		os.Exit(2)
	}
	configPath = path
	config = NewConfig()
	if err := config.validate(); err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(2)
	}
	if err := prepareSessionFile(config.SessionFile); err != nil {
		slog.Error("session file", "err", redactSecrets(err.Error()))
		os.Exit(1)
	}
	store, err := storage.Open(config.SessionFile)
	if err != nil {
		slog.Error("open session", "err", redactSecrets(err.Error()))
		os.Exit(1)
	}
	defer store.Close()
	tightenOwnerOnly(config.SessionFile, 0o600)

	SetupS3()
	tgBot, err := botapi.New(config.Token, botapi.Options{
		AppID:   config.APIID,
		AppHash: config.APIHash,
		Storage: store,
	})
	if err != nil {
		slog.Error("create bot", "err", redactSecrets(err.Error()))
		os.Exit(1)
	}
	handlerGroup = NewHandlerGroup()
	tgBot.OnCommand("start", "Start the bot", WrapHandlerGroupFunc(handlerGroup.Start))
	tgBot.OnCommand("list", "List my applications", WrapHandlerGroupFunc(handlerGroup.List))
	tgBot.OnCallbackQuery(WrapHandlerGroupFunc(handlerGroup.ListSwitchPage), botapi.CallbackPrefix("list_"))
	tgBot.OnMessage(WrapHandlerGroupFunc(defaultHandler))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := tgBot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("bot stopped", "err", redactSecrets(err.Error()))
		os.Exit(1)
	}
}

func defaultHandler(update *Update) {
	if update.Message == nil {
		return
	}
	blocked := false
	withSession(update.Message.Chat.ID, func(session *Session) {
		blocked = session.State != SessionStateDefault
	})
	if blocked {
		return
	}
	if update.Message.Document == nil {
		update.MustSendReplyMessage("Please upload a .ipa file")
		return
	}
	handlerGroup.UploadIPA(update)
}
