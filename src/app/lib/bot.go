package lib

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
)

type NotifyMessage struct {
	Text   string
	ChatId *int64
}

type BotNotifier interface {
	Notify(NotifyMessage)
}

type TelegramNotifier struct {
	botToken string
	chatId   string
}

func NewTelegramNotifier(botToken string, chatId string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatId:   chatId,
	}
}

var Replacer = strings.NewReplacer("_", "\\_",
	//"[", "\\[",
	//"]", "\\]",
	"(", "\\(",
	")", "\\)",
	//"`", "\\`",
	//">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	//"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!")

func (t *TelegramNotifier) Notify(msg NotifyMessage) {
	tg := TgBotInstance()
	if tg == nil {
		return
	}

	defer func() {
		rescue.Recover()
	}()

	tgMsg := tgbotapi.NewMessage(cast.ToInt64(t.chatId), Replacer.Replace(msg.Text))
	if msg.ChatId != nil {
		tgMsg.ChatID = *msg.ChatId
	}
	logx.ContextWithFields(context.Background(), logx.Field("msg", tgMsg.Text), logx.Field("chatId", tgMsg.ChatID))
	tgMsg.ParseMode = tgbotapi.ModeMarkdownV2
	tgMsg.DisableWebPagePreview = false
	_, e := tg.Send(tgMsg)
	if e != nil {
		logx.Errorw("send telegram message error", logx.Field("error", e))
	} else {
		logx.Infow("send telegram message")
	}

}
