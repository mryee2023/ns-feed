package lib

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/imroc/req/v3"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLinuxDoFeed(t *testing.T) {
	feedUrl := "https://hostloc.com/forum.php?fid=45&mod=rss"
	//feedUrl = "https://rsshub.app/telegram/channel/nodeloc_rss"
	reqClient := req.C().ImpersonateChrome()
	resp, err := reqClient.R().Get(feedUrl)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(resp.String())
	assert.NoError(t, err)
	assert.NotNil(t, feed)

	for _, item := range feed.Items {
		fmt.Println(item.Title+" "+item.Link, " ", item.Published)

	}
}

func Test_hasKeyword(t *testing.T) {
	type args struct {
		title    string
		keywords []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "正则关键字测试",
			args: args{
				title:    "剩余价值push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{`(?=.*港仔)(?=.*出)`, `港仔`},
			},
			want: true,
		},
		{
			name: "正则关键字测试2",
			args: args{
				title:    "剩余价值push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{`(?=.*(港仔|ggy|claw).*)`},
			},
			want: true,
		},
		{
			name: "存量关键字匹配",
			args: args{
				title:    "剩余价值➕push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{"bgp", "探针", "bgp.gd", "港仔", "mk", "boil", "zgo", "lala", "bage"},
			},
			want: true,
		},
		{
			name: "逻辑运算符关键字匹配",
			args: args{
				title:    "剩余价值➕push出港仔CMHK NAT 续费 13.88u/月",
				keywords: []string{"bgp", "探针", "bgp.gd", "港仔~NAT", "mk", "boil", "zgo", "lala", "bage"},
			},
			want: false,
		},
		{
			name: "逻辑运算符关键字匹配_斯巴达_1",
			args: args{
				title:    "[收]斯巴达小鸡一个",
				keywords: []string{"斯巴达"},
			},
			want: true,
		},
		{
			name: "逻辑运算符关键字匹配_斯巴达_2",
			args: args{
				title:    "[收]斯巴达小鸡一个",
				keywords: []string{"斯巴达~收"},
			},
			want: false,
		},
		{
			name: "逻辑运算符关键字匹配_斯巴达_3",
			args: args{
				title:    "油管 YouTube Premium家庭组 任意区年66.99",
				keywords: []string{"youtube"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasKeyword(tt.args.title, tt.args.keywords); got != tt.want {
				t.Errorf("hasKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBotButton(t *testing.T) {
	// 创建一个新的 Bot
	bot, err := tgbotapi.NewBotAPI("7727116717:AAH31RbD5ygRkuWGO1EaCcfKybAoirykxaY")
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true

	// 用于跟踪用户状态的map
	userStates := make(map[int64]string)
	// 用于跟踪用户当前选择的新闻类别
	userCategories := make(map[int64]string)

	// 模拟的RSS源数据（实际应用中这些数据应该来自数据库）
	mockRSSFeeds := map[string][]struct {
		ID    string
		Title string
		URL   string
	}{
		"tech_news": {
			{ID: "tech1", Title: "最新科技新闻1", URL: "http://example.com/tech1"},
			{ID: "tech2", Title: "最新科技新闻2", URL: "http://example.com/tech2"},
			{ID: "tech3", Title: "最新科技新闻3", URL: "http://example.com/tech3"},
		},
		"sports_news": {
			{ID: "sport1", Title: "体育新闻1", URL: "http://example.com/sport1"},
			{ID: "sport2", Title: "体育新闻2", URL: "http://example.com/sport2"},
		},
		// 可以添加其他类别的数据
	}

	// 创建主菜单按钮
	mainMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("科技新闻", "tech_news"),
			tgbotapi.NewInlineKeyboardButtonData("体育新闻", "sports_news"),
			tgbotapi.NewInlineKeyboardButtonData("财经新闻", "finance_news"),
			tgbotapi.NewInlineKeyboardButtonData("娱乐新闻", "entertainment_news"),
			tgbotapi.NewInlineKeyboardButtonData("生活新闻", "life_news"),
		),
	)

	// 创建子菜单按钮
	subMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("查看", "view"),
			tgbotapi.NewInlineKeyboardButtonData("添加", "add"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("返回主菜单", "back_to_main"),
		),
	)

	// 创建取消按钮
	cancelMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("取消添加", "cancel_add"),
		),
	)

	// 创建RSS源列表的按钮（带删除功能）
	createRSSListMarkup := func(category string) (string, tgbotapi.InlineKeyboardMarkup) {
		feeds, exists := mockRSSFeeds[category]
		if !exists || len(feeds) == 0 {
			return "当前分类没有RSS源", tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("返回主菜单", "back_to_main"),
				),
			)
		}

		var text string
		var rows [][]tgbotapi.InlineKeyboardButton

		// 添加每个RSS源和其删除按钮
		for i, feed := range feeds {
			text += fmt.Sprintf("%d. %s\n", i+1, feed.Title)
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("🗑️ 删除 #"+feed.ID, "delete_"+feed.ID),
			})
		}

		// 添加返回按钮
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("返回主菜单", "back_to_main"),
		})

		return text, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	}

	// 设置 Webhook 或轮询（这里使用轮询）
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID

			// 检查用户是否在等待输入状态
			if state, exists := userStates[chatID]; exists && state == "waiting_for_url" {
				// 用户在输入模式，处理输入的URL
				inputURL := update.Message.Text
				category := userCategories[chatID]

				// 这里可以添加URL验证逻辑
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("已收到您要添加到 %s 的链接：%s", category, inputURL))
				msg.ReplyMarkup = mainMenu // 添加完成后显示主菜单
				bot.Send(msg)

				// 清除用户状态
				delete(userStates, chatID)
				delete(userCategories, chatID)
			} else {
				// 普通消息，显示主菜单
				msg := tgbotapi.NewMessage(chatID, "请选择新闻类别：")
				msg.ReplyMarkup = mainMenu
				bot.Send(msg)
			}
		}

		// 处理回调数据
		if update.CallbackQuery != nil {
			// 回调查询的处理
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			bot.Send(callback)

			chatID := update.CallbackQuery.Message.Chat.ID
			var newMarkup tgbotapi.InlineKeyboardMarkup
			var responseText string

			// 检查是否是删除操作
			if strings.HasPrefix(update.CallbackQuery.Data, "delete_") {
				feedID := strings.TrimPrefix(update.CallbackQuery.Data, "delete_")
				category := userCategories[chatID]

				// 这里应该添加实际的删除逻辑
				responseText = fmt.Sprintf("已删除RSS源 (ID: %s)", feedID)

				// 重新显示更新后的列表
				listText, listMarkup := createRSSListMarkup(category)
				msg := tgbotapi.NewMessage(chatID, listText)
				msg.ReplyMarkup = listMarkup
				bot.Send(msg)
				continue
			}

			switch update.CallbackQuery.Data {
			case "tech_news", "sports_news", "finance_news", "entertainment_news", "life_news":
				newMarkup = subMenu
				responseText = "请选择操作："
				// 保存用户选择的类别
				userCategories[chatID] = update.CallbackQuery.Data
			case "back_to_main":
				newMarkup = mainMenu
				responseText = "请选择新闻类别："
				// 清除用户状态
				delete(userStates, chatID)
				delete(userCategories, chatID)
			case "view":
				category := userCategories[chatID]
				// 获取RSS源列表和对应的按钮
				listText, listMarkup := createRSSListMarkup(category)

				// 删除原有的菜单消息
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, update.CallbackQuery.Message.MessageID)
				bot.Send(deleteMsg)

				// 发送RSS源列表
				msg := tgbotapi.NewMessage(chatID, listText)
				msg.ReplyMarkup = listMarkup
				bot.Send(msg)
				continue
			case "add":
				// 设置用户状态为等待输入
				userStates[chatID] = "waiting_for_url"
				category := userCategories[chatID]

				// 删除原有的菜单消息
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, update.CallbackQuery.Message.MessageID)
				bot.Send(deleteMsg)

				// 发送新的提示消息
				tipMsg := fmt.Sprintf("您正在为 %s 添加新的RSS源\n\n"+
					"请按以下格式发送信息：\n"+
					"1. RSS feed的URL地址\n"+
					"2. 确保URL是有效的RSS feed源\n"+
					"3. 发送完成后会自动返回主菜单\n\n"+
					"您可以随时点击下方的「取消添加」按钮返回主菜单", category)

				msg := tgbotapi.NewMessage(chatID, tipMsg)
				msg.ReplyMarkup = cancelMenu
				bot.Send(msg)
				continue
			case "cancel_add":
				// 清除用户状态
				delete(userStates, chatID)
				delete(userCategories, chatID)

				// 发送主菜单
				msg := tgbotapi.NewMessage(chatID, "已取消添加，请选择新闻类别：")
				msg.ReplyMarkup = mainMenu
				bot.Send(msg)
				continue
			default:
				newMarkup = mainMenu
				responseText = "未知操作，请重新选择："
			}

			// 编辑现有消息的按钮
			edit := tgbotapi.NewEditMessageText(
				chatID,
				update.CallbackQuery.Message.MessageID,
				responseText,
			)
			edit.ReplyMarkup = &newMarkup
			bot.Send(edit)
		}
	}
}

