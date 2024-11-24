package lib

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/matoous/go-nanoid/v2"
	"github.com/mmcdole/gofeed"
	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
	"github.com/zeromicro/go-zero/core/threading"
	"ns-rss/src/app/config"
)

// 存放发送历史
//
//	{
//	   "chatId":{"link1":{},"link2":{}}
//	}
var history = make(map[int64]map[string]struct{})

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
	go func() {
		defer func() {
			rescue.Recover()
		}()
		tk := time.NewTicker(10 * time.Second)
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
		if strings.Contains(strings.ToLower(title), strings.ToLower(keyword)) {
			return true
		}
	}
	return false

}

var mutex sync.Mutex

func (f *NsFeed) postToChannel(c *config.ChannelInfo, feed *gofeed.Feed) {
	if len(c.Keywords) == 0 {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	var his map[string]struct{}
	his, ok := history[c.ChatId]
	if !ok {
		his = make(map[string]struct{})
		history[c.ChatId] = his
	}
	for _, item := range feed.Items {
		_, exists := history[c.ChatId][item.Link]
		if hasKeyword(item.Title, c.Keywords) && !exists {
			history[c.ChatId][item.Link] = struct{}{}
			if f.bot != nil {
				msg := NotifyMessage{
					Text: fmt.Sprintf("📢 *%s*\n\n🕐 %s\n\n👉 %s",
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

func (f *NsFeed) fetchRss() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(f.svc.Config.NsFeed, ctx)
	if err != nil {
		f.logger.Errorw("fetch rss failed", logx.Field("err", err))
		return
	}
	if feed == nil {
		f.logger.Errorw("fetch rss failed", logx.Field("err", "feed is nil"))
		return
	}

	var wg threading.RoutineGroup
	for _, channel := range f.svc.Config.Channels {
		channel := channel
		wg.RunSafe(func() {
			f.postToChannel(channel, feed)
		})
	}
	wg.Wait()
	//
	//for _, item := range feed.Items {
	//	item := item
	//	wg.RunSafe(func() {})
	//_, exists := history[item.Link]
	//if hasKeyword(item.Title, f.svc.Config.Keywords) && !exists {
	//	history[item.Link] = struct{}{}
	//	if f.bot != nil {
	//		msg := NotifyMessage{
	//			Text: fmt.Sprintf("📢 *%s*\n\n🕐 %s\n\n👉 [%s](%s)",
	//				item.Title,
	//				item.PublishedParsed.Add(time.Hour*8).Format("2006-01-02 15:04:05"),
	//				item.Link, item.Link),
	//			ChatId: &f.svc.Config.TgChatId,
	//		}
	//		f.Add(msg)
	//	}
	//}
	//}

}
