package lib

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"ns-rss/src/app/config"
	"ns-rss/src/app/db"

	"github.com/dlclark/regexp2"
	"github.com/imroc/req/v3"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
	"github.com/zeromicro/go-zero/core/threading"
)

//var history = make(map[int64]map[string]struct{})

// 发送计数
//var noticeHistory = make(map[string]int64)

type NsFeed struct {
	sync.Mutex
	ctx          context.Context
	svc          *ServiceCtx
	logger       logx.Logger
	bot          BotNotifier
	msgQueue     chan *NotifyMessage // 消息队列
	Config       *config.Config
	LastUpdate   time.Time
	interval     time.Duration // 当前请求间隔
	minInterval  time.Duration // 最小间隔
	maxInterval  time.Duration // 最大间隔
	successCount int           // 连续成功次数
	failureCount int           // 连续失败次数
}

func NewNsFeed(ctx context.Context, svc *ServiceCtx, config *config.Config) *NsFeed {
	return &NsFeed{
		ctx:         ctx,
		svc:         svc,
		logger:      logx.WithContext(ctx).WithFields(logx.Field("lib", "ns_feed")),
		Config:      config,
		interval:    10 * time.Second,                // 初始间隔
		minInterval: 10 * time.Second,                // 最小间隔
		maxInterval: 5 * time.Minute,                 // 最大间隔
		msgQueue:    make(chan *NotifyMessage, 1000), // 创建消息队列，缓冲大小为1000
	}
}

func (f *NsFeed) SetBot(bot BotNotifier) *NsFeed {
	f.bot = bot

	// 启动消息队列消费者
	go f.startQueueConsumer()

	return f
}

// 启动消息队列消费者，控制消费速率为每秒20条消息
func (f *NsFeed) startQueueConsumer() {
	defer rescue.Recover()

	f.logger.Infow("starting message queue consumer")

	// 计算消息发送间隔，保证每秒最多发送20条消息
	interval := time.Millisecond * 50 // 1000ms / 20 = 50ms

	// 使用动态 ticker 模式，只在有消息时才激活 ticker
	var ticker *time.Ticker
	var tickerC <-chan time.Time

	// 用于限制发送速率的时间点
	var nextSendTime time.Time

	for {
		select {
		case <-f.ctx.Done():
			f.logger.Infow("message queue consumer stopped due to context done")
			if ticker != nil {
				ticker.Stop()
			}
			return

		case msg := <-f.msgQueue:
			// 只有在收到消息时才创建或使用 ticker
			now := time.Now()

			// 如果已经到了可以发送的时间，直接发送
			if now.After(nextSendTime) {
				if msg != nil && f.bot != nil {
					f.logger.Debugw("sending message from queue immediately", logx.Field("chatId", msg.ChatId))
					f.bot.Notify(*msg)
					nextSendTime = now.Add(interval)
				}
				continue
			}

			// 如果还没到发送时间，需要等待
			if ticker == nil {
				ticker = time.NewTicker(interval)
				tickerC = ticker.C
			}

			// 等待发送时间
			select {
			case <-tickerC:
				if msg != nil && f.bot != nil {
					f.logger.Debugw("sending message from queue after wait", logx.Field("chatId", msg.ChatId))
					f.bot.Notify(*msg)
					nextSendTime = time.Now().Add(interval)
				}
			case <-f.ctx.Done():
				f.logger.Infow("message queue consumer stopped during wait")
				ticker.Stop()
				return
			}
		}

		// 如果队列为空，停止 ticker 以节省资源
		if len(f.msgQueue) == 0 && ticker != nil {
			ticker.Stop()
			ticker = nil
			tickerC = nil
		}
	}
}

func (f *NsFeed) Add(msg NotifyMessage) {
	// 将消息添加到队列
	select {
	case f.msgQueue <- &msg:
		f.logger.Debugw("added message to queue", logx.Field("chatId", msg.ChatId))
	default:
		// 队列已满，记录日志
		f.logger.Infow("message queue is full, message dropped", logx.Field("chatId", msg.ChatId))
	}
}

