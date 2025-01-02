package db

import (
	"errors"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

var pb *pocketbase.PocketBase

// InitPocketBase initializes the PocketBase instance
func InitPocketBase(dbPath string) error {
	pb = pocketbase.New()

	// 配置数据库路径
	pb.RootCmd.SetArgs([]string{"serve", "--dir", dbPath})

	// 创建必要的集合
	pb.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		app := e.App
		collections := []func(app core.App) error{
			createSubscribesCollection,
			createNotifyHistoriesCollection,
			createFeedConfigsCollection,
			createSubscribeConfigsCollection,
		}

		for _, createCollection := range collections {
			if err := createCollection(app); err != nil {
				return err
			}
		}

		// 默认添加 NodeSeek feed
		collection := app.Dao().FindCollectionByNameOrId("feed_configs")
		if collection != nil {
			record, _ := app.Dao().FindFirstRecord(collection, "feed_id = {}", "ns")
			if record == nil {
				record = models.NewRecord(collection)
				record.Set("name", "NodeSeek")
				record.Set("feed_url", "https://rss.nodeseek.com")
				record.Set("feed_id", "ns")
				if err := app.Dao().SaveRecord(record); err != nil {
					return err
				}
			}
		}

		return nil
	})

	// 启动 PocketBase
	go func() {
		if err := pb.Start(); err != nil {
			log.Fatal("Failed to start PocketBase:", err)
		}
	}()

	return nil
}

// MigrateFromSQLite migrates data from SQLite to PocketBase
func MigrateFromSQLite(oldDbPath string) error {
	if pb == nil {
		return errors.New("PocketBase instance not initialized")
	}
	return MigrateData(oldDbPath, pb)
}

// createSubscribesCollection creates the subscribes collection schema
func createSubscribesCollection(app core.App) error {
	collection := &models.Collection{
		Name:       "subscribes",
		Type:       models.CollectionTypeBase,
		ListRule:   nil,
		ViewRule:   nil,
		CreateRule: nil,
		UpdateRule: nil,
		DeleteRule: nil,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "name",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "chat_id",
				Type:     schema.FieldTypeNumber,
				Required: true,
				Unique:   true,
			},
			&schema.SchemaField{
				Name:     "keywords",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "status",
				Type:     schema.FieldTypeText,
				Required: false,
			},
			&schema.SchemaField{
				Name:     "type",
				Type:     schema.FieldTypeText,
				Required: false,
			},
		),
	}

	return app.Dao().SaveCollection(collection)
}

// createNotifyHistoriesCollection creates the notify_histories collection schema
func createNotifyHistoriesCollection(app core.App) error {
	collection := &models.Collection{
		Name:       "notify_histories",
		Type:       models.CollectionTypeBase,
		ListRule:   nil,
		ViewRule:   nil,
		CreateRule: nil,
		UpdateRule: nil,
		DeleteRule: nil,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "chat_id",
				Type:     schema.FieldTypeNumber,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "url",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "title",
				Type:     schema.FieldTypeText,
				Required: true,
			},
		),
		Indexes: []string{
			"CREATE INDEX idx_notify_chat_id ON notify_histories (chat_id)",
			"CREATE INDEX idx_notify_url ON notify_histories (url)",
			"CREATE UNIQUE INDEX idx_notify_chat_url ON notify_histories (chat_id, url)",
		},
	}

	return app.Dao().SaveCollection(collection)
}

// createFeedConfigsCollection creates the feed_configs collection schema
func createFeedConfigsCollection(app core.App) error {
	collection := &models.Collection{
		Name:       "feed_configs",
		Type:       models.CollectionTypeBase,
		ListRule:   nil,
		ViewRule:   nil,
		CreateRule: nil,
		UpdateRule: nil,
		DeleteRule: nil,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "name",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "feed_url",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "feed_id",
				Type:     schema.FieldTypeText,
				Required: true,
				Unique:   true,
			},
		),
	}

	return app.Dao().SaveCollection(collection)
}

// createSubscribeConfigsCollection creates the subscribe_configs collection schema
func createSubscribeConfigsCollection(app core.App) error {
	collection := &models.Collection{
		Name:       "subscribe_configs",
		Type:       models.CollectionTypeBase,
		ListRule:   nil,
		ViewRule:   nil,
		CreateRule: nil,
		UpdateRule: nil,
		DeleteRule: nil,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "chat_id",
				Type:     schema.FieldTypeNumber,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "feed_id",
				Type:     schema.FieldTypeText,
				Required: true,
			},
		),
		Indexes: []string{
			"CREATE UNIQUE INDEX idx_subscribe_chat_feed ON subscribe_configs (chat_id, feed_id)",
		},
	}

	return app.Dao().SaveCollection(collection)
}

// GetPB returns the PocketBase instance
func GetPB() *pocketbase.PocketBase {
	return pb
}
