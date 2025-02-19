package lib

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	json "github.com/bytedance/sonic"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/rescue"

	"ns-rss/src/app/config"
	"ns-rss/src/app/db"
	"ns-rss/src/app/vars"
)

const (
	cmdFeed   = "/feed" //查看当前支持的RSS源
	cmdHelp   = "/help"
	cmdStatus = "/status"
	cmdAdd    = "/add"
)

var helpText = `

/feed 查看当前支持的RSS源

/help 查看帮助说明

/add feedId 关键字1 关键字2 关键字3.... 增加新的关键字

任何使用上的帮助或建议可以联系大管家 @hello\_cello\_bot
`

var (
	tgBot          *tgbotapi.BotAPI
	mainMenu       tgbotapi.InlineKeyboardMarkup
	lastMessageIDs sync.Map // 存储每个chat的最后一条消息ID
)

// ChatInfo 存储聊天相关信息
type ChatInfo struct {
	Name     string
	ChatID   int64
	ChatType string
	Text     string
}

// CommandHandler 命令处理函数类型
type CommandHandler func(*db.Subscribe, []string) (*tgbotapi.MessageConfig, error)

// 命令处理器映射
var commandHandlers = map[string]CommandHandler{
	cmdFeed: handleFeed,
	cmdAdd:  handleAdd,
	cmdHelp: handleHelp,
}

func InitTgBotListen(cnf *config.Config) {
	defer rescue.Recover()

	var err error
	tgBot, err = tgbotapi.NewBotAPI(cnf.TgToken)
	if err != nil {
		log.Fatalf("tgbotapi init failure: %v", err)
	}
	tgBot.Debug = false

	log.Infof("Authorized on account %s", tgBot.Self.UserName)
	go updates(cnf)
}

var backToMain = tgbotapi.NewInlineKeyboardButtonData("🔙返回主菜单",
	(&vars.CallbackEvent[vars.CallbackBackToMain]{
		Data: vars.CallbackBackToMain{},
	}).Param(),
)

