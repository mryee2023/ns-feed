package db

import (
	"errors"
	"log"
	"path/filepath"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

var (
	pb               *pocketbase.PocketBase
	initOnce         sync.Once
	collectionsReady chan struct{}
	initialized      bool
)

func init() {
	collectionsReady = make(chan struct{})
}

// InitPocketBase initializes the PocketBase instance
func InitPocketBase(dbPath string) error {
	var initErr error
	initOnce.Do(func() {
		// 如果路径以 pb_data 结尾，直接使用
		// 否则，在路径下创建 pb_data 目录
		if filepath.Base(dbPath) != "pb_data" {
			dbPath = filepath.Join(dbPath, "pb_data")
		}

		pb = pocketbase.New()

		// 配置数据库路径
		pb.RootCmd.SetArgs([]string{"serve", "--dir", dbPath})

		// 创建必要的集合
		pb.OnBeforeServe().Add(func(e *core.ServeEvent) error {
			defer close(collectionsReady) // 标记集合创建完成

			app := e.App
			collections := []func(app core.App) error{
				createSubscribesCollection,
				createNotifyHistoriesCollection,
				createFeedConfigsCollection,
				createSubscribeConfigsCollection,
			}

			for _, createCollection := range collections {
				if err := createCollection(app); err != nil {
					if err.Error() != "already exists" {
						initErr = err
						return err
					}
				}
			}

			initialized = true
			return nil
		})

		// 启动 PocketBase
		go func() {
			if err := pb.Start(); err != nil {
				if err.Error() != "server closed" {
					log.Fatal("Failed to start PocketBase:", err)
				}
			}
		}()
	})

	if initErr != nil {
		return initErr
	}

	// 等待初始化完成
	<-collectionsReady
	if !initialized {
		return errors.New("failed to initialize PocketBase")
	}

	return nil
}

// WaitForCollections 等待集合创建完成
func WaitForCollections() {
	<-collectionsReady
}

// MigrateFromSQLite migrates data from SQLite to PocketBase
func MigrateFromSQLite(oldDbPath string) error {
	if pb == nil {
		return errors.New("PocketBase instance not initialized")
	}
	WaitForCollections()
	return MigrateData(oldDbPath, pb)
}

// createSubscribesCollection creates the subscribes collection schema
func createSubscribesCollection(app core.App) error {
	// 检查集合是否已存在
	if existing, _ := app.Dao().FindCollectionByNameOrId("subscribes"); existing != nil {
		return errors.New("already exists")
	}

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
	// 检查集合是否已存在
	if existing, _ := app.Dao().FindCollectionByNameOrId("notify_histories"); existing != nil {
		return errors.New("already exists")
	}

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
	// 检查集合是否已存在
	if existing, _ := app.Dao().FindCollectionByNameOrId("feed_configs"); existing != nil {
		return errors.New("already exists")
	}

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
	// 检查集合是否已存在
	if existing, _ := app.Dao().FindCollectionByNameOrId("subscribe_configs"); existing != nil {
		return errors.New("already exists")
	}

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
				Name:     "keywords",
				Type:     schema.FieldTypeText,
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
	if !initialized {
		log.Fatal("PocketBase not initialized")
	}
	return pb
}
