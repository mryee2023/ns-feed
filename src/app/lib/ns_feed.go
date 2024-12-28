package lib

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/matoous/go-nanoid/v2"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
	"github.com/zeromicro/go-zero/core/threading"
	"ns-rss/src/app/db"
)

//var history = make(map[int64]map[string]struct{})

// 发送计数
//var noticeHistory = make(map[string]int64)

type NsFeed struct {
	ctx    context.Context
	svc    *ServiceCtx
	logger logx.Logger
	bot    BotNotifier
	q      *collection.TimingWheel
}

func NewNsFeed(ctx context.Context, svc *ServiceCtx) *NsFeed {
	return &NsFeed{
		ctx:    ctx,
		svc:    svc,
		logger: logx.WithContext(ctx).WithFields(logx.Field("lib", "ns_feed")),
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

	ds, e := time.ParseDuration(f.svc.Config.FetchTimeInterval)
	if e != nil {
		f.logger.Errorw("parse duration failed", logx.Field("err", e), logx.Field("FetchTimeInterval", f.svc.Config.FetchTimeInterval))
		ds = 10 * time.Second
	}
	if ds < 10*time.Second {
		ds = 10 * time.Second
	}
	go func() {
		defer func() {
			rescue.Recover()
		}()
		tk := time.NewTicker(ds)
		defer tk.Stop()
		for {
			select {
			case <-f.ctx.Done():
				return
			case <-tk.C:
				f.fetchRss()
			}
		}
	}()

}

func hasKeyword(title string, keywords []string) bool {
	for _, keyword := range keywords {
		keyword = strings.Trim(keyword, "{}")
		if strings.Contains(strings.ToLower(title), strings.ToLower(keyword)) {
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

//var mutex sync.Mutex

func (f *NsFeed) sendMessage(c *MessageOption, feedName string, items []*gofeed.Item) {

	for _, item := range items {
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
				f.Add(msg)
			}
		}
	}

}

var isRunning bool

func (f *NsFeed) fetchRss() {
	if isRunning {
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
			fp := gofeed.NewParser()
			feed, err := fp.ParseURLWithContext(cnf.FeedUrl, ctx)
			if err != nil {
				f.logger.Errorw("fetch rss failed", logx.Field("err", err))
				return
			}
			if feed == nil {
				f.logger.Errorw("fetch rss failed", logx.Field("err", "feed is nil"))
				return
			}
			var items []*gofeed.Item
			for _, item := range feed.Items {
				items = append(items, item)
				//fmt.Println(cnf.FeedId, ",", item.Title, ",", item.Link)
			}
			mux.Lock()
			feedItems[cnf.FeedId] = items
			mux.Unlock()
		})
	}
	wg.Wait()

	subscribes := db.ListSubscribes()
	subscribes = funk.Filter(subscribes, func(c *db.Subscribe) bool {
		return c.Status == "on"
	}).([]*db.Subscribe)

	var swg sync.WaitGroup

	for fid, items := range feedItems {
		swg.Add(1)
		go func(feedId string, items []*gofeed.Item) {
			defer func() {
				swg.Done()
			}()
			for _, c := range subscribes {
				subKeys := db.ListSubscribeFeedWith(c.ChatId, fid)
				if subKeys.ID == 0 {
					continue
				}
				f.sendMessage(&MessageOption{
					ChatId:   c.ChatId,
					FeedName: feedId,
					Keywords: subKeys.KeywordsArray,
				}, feedId, items)
			}
		}(fid, items)

		//for _, c := range subscribes {
		//	subKeys := db.ListSubscribeFeedWith(c.ChatId, fid)
		//	if subKeys.ID == 0 {
		//		continue
		//	}
		//	c.KeywordsArray = subKeys.KeywordsArray
		//	//wg.RunSafe(func() {
		//	f.sendMessage(c, fid, items)
		//	//})
		//
		//}
		//swg.Done()
	}
	swg.Wait()

}
