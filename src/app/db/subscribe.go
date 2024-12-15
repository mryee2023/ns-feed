package db

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Subscribe struct {
	ID            uint     `gorm:"primaryKey,autoIncrement"`
	Name          string   `gorm:"not null"`
	ChatId        int64    `gorm:"not null;uniqueIndex"`
	Keywords      string   `gorm:"not null"`
	KeywordsArray []string `gorm:"-"`
	Status        string
	Type          string
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// BeforeSave 在保存到数据库前将 KeywordsArray 序列化为 Keywords
func (s *Subscribe) BeforeSave(tx *gorm.DB) error {
	if len(s.KeywordsArray) > 0 {
		keywords, err := json.Marshal(s.KeywordsArray)
		if err != nil {
			return err
		}
		s.Keywords = string(keywords)
	}
	return nil
}

// AfterFind 在从数据库读取后将 Keywords 反序列化为 KeywordsArray
func (s *Subscribe) AfterFind(tx *gorm.DB) error {
	if s.Keywords != "" {
		return json.Unmarshal([]byte(s.Keywords), &s.KeywordsArray)
	}
	return nil
}

var db *gorm.DB

// InitDB initializes the database connection
func InitDB(dbPath string) error {
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second,   // Slow SQL threshold
				LogLevel:                  logger.Silent, // Log level
				IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,          // Don't include params in the SQL log
				Colorful:                  false,         // Disable color
			},
		),
	})
	if err != nil {
		return err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&Subscribe{}, &NotifyHistory{})
	if err != nil {
		return err
	}

	return nil
}

// AddSubscribe creates a new subscription
func AddSubscribe(sub *Subscribe) error {
	var exists *Subscribe
	db.Where("chat_id = ?", sub.ChatId).First(&exists)
	if exists != nil && exists.ID > 0 {
		return nil
	}
	return db.Create(sub).Error
}

// GetSubscribeWithChatId retrieves a subscription by ChatId
func GetSubscribeWithChatId(chatId int64) *Subscribe {
	var sub *Subscribe
	db.Where("chat_id = ?", chatId).First(&sub)
	if sub == nil || sub.ID == 0 {
		return nil
	}
	return sub
}

// UpdateSubscribe updates an existing subscription
func UpdateSubscribe(sub *Subscribe) error {
	return db.Save(sub).Error
}

// DeleteSubscribe deletes a subscription by ID
func DeleteSubscribe(chatId int64) error {
	return db.Where("chat_id = ?", chatId).Delete(&Subscribe{}).Error
}

// ListSubscribes returns all subscriptions
func ListSubscribes() []*Subscribe {
	var subs []*Subscribe
	db.Find(&subs)
	return subs
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return db
}
