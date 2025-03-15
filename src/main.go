package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"ns-rss/src/app"
	"ns-rss/src/app/bot_http"
	config2 "ns-rss/src/app/config"
	"ns-rss/src/app/db"
	"ns-rss/src/app/lib"

	"github.com/golang-module/carbon/v2"
	log "github.com/sirupsen/logrus"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/rescue"
	"gopkg.in/yaml.v3"
)

var (
	// 配置文件
	configFile = flag.String("f", "", "配置文件路径")

	// 数据库相关
	dbFile = flag.String("db", "/db/sqlite.db", "SQLite数据库文件路径")

	// Telegram相关
	tgToken = flag.String("token", "", "Telegram Bot Token")
	adminId = flag.Int64("admin", 0, "管理员的 Telegram Chat ID")

	// RSS相关
	nsFeed        = flag.String("feed", "https://rss.nodeseek.com", "NodeSeek RSS feed URL")
	fetchInterval = flag.Duration("interval", 10*time.Second, "RSS抓取间隔")

	// HTTP服务相关
	port = flag.String("port", ":8080", "HTTP服务端口")
)

func main() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	log.SetLevel(log.InfoLevel)
	logx.SetLevel(logx.InfoLevel)
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

	// 初始化数据库
	if err := db.InitDB(*dbFile); err != nil {
		log.Fatalf("init db failure:%v", err)
	}

	// 如果提供了配置文件，从配置文件读取配置
	var config config2.Config
	if *configFile != "" {
		b, err := os.ReadFile(*configFile)
		if err != nil {
			log.Fatalf("load config failure :%s, %v", *configFile, err)
		}

		err = yaml.Unmarshal(b, &config)
		if err != nil {
			log.Fatalf("unmarshal config failure: %v", err)
		}

		// 使用配置文件中的值
		if config.Port != "" {
			*port = config.Port
		}
		if config.TgToken != "" {
			*tgToken = config.TgToken
		}
		if config.AdminId != 0 {
			*adminId = config.AdminId
		}
		if config.NsFeed != "" {
			*nsFeed = config.NsFeed
		}
		if config.FetchTimeInterval != "" {
			interval, err := time.ParseDuration(config.FetchTimeInterval)
			if err == nil {
				*fetchInterval = interval
			}
		}
	}

	// 验证必要参数
	if *tgToken == "" {
		log.Fatal("Telegram bot token is required")
	}
	if *adminId == 0 {
		log.Fatal("Admin chat ID is required")
	}
	app.SetConfig(&config)
	// 初始化机器人

	app.InitBot(tgToken, adminId)
	if app.GetBotInstance() == nil {
		log.Fatal("error: invalid bot platform")
	}

	// 设置关闭和恢复处理
	proc.AddShutdownListener(func() {
		app.GetBotInstance().Notify(lib.NotifyMessage{Text: "⚠️ NodeSeek Feed服务已停止", ChatId: adminId})
		log.Info("service shutdown")
	})

	rescue.Recover(func() {
		app.GetBotInstance().Notify(lib.NotifyMessage{Text: "⚠️ NodeSeek Feed服务发生异常", ChatId: adminId})
	})

	// 初始化服务
	lib.InitTgBotListen(&config)
	svc := lib.NewServiceCtx(lib.TgBotInstance(), &config)

	// 启动RSS抓取
	go func() {
		if !config.Online {
			log.Info("NodeSeek Feed服务已离线")
			return
		}
		feeder := lib.NewNsFeed(context.Background(), svc, &config)
		feeder.SetBot(app.GetBotInstance())
		feeder.Start()
	}()

	// 启动HTTP服务
	for k, v := range bot_http.RouteHandler {
		http.HandleFunc(k, v)
	}

	log.Info("NodeSeek Feed服务启动成功")
	app.GetBotInstance().Notify(lib.NotifyMessage{Text: "✅ NodeSeek Feed服务已启动", ChatId: adminId})

	log.Infof("Service start success, Listen On %s", *port)
	if err := http.ListenAndServe(*port, nil); err != nil {
		log.Fatalf("start web server failure : %v", err)
	}

}
