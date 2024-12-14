package lib

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/rescue"
	"ns-rss/src/app"
	"ns-rss/src/app/config"
)

var help = `
/list 列出当前所有关键字

/add 关键字1 关键字2 关键字3.... 增加新的关键字

/delete 关键字1 关键字2 关键字3.... 删除关键字

/on 开启关键字通知

/off 关闭关键字通知

/quit 退出关键字通知

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

func curl() string {
	// 准备命令
	cmd := exec.Command("curl", "ip.sb", "-4")

	// 捕获输出
	var out bytes.Buffer
	cmd.Stdout = &out

	// 执行命令
	err := cmd.Run()
	if err != nil {
		return ""
	}
	return out.String()
}

func processMessage(cfg *config.Config, update tgbotapi.Update) {
	defer func() {
		rescue.Recover()
	}()
	var name string
	var chatId int64
	var text string
	var chatType = config.ChatTypeChat
	//判断来源类型
	if update.ChannelPost != nil {
		chatType = config.ChatTypeChannel
		name = update.ChannelPost.Chat.Title
		chatId = update.ChannelPost.Chat.ID
		text = update.ChannelPost.Text
	} else if update.Message != nil && update.Message.Chat.IsGroup() {
		chatType = config.ChatTypeGroup
		name = update.Message.Chat.Title
		chatId = update.Message.Chat.ID
		text = update.Message.Text
	} else if update.Message != nil {
		chatType = config.ChatTypeChat
		name = update.Message.Chat.Title
		chatId = update.Message.Chat.ID
		text = update.Message.Text
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	entry := log.WithField("message", text).
		WithField("from", name)
	entry.Info("receive message")

	processMutex.Lock()
	defer processMutex.Unlock()
	//判断个人是否在配置文件中
	var currentChannel *config.Subscribe
	for i, info := range cfg.Subscribes {
		//兼容原数据
		if strings.TrimSpace(info.Type) == "" {
			info.Type = config.ChatTypeChannel
		}
		if info.ChatId == chatId && info.Type == chatType {
			currentChannel = cfg.Subscribes[i]
			break
		}
	}
	if currentChannel == nil {
		currentChannel = &config.Subscribe{
			Name:     name,
			ChatId:   chatId,
			Keywords: []string{},
			Type:     chatType,
		}
		cfg.Subscribes = append(cfg.Subscribes, currentChannel)
		cfg.Storage(app.ConfigFilePath)
		//第一次添加，发送欢迎消息
		tgBot.Send(tgbotapi.NewMessage(chatId, `欢迎使用 NS 论坛关键字通知功能，这是您的首次使用, 请用 /help 查看帮助说明。`))
	}

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
	case strings.HasPrefix(text, "/quit"):
		var channels []*config.Subscribe
		for i, info := range cfg.Subscribes {
			if info.ChatId != chatId {
				channels = append(channels, cfg.Subscribes[i])
			}
		}
		cfg.Subscribes = channels
		cfg.Storage(app.ConfigFilePath)
		m := tgbotapi.NewMessage(currentChannel.ChatId, "Bye~您现在可以移除本机器人了\n期待您的再次使用")
		msg = &m
	case strings.HasPrefix(text, "/status") && currentChannel.ChatId == cfg.AdminId:
		//汇总当前状态
		subscribers := len(cfg.Subscribes)
		//当天发送次数
		notifyLock.Lock()
		defer notifyLock.Unlock()
		todaySend := int64(0)
		k := time.Now().Format(carbon.DateFormat)
		if v, ok := noticeHistory[k]; ok {
			todaySend = v
		}
		var ip = curl()
		if strings.TrimSpace(ip) == "" {
			ip = "未知"
		} else {
			mask := strings.Split(ip, ".")
			ip = mask[0] + ".\\*." + mask[2] + "." + mask[3]
		}
		var message = fmt.Sprintf("当前状态: \n订阅数: %d \n当天发送: %d \n当前IP: %s", subscribers, todaySend, ip)
		m := tgbotapi.NewMessage(currentChannel.ChatId, message)
		msg = &m
	default:
		return
	}

	if err != nil {
		tgBot.Send(tgbotapi.NewMessage(currentChannel.ChatId, err.Error()))
		return
	}
	msg.ParseMode = tgbotapi.ModeMarkdown
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

func processDeleteEvent(cfg *config.Config, postText string, currentChannel *config.Subscribe) (*tgbotapi.MessageConfig, error) {
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

func processAddEvent(cfg *config.Config, postText string, currentChannel *config.Subscribe) (*tgbotapi.MessageConfig, error) {
	words := strings.Split(postText, " ")
	words = funk.FilterString(words, func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
	words = funk.UniqString(words)
	if len(words) == 1 {
		return nil, errors.New("请输入你要添加的关键字, 例如: /add keyword")
	}
	words = funk.Map(words, func(s string) string {
		v := strings.TrimSpace(s)
		v = strings.Trim(v, "{}")
		return v
	}).([]string)
	currentChannel.Keywords = append(currentChannel.Keywords, words[1:]...)

	cfg.Storage(app.ConfigFilePath)
	msg := tgbotapi.NewMessage(currentChannel.ChatId, "关键字添加成功 "+strings.Join(words[1:], " , "))
	return &msg, nil
}

func processListEvent(cfg *config.Config, channel *config.Subscribe) (*tgbotapi.MessageConfig, error) {
	var keywords []string
	for _, info := range cfg.Subscribes {
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
