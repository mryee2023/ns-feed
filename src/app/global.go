package app

import (
	json "github.com/bytedance/sonic"

	"github.com/spf13/cast"
	config2 "ns-rss/src/app/config"
	"ns-rss/src/app/lib"
)

var config *config2.Config

func SetConfig(cnf *config2.Config) {
	config = cnf

}
func GetConfig() *config2.Config {
	return config
}
func ToJson(v interface{}) string {
	b, e := json.Marshal(v)
	if e != nil {
		return ""
	}
	return string(b)
}

var bot lib.BotNotifier

func InitBot(token *string, adminId *int64) {
	if bot != nil {
		return
	}
	bot = lib.NewTelegramNotifier(*token, cast.ToString(*adminId))
}

func GetBotInstance() lib.BotNotifier {
	return bot
}
