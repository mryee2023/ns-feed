package db

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
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
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return errors.New("collection not found")
	}

	if err := cnf.BeforeSave(); err != nil {
		return err
	}

	record, _ := GetPB().Dao().FindFirstRecordByFilter("subscribe_configs", "chat_id = {:cid} && feed_id = {:fid}",
		dbx.Params{
			"cid": cnf.ChatId,
			"fid": cnf.FeedId,
		},
	)
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
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return nil
	}

	expr := dbx.NewExp("chat_id = {:cid}", dbx.Params{
		"cid": chatId,
	})
	records, err := GetPB().Dao().FindRecordsByExpr("subscribe_configs", expr)
	if err != nil {
		return nil
	}

	m := make(map[string][]string)
	for _, record := range records {
		config := &SubscribeConfig{
			Keywords: record.GetString("keywords"),
			FeedId:   record.GetString("feed_id"),
		}
		config.AfterFind()
		m[config.FeedId] = config.KeywordsArray
	}
	return m
}

func ListSubscribeFeedWith(chatId int64, feedId string) SubscribeConfig {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return SubscribeConfig{}
	}

	record, err := GetPB().Dao().FindFirstRecordByFilter("subscribe_configs", "chat_id = {:cid} && feed_id = {:fid}",
		dbx.Params{
			"cid": chatId,
			"fid": feedId,
		})
	if err != nil {
		return SubscribeConfig{}
	}

	config := SubscribeConfig{
		ID:        record.Id,
		ChatId:    int64(record.GetInt("chat_id")),
		Keywords:  record.GetString("keywords"),
		FeedId:    record.GetString("feed_id"),
		CreatedAt: record.GetDateTime("created").Time(),
		UpdatedAt: record.GetDateTime("updated").Time(),
	}
	config.AfterFind()
	return config
}
