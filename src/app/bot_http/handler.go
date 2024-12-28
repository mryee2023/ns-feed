package bot_http

import (
	"net/http"

	"github.com/thoas/go-funk"
	"ns-rss/src/app"
	"ns-rss/src/app/db"
	"ns-rss/src/app/lib"
)

type BotHttpHandler func(writer http.ResponseWriter, request *http.Request)

func validateToken(writer http.ResponseWriter, request *http.Request) bool {
	token := request.Header.Get("accessKey")
	v := token == app.GetConfig().AccessKey
	if !v {
		writer.WriteHeader(http.StatusUnauthorized)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":401,"msg":"Unauthorized"}`))
	}
	return v
}

// RouteHandler 命令处理器映射
var RouteHandler = map[string]BotHttpHandler{
	"/ping":                httpHandlerPing,
	"/api/feed":            httpHandlerFeed,
	"/api/subscribe/trans": httpHandlerSubscribeTrans,
	"/api/notice":          httpHandlerNotice,
}

func httpHandlerPing(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"code":1000,"msg":"pong"}`))
}

func httpHandlerFeed(writer http.ResponseWriter, request *http.Request) {
	if validateToken(writer, request) == false {
		return
	}
	if request.Method == "GET" {
		feeds := db.ListAllFeedConfig()
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(app.ToJson(feeds)))
		return
	}

	if err := request.ParseForm(); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	feedId := request.FormValue("feed_id")
	feedUrl := request.FormValue("feed_url")
	feedName := request.FormValue("feed_name")
	if feedId == "" || feedUrl == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	db.AddOrUpdateFeed(db.FeedConfig{
		Name:    feedName,
		FeedUrl: feedUrl,
		FeedId:  feedId,
	})

	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"code":1000,"msg":"success"}`))
}

func httpHandlerSubscribeTrans(writer http.ResponseWriter, request *http.Request) {
	// 转换订阅数据
	//查询所有订阅者
	if validateToken(writer, request) == false {
		return
	}
	subs := db.ListSubscribes()
	for _, sub := range subs {
		if len(sub.KeywordsArray) == 0 {
			continue
		}
		db.AddSubscribeConfig(db.SubscribeConfig{
			ChatId:        sub.ChatId,
			KeywordsArray: sub.KeywordsArray,
			FeedId:        "ns",
		})

	}
}

func httpHandlerNotice(writer http.ResponseWriter, request *http.Request) {
	if validateToken(writer, request) == false {
		return
	}
	if request.Method != "POST" {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := request.ParseForm(); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	text := request.FormValue("text")
	if text == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	//查询所有订阅者，并发送通知
	subs := db.ListSubscribes()
	subs = funk.Filter(subs, func(c *db.Subscribe) bool {
		return c.Status == "on"
	}).([]*db.Subscribe)
	for _, sub := range subs {
		app.GetBotInstance().Notify(lib.NotifyMessage{Text: text, ChatId: &sub.ChatId})
	}

	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(`{"code":1000,"msg":"success"}`))
}
