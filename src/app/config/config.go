package config

import (
	"os"

	"github.com/zeromicro/go-zero/core/logx"
	"gopkg.in/yaml.v3"
)

var ChatType string

const (
	ChatTypeChat    = "chat"
	ChatTypeGroup   = "group"
	ChatTypeChannel = "channel"
)

type ChannelInfo struct {
	Name     string   `yaml:"name"`
	ChatId   int64    `yaml:"chatId"`
	Keywords []string `yaml:"keywords"`
	Status   string   `yaml:"status"`
	Type     string   `yaml:"type"` //channel, group, chat
}

type Config struct {
	TgToken     string         `yaml:"tgToken"`
	TgChatId    int64          `yaml:"tgChatId"`
	NsFeed      string         `yaml:"nsFeed"`
	AlterChatId int64          `yaml:"alterChatId"`
	Channels    []*ChannelInfo `yaml:"channels"`
}

func (c *Config) Storage(path string) {
	b, e := yaml.Marshal(c)
	if e != nil {
		logx.Errorf("yaml.Marshal(c) error:%v", e)
		return
	}
	err := os.WriteFile(path, b, 0644)
	if err != nil {
		logx.Errorf("os.WriteFile error:%v", err)
		return
	}
}
