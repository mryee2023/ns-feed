package lib

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
)

type NotifyMessage struct {
	Text    string
	ChatId  *int64
	MsgType string //chat, group, channel
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
	"|", "\\|",
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

	tgMsg.ParseMode = tgbotapi.ModeMarkdownV2
	tgMsg.DisableWebPagePreview = false
	v, e := tg.Send(tgMsg)
	if e != nil {
		logx.Errorw("send telegram message failure", logx.Field("error", e), logx.Field("msg", msg.Text), logx.Field("chatId", tgMsg.ChatID))
	} else {
		logx.Infow("send telegram message success", logx.Field("result", v.MessageID), logx.Field("msg", msg.Text), logx.Field("chatId", tgMsg.ChatID))
	}

}
