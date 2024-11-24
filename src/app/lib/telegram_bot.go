package lib

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
	"ns-rss/src/app"
	"ns-rss/src/app/config"
)

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

func processChannelPost(cfg *config.Config, update tgbotapi.Update) {
	channel := update.ChannelPost.Chat
	post := update.ChannelPost
	var msg tgbotapi.MessageConfig
	logx.Infow("receive channel post", logx.Field("channel", channel.ID), logx.Field("channel_name", channel.Title), logx.Field("post", post.Text))
	var currentChannel *config.ChannelInfo
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
	}
	if strings.TrimSpace(post.Text) == "" {
		return
	}
	post.Text = strings.TrimSpace(post.Text)

	if strings.HasPrefix(post.Text, "/add") {
		words := strings.Split(post.Text, " ")
		words = funk.FilterString(words, func(s string) bool {
			return strings.TrimSpace(s) != ""
		})
		words = funk.UniqString(words)
		if len(words) == 1 {
			msg = tgbotapi.NewMessage(channel.ID, "请输入你要添加的关键字, 例如: /add keyword")
			_, err := tgBot.Send(msg)
			if err != nil {
				log.Errorf("send message to channel failure: %v", err)
			}
			return
		}
		currentChannel.Keywords = append(currentChannel.Keywords, words[1:]...)

		cfg.Storage(app.ConfigFilePath)
		msg = tgbotapi.NewMessage(channel.ID, "关键字添加成功 "+strings.Join(words[1:], " , "))
	} else if strings.HasPrefix(post.Text, "/delete") {
		words := strings.Split(post.Text, " ")
		var keywords []string
		var deletes []string
		if len(words) == 1 {
			msg = tgbotapi.NewMessage(channel.ID, "请输入你要删除的关键字, 例如: /delete keyword")
			_, err := tgBot.Send(msg)
			if err != nil {
				log.Errorf("send message to channel failure: %v", err)
			}
			return
		}
		words = words[1:]
		words = funk.FilterString(words, func(s string) bool {
			return strings.TrimSpace(s) != ""
		})
		words = funk.UniqString(words)
		for _, word := range words {
			for _, v := range currentChannel.Keywords {
				if strings.ToLower(v) == strings.ToLower(word) {
					deletes = append(deletes, word)
				} else {
					keywords = append(keywords, v)
				}
			}
		}
		currentChannel.Keywords = keywords
		cfg.Storage(app.ConfigFilePath)
		msg = tgbotapi.NewMessage(channel.ID, "关键字删除成功 "+strings.Join(deletes, " , "))
	} else if strings.HasPrefix(post.Text, "/list") {
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
		msg = tgbotapi.NewMessage(channel.ID, "当前关键字: "+strings.Join(keywords, " , "))
	} else {
		return
		//msg = tgbotapi.NewMessage(channel.ID, "操作指南:\n/list 列出当前所有关键字\n /add {keyword1} {keyword2} ...... 增加新的关键字\n /delete {keyword1} {keyword2} ...... 删除关键字")
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := tgBot.Send(msg)
	if err != nil {
		log.Errorf("send message to channel failure: %v", err)
	}
}

func TgBotInstance() *tgbotapi.BotAPI {
	return tgBot
}