func hasKeywordWithRegex(title string, keyword string) bool {
	//尝试转为正则
	re, err := regexp2.Compile(keyword, regexp2.IgnoreCase)
	if err != nil {
		return false
	}
	r, _ := re.MatchString(title)
	return r
}

// 使用缓存存储已编译的正则表达式
var regexCache = sync.Map{}

// 获取缓存的正则表达式
func getCachedRegex(keyword string) (*regexp2.Regexp, bool) {
	if re, ok := regexCache.Load(keyword); ok {
		return re.(*regexp2.Regexp), true
	}

	re, err := regexp2.Compile(keyword, regexp2.IgnoreCase)
	if err != nil {
		return nil, false
	}

	regexCache.Store(keyword, re)
	return re, true
}

// 优化后的正则匹配函数
func hasKeywordWithRegexCached(title string, keyword string) bool {
	re, ok := getCachedRegex(keyword)
	if !ok {
		return false
	}
	r, _ := re.MatchString(title)
	return r
}

func hasKeyword(title string, keywords []string) bool {
	title = strings.ToLower(title)
	for _, keyword := range keywords {
		// 首先尝试表达式匹配，这通常更快
		if hasKeywordWithExpression(title, keyword) {
			return true
		}

		// 如果表达式匹配失败，再尝试正则匹配
		if hasKeywordWithRegexCached(title, keyword) {
			return true
		}
	}
	return false
}

// matchExpression 匹配表达式函数
// expr: 表达式字符串，支持 + (与), | (或), ~ (排除)
// text: 要匹配的文本
// 返回: 是否匹配的布尔值
func matchExpression(expr, text string) bool {
	// 首先处理 | (或) 运算符，将表达式按 | 分割
	orParts := strings.Split(expr, "|")

	// 任意一个 or 条件满足即返回 true
	for _, orPart := range orParts {
		if matchAndExpression(strings.TrimSpace(orPart), text) {
			return true
		}
	}
	return false
}

// matchAndExpression 处理 + (与) 和 ~ (排除) 的逻辑
func matchAndExpression(expr, text string) bool {
	// 将表达式按 + 分割
	andParts := strings.Split(expr, "+")

	// 检查每个条件
	for _, part := range andParts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}

		// 处理排除(~)逻辑
		if strings.HasPrefix(part, "~") {
			excludeTerm := strings.TrimPrefix(part, "~")
			if strings.Contains(text, excludeTerm) {
				return false
			}
		} else {
			// 处理包含逻辑
			if !strings.Contains(text, part) {
				return false
			}
		}
	}
	return true
}

func hasKeywordWithExpression(title string, keyword string) bool {

	return matchExpression(keyword, title)

	// keyword = strings.Trim(keyword, "{}")
	// keyword = strings.ToLower(keyword)
	// // 处理或关系 (|)
	// orParts := strings.Split(keyword, "|")
	// if len(orParts) == 1 {
	// 	return strings.Contains(title, keyword)
	// }
	// for _, orPart := range orParts {
	// 	orPart = strings.TrimSpace(orPart)

	// 	// 处理与关系 (+) 和非关系 (~)
	// 	andParts := strings.Split(orPart, "+")
	// 	allAndPartsMatch := true

	// 	for _, andPart := range andParts {
	// 		andPart = strings.TrimSpace(andPart)

	// 		// 处理非关系 (~)
	// 		notParts := strings.Split(andPart, "~")
	// 		mainKeyword := strings.TrimSpace(notParts[0])

	// 		// 检查主关键字是否存在
	// 		if !strings.Contains(title, mainKeyword) {
	// 			allAndPartsMatch = false
	// 			break
	// 		}

	// 		// 检查排除关键字
	// 		for i := 1; i < len(notParts); i++ {
	// 			notKeyword := strings.TrimSpace(notParts[i])
	// 			if strings.Contains(title, notKeyword) {
	// 				allAndPartsMatch = false
	// 				break
	// 			}
	// 		}

	// 		if !allAndPartsMatch {
	// 			break
	// 		}
	// 	}

	// 	// 如果所有 AND 条件都匹配，返回 true
	// 	if allAndPartsMatch {
	// 		return true
	// 	}
	// }
	// return false
}

type MessageOption struct {
	ChatId   int64
	FeedName string
	Keywords []string
}

