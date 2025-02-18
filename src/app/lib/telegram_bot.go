package lib

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/rescue"
	"ns-rss/src/app/config"
	"ns-rss/src/app/db"
)

const (
	cmdFeed   = "/feed" //查看当前支持的RSS源
	cmdHelp   = "/help"
	cmdList   = "/list"
	cmdAdd    = "/add"
	cmdDelete = "/delete"
	cmdOn     = "/on"
	cmdOff    = "/off"
	cmdQuit   = "/quit"
	cmdStatus = "/status"
	cmdStart  = "/start"
)

var helpText = `
/start 开始使用关键字通知
/feed 查看当前支持的RSS源
/help 查看帮助说明
/list 列出当前所有关键字
/add feedId 关键字1 关键字2 关键字3.... 增加新的关键字
/delete feedId  关键字1 关键字2 关键字3.... 删除关键字
/on 开启关键字通知
/off 关闭关键字通知
/quit 退出关键字通知

任何使用上的帮助或建议可以联系大管家 @hello\_cello\_bot
`

var (
	tgBot *tgbotapi.BotAPI
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
	cmdList:   handleList,
	cmdFeed:   handleFeed,
	cmdAdd:    handleAdd,
	cmdDelete: handleDelete,
	cmdHelp:   handleHelp,
	cmdOn:     handleOn,
	cmdOff:    handleOff,
	cmdQuit:   handleQuit,
	cmdStart:  handleStart,
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

var mainMenu tgbotapi.InlineKeyboardMarkup
var backToMain = tgbotapi.NewInlineKeyboardButtonData("返回主菜单", "back_to_main")

//var subMenu = tgbotapi.NewInlineKeyboardMarkup(
//	tgbotapi.NewInlineKeyboardRow(
//		tgbotapi.NewInlineKeyboardButtonData("查看我的关键字", "view"),
//		tgbotapi.NewInlineKeyboardButtonData("添加新的关键字", "add"),
//	),
//	tgbotapi.NewInlineKeyboardRow(
//		backToMain,
//	),
//)

// 创建取消按钮
var cancelMenu = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("取消添加", "cancel_add"),
	),
)

func createRssListMarkup(chatId int64, feedId string) (string, tgbotapi.InlineKeyboardMarkup) {
	conf := db.ListSubscribeFeedWith(chatId, feedId)
	if len(conf.KeywordsArray) == 0 {
		return "您还没有添加关键字", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				backToMain,
			),
		)
	}

	var text string
	var rows [][]tgbotapi.InlineKeyboardButton

	// 添加每个RSS源和其删除按钮
	for _, feed := range conf.KeywordsArray {
		if feed == "" {
			continue
		}
		text += fmt.Sprintf("%s\n", feed)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ 删除 #", "delete_"+feed),
		})
	}

	// 添加返回按钮
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		backToMain,
	})

	return text, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func updates(cfg *config.Config) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tgBot.GetUpdatesChan(u)

	var buttons []tgbotapi.InlineKeyboardButton

	feeds := db.ListAllFeedConfig()
	for _, v := range feeds {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(v.Name, v.FeedId))
	}
	mainMenu = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)
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

		fmt.Println("update.CallbackQuery", update.CallbackQuery)

		// 回调查询的处理
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		tgBot.Send(callback)

		chatID := update.CallbackQuery.Message.Chat.ID
		//var newMarkup tgbotapi.InlineKeyboardMarkup
		//var responseText string

		// 检查是否是删除操作
		if strings.HasPrefix(update.CallbackQuery.Data, "delete_") {
			feedID := strings.TrimPrefix(update.CallbackQuery.Data, "delete_")

			// 这里应该添加实际的删除逻辑
			//responseText = fmt.Sprintf("已删除RSS源 (ID: %s)", feedID)

			// 重新显示更新后的列表
			listText, listMarkup := createRssListMarkup(chatID, feedID)
			msg := tgbotapi.NewMessage(chatID, listText)
			msg.ReplyMarkup = listMarkup
			sendMessage(&msg)
			return
		}

		switch update.CallbackQuery.Data {

		case "back_to_main":
			//newMarkup = mainMenu
			msg := tgbotapi.NewMessage(chatID, "asdfasdfasdf")
			msg.ReplyMarkup = backToMain
			sendMessage(&msg)
			return

		case "add":
			// 设置用户状态为等待输入

			// 发送新的提示消息
			tipMsg := fmt.Sprintf("您正在为 %s 添加新的RSS源\n\n"+
				"请按以下格式发送信息：\n"+
				"1. RSS feed的URL地址\n"+
				"2. 确保URL是有效的RSS feed源\n"+
				"3. 发送完成后会自动返回主菜单\n\n"+
				"您可以随时点击下方的「取消添加」按钮返回主菜单", "feedId")

			msg := tgbotapi.NewMessage(chatID, tipMsg)
			msg.ReplyMarkup = cancelMenu
			sendMessage(&msg)
			return
		case "cancel_add":
			// 清除用户状态
			//delete(userStates, chatID)
			//delete(userCategories, chatID)

			// 发送主菜单
			msg := tgbotapi.NewMessage(chatID, "已取消添加，请选择新闻类别：")
			msg.ReplyMarkup = mainMenu
			sendMessage(&msg)
		default:
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
			WithField("error", err).
			Error("send message failure")
	} else {
		log.WithField("msg", msg.Text).
			WithField("result id", result.MessageID).
			Info("send message success")
	}
}