func TestStructuredBotButtons(t *testing.T) {
	// 创建主菜单按钮
	mainMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("科技新闻", "tech_news"),
			tgbotapi.NewInlineKeyboardButtonData("体育新闻", "sports_news"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("财经新闻", "finance_news"),
			tgbotapi.NewInlineKeyboardButtonData("娱乐新闻", "entertainment_news"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("生活新闻", "life_news"),
		),
	)

	// 创建子菜单按钮（当用户点击主菜单项后显示）
	subMenu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("查看", "view"),
			tgbotapi.NewInlineKeyboardButtonData("添加", "add"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("返回主菜单", "back_to_main"),
		),
	)

	// 示例：如何在处理回调时使用这些按钮
	handleCallback := func(callbackData string) tgbotapi.InlineKeyboardMarkup {
		switch callbackData {
		case "tech_news", "sports_news", "finance_news", "entertainment_news", "life_news":
			return subMenu
		case "back_to_main":
			return mainMenu
		default:
			return mainMenu
		}
	}

	// 测试按钮结构
	testCases := []struct {
		callback string
		want     int // 期望的按钮行数
	}{
		{"tech_news", 2},    // 子菜单应该有2行
		{"back_to_main", 3}, // 主菜单应该有3行
	}

	for _, tc := range testCases {
		result := handleCallback(tc.callback)
		if len(result.InlineKeyboard) != tc.want {
			t.Errorf("handleCallback(%s) got %d rows, want %d rows",
				tc.callback, len(result.InlineKeyboard), tc.want)
		}
	}
}

func TestNsFeed_loadRssData(t *testing.T) {

	type args struct {
		url string
		ctx context.Context
	}
	tests := []struct {
		name string

		args args

		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "正常加载RSS数据",
			args: args{
				url: "https://rsshub.app/telegram/channel/nodeloc_rss",
				ctx: context.Background(),
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewNsFeed(context.Background(), &ServiceCtx{})
			_, err := f.loadRssData(tt.args.url, tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("loadRssData(%v, %v)", tt.args.url, tt.args.ctx)) {
				return
			}

		})
	}
}
