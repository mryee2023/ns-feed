package db

import (
	"time"

	"github.com/pocketbase/pocketbase/models"
)

type NotifyHistory struct {
	ID        string    `json:"id"`
	ChatId    int64     `json:"chat_id"`
	Url       string    `json:"url"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created"`
}

// AddNotifyHistory creates a new notification history
func AddNotifyHistory(nh *NotifyHistory) error {
	collection := GetPB().Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return errors.New("collection not found")
	}

	// Check if notification already exists
	record, _ := GetPB().Dao().FindFirstRecord(collection, "chat_id = {} && url = {}", nh.ChatId, nh.Url)
	if record != nil {
		return nil
	}

	// Create new record
	record = models.NewRecord(collection)
	record.Set("chat_id", nh.ChatId)
	record.Set("url", nh.Url)
	record.Set("title", nh.Title)

	return GetPB().Dao().SaveRecord(record)
}

// GetNotifyHistory retrieves a notification history by ChatId and Url
func GetNotifyHistory(chatId int64, url string) *NotifyHistory {
	collection := GetPB().Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return nil
	}

	record, err := GetPB().Dao().FindFirstRecord(collection, "chat_id = {} && url = {}", chatId, url)
	if err != nil {
		return nil
	}

	nh := &NotifyHistory{
		ID:        record.Id,
		ChatId:    record.GetInt("chat_id"),
		Url:       record.GetString("url"),
		Title:     record.GetString("title"),
		CreatedAt: record.GetDateTime("created"),
	}

	return nh
}

func GetNotifyCountByDateTime(start, end time.Time) int64 {
	collection := GetPB().Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return 0
	}

	records, err := GetPB().Dao().FindRecordsByExpr(collection, "created >= {} && created < {}", start, end)
	if err != nil {
		return 0
	}

	return int64(len(records))
}
