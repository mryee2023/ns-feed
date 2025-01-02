package db

import (
	"errors"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
	"github.com/spf13/cast"
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
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return errors.New("collection not found")
	}

	// Check if notification already exists
	record, _ := GetPB().Dao().FindFirstRecordByFilter("notify_histories", "chat_id = {:cid} && url = {:url}", dbx.Params{
		"cid": nh.ChatId,
		"url": nh.Url,
	})
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
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return nil
	}

	record, err := GetPB().Dao().FindFirstRecordByFilter("notify_histories", "chat_id = {:chatId} && url = {:url}",
		dbx.Params{
			"chatId": chatId,
			"url":    url,
		},
	)
	if err != nil {
		return nil
	}

	nh := &NotifyHistory{
		ID:        record.Id,
		ChatId:    cast.ToInt64(record.GetInt("chat_id")),
		Url:       record.GetString("url"),
		Title:     record.GetString("title"),
		CreatedAt: record.GetDateTime("created").Time(),
	}

	return nh
}

func GetNotifyCountByDateTime(start, end time.Time) int64 {
	collection, _ := GetPB().Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return 0
	}
	exp := dbx.NewExp("created >= {:a} && created < {:b}", dbx.Params{
		"a": start,
		"b": end,
	})
	records, err := GetPB().Dao().FindRecordsByExpr("notify_histories", exp)
	if err != nil {
		return 0
	}

	return int64(len(records))
}
