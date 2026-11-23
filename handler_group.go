package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/botapi"
	"github.com/life4/genesis/slices"
)

var pagePattern = regexp.MustCompile(`\(Page: (\d+)/(\d+)\)`)

type HandlerGroup struct {
}
type Update struct {
	*botapi.Update
	bot *botapi.Bot
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
func (o Update) MustSendReplyMessage(text string) *botapi.Message {
	msg, err := o.SendReplyMessage(text)
	if err != nil {
		panic(err)
	}
	return msg
}
func (o Update) SendReplyMessage(text string) (*botapi.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("send message failed", "err", redactSecrets(fmt.Sprint(err)))
		}
	}()
	text = redactSecrets(text)
	opts := []botapi.SendOption{botapi.WithParseMode(botapi.ParseModeHTML)}
	if o.Message != nil {
		opts = append(opts, botapi.ReplyTo(o.Message.MessageID))
		return o.bot.SendMessage(o.ctx, botapi.ID(o.Message.Chat.ID), text, opts...)
	} else if o.CallbackQuery != nil && o.CallbackQuery.Message != nil {
		return o.bot.SendMessage(o.ctx, botapi.ID(o.CallbackQuery.Message.Chat.ID), text, opts...)
	}
	panic("no target to reply")
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
		panic(userErr("That command needs more detail."))
	}
	return payload
}
func (o Update) botUsername() string {
	me := o.bot.Self()
	if me == nil || me.Username == "" {
		panic(errors.New("bot identity is unavailable"))
	}
	return me.Username
}

func isPrivateChat(update *botapi.Update) bool {
	if update == nil {
		return false
	}
	if update.Message != nil {
		return update.Message.Chat.Type == botapi.ChatTypePrivate
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		return update.CallbackQuery.Message.Chat.Type == botapi.ChatTypePrivate
	}
	return false
}

func WrapHandlerGroupFunc(fun func(update *Update)) botapi.Handler {
	return func(c *botapi.Context) error {
		u := &Update{Update: c.Update, bot: c.Bot, ctx: c}
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			reply, logDetail := handlerFailureMessage(rec)
			if logDetail != "" {
				slog.Error("handler failed", "err", logDetail)
			}
			if _, err := u.SendReplyMessage(reply); err != nil {
				slog.Error("cannot send message", "err", redactSecrets(err.Error()))
			}
		}()
		if !isPrivateChat(c.Update) {
			if c.Update != nil && (c.Update.Message != nil || (c.Update.CallbackQuery != nil && c.Update.CallbackQuery.Message != nil)) {
				_, _ = u.SendReplyMessage("This bot only works in private chats.")
			}
			return nil
		}
		fun(u)
		return nil
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
	if update.Message == nil {
		panic(userErr("Application not found."))
	}
	payload := update.MustGetPayload()
	if !strings.HasPrefix(payload, "del_") || len(payload) <= 4 {
		panic(userErr("Application not found."))
	}
	appUUID := payload[4:]
	expireChat(update.Message.Chat.ID, time.Now())
	var deletedName string
	withSession(update.Message.Chat.ID, func(session *Session) {
		app, err := slices.Find(session.Applications, func(el Application) bool {
			return el.UUID == appUUID
		})
		if err != nil {
			panic(userErr("Application not found."))
		}
		if err := DeleteApplicationObjects(app.UUID); err != nil {
			slog.Error("delete application files", "err", redactSecrets(err.Error()))
			panic(userErr("Could not delete the stored files. Try again later."))
		}
		session.Applications = slices.Delete(session.Applications, app)
		session.Save()
		deletedName = app.Name
	})
	update.MustSendReplyMessage("Application <b>" + htmlText(deletedName) + "</b> has been deleted.")
}
func (o *HandlerGroup) GetApp(update *Update) {
	if update.Message == nil {
		panic(userErr("Application not found."))
	}
	payload := update.MustGetPayload()
	if !strings.HasPrefix(payload, "app_") || len(payload) <= 4 {
		panic(userErr("Application not found."))
	}
	appUUID := payload[4:]
	expireChat(update.Message.Chat.ID, time.Now())
	var app Application
	var found bool
	withSession(update.Message.Chat.ID, func(session *Session) {
		foundApp, err := slices.Find(session.Applications, func(el Application) bool {
			return el.UUID == appUUID
		})
		if err != nil {
			return
		}
		app = foundApp
		found = true
	})
	if !found {
		panic(userErr("Application not found."))
	}
	update.MustSendReplyMessage(BuildAppInfoTemplate(app))
}

func listKeyboard() *botapi.InlineKeyboardMarkup {
	return botapi.InlineKeyboard([]botapi.InlineKeyboardButton{
		botapi.InlineButtonData("Previous Page", "list_previous_page"),
		botapi.InlineButtonData("Next Page", "list_next_page"),
	})
}

