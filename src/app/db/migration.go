package db

import (
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// OldSubscribe represents the old SQLite Subscribe model
type OldSubscribe struct {
	ID            uint      `gorm:"primaryKey,autoIncrement"`
	Name          string    `gorm:"not null"`
	ChatId        int64     `gorm:"not null;uniqueIndex"`
	Keywords      string    `gorm:"not null"`
	KeywordsArray []string  `gorm:"-"`
	Status        string
	Type          string
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// OldNotifyHistory represents the old SQLite NotifyHistory model
type OldNotifyHistory struct {
	ID        uint      `gorm:"primaryKey,autoIncrement"`
	ChatId    int64     `gorm:"not null;index"`
	Url       string    `gorm:"not null;index"`
	Title     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// OldFeedConfig represents the old SQLite FeedConfig model
type OldFeedConfig struct {
	ID      uint   `gorm:"primaryKey,autoIncrement"`
	Name    string `gorm:"not null"`
	FeedUrl string `gorm:"not null"`
	FeedId  string `gorm:"not null;uniqueIndex"`
}

// OldSubscribeConfig represents the old SQLite SubscribeConfig model
type OldSubscribeConfig struct {
	ID            uint      `gorm:"primaryKey,autoIncrement"`
	ChatId        int64     `gorm:"not null,index"`
	Keywords      string    `gorm:"not null"`
	KeywordsArray []string  `gorm:"-"`
	FeedId        string    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// MigrateData migrates data from SQLite to PocketBase
func MigrateData(oldDbPath string, pb *pocketbase.PocketBase) error {
	// Connect to old SQLite database
	oldDb, err := gorm.Open(sqlite.Open(oldDbPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to old database: %v", err)
	}

	// Migrate subscribes
	if err := migrateSubscribes(oldDb, pb); err != nil {
		return fmt.Errorf("failed to migrate subscribes: %v", err)
	}

	// Migrate notify histories
	if err := migrateNotifyHistories(oldDb, pb); err != nil {
		return fmt.Errorf("failed to migrate notify histories: %v", err)
	}

	// Migrate feed configs
	if err := migrateFeedConfigs(oldDb, pb); err != nil {
		return fmt.Errorf("failed to migrate feed configs: %v", err)
	}

	// Migrate subscribe configs
	if err := migrateSubscribeConfigs(oldDb, pb); err != nil {
		return fmt.Errorf("failed to migrate subscribe configs: %v", err)
	}

	log.Println("Data migration completed successfully")
	return nil
}

func migrateSubscribes(oldDb *gorm.DB, pb *pocketbase.PocketBase) error {
	var oldSubscribes []OldSubscribe
	if err := oldDb.Find(&oldSubscribes).Error; err != nil {
		return err
	}

	collection := pb.Dao().FindCollectionByNameOrId("subscribes")
	if collection == nil {
		return fmt.Errorf("subscribes collection not found")
	}

	for _, old := range oldSubscribes {
		record := models.NewRecord(collection)
		record.Set("name", old.Name)
		record.Set("chat_id", old.ChatId)
		record.Set("keywords", old.Keywords)
		record.Set("status", old.Status)
		record.Set("type", old.Type)
		record.Set("created", old.CreatedAt)
		record.Set("updated", old.UpdatedAt)

		if err := pb.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save subscribe record: %v", err)
		}
	}

	return nil
}

func migrateNotifyHistories(oldDb *gorm.DB, pb *pocketbase.PocketBase) error {
	var oldHistories []OldNotifyHistory
	if err := oldDb.Find(&oldHistories).Error; err != nil {
		return err
	}

	collection := pb.Dao().FindCollectionByNameOrId("notify_histories")
	if collection == nil {
		return fmt.Errorf("notify_histories collection not found")
	}

	for _, old := range oldHistories {
		record := models.NewRecord(collection)
		record.Set("chat_id", old.ChatId)
		record.Set("url", old.Url)
		record.Set("title", old.Title)
		record.Set("created", old.CreatedAt)

		if err := pb.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save notify history record: %v", err)
		}
	}

	return nil
}

func migrateFeedConfigs(oldDb *gorm.DB, pb *pocketbase.PocketBase) error {
	var oldConfigs []OldFeedConfig
	if err := oldDb.Find(&oldConfigs).Error; err != nil {
		return err
	}

	collection := pb.Dao().FindCollectionByNameOrId("feed_configs")
	if collection == nil {
		return fmt.Errorf("feed_configs collection not found")
	}

	for _, old := range oldConfigs {
		record := models.NewRecord(collection)
		record.Set("name", old.Name)
		record.Set("feed_url", old.FeedUrl)
		record.Set("feed_id", old.FeedId)

		if err := pb.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save feed config record: %v", err)
		}
	}

	return nil
}

func migrateSubscribeConfigs(oldDb *gorm.DB, pb *pocketbase.PocketBase) error {
	var oldConfigs []OldSubscribeConfig
	if err := oldDb.Find(&oldConfigs).Error; err != nil {
		return err
	}

	collection := pb.Dao().FindCollectionByNameOrId("subscribe_configs")
	if collection == nil {
		return fmt.Errorf("subscribe_configs collection not found")
	}

	for _, old := range oldConfigs {
		record := models.NewRecord(collection)
		record.Set("chat_id", old.ChatId)
		record.Set("keywords", old.Keywords)
		record.Set("feed_id", old.FeedId)
		record.Set("created", old.CreatedAt)
		record.Set("updated", old.UpdatedAt)

		if err := pb.Dao().SaveRecord(record); err != nil {
			return fmt.Errorf("failed to save subscribe config record: %v", err)
		}
	}

	return nil
}
