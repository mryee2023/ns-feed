package db

import (
	"time"

	json "github.com/bytedance/sonic"

	"gorm.io/gorm"
)

type SubscribeConfig struct {
	ID            uint      `gorm:"primaryKey,autoIncrement"`
	ChatId        int64     `gorm:"not null,index"`
	Keywords      string    `gorm:"not null"`
	KeywordsArray []string  `gorm:"-"`
	FeedId        string    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (s *SubscribeConfig) TableName() string {
	return "subscribe_config"
}

func AddSubscribeConfig(cnf SubscribeConfig) {

	var exists SubscribeConfig
	db.Where("chat_id = ? AND feed_id = ?", cnf.ChatId, cnf.FeedId).First(&exists)
	if exists.ID > 0 {
		db.Save(&cnf)
		return
	}
	db.Create(&cnf)

}

// BeforeSave 在保存到数据库前将 KeywordsArray 序列化为 Keywords
func (s *SubscribeConfig) BeforeSave(tx *gorm.DB) error {
	if len(s.KeywordsArray) > 0 {
		keywords, err := json.Marshal(s.KeywordsArray)
		if err != nil {
			return err
		}
		s.Keywords = string(keywords)
	} else {
		s.Keywords = ""
	}
	return nil
}

// AfterFind 在从数据库读取后将 Keywords 反序列化为 KeywordsArray
func (s *SubscribeConfig) AfterFind(tx *gorm.DB) error {
	if s.Keywords != "" {
		return json.Unmarshal([]byte(s.Keywords), &s.KeywordsArray)
	}
	return nil
}

func ListSubscribeFeedConfig(chatId int64) map[string][]string {
	var cnf []*SubscribeConfig
	db.Where("chat_id = ?", chatId).Find(&cnf)
	m := make(map[string][]string)
	for _, c := range cnf {
		m[c.FeedId] = c.KeywordsArray
	}
	return m
}

func ListSubscribeFeedWith(chatId int64, feedId string) SubscribeConfig {
	var cnf SubscribeConfig
	db.Where("chat_id = ? and feed_id = ?", chatId, feedId).First(&cnf)
	return cnf
}
