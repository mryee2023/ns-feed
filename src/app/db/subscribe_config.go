package db

import (
	"encoding/json"
	"time"
)

type SubscribeConfig struct {
	ID            string    `json:"id"`
	ChatId        int64     `json:"chat_id"`
	Keywords      string    `json:"keywords"`
	KeywordsArray []string  `json:"-"`
	FeedId        string    `json:"feed_id"`
	CreatedAt     time.Time `json:"created"`
	UpdatedAt     time.Time `json:"updated"`
}

// BeforeSave 在保存到数据库前将 KeywordsArray 序列化为 Keywords
func (s *SubscribeConfig) BeforeSave() error {
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
func (s *SubscribeConfig) AfterFind() error {
	if s.Keywords != "" {
		return json.Unmarshal([]byte(s.Keywords), &s.KeywordsArray)
	}
	return nil
}

func AddSubscribeConfig(cnf SubscribeConfig) error {
	collection := GetPB().Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return errors.New("collection not found")
	}

	if err := cnf.BeforeSave(); err != nil {
		return err
	}

	record, _ := GetPB().Dao().FindFirstRecord(collection, "chat_id = {} && feed_id = {}", cnf.ChatId, cnf.FeedId)
	if record != nil {
		// Update existing record
		record.Set("keywords", cnf.Keywords)
		return GetPB().Dao().SaveRecord(record)
	}

	// Create new record
	record = models.NewRecord(collection)
	record.Set("chat_id", cnf.ChatId)
	record.Set("feed_id", cnf.FeedId)
	record.Set("keywords", cnf.Keywords)

	return GetPB().Dao().SaveRecord(record)
}

func ListSubscribeFeedConfig(chatId int64) map[string][]string {
	collection := GetPB().Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return nil
	}

	records, err := GetPB().Dao().FindRecordsByExpr(collection, "chat_id = {}", chatId)
	if err != nil {
		return nil
	}

	m := make(map[string][]string)
	for _, record := range records {
		config := &SubscribeConfig{
			Keywords: record.GetString("keywords"),
			FeedId:  record.GetString("feed_id"),
		}
		config.AfterFind()
		m[config.FeedId] = config.KeywordsArray
	}
	return m
}

func ListSubscribeFeedWith(chatId int64, feedId string) SubscribeConfig {
	collection := GetPB().Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return SubscribeConfig{}
	}

	record, err := GetPB().Dao().FindFirstRecord(collection, "chat_id = {} && feed_id = {}", chatId, feedId)
	if err != nil {
		return SubscribeConfig{}
	}

	config := SubscribeConfig{
		ID:        record.Id,
		ChatId:    record.GetInt("chat_id"),
		Keywords:  record.GetString("keywords"),
		FeedId:    record.GetString("feed_id"),
		CreatedAt: record.GetDateTime("created"),
		UpdatedAt: record.GetDateTime("updated"),
	}
	config.AfterFind()
	return config
}
