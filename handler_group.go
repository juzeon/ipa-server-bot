package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"github.com/life4/genesis/slices"
	"howett.net/plist"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

type HandlerGroup struct {
}
type Update struct {
	*models.Update
	bot *bot.Bot
	ctx context.Context
}

func (o Update) GetCommand() string {
	return strings.Split(strings.Split(o.Message.Text[1:], "@")[0], " ")[0]
}
func (o Update) GetArgumentString() string {
	arr := strings.Split(o.Message.Text, " ")
	if len(arr) <= 1 {
		return ""
	}
	return strings.Join(arr[1:], " ")
}
func (o Update) MustSendReplyMessage(text string) *models.Message {
	msg, err := o.SendReplyMessage(text)
	if err != nil {
		panic(err)
	}
	return msg
}
func (o Update) SendReplyMessage(text string) (*models.Message, error) {
	return o.bot.SendMessage(o.ctx, &bot.SendMessageParams{
		ChatID:           o.Message.Chat.ID,
		Text:             text,
		ReplyToMessageID: o.Message.ID,
		ParseMode:        "HTML",
	})
}
func (o Update) GetPayload() string {
	text := ""
	if o.Message.ReplyToMessage != nil {
		text = o.Message.ReplyToMessage.Text
		if text == "" {
			text = o.Message.ReplyToMessage.Caption
		}
	}
	if arg := o.GetArgumentString(); arg != "" {
		if text != "" {
			text += "\n\n"
		}
		text += arg
	}
	return text
}
func (o Update) MustGetPayload() string {
	payload := o.GetPayload()
	if payload == "" {
		panic("cannot get payload")
	}
	return payload
}

func WrapHandlerGroupFunc(fun func(update *Update)) bot.HandlerFunc {
	return func(ctx context.Context, botIns *bot.Bot, update *models.Update) {
		u := &Update{Update: update, bot: botIns, ctx: ctx}
		defer func() {
			if err := recover(); err != nil {
				slog.Error("recover from panic", err)
				_, err = u.SendReplyMessage(fmt.Sprintf("%v", err))
				if err != nil {
					slog.Error("cannot send error msg: ", err)
				}
			}
		}()
		fun(u)
	}
}
func NewHandlerGroup() *HandlerGroup {
	return &HandlerGroup{}
}
func (o *HandlerGroup) Start(update *Update) {
	payload := update.GetPayload()
	if strings.HasPrefix(payload, "app_") {
		o.GetApp(update)
	} else if strings.HasPrefix(payload, "del_") {
		o.DelApp(update)
	} else {
		update.MustSendReplyMessage("Send me a signed .ipa file and " +
			"I will generate a link to install it on your iOS devices directly!")
	}
}
func (o *HandlerGroup) DelApp(update *Update) {
	appUUID := update.MustGetPayload()[4:]
	session := NewSession(update.Message.Chat.ID)
	app, err := slices.Find(session.Applications, func(el Application) bool {
		return el.UUID == appUUID
	})
	if err != nil {
		panic(err)
	}
	session.Applications = slices.Delete(session.Applications, app)
	session.Save()
	update.MustSendReplyMessage("Application <b>" + app.Name + "</b> has been deleted.")
}
func (o *HandlerGroup) GetApp(update *Update) {
	appUUID := update.MustGetPayload()[4:]
	session := NewSession(update.Message.Chat.ID)
	app, err := slices.Find(session.Applications, func(el Application) bool {
		return el.UUID == appUUID
	})
	if err != nil {
		panic(err)
	}
	update.MustSendReplyMessage(BuildAppInfoTemplate(app))
}
func (o *HandlerGroup) List(update *Update) {
	session := NewSession(update.Message.Chat.ID)
	botUser, err := update.bot.GetMe(update.ctx)
	if err != nil {
		panic(err)
	}
	_, err = update.bot.SendMessage(update.ctx, &bot.SendMessageParams{
		ChatID:                update.Message.Chat.ID,
		Text:                  BuildAppListTemplate(session.Applications, botUser.Username),
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	if err != nil {
		panic(err)
	}
}
func (o *HandlerGroup) UploadIPA(update *Update) {
	if !strings.HasSuffix(update.Update.Message.Document.FileName, ".ipa") {
		update.MustSendReplyMessage("Please upload a .ipa file")
		return
	}
	ipaFile, err := update.bot.GetFile(update.ctx, &bot.GetFileParams{FileID: update.Update.Message.Document.FileID})
	if err != nil {
		panic(err)
	}
	processingMessage := update.MustSendReplyMessage("Processing your .ipa file...")
	defer update.bot.DeleteMessage(update.ctx, &bot.DeleteMessageParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: processingMessage.ID,
	})
	ipaBytes, err := DownloadTelegramFile(ipaFile.FilePath)
	if err != nil {
		panic(err)
	}
	r, err := zip.NewReader(bytes.NewReader(ipaBytes), int64(len(ipaBytes)))
	if err != nil {
		panic(err)
	}
	uid := uuid.New().String()
	application := Application{CreatedAt: time.Now(), UUID: uid}
	for _, file := range r.File {
		readName := ""
		if strings.HasSuffix(file.Name, ".app/embedded.mobileprovision") {
			readName = "mobileprovision"
		}
		if strings.HasSuffix(file.Name, ".app/Info.plist") {
			readName = "info.plist"
		}
		if readName == "" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			panic(err)
		}
		v, err := io.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		switch readName {
		case "mobileprovision":
			if group := regexp.MustCompile("<plist([\\s\\S]*?)</plist>").FindSubmatch(v); len(group) > 0 {
				mobileprovision := string(group[0])
				if g := regexp.MustCompile("<key>CreationDate</key>[\\s\\S]*?<date>(.*?)</date>").
					FindStringSubmatch(mobileprovision); len(g) > 0 {
					application.CertCreatedAt, _ = time.Parse(time.RFC3339, g[1])
				}
				if g := regexp.MustCompile("<key>ExpirationDate</key>[\\s\\S]*?<date>(.*?)</date>").
					FindStringSubmatch(mobileprovision); len(g) > 0 {
					application.CertExpiredAt, _ = time.Parse(time.RFC3339, g[1])
				}
				if g := regexp.MustCompile("<key>TeamName</key>[\\s\\S]*?<string>(.*?)</string>").
					FindStringSubmatch(mobileprovision); len(g) > 0 {
					application.CertOrg = g[1]
				}
			}
		case "info.plist":
			plistV := map[string]any{}
			_, err = plist.Unmarshal(v, &plistV)
			if err != nil {
				panic(err)
			}
			application.Name = plistV["CFBundleDisplayName"].(string)
			application.Package = plistV["CFBundleIdentifier"].(string)
			application.Version = plistV["CFBundleShortVersionString"].(string)
		}
	}
	ipaURL, err := UploadS3(ipaBytes, uid+"/0.ipa", "application/octet-stream")
	if err != nil {
		panic(err)
	}
	application.IPA = ipaURL
	plistContent := application.BuildPlistContent()
	plistURL, err := UploadS3([]byte(plistContent), uid+"/manifest.plist", "text/xml")
	if err != nil {
		panic(err)
	}
	application.Plist = plistURL
	installURL, err := UploadS3([]byte(application.BuildInstallPageContent()),
		uid+"/install.html", "text/html")
	if err != nil {
		panic(err)
	}
	application.InstallPage = installURL
	session := NewSession(update.Message.Chat.ID)
	session.Applications = append(session.Applications, application)
	session.Save()
	update.MustSendReplyMessage(BuildAppInfoTemplate(application))
}