func (o *HandlerGroup) List(update *Update) {
	if update.Message == nil {
		return
	}
	expireChat(update.Message.Chat.ID, time.Now())
	var apps []Application
	withSession(update.Message.Chat.ID, func(session *Session) {
		apps = append([]Application(nil), session.Applications...)
	})
	str, err := BuildAppListTemplate(apps, update.botUsername(), 1)
	if err != nil {
		update.MustSendReplyMessage("error building app list template: " + err.Error())
		return
	}
	_, err = update.bot.SendMessage(update.ctx, botapi.ID(update.Message.Chat.ID), str,
		botapi.WithParseMode(botapi.ParseModeHTML),
		botapi.DisableWebPagePreview(),
		botapi.WithReplyMarkup(listKeyboard()),
	)
	if err != nil {
		panic(err)
	}
}
func (o *HandlerGroup) ListSwitchPage(update *Update) {
	answerText := ""
	defer func() {
		if update.CallbackQuery == nil {
			return
		}
		if err := update.bot.AnswerCallbackQuery(update.ctx, update.CallbackQuery.ID, botapi.WithCallbackText(answerText)); err != nil {
			slog.Error("answer callback", "err", redactSecrets(err.Error()))
		}
	}()
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return
	}
	match := pagePattern.FindStringSubmatch(update.CallbackQuery.Message.Text)
	if len(match) < 2 {
		panic(userErr("This page is no longer available."))
	}
	currentPage, err := strconv.Atoi(match[1])
	if err != nil {
		panic(userErr("This page is no longer available."))
	}
	if len(update.CallbackQuery.Data) < 5 {
		return
	}
	expireChat(update.CallbackQuery.Message.Chat.ID, time.Now())
	var apps []Application
	withSession(update.CallbackQuery.Message.Chat.ID, func(session *Session) {
		apps = append([]Application(nil), session.Applications...)
	})
	template := ""
	switch update.CallbackQuery.Data[5:] {
	case "previous_page":
		t, err := BuildAppListTemplate(apps, update.botUsername(), currentPage-1)
		if err != nil {
			answerText = err.Error()
			return
		}
		template = t
	case "next_page":
		t, err := BuildAppListTemplate(apps, update.botUsername(), currentPage+1)
		if err != nil {
			answerText = err.Error()
			return
		}
		template = t
	default:
		return
	}
	_, err = update.bot.EditMessageText(update.ctx, botapi.ID(update.CallbackQuery.Message.Chat.ID), update.CallbackQuery.Message.MessageID, template,
		botapi.WithParseMode(botapi.ParseModeHTML),
		botapi.DisableWebPagePreview(),
		botapi.WithReplyMarkup(listKeyboard()),
	)
	if err != nil {
		panic(err)
	}
}
func (o *HandlerGroup) UploadIPA(update *Update) {
	if update.Message == nil || update.Message.Document == nil || !isIPAFileName(update.Message.Document.FileName) {
		update.MustSendReplyMessage("Please upload a .ipa file")
		return
	}
	if update.Message.Document.FileSize > maxIPABytes {
		update.MustSendReplyMessage("The IPA file is too large.")
		return
	}
	expireChat(update.Message.Chat.ID, time.Now())
	tooMany := false
	withSession(update.Message.Chat.ID, func(session *Session) {
		tooMany = len(session.Applications) >= maxApplicationsPerChat
	})
	if tooMany {
		update.MustSendReplyMessage("You have too many applications. Delete one before uploading another.")
		return
	}
	if !acquireIPASlot() {
		update.MustSendReplyMessage("The bot is busy processing other files. Try again in a moment.")
		return
	}
	defer releaseIPASlot()
	processingMessage := update.MustSendReplyMessage("Processing your .ipa file...")
	defer update.bot.DeleteMessage(update.ctx, botapi.ID(update.Message.Chat.ID), processingMessage.MessageID)
	slog.Info("Begin download ipa")
	tmpPath, infoSize, err := writeLimitedTemp(maxIPABytes, func(w io.Writer) (int64, error) {
		return update.bot.DownloadFile(update.ctx, update.Message.Document.FileID, w)
	})
	if err != nil {
		if errors.Is(err, errTooLarge) {
			panic(userErr("The IPA file is too large."))
		}
		panic(err)
	}
	defer os.Remove(tmpPath)
	if infoSize == 0 {
		panic(userErr("The file is not a valid IPA."))
	}
	parsed, err := parseIPA(tmpPath)
	if err != nil {
		panic(err)
	}
	uid := uuid.New().String()
	slog.Info("parsed ipa", "uuid-generated", uid)
	application := parsed
	application.CreatedAt = time.Now()
	application.UUID = uid
	slog.Info("Application info", "name", application.Name, "package", application.Package)
	slog.Info("Uploading to S3...")
	ipaReader, err := os.Open(tmpPath)
	if err != nil {
		panic(err)
	}
	ipaURL, err := UploadS3(ipaReader, infoSize, uid+"/0.ipa", "application/octet-stream")
	closeErr := ipaReader.Close()
	if err != nil {
		panic(err)
	}
	if closeErr != nil {
		panic(closeErr)
	}
	slog.Info("Successfully uploaded to S3", "ipa-url", ipaURL)
	application.IPA = ipaURL
	plistContent, err := application.BuildPlistContent()
	if err != nil {
		panic(userErr("Could not build the install manifest."))
	}
	plistURL, err := UploadS3(bytes.NewReader([]byte(plistContent)), int64(len(plistContent)), uid+"/manifest.plist", "text/xml")
	if err != nil {
		panic(err)
	}
	application.Plist = plistURL
	installContent, err := application.BuildInstallPageContent()
	if err != nil {
		panic(userErr("Could not build the install page."))
	}
	installURL, err := UploadS3(bytes.NewReader([]byte(installContent)), int64(len(installContent)),
		uid+"/install.html", "text/html; charset=utf-8")
	if err != nil {
		panic(err)
	}
	application.InstallPage = installURL
	slog.Info("Final application", "name", application.Name)
	saved := false
	withSession(update.Message.Chat.ID, func(session *Session) {
		if len(session.Applications) >= maxApplicationsPerChat {
			return
		}
		session.Applications = append(session.Applications, application)
		session.Save()
		saved = true
	})
	if !saved {
		if err := DeleteApplicationObjects(uid); err != nil {
			slog.Error("cleanup rejected upload", "err", redactSecrets(err.Error()))
		}
		panic(userErr("You have too many applications. Delete one before uploading another."))
	}
	update.MustSendReplyMessage(BuildAppInfoTemplate(application))
}