// 命令处理函数
func handleList(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {

	keys := db.ListSubscribeFeedConfig(sub.ChatId)
	var sb strings.Builder
	for k, v := range keys {
		sb.WriteString(fmt.Sprintf("feed源: %s, 关键字: %s\n", k, strings.Join(v, " , ")))
	}
	msg := tgbotapi.NewMessage(sub.ChatId, "当前配置的关键字: \n"+sb.String())
	return &msg, nil
}

// 命令处理函数
func handleFeed(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {

	//feeds := db.ListAllFeedConfig()
	//var feedId []string
	//for _, v := range feeds {
	//	feedId = append(feedId, "名称: "+v.Name+" , 标识: **"+v.FeedId+"**")
	//}
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
		return nil, errors.New("该feedId不存在, 请先使用 /feed 查看支持的feedId")
	}

	args = args[1:]

	args = funk.Map(args, func(s string) string {
		return strings.Trim(strings.TrimSpace(s), "{}")
	}).([]string)

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

	msg := tgbotapi.NewMessage(sub.ChatId, "关键字添加成功 "+strings.Join(args, " , "))
	return &msg, nil
}

func handleDelete(sub *db.Subscribe, args []string) (*tgbotapi.MessageConfig, error) {
	if len(args) == 0 || len(args) == 1 {
		return nil, errors.New("请输入你要删除的关键字, 例如: /delete feedId keyword")
	}

	feedId := args[0]

	// 检查是否存在该feedId
	v := db.GetFeedConfigWithFeedId(feedId)
	if v.ID == 0 {
		return nil, errors.New("该feedId不存在, 请先使用 /feed 查看支持的feedId")
	}

	exists := db.ListSubscribeFeedWith(sub.ChatId, feedId)
	if exists.ID == 0 {
		return nil, errors.New("您还未添加过该feedId的关键字")
	}

	deletes := make(map[string]struct{})
	var delWords []string

	for _, word := range args {
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

	msg := tgbotapi.NewMessage(sub.ChatId, "关键字删除成功 "+strings.Join(delWords, " , "))
	return &msg, nil
}

func handleHelp(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {
	msg := tgbotapi.NewMessage(sub.ChatId, helpText)
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

func handleStart(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {
	sub.Status = "on"
	db.UpdateSubscribe(sub)
	msg := tgbotapi.NewMessage(sub.ChatId, "欢迎回来, 请用 /help 查看帮助说明。")
	return &msg, nil

}

func handleQuit(sub *db.Subscribe, _ []string) (*tgbotapi.MessageConfig, error) {
	sub.Status = "quit"
	db.UpdateSubscribe(sub)
	msg := tgbotapi.NewMessage(sub.ChatId, "Bye~您现在可以移除本机器人了\n期待您的再次使用")
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