func removeHash(u string) (string, error) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	// 将锚点部分设置为空
	parsedUrl.Fragment = ""
	return parsedUrl.String(), nil
}

func (f *NsFeed) sendMessage(c *MessageOption, feedName string, items []*gofeed.Item) {

	for _, item := range items {
		item.Link, _ = removeHash(item.Link)
		if item.Link == "" {
			continue
		}
		exists := db.GetNotifyHistory(c.ChatId, item.Link) != nil
		if hasKeyword(item.Title, c.Keywords) && !exists {

			db.AddNotifyHistory(&db.NotifyHistory{
				ChatId: c.ChatId,
				Url:    item.Link,
				Title:  item.Title,
			})
			if f.bot != nil {
				msg := NotifyMessage{
					Text: fmt.Sprintf("📢  *%s*\n\n🕐 %s\n\n👉 %s",
						item.Title,
						item.PublishedParsed.Add(time.Hour*8).Format("2006-01-02 15:04:05"),
						item.Link),
					ChatId: &c.ChatId,
				}

				f.bot.Notify(msg)
			}
		}
	}

}

var isRunning bool

func (f *NsFeed) loadRssData(url string, ctx context.Context) (*gofeed.Feed, error) {
	defer func() {
		rescue.Recover()
	}()
	fp := gofeed.NewParser()
	//fix:
	if strings.Contains(url, "nodeloc_rss") {
		return fp.ParseURLWithContext(url, ctx)
	}
	reqClient := req.C().ImpersonateChrome()
	resp, err := reqClient.R().Get(url)
	if err != nil {
		return nil, err
	}

	return fp.ParseString(resp.String())
}

func (f *NsFeed) fetchRss() {
	if isRunning {
		fmt.Println("fetch rss is running")
		return
	}
	isRunning = true
	defer func() {
		isRunning = false
	}()
	feedCnf := db.ListAllFeedConfig()

	if len(feedCnf) == 0 {
		f.logger.Errorw("fetch rss failed", logx.Field("err", "feed config is empty"))
		return
	}

	// 第一步：抓取所有RSS源的数据
	feedItems := make(map[string][]*gofeed.Item)
	var mux sync.Mutex
	var wg threading.RoutineGroup
	for _, cnf := range feedCnf {
		cnf := cnf
		wg.RunSafe(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			feed, err := f.loadRssData(cnf.FeedUrl, ctx)
			if err != nil {
				f.logger.Errorw("fetch rss failed", logx.Field("err", err), logx.Field("feedUrl", cnf.FeedUrl))
				return
			}
			if feed == nil {
				f.logger.Errorw("fetch rss failed", logx.Field("err", "feed is nil"), logx.Field("feedUrl", cnf.FeedUrl))
				return
			}
			var items []*gofeed.Item
			for _, item := range feed.Items {
				items = append(items, item)
			}
			mux.Lock()
			feedItems[cnf.FeedId] = items
			mux.Unlock()
		})
	}
	wg.Wait()

	// 第二步：获取所有活跃订阅
	subscribes := db.ListSubscribes()
	subscribes = funk.Filter(subscribes, func(c *db.Subscribe) bool {
		c.Status = strings.ToLower(c.Status)
		c.Status = strings.TrimSpace(c.Status)
		return c.Status == "on" || c.Status == ""
	}).([]*db.Subscribe)

	// 第三步：使用工作池模式处理订阅消息
	// 创建任务通道
	type subscribeTask struct {
		subscribe *db.Subscribe
		feedId    string
		items     []*gofeed.Item
	}

	// 估算任务总数
	taskCount := 0
	for range subscribes {
		for range feedItems {
			taskCount++
		}
	}

	// 创建任务通道，缓冲大小为任务总数
	taskChan := make(chan subscribeTask, taskCount)

	// 创建工作池
	const workerCount = 5 // 工作协程数量，可以根据实际情况调整
	var workerWg sync.WaitGroup

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			defer rescue.Recover()

			for task := range taskChan {
				subKeys := db.ListSubscribeFeedWith(task.subscribe.ChatId, task.feedId)
				if subKeys.ID == 0 {
					continue
				}

				f.sendMessage(&MessageOption{
					ChatId:   task.subscribe.ChatId,
					FeedName: task.feedId,
					Keywords: subKeys.KeywordsArray,
				}, task.feedId, task.items)
			}
		}()
	}

	// 分发任务
	for _, subscribe := range subscribes {
		for feedId, items := range feedItems {
			taskChan <- subscribeTask{
				subscribe: subscribe,
				feedId:    feedId,
				items:     items,
			}
		}
	}

	// 关闭任务通道，表示没有更多任务
	close(taskChan)

	// 等待所有工作协程完成
	workerWg.Wait()
}

