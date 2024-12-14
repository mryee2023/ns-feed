package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "net/http/pprof"

	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/rescue"
	"gopkg.in/yaml.v3"
	"ns-rss/src/app"
	config2 "ns-rss/src/app/config"
	"ns-rss/src/app/db"
	"ns-rss/src/app/lib"
)

var configFile = flag.String("f", "/etc/config.yaml", "the config file")
var dbFile = flag.String("db", "/db/sqlite.db", "the db file")
var config config2.Config
var bot lib.BotNotifier

func getAbsolutePath() string {
	// 获取当前可执行文件的路径
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("os.Executable() failed: %v", err)
	}
	// 获取绝对路径
	return filepath.Dir(exe)
}

func syncSubscribes(subs []*config2.Subscribe) {
	//write to db
	for _, subscribe := range subs {
		if len(subscribe.Keywords) > 0 {
			db.AddSubscribe(&db.Subscribe{
				Name:      subscribe.Name,
				ChatId:    subscribe.ChatId,
				Keywords:  app.ToJson(subscribe.Keywords),
				Status:    subscribe.Status,
				Type:      subscribe.Type,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
		}
	}
}

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		PrettyPrint:     false,
	})

	log.SetLevel(log.InfoLevel)
	carbon.SetDefault(carbon.Default{
		Layout:       carbon.DateTimeLayout,
		Timezone:     carbon.PRC,
		WeekStartsAt: carbon.Monday,
		Locale:       "zh-CN",
	})

	defer func() {
		rescue.Recover()
	}()

	log.SetOutput(os.Stdout)

	flag.Parse()

	if e := db.InitDB(*dbFile); e != nil {
		log.Fatalf("init db failure:%v", e)
	}

	b, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("load config failure :%s, %v", *configFile, err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("unmarshal config failure: %v", err)
	}

	bot = lib.NewTelegramNotifier(config.TgToken, cast.ToString(config.AdminId))
	if bot == nil {
		log.Fatalf("error: invalid bot platform")
	}

	syncSubscribes(config.Subscribes)

	proc.AddShutdownListener(func() {
		bot.Notify(lib.NotifyMessage{Text: "⚠️ NodeSeek Feed服务已停止", ChatId: &config.AdminId})
		log.Info("service shutdown")
	})

	lib.InitTgBotListen(&config)
	svc := lib.NewServiceCtx(lib.TgBotInstance(), &config)
	app.ConfigFilePath = *configFile
	bot.Notify(lib.NotifyMessage{Text: fmt.Sprintf("📢 NodeSeek Feed服务已启动。"), ChatId: &config.AdminId})

	go func() {
		feeder := lib.NewNsFeed(context.Background(), svc)
		feeder.SetBot(bot)
		feeder.Start()
	}()
	var port = ":8080"
	if config.Port != "" {
		port = config.Port
	}

	// 定义路由
	http.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		var rtn = make(map[string]interface{})
		rtn["code"] = 1000
		rtn["msg"] = time.Now().Format("2006-01-02 15:04:05")
		_, _ = writer.Write([]byte(`{"code":1000,"msg":"pong"}`))
	})
	log.Infof("Service start success,Listen On " + port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("start web server failure : %v", err)
	}
	//开启web服务

	select {}

}