func updates(cfg *config.Config) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgBot.GetUpdatesChan(u)

	var buttons []tgbotapi.InlineKeyboardButton

	feeds := db.ListAllFeedConfig()
	for _, v := range feeds {
		event := &vars.CallbackEvent[vars.CallbackFeedData]{
			Data: vars.CallbackFeedData{
				FeedId: v.FeedId,
			},
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(v.Name, event.Param()))

	}
	mainMenu = tgbotapi.NewInlineKeyboardMarkup()
	for _, button := range buttons {
		mainMenu.InlineKeyboard = append(mainMenu.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	for update := range updates {
		processMessage(cfg, update)
	}
}

// extractChatInfo 从更新中提取聊天信息
func extractChatInfo(update tgbotapi.Update) *ChatInfo {
	switch {
	case update.ChannelPost != nil:
		return &ChatInfo{
			Name:     update.ChannelPost.Chat.Title,
			ChatID:   update.ChannelPost.Chat.ID,
			ChatType: config.ChatTypeChannel,
			Text:     strings.TrimSpace(update.ChannelPost.Text),
		}
	case update.Message != nil && update.Message.Chat.IsGroup():
		return &ChatInfo{
			Name:     update.Message.Chat.Title,
			ChatID:   update.Message.Chat.ID,
			ChatType: config.ChatTypeGroup,
			Text:     strings.TrimSpace(update.Message.Text),
		}
	case update.Message != nil:
		return &ChatInfo{
			Name:     update.Message.Chat.Title,
			ChatID:   update.Message.Chat.ID,
			ChatType: config.ChatTypeChat,
			Text:     strings.TrimSpace(update.Message.Text),
		}
	case update.CallbackQuery != nil:
		return &ChatInfo{
			Name:     update.CallbackQuery.From.UserName,
			ChatID:   update.CallbackQuery.Message.Chat.ID,
			ChatType: config.ChatTypeCallback,
			Text:     strings.TrimSpace(update.CallbackQuery.Data),
		}
	default:
		return nil
	}
}

func processMessage(cfg *config.Config, update tgbotapi.Update) {
	defer rescue.Recover()

	chatInfo := extractChatInfo(update)
	if chatInfo == nil || chatInfo.Text == "" {
		return
	}

	entry := log.WithField("message", chatInfo.Text).
		WithField("from", chatInfo.Name)
	entry.Info("receive message")

	subscriber := ensureSubscriber(chatInfo)
	if subscriber == nil || subscriber.Status == "quit" {
		return
	}

	// 处理回调数据
	if update.CallbackQuery != nil {
		log.WithFields(log.Fields{
			"callback_data": update.CallbackQuery.Data,
			"from":          update.CallbackQuery.From.UserName,
		}).Info("Received callback query")

		// 确认收到回调
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
		tgBot.Send(callback)

		chatID := update.CallbackQuery.Message.Chat.ID
		callbackData := update.CallbackQuery.Data

		// 解析回调数据
		var event vars.CallbackEvent[vars.CallbackFeedData]
		if err := json.Unmarshal([]byte(callbackData), &event); err != nil {
			log.WithError(err).WithField("data", callbackData).Error("Failed to unmarshal callback data")
			return
		}

		log.WithFields(log.Fields{
			"event": event.Event,
			"data":  event.Data,
		}).Info("Parsed callback event")

		// 根据事件类型处理
		switch event.Event {
		case string(vars.EventSelectFeed):
			// 获取完整的feed信息
			feed := db.GetFeedConfigWithFeedId(event.Data.FeedId)

			// 获取关键字列表

			subscribe := db.ListSubscribeFeedWith(chatID, feed.FeedId)
			if len(subscribe.KeywordsArray) > 0 {
				var keywords []tgbotapi.InlineKeyboardButton
				for _, v := range subscribe.KeywordsArray {
					v = "🗑️ " + v
					data := vars.CallbackEvent[vars.CallbackDeleteKeyword]{
						Data: vars.CallbackDeleteKeyword{
							Keyword: v,
							FeedId:  feed.FeedId,
						},
					}
					//判断一下长度
					if len(data.Param()) > 64 {
						continue
					}
					keywords = append(keywords, tgbotapi.NewInlineKeyboardButtonData(v, data.Param()))
				}
				keyboard := tgbotapi.NewInlineKeyboardMarkup()

				// 创建添加关键字的事件
				addEvent := vars.CallbackEvent[vars.CallbackAddKeyword]{
					Data: vars.CallbackAddKeyword{
						FeedId: event.Data.FeedId,
					},
				}

				for _, keyword := range keywords {
					keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(keyword))
				}

				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✍️ 添加关键字", addEvent.Param())),
				)
				keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(backToMain))

				msg := tgbotapi.NewMessage(chatID, "以下是您已添加的 "+feed.Name+" 关键字:")
				msg.ReplyMarkup = keyboard
				msg.ParseMode = tgbotapi.ModeHTML
				sendMessage(&msg)
			} else {
				// 创建添加关键字的事件
				addEvent := vars.CallbackEvent[vars.CallbackAddKeyword]{
					Data: vars.CallbackAddKeyword{
						FeedId: event.Data.FeedId,
					},
				}

				msg := tgbotapi.NewMessage(chatID, "未设置关键字，请点击下方按钮添加")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("添加关键字", addEvent.Param()),
						backToMain,
					),
				)
				sendMessage(&msg)
			}

			return

		case string(vars.EventDeleteKeyword):
			var deleteEvent vars.CallbackEvent[vars.CallbackDeleteKeyword]
			if err := json.Unmarshal([]byte(callbackData), &deleteEvent); err != nil {
				return
			}
			// 显示确认删除界面
			confirmEvent := vars.CallbackEvent[vars.CallbackConfirmDelete]{
				Data: vars.CallbackConfirmDelete{
					Keyword: deleteEvent.Data.Keyword,
					FeedId:  deleteEvent.Data.FeedId,
				},
			}

			// 创建返回事件
			backEvent := vars.CallbackEvent[vars.CallbackFeedData]{
				Data: vars.CallbackFeedData{
					FeedId: event.Data.FeedId,
				},
			}

			text := fmt.Sprintf("确定要删除关键字 \"%s\" 吗？", deleteEvent.Data.Keyword)
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ 确认删除", confirmEvent.Param()),
					tgbotapi.NewInlineKeyboardButtonData("❌ 取消", backEvent.Param()),
				),
			)
			sendMessage(&msg)
			return

		case string(vars.EventConfirmDelete):

			var deleteEvent vars.CallbackEvent[vars.CallbackConfirmDelete]
			json.Unmarshal([]byte(callbackData), &deleteEvent)

			_, err := handleDelete(subscriber, []string{event.Data.FeedId, deleteEvent.Data.Keyword})
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, err.Error())
				sendMessage(&msg)
				return
			}
			// 返回到Feed详情
			backEvent := vars.CallbackEvent[vars.CallbackFeedData]{
				Data: vars.CallbackFeedData{
					FeedId: event.Data.FeedId,
				},
			}

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("已删除关键字 %s", deleteEvent.Data.Keyword))
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("返回列表", backEvent.Param()),
				),
			)
			sendMessage(&msg)
			return

		case string(vars.EventAddKeyword):
			feed := db.GetFeedConfigWithFeedId(event.Data.FeedId)
			if feed.FeedId == "" {
				msg := tgbotapi.NewMessage(chatID, "未找到对应的Feed源")
				sendMessage(&msg)
				return
			}

			text := fmt.Sprintf("请输入想要添加的关键字，格式如下：\n"+
				"/add %s 关键字1 正则表达式 ...\n\n"+
				"示例：\n"+
				"/add %s 科技 ", feed.FeedId, feed.FeedId)

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					backToMain,
				),
			)
			sendMessage(&msg)
			return

		case string(vars.EventBackToMain):
			msg := tgbotapi.NewMessage(chatID, "请选择Feed源:")
			msg.ReplyMarkup = mainMenu
			sendMessage(&msg)
			return
		case string(vars.EventOn):
			msg, _ := handleOn(subscriber, nil)
			sendMessage(msg)
			return
		case string(vars.EventOff):
			msg, _ := handleOff(subscriber, nil)
			sendMessage(msg)
			return
		}
		return
	}

	cmd, args := parseCommand(chatInfo.Text)
	if cmd == "" {
		return
	}

	// 特殊处理 status 命令
	if cmd == cmdStatus && subscriber.ChatId == cfg.AdminId {
		handleStatus(subscriber)
		return
	}
	defer func() {
		SubCacheInstance().Del(subscriber.ChatId)
		SubCacheInstance().ReloadAll()
	}()
	handler, exists := commandHandlers[cmd]
	if !exists {
		return
	}

	msg, err := handler(subscriber, args)
	if err != nil {

		errMsg := tgbotapi.NewMessage(subscriber.ChatId, err.Error())
		sendMessage(&errMsg)
		return
	}
	if msg == nil {
		return
	}

	sendMessage(msg)
}