func (f *NsFeed) adjustInterval(rss string, success bool) {
	f.Lock()
	defer f.Unlock()
	l := logx.WithContext(context.Background()).WithFields(logx.Field("rss", rss))
	if success {
		f.failureCount = 0
		f.successCount++

		// 连续成功10次后，尝试减少间隔
		if f.successCount >= 10 {
			newInterval := f.interval - (5 * time.Second)
			if newInterval >= f.minInterval {
				f.interval = newInterval
				l.Infow(fmt.Sprintf("RSS请求稳定，减少间隔至 %v", f.interval))
			}
			f.successCount = 0
		}
	} else {
		f.successCount = 0
		f.failureCount++

		// 失败后立即增加间隔
		multiplier := float64(f.failureCount)
		if multiplier > 4 {
			multiplier = 4 // 限制最大倍数
		}

		newInterval := time.Duration(float64(f.interval) * (1 + (0.5 * multiplier)))
		if newInterval <= f.maxInterval {
			f.interval = newInterval
			l.Infow(fmt.Sprintf("RSS请求失败，增加间隔至 %v", f.interval))
		} else {
			f.interval = f.maxInterval
			l.Infow(fmt.Sprintf("RSS请求达到最大间隔 %v", f.maxInterval))
		}
	}
}

func (f *NsFeed) fetchRssAdaptive(feed *db.FeedConfig) error {
	defer rescue.Recover()

	resp, err := f.loadRssData(feed.FeedUrl, f.ctx)

	if err != nil || resp == nil {
		logx.Errorw("获取RSS失败",
			logx.Field("err", err),
			logx.Field("feedUrl", feed.FeedUrl),
		)

		f.adjustInterval(feed.FeedUrl, false)
		return err
	}

	// 请求成功
	f.adjustInterval(feed.FeedUrl, true)

	if len(resp.Items) == 0 {
		return nil
	}

	f.Lock()
	defer f.Unlock()

	// 获取所有活跃订阅
	subscribes := db.ListSubscribes()
	subscribes = funk.Filter(subscribes, func(c *db.Subscribe) bool {
		c.Status = strings.ToLower(c.Status)
		c.Status = strings.TrimSpace(c.Status)
		return c.Status == "on" || c.Status == ""
	}).([]*db.Subscribe)

	// 使用工作池模式处理订阅消息
	// 创建任务通道
	type subscribeTask struct {
		subscribe *db.Subscribe
		item      *gofeed.Item
	}

	// 估算任务总数
	taskCount := len(subscribes) * len(resp.Items)
	if taskCount == 0 {
		return nil
	}

	// 创建任务通道，缓冲大小为任务总数
	taskChan := make(chan subscribeTask, taskCount)

	// 创建工作池
	const workerCount = 5 // 工作协程数量，可以根据实际情况调整
	var workerWg sync.WaitGroup

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			defer rescue.Recover()

			for task := range taskChan {
				subKeys := db.ListSubscribeFeedWith(task.subscribe.ChatId, feed.FeedId)
				if subKeys.ID == 0 {
					continue
				}

				f.sendMessage(&MessageOption{
					ChatId:   task.subscribe.ChatId,
					FeedName: feed.Name,
					Keywords: subKeys.KeywordsArray,
				}, feed.Name, []*gofeed.Item{task.item})
			}
		}()
	}

	// 分发任务
	for _, subscribe := range subscribes {
		for _, item := range resp.Items {
			taskChan <- subscribeTask{
				subscribe: subscribe,
				item:      item,
			}
		}
	}

	// 关闭任务通道，表示没有更多任务
	close(taskChan)

	// 等待所有工作协程完成
	workerWg.Wait()

	return nil
}

