package lib

import (
	"errors"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/rescue"
	"ns-rss/src/app"
	"ns-rss/src/app/config"
)

var help = `
/list 列出当前所有关键字

/add {keyword} 增加新的关键字

/delete {keyword} 删除关键字

/on 开启关键字通知

/off 关闭关键字通知

任何使用上的帮助或建议可以联系大管家 @hello\_cello\_bot

`
var tgBot *tgbotapi.BotAPI

func InitTgBotListen(cnf *config.Config) {

	defer func() {
		rescue.Recover()
	}()
	var err error
	tgBot, err = tgbotapi.NewBotAPI(cnf.TgToken)
	if err != nil {
		log.Fatalf("tgbotapi init failure: %v", err)
	}
	tgBot.Debug = false

	log.Infof("Authorized on account %s", tgBot.Self.UserName)

	go updates(cnf)

}

func updates(cfg *config.Config) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgBot.GetUpdatesChan(u)
	for update := range updates {
		processMessage(cfg, update)
	}
}

var processMutex sync.Mutex

func processMessage(cfg *config.Config, update tgbotapi.Update) {
	defer func() {
		rescue.Recover()
	}()
	message := update.Message
	if message == nil {
		return
	}
	entry := log.WithField("message", message.Text).
		WithField("from", message)
	entry.Info("receive message")
	var chatType = config.ChatTypeChat
	//判断来源类型
	if message.Chat.IsChannel() {
		chatType = config.ChatTypeChannel
	} else if message.Chat.IsGroup() {
		chatType = config.ChatTypeGroup
	}

	processMutex.Lock()
	defer processMutex.Unlock()
	//判断个人是否在配置文件中
	var currentChannel *config.ChannelInfo
	for i, info := range cfg.Channels {
		//兼容原数据
		if strings.TrimSpace(info.Type) == "" {
			info.Type = config.ChatTypeChannel
		}
		if info.ChatId == message.From.ID && info.Type == chatType {
			currentChannel = cfg.Channels[i]
			break
		}
	}
	if currentChannel == nil {
		currentChannel = &config.ChannelInfo{
			Name:     message.From.UserName,
			ChatId:   message.From.ID,
			Keywords: []string{},
			Type:     chatType,
		}
		cfg.Channels = append(cfg.Channels, currentChannel)
		cfg.Storage(app.ConfigFilePath)
		//第一次添加，发送欢迎消息
		tgBot.Send(tgbotapi.NewMessage(message.Chat.ID, "欢迎使用 NS 论坛关键字通知功能，这是您的首次使用, 请用 /help 查看帮助说明"))
	}

	text := update.Message.Text

	//处理命令
	if strings.TrimSpace(text) == "" {
		return
	}
	text = strings.TrimSpace(text)
	var msg *tgbotapi.MessageConfig
	var err error
	switch {
	case strings.HasPrefix(text, "/list"):
		msg, err = processListEvent(cfg, currentChannel)
	case strings.HasPrefix(text, "/add"):
		msg, err = processAddEvent(cfg, text, currentChannel)
	case strings.HasPrefix(text, "/delete"):
		msg, err = processDeleteEvent(cfg, text, currentChannel)
	case strings.HasPrefix(text, "/help"):
		m := tgbotapi.NewMessage(currentChannel.ChatId, help)
		msg = &m
	case strings.HasPrefix(text, "/on"):
		currentChannel.Status = "on"
		cfg.Storage(app.ConfigFilePath)
		m := tgbotapi.NewMessage(currentChannel.ChatId, "关键字通知已成功开启")
		msg = &m
	case strings.HasPrefix(text, "/off"):
		currentChannel.Status = "off"
		cfg.Storage(app.ConfigFilePath)
		m := tgbotapi.NewMessage(currentChannel.ChatId, "关键字通知已成功关闭")
		msg = &m
	default:
		return
	}

	msg.ParseMode = tgbotapi.ModeMarkdown
	if err != nil {
		tgBot.Send(tgbotapi.NewMessage(currentChannel.ChatId, err.Error()))
		return
	}

	result, err := tgBot.Send(msg)
	log.WithField("msg", msg.Text)
	if err != nil {
		log.WithField("msg", msg.Text).
			WithField("error", err).
			Error("send message  failure")
	} else {
		log.WithField("msg", msg.Text).
			WithField("result id", result.MessageID).
			Info("send message success")
	}
}

func processDeleteEvent(cfg *config.Config, postText string, currentChannel *config.ChannelInfo) (*tgbotapi.MessageConfig, error) {
	words := strings.Split(postText, " ")
	var deletes = make(map[string]struct{})
	var delWords []string
	words = funk.FilterString(words, func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	words = funk.UniqString(words)

	if len(words) == 1 {
		return nil, errors.New("请输入你要删除的关键字, 例如: /delete keyword")
	}
	words = words[1:]
	for _, word := range words {
		if word == "" {
			continue
		}

		for _, v := range currentChannel.Keywords {
			_, ok := deletes[v]
			if strings.ToLower(v) == strings.ToLower(word) && !ok {
				deletes[word] = struct{}{}
				delWords = append(delWords, word)
			}
		}
	}

	var newWords []string

	for _, v := range currentChannel.Keywords {
		if _, ok := deletes[v]; !ok {
			newWords = append(newWords, v)
		}
	}

	currentChannel.Keywords = newWords
	cfg.Storage(app.ConfigFilePath)
	msg := tgbotapi.NewMessage(currentChannel.ChatId, "关键字删除成功 "+strings.Join(delWords, " , "))
	return &msg, nil
}

func processAddEvent(cfg *config.Config, postText string, currentChannel *config.ChannelInfo) (*tgbotapi.MessageConfig, error) {
	words := strings.Split(postText, " ")
	words = funk.FilterString(words, func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	words = funk.UniqString(words)
	if len(words) == 1 {
		return nil, errors.New("请输入你要添加的关键字, 例如: /add keyword")
	}
	currentChannel.Keywords = append(currentChannel.Keywords, words[1:]...)

	cfg.Storage(app.ConfigFilePath)
	msg := tgbotapi.NewMessage(currentChannel.ChatId, "关键字添加成功 "+strings.Join(words[1:], " , "))
	return &msg, nil
}

func processListEvent(cfg *config.Config, channel *config.ChannelInfo) (*tgbotapi.MessageConfig, error) {
	var keywords []string
	for _, info := range cfg.Channels {
		if info.ChatId == channel.ChatId {
			keywords = append(keywords, info.Keywords...)
		}
	}
	keywords = funk.UniqString(keywords)
	keywords = funk.FilterString(keywords, func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	msg := tgbotapi.NewMessage(channel.ChatId, "当前关键字: "+strings.Join(keywords, " , "))
	return &msg, nil
}

func TgBotInstance() *tgbotapi.BotAPI {
	return tgBot
}
