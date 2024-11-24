package lib

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"ns-rss/src/app/config"
)

type ServiceCtx struct {
	TgBotAPi *tgbotapi.BotAPI
	Config   *config.Config
}

func NewServiceCtx(tgBotAPi *tgbotapi.BotAPI, config *config.Config) *ServiceCtx {
	return &ServiceCtx{
		TgBotAPi: tgBotAPi,
		Config:   config,
	}
}

func (s *ServiceCtx) SetConfigPath(path string) *ServiceCtx {
	return s
}