// ensureSubscriber 确保订阅者存在
func ensureSubscriber(info *ChatInfo) *db.Subscribe {
	subscriber := db.GetSubscribeWithChatId(info.ChatID)
	if subscriber == nil {
		tgBot.Send(tgbotapi.NewMessage(info.ChatID, "欢迎使用 NS 论坛关键字通知功能，这是您的首次使用, 请用 /help 查看帮助说明。"))
		db.AddSubscribe(&db.Subscribe{
			Name:      info.Name,
			ChatId:    info.ChatID,
			Status:    "on",
			Type:      info.ChatType,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
		subscriber = db.GetSubscribeWithChatId(info.ChatID)
	}
	return subscriber
}

// parseCommand 解析命令和参数
func parseCommand(text string) (string, []string) {
	parts := splitAndClean(text)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// splitAndClean 分割并清理字符串
func splitAndClean(text string) []string {
	words := strings.Split(text, " ")
	return funk.FilterString(words, func(s string) bool {
		return strings.TrimSpace(s) != ""
	})
}

// sendMessage 发送消息
func sendMessage(msg *tgbotapi.MessageConfig) {
	msg.ParseMode = tgbotapi.ModeMarkdown
	result, err := tgBot.Send(msg)
	if err != nil {
		log.WithField("msg", msg.Text).
			WithError(err).
			Error("Failed to send message")
		return
	}

	// 如果消息带有ReplyMarkup
	if msg.ReplyMarkup != nil {
		// 获取上一条消息的ID
		if lastID, ok := lastMessageIDs.Load(msg.ChatID); ok {
			// 删除上一条消息
			go func(chatID int64, messageID int) {
				time.Sleep(3 * time.Second)
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
				_, err := tgBot.Request(deleteMsg)
				if err != nil {
					log.WithError(err).
						WithField("chat_id", chatID).
						WithField("message_id", messageID).
						Error("Failed to delete message")
				}
			}(msg.ChatID, lastID.(int))
		}

		// 更新最后一条消息的ID
		lastMessageIDs.Store(msg.ChatID, result.MessageID)
	}

	log.WithField("msg", msg.Text).
		WithField("chat_id", msg.ChatID).
		Info("Message sent")
}

// 命令处理函数
func handleFeed(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {

	msg := tgbotapi.NewMessage(sub.ChatId, "当前支持的feed源, 请点击选择:")
	msg.ReplyMarkup = mainMenu
	return &msg, nil
}

func handleAdd(sub *db.Subscribe, args []string) (*tgbotapi.MessageConfig, error) {
	if len(args) == 0 || len(args) == 1 {
		return nil, errors.New("请输入你要添加的关键字, 例如: /add feedId keyword")
	}

	feedId := args[0]

	// 检查是否存在该feedId
	v := db.GetFeedConfigWithFeedId(feedId)
	if v.ID == 0 {
		return nil, errors.New("未找到该feed")
	}

	args = args[1:]

	args = funk.Map(args, func(s string) string {
		return strings.Trim(strings.TrimSpace(s), "{}")
	}).([]string)

	// 检查每个关键字的callback_data长度
	var invalidKeywords []string
	for _, keyword := range args {
		data := vars.CallbackEvent[vars.CallbackDeleteKeyword]{
			Data: vars.CallbackDeleteKeyword{
				Keyword: keyword,
				FeedId:  feedId,
			},
		}
		if len(data.Param()) > 64 {
			invalidKeywords = append(invalidKeywords, keyword)
		}
	}

	if len(invalidKeywords) > 0 {
		return nil, fmt.Errorf("以下关键字太长，请缩短后重新添加：\n%s", strings.Join(invalidKeywords, "\n"))
	}

	//更新db
	exists := db.ListSubscribeFeedWith(sub.ChatId, feedId)
	if exists.ID > 0 {
		//取一下并集
		args = append(args, exists.KeywordsArray...)
		args = funk.UniqString(args)
		exists.KeywordsArray = args
	} else {
		exists = db.SubscribeConfig{
			ChatId:        sub.ChatId,
			KeywordsArray: args,
			FeedId:        feedId,
		}
	}
	db.AddSubscribeConfig(exists)
	msg := tgbotapi.NewMessage(sub.ChatId, "🎉关键字添加成功")
	// 获取关键字列表

	if len(exists.KeywordsArray) > 0 {
		var keywords []tgbotapi.InlineKeyboardButton
		for _, v := range exists.KeywordsArray {
			data := vars.CallbackEvent[vars.CallbackDeleteKeyword]{
				Data: vars.CallbackDeleteKeyword{
					Keyword: v,
					FeedId:  feedId,
				},
			}

			text := "🗑️" + v
			keywords = append(keywords, tgbotapi.NewInlineKeyboardButtonData(text, data.Param()))
		}
		keyboard := tgbotapi.NewInlineKeyboardMarkup()

		// 创建添加关键字的事件
		addEvent := vars.CallbackEvent[vars.CallbackAddKeyword]{
			Data: vars.CallbackAddKeyword{
				FeedId: feedId,
			},
		}

		for _, keyword := range keywords {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(keyword))
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✍️ 添加关键字", addEvent.Param())),
		)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(backToMain))

		msg.ReplyMarkup = keyboard
	}
	return &msg, nil
}

func handleDelete(sub *db.Subscribe, args []string) (*tgbotapi.MessageConfig, error) {
	if len(args) == 0 || len(args) == 1 {
		return nil, errors.New("请选择你要删除的关键字")
	}

	feedId := args[0]

	exists := db.ListSubscribeFeedWith(sub.ChatId, feedId)
	if exists.ID == 0 {
		return nil, errors.New("您还未添加过该feedId的关键字")
	}

	deletes := make(map[string]struct{})
	var delWords []string

	for _, word := range args[1:] {
		for _, v := range exists.KeywordsArray {
			if strings.ToLower(v) == strings.ToLower(word) {
				deletes[v] = struct{}{}
				delWords = append(delWords, word)
			}
		}
	}

	var newWords []string
	for _, v := range exists.KeywordsArray {
		if _, ok := deletes[v]; !ok {
			newWords = append(newWords, v)
		}
	}

	exists.KeywordsArray = newWords
	db.AddSubscribeConfig(exists)
	return nil, nil
}

func handleHelp(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {
	msg := tgbotapi.NewMessage(sub.ChatId, helpText)
	on := vars.CallbackEvent[vars.CallbackStatusOn]{
		Data: vars.CallbackStatusOn{
			ChatId: sub.ChatId,
		},
	}
	button := tgbotapi.NewInlineKeyboardButtonData("开启关键字通知", on.Param())
	if sub.Status == "on" || sub.Status == "" {
		off := vars.CallbackEvent[vars.CallbackStatusOff]{
			Data: vars.CallbackStatusOff{
				ChatId: sub.ChatId,
			},
		}
		button = tgbotapi.NewInlineKeyboardButtonData("关闭关键字通知", off.Param())
	}

	keyword := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			button,
		))
	msg.ReplyMarkup = keyword
	return &msg, nil
}

func handleOn(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {
	sub.Status = "on"
	db.UpdateSubscribe(sub)
	msg := tgbotapi.NewMessage(sub.ChatId, "关键字通知已成功开启")
	return &msg, nil
}

func handleOff(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {
	sub.Status = "off"
	db.UpdateSubscribe(sub)
	msg := tgbotapi.NewMessage(sub.ChatId, "关键字通知已成功关闭")
	return &msg, nil
}

func handleStatus(sub *db.Subscribe) {
	subscribers := db.ListSubscribes()
	todaySend := db.GetNotifyCountByDateTime(carbon.Now().StartOfDay().StdTime(), time.Now())

	ip := getPublicIP()
	if ip != "" {
		parts := strings.Split(ip, ".")
		ip = fmt.Sprintf("%s.\\*.%s.%s", parts[0], parts[2], parts[3])
	} else {
		ip = "未知"
	}

	message := fmt.Sprintf("当前状态: \n订阅数: %d \n当天发送: %d \n当前IP: %s",
		len(subscribers), todaySend, ip)
	msg := tgbotapi.NewMessage(sub.ChatId, message)
	sendMessage(&msg)
}

func getPublicIP() string {
	cmd := exec.Command("curl", "ip.sb", "-4")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

func TgBotInstance() *tgbotapi.BotAPI {
	return tgBot
}
