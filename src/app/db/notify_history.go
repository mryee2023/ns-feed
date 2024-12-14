package db

import (
	"time"
)

type NotifyHistory struct {
	ID        uint      `gorm:"primaryKey,autoIncrement"`
	ChatId    int64     `gorm:"not null;index"`
	Url       string    `gorm:"not null;index"`
	Title     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (n NotifyHistory) TableName() string {
	return "notify_history"
}

// AddNotifyHistory creates a new notification history
func AddNotifyHistory(nh *NotifyHistory) error {
	var exists *NotifyHistory
	db.Where("chat_id = ? AND url = ?", nh.ChatId, nh.Url).First(&exists)
	if exists != nil && exists.ID > 0 {
		return nil
	}
	return db.Create(nh).Error
}

// GetNotifyHistory retrieves a notification history by ChatId and Url
func GetNotifyHistory(chatId int64, url string) *NotifyHistory {
	var nh *NotifyHistory
	db.Where("chat_id = ? AND url = ?", chatId, url).First(&nh)
	if nh == nil || nh.ID == 0 {
		return nil
	}
	return nh
}

func GetNotifyCountByDateTime(start, end time.Time) int64 {
	var count int64
	db.Model(&NotifyHistory{}).Where("created_at >= ? and created_at<?", start, end).Count(&count)
	return count
}
