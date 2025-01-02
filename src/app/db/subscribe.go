package db

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
)

type Subscribe struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ChatId        int64     `json:"chat_id"`
	Keywords      string    `json:"keywords"`
	KeywordsArray []string  `json:"-"`
	Status        string    `json:"status"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created"`
	UpdatedAt     time.Time `json:"updated"`
}

// BeforeSave 在保存到数据库前将 KeywordsArray 序列化为 Keywords
func (s *Subscribe) BeforeSave() error {
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
func (s *Subscribe) AfterFind() error {
	if s.Keywords != "" {
		return json.Unmarshal([]byte(s.Keywords), &s.KeywordsArray)
	}
	return nil
}

// InitDB initializes the database connection
func InitDB(dbPath string) error {
	return InitPocketBase(dbPath)
}

// AddSubscribe creates a new subscription
func AddSubscribe(sub *Subscribe) error {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribes")
	if collection == nil {
		return errors.New("collection not found")
	}

	// Check if subscription already exists
	record, _ := GetPB().Dao().FindFirstRecordByFilter("subscribes", "chat_id = {:a}", dbx.Params{
		"a": sub.ChatId,
	})
	if record != nil {
		return nil
	}

	// Create new record
	record = models.NewRecord(collection)
	if err := sub.BeforeSave(); err != nil {
		return err
	}

	record.Set("name", sub.Name)
	record.Set("chat_id", sub.ChatId)
	record.Set("keywords", sub.Keywords)
	record.Set("status", sub.Status)
	record.Set("type", sub.Type)

	return GetPB().Dao().SaveRecord(record)
}

// GetSubscribeWithChatId retrieves a subscription by ChatId
func GetSubscribeWithChatId(chatId int64) *Subscribe {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribes")
	if collection == nil {
		return nil
	}

	record, err := GetPB().Dao().FindFirstRecordByFilter("subscribes", "chat_id = {:id}", dbx.Params{
		"id": chatId,
	})
	if err != nil {
		return nil
	}

	sub := &Subscribe{
		ID:        record.Id,
		Name:      record.GetString("name"),
		ChatId:    int64(record.GetInt("chat_id")),
		Keywords:  record.GetString("keywords"),
		Status:    record.GetString("status"),
		Type:      record.GetString("type"),
		CreatedAt: record.GetDateTime("created").Time(),
		UpdatedAt: record.GetDateTime("updated").Time(),
	}

	sub.AfterFind()
	return sub
}

// UpdateSubscribe updates an existing subscription
func UpdateSubscribe(sub *Subscribe) error {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribes")
	if collection == nil {
		return errors.New("collection not found")
	}

	record, err := GetPB().Dao().FindFirstRecordByFilter("subscribes", "chat_id = {:id}",
		dbx.Params{"id": sub.ChatId})
	if err != nil {
		return err
	}

	if err := sub.BeforeSave(); err != nil {
		return err
	}

	record.Set("name", sub.Name)
	record.Set("keywords", sub.Keywords)
	record.Set("status", sub.Status)
	record.Set("type", sub.Type)

	return GetPB().Dao().SaveRecord(record)
}

// DeleteSubscribe deletes a subscription by ChatId
func DeleteSubscribe(chatId int64) error {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribes")
	if collection == nil {
		return errors.New("collection not found")
	}

	record, err := GetPB().Dao().FindFirstRecordByFilter("subscribes", "chat_id = {:id}",
		dbx.Params{"id": chatId})
	if err != nil {
		return err
	}

	return GetPB().Dao().DeleteRecord(record)
}

// ListSubscribes returns all subscriptions
func ListSubscribes() []*Subscribe {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("subscribes")
	if collection == nil {
		return nil
	}

	records, err := GetPB().Dao().FindRecordsByExpr("subscribes")
	if err != nil {
		return nil
	}

	var subs []*Subscribe
	for _, record := range records {
		sub := &Subscribe{

			ID:        record.Id,
			Name:      record.GetString("name"),
			ChatId:    int64(record.GetInt("chat_id")),
			Keywords:  record.GetString("keywords"),
			Status:    record.GetString("status"),
			Type:      record.GetString("type"),
			CreatedAt: record.GetDateTime("created").Time(),
			UpdatedAt: record.GetDateTime("updated").Time(),
		}
		sub.AfterFind()
		subs = append(subs, sub)
	}

	return subs
}
