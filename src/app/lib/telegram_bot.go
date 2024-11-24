package lib

import (
	"errors"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/logx"
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
`
var tgBot *tgbotapi.BotAPI

//var startTime = time.Now()

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
		if update.ChannelPost != nil { //频道发过来的消息
			processChannelPost(cfg, update)
			continue
		}
		if update.Message == nil {
			continue
		}
		if !update.Message.IsCommand() {
			continue
		}

		//var msg tgbotapi.MessageConfig
		//switch update.Message.Command() {
		//case "list":
		//	m := strings.Join(cfg.Keywords, " , ")
		//	msg = tgbotapi.NewMessage(update.Message.Chat.ID, m)
		//case "add":
		//	if len(update.Message.CommandArguments()) == 0 {
		//		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "请输入你要添加的关键字, 例如: /add keyword")
		//		break
		//	}
		//	cfg.Keywords = append(cfg.Keywords, update.Message.CommandArguments())
		//	msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("关键字添加成功 %s ", update.Message.CommandArguments()))
		//	cfg.Storage(app.ConfigFilePath)
		//case "delete":
		//	if len(update.Message.CommandArguments()) == 0 {
		//		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "请输入你要删除的关键字, 例如: /delete keyword")
		//		break
		//	}
		//	var keywords []string
		//	for _, v := range cfg.Keywords {
		//		if strings.ToLower(v) == strings.ToLower(update.Message.CommandArguments()) {
		//			msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("关键字删除成功  %s", update.Message.CommandArguments()))
		//		} else {
		//			keywords = append(keywords, v)
		//		}
		//	}
		//	cfg.Keywords = keywords
		//	cfg.Storage(app.ConfigFilePath)
		//default:
		//	msg = tgbotapi.NewMessage(update.Message.Chat.ID, "/list 列出当前所有关键字\n /add {keyword} 增加新的关键字\n /delete {keyword} 删除关键字")
		//}
		//msg.ParseMode = tgbotapi.ModeMarkdown
		//msg.ReplyToMessageID = update.Message.MessageID
		//
		//_, err := tgBot.Send(msg)
		//if err != nil {
		//	log.Errorf("send message failure: %v", err)
		//}
	}
}

var processMutex sync.Mutex

func processChannelPost(cfg *config.Config, update tgbotapi.Update) {
	channel := update.ChannelPost.Chat
	post := update.ChannelPost
	var msg *tgbotapi.MessageConfig
	var err error
	logx.Infow("receive channel post", logx.Field("channel", channel.ID), logx.Field("channel_name", channel.Title), logx.Field("post", post.Text))
	var currentChannel *config.ChannelInfo
	processMutex.Lock()
	defer processMutex.Unlock()
	for i, info := range cfg.Channels {
		if info.ChatId == channel.ID {
			currentChannel = cfg.Channels[i]
			break
		}
	}
	if currentChannel == nil {
		currentChannel = &config.ChannelInfo{
			Name:     channel.Title,
			ChatId:   channel.ID,
			Keywords: []string{},
		}
		cfg.Channels = append(cfg.Channels, currentChannel)
		//第一次添加，发送欢迎消息
		tgBot.Send(tgbotapi.NewMessage(channel.ID, "欢迎使用 NS 论坛关键字通知功能，这是您的首次使用, 请用 /help 查看帮助说明"))
	}
	if strings.TrimSpace(post.Text) == "" {
		return
	}
	post.Text = strings.TrimSpace(post.Text)

	switch {
	case strings.HasPrefix(post.Text, "/list"):
		msg, err = processListEvent(cfg, channel)
	case strings.HasPrefix(post.Text, "/add"):
		msg, err = processAddEvent(cfg, post.Text, channel, currentChannel)
	case strings.HasPrefix(post.Text, "/delete"):
		msg, err = processDeleteEvent(cfg, post.Text, channel, currentChannel)
	case strings.HasPrefix(post.Text, "/help"):
		m := tgbotapi.NewMessage(channel.ID, help)
		msg = &m
	case strings.HasPrefix(post.Text, "/on"):
		currentChannel.Status = "on"
		cfg.Storage(app.ConfigFilePath)
		m := tgbotapi.NewMessage(channel.ID, "关键字通知已成功开启")
		msg = &m
	case strings.HasPrefix(post.Text, "/off"):
		currentChannel.Status = "off"
		cfg.Storage(app.ConfigFilePath)
		m := tgbotapi.NewMessage(channel.ID, "关键字通知已成功关闭")
		msg = &m
	default:
		return
	}

	msg.ParseMode = tgbotapi.ModeMarkdown
	if err != nil {
		tgBot.Send(tgbotapi.NewMessage(channel.ID, err.Error()))
		return
	}

	_, err = tgBot.Send(msg)
	if err != nil {
		log.Errorf("send message to channel failure: %v", err)
	}
}

func processDeleteEvent(cfg *config.Config, postText string, channel *tgbotapi.Chat, currentChannel *config.ChannelInfo) (*tgbotapi.MessageConfig, error) {
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
	msg := tgbotapi.NewMessage(channel.ID, "关键字删除成功 "+strings.Join(delWords, " , "))
	return &msg, nil
}

func processAddEvent(cfg *config.Config, postText string, channel *tgbotapi.Chat, currentChannel *config.ChannelInfo) (*tgbotapi.MessageConfig, error) {
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
	msg := tgbotapi.NewMessage(channel.ID, "关键字添加成功 "+strings.Join(words[1:], " , "))
	return &msg, nil
}

func processListEvent(cfg *config.Config, channel *tgbotapi.Chat) (*tgbotapi.MessageConfig, error) {
	var keywords []string
	for _, info := range cfg.Channels {
		if info.ChatId == channel.ID {
			keywords = append(keywords, info.Keywords...)
		}
	}
	keywords = funk.UniqString(keywords)
	keywords = funk.FilterString(keywords, func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	msg := tgbotapi.NewMessage(channel.ID, "当前关键字: "+strings.Join(keywords, " , "))
	return &msg, nil
}

func TgBotInstance() *tgbotapi.BotAPI {
	return tgBot
}
