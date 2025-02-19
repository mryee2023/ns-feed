package lib

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/imroc/req/v3"
	"github.com/matoous/go-nanoid/v2"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
	"github.com/zeromicro/go-zero/core/threading"
	"ns-rss/src/app/config"
	"ns-rss/src/app/db"
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
	q            *collection.TimingWheel
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
		interval:    10 * time.Second, // 初始间隔
		minInterval: 10 * time.Second, // 最小间隔
		maxInterval: 5 * time.Minute,  // 最大间隔
	}
}

func (f *NsFeed) SetBot(bot BotNotifier) *NsFeed {
	f.bot = bot

	f.q, _ = collection.NewTimingWheel(time.Second, 120, func(key, value any) {
		if v, ok := value.(*NotifyMessage); ok {
			bot.Notify(NotifyMessage{
				Text:   v.Text,
				ChatId: v.ChatId,
			})
		}
	})

	return f
}

func (f *NsFeed) Add(msg NotifyMessage) {
	id, _ := gonanoid.New()
	f.q.SetTimer(id, &msg, time.Second*3)
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

func hasKeywordWithRegex(title string, keyword string) bool {
	//尝试转为正则
	re, err := regexp2.Compile(keyword, regexp2.None)
	if err != nil {
		return false
	}
	r, _ := re.MatchString(title)
	return r
}

func hasKeyword(title string, keywords []string) bool {
	title = strings.ToLower(title)
	for _, keyword := range keywords {
		resultChan := make(chan bool, 2)
		// 并行执行两个检查函数
		go func() {
			resultChan <- hasKeywordWithExpression(title, keyword)
		}()
		go func() {
			resultChan <- hasKeywordWithRegex(title, keyword)
		}()

		// 只要有一个返回 true 就可以了
		if <-resultChan || <-resultChan {
			return true
		}
	}
	return false
}

func hasKeywordWithExpression(title string, keyword string) bool {
	keyword = strings.Trim(keyword, "{}")
	keyword = strings.ToLower(keyword)
	// 处理或关系 (|)
	orParts := strings.Split(keyword, "|")
	if len(orParts) == 1 {
		return strings.Contains(title, keyword)
	}
	for _, orPart := range orParts {
		orPart = strings.TrimSpace(orPart)

		// 处理与关系 (+) 和非关系 (~)
		andParts := strings.Split(orPart, "+")
		allAndPartsMatch := true

		for _, andPart := range andParts {
			andPart = strings.TrimSpace(andPart)

			// 处理非关系 (~)
			notParts := strings.Split(andPart, "~")
			mainKeyword := strings.TrimSpace(notParts[0])

			// 检查主关键字是否存在
			if !strings.Contains(title, mainKeyword) {
				allAndPartsMatch = false
				break
			}

			// 检查排除关键字
			for i := 1; i < len(notParts); i++ {
				notKeyword := strings.TrimSpace(notParts[i])
				if strings.Contains(title, notKeyword) {
					allAndPartsMatch = false
					break
				}
			}

			if !allAndPartsMatch {
				break
			}
		}

		// 如果所有 AND 条件都匹配，返回 true
		if allAndPartsMatch {
			return true
		}
	}
	return false
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

	subscribes := db.ListSubscribes()
	subscribes = funk.Filter(subscribes, func(c *db.Subscribe) bool {
		c.Status = strings.ToLower(c.Status)
		c.Status = strings.TrimSpace(c.Status)
		return c.Status == "on" || c.Status == ""
	}).([]*db.Subscribe)

	var swg sync.WaitGroup

	for _, subscribe := range subscribes {
		subscribe := subscribe
		swg.Add(1)
		go func() {
			defer func() {
				swg.Done()
				rescue.Recover()
			}()
			for fid, items := range feedItems {
				subKeys := db.ListSubscribeFeedWith(subscribe.ChatId, fid)
				if subKeys.ID == 0 {
					continue
				}

				f.sendMessage(&MessageOption{
					ChatId:   subscribe.ChatId,
					FeedName: fid,
					Keywords: subKeys.KeywordsArray,
				}, fid, items)
			}
		}()
	}
	swg.Wait()
}

func (f *NsFeed) adjustInterval(success bool) {
	f.Lock()
	defer f.Unlock()

	if success {
		f.failureCount = 0
		f.successCount++

		// 连续成功10次后，尝试减少间隔
		if f.successCount >= 10 {
			newInterval := f.interval - (5 * time.Second)
			if newInterval >= f.minInterval {
				f.interval = newInterval
				log.Infof("RSS请求稳定，减少间隔至 %v", f.interval)
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
			log.Warnf("RSS请求失败，增加间隔至 %v", f.interval)
		} else {
			f.interval = f.maxInterval
			log.Warnf("RSS请求达到最大间隔 %v", f.maxInterval)
		}
	}
}

func (f *NsFeed) fetchRssAdaptive(feed *db.FeedConfig) {
	defer rescue.Recover()

	resp, err := f.loadRssData(feed.FeedUrl, f.ctx)

	if err != nil || resp == nil {
		log.WithError(err).Error("获取RSS失败")
		f.adjustInterval(false)
		return
	}

	// 请求成功
	f.adjustInterval(true)

	if len(resp.Items) == 0 {
		return
	}

	f.Lock()
	defer f.Unlock()

	// 其他处理逻辑保持不变
	subscribes := db.ListSubscribes()
	subscribes = funk.Filter(subscribes, func(c *db.Subscribe) bool {
		c.Status = strings.ToLower(c.Status)
		c.Status = strings.TrimSpace(c.Status)
		return c.Status == "on" || c.Status == ""
	}).([]*db.Subscribe)

	var swg sync.WaitGroup

	for _, subscribe := range subscribes {
		subscribe := subscribe
		swg.Add(1)
		go func() {
			defer func() {
				swg.Done()
				rescue.Recover()
			}()
			for _, items := range resp.Items {
				subKeys := db.ListSubscribeFeedWith(subscribe.ChatId, feed.FeedId)
				if subKeys.ID == 0 {
					continue
				}

				f.sendMessage(&MessageOption{
					ChatId:   subscribe.ChatId,
					FeedName: feed.Name,
					Keywords: subKeys.KeywordsArray,
				}, feed.Name, []*gofeed.Item{items})
			}
		}()
	}
	swg.Wait()
}

func (f *NsFeed) startAdaptiveFetch() {
	go func() {
		defer func() {
			rescue.Recover()
		}()
		for {
			f.Lock()
			currentInterval := f.interval
			f.Unlock()

			time.Sleep(currentInterval)

			feeds := db.ListAllFeedConfig()
			for _, feed := range feeds {
				f.fetchRssAdaptive(&feed)
			}
		}
	}()
}