func (f *NsFeed) startAdaptiveFetch() {
	// 创建一个工作池来处理RSS源的抓取
	const workerCount = 3 // 工作协程数量，可以根据实际情况调整

	// 创建任务通道
	type fetchTask struct {
		feed          db.FeedConfig
		interval      time.Duration
		minInterval   time.Duration
		maxInterval   time.Duration
		successCount  int
		failureCount  int
		nextFetchTime time.Time
	}

	// 初始化任务列表
	var tasks []fetchTask
	for _, feed := range db.ListAllFeedConfig() {
		tasks = append(tasks, fetchTask{
			feed:          feed,
			interval:      10 * time.Second,
			minInterval:   10 * time.Second,
			maxInterval:   5 * time.Minute,
			successCount:  0,
			failureCount:  0,
			nextFetchTime: time.Now(),
		})
	}

	// 启动调度器
	go func() {
		defer rescue.Recover()

		// 创建任务通道
		taskChan := make(chan *fetchTask, len(tasks))

		// 启动工作协程
		var wg sync.WaitGroup
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer rescue.Recover()

				for task := range taskChan {
					ctx := logx.ContextWithFields(context.Background(), logx.Field("rss", task.feed.FeedUrl))

					if err := f.fetchRssAdaptive(&task.feed); err != nil {
						task.failureCount++
						task.successCount = 0

						multiplier := float64(task.failureCount)
						if multiplier > 4 {
							multiplier = 4 // 限制最大倍数
						}

						newInterval := time.Duration(float64(task.interval) * (1 + (0.5 * multiplier)))
						if newInterval <= task.maxInterval {
							task.interval = newInterval
							logx.WithContext(ctx).Infow(fmt.Sprintf("RSS请求失败，增加间隔至 %v", task.interval))
						} else {
							task.interval = task.maxInterval
							logx.WithContext(ctx).Infow(fmt.Sprintf("RSS请求达到最大间隔 %v", task.maxInterval))
						}
					} else {
						task.failureCount = 0
						task.successCount++

						if task.successCount >= 10 {
							newInterval := task.interval - (5 * time.Second)
							if newInterval >= task.minInterval {
								task.interval = newInterval
								logx.WithContext(ctx).Infow(fmt.Sprintf("RSS请求稳定，减少间隔至 %v", task.interval))
							}
							task.successCount = 0
						}
					}

					// 更新下次抓取时间
					task.nextFetchTime = time.Now().Add(task.interval)
				}
			}()
		}

		// 主循环，调度任务
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-f.ctx.Done():
				close(taskChan)
				wg.Wait()
				return

			case <-ticker.C:
				now := time.Now()

				// 检查每个任务，如果到了执行时间就发送到任务通道
				for i := range tasks {
					if now.After(tasks[i].nextFetchTime) {
						select {
						case taskChan <- &tasks[i]:
							// 临时设置下次执行时间为很久以后，防止重复调度
							// 实际的下次执行时间会在任务完成后更新
							tasks[i].nextFetchTime = now.Add(24 * time.Hour)
						default:
							// 任务通道已满，跳过
						}
					}
				}
			}
		}
	}()
}

func (f *NsFeed) Start() {
	defer func() {
		rescue.Recover()
	}()
	f.logger.Infow("start ns feed......")
	//
	//ds, e := time.ParseDuration(f.svc.Config.FetchTimeInterval)
	//if e != nil {
	//	f.logger.Errorw("parse duration failed", logx.Field("err", e), logx.Field("FetchTimeInterval", f.svc.Config.FetchTimeInterval))
	//	ds = 10 * time.Second
	//}
	//if ds < 10*time.Second {
	//	ds = 10 * time.Second
	//}
	//go func() {
	//	defer func() {
	//		rescue.Recover()
	//	}()
	//	tk := time.NewTicker(ds)
	//	defer tk.Stop()
	//	for {
	//		select {
	//		case <-f.ctx.Done():
	//			return
	//		case <-tk.C:
	//			f.fetchRss()
	//		}
	//	}
	//}()

	f.startAdaptiveFetch()
}
