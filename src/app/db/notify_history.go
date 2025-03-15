package db

import (
	"fmt"
	"sync"
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

// 添加内存缓存
var notifyCache = struct {
	sync.RWMutex
	cache map[string]bool // key: chatId_url
}{cache: make(map[string]bool)}

// 定期清理缓存的函数
func startCacheCleaner() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		notifyCache.Lock()
		notifyCache.cache = make(map[string]bool)
		notifyCache.Unlock()
	}
}

// 初始化时启动缓存清理
func init() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// 恢复并记录错误
				fmt.Println("Cache cleaner panic:", r)
			}
		}()
		startCacheCleaner()
	}()
}

// AddNotifyHistory creates a new notification history
func AddNotifyHistory(nh *NotifyHistory) error {
	// 先检查缓存
	cacheKey := fmt.Sprintf("%d_%s", nh.ChatId, nh.Url)

	notifyCache.RLock()
	exists, found := notifyCache.cache[cacheKey]
	notifyCache.RUnlock()

	if found && exists {
		return nil
	}

	// 缓存未命中，检查数据库
	var existingNh *NotifyHistory
	db.Where("chat_id = ? AND url = ?", nh.ChatId, nh.Url).First(&existingNh)
	if existingNh != nil && existingNh.ID > 0 {
		// 更新缓存
		notifyCache.Lock()
		notifyCache.cache[cacheKey] = true
		notifyCache.Unlock()
		return nil
	}

	// 不存在，创建新记录
	err := db.Create(nh).Error
	if err == nil {
		// 更新缓存
		notifyCache.Lock()
		notifyCache.cache[cacheKey] = true
		notifyCache.Unlock()
	}
	return err
}

// GetNotifyHistory retrieves a notification history by ChatId and Url
func GetNotifyHistory(chatId int64, url string) *NotifyHistory {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%d_%s", chatId, url)

	// 先检查缓存
	notifyCache.RLock()
	exists, found := notifyCache.cache[cacheKey]
	notifyCache.RUnlock()

	if found && exists {
		// 缓存命中，返回一个非空对象
		return &NotifyHistory{ID: 1}
	}

	// 缓存未命中，查询数据库
	var nh *NotifyHistory
	db.Where("chat_id = ? AND url = ?", chatId, url).First(&nh)

	// 更新缓存
	if nh != nil && nh.ID > 0 {
		notifyCache.Lock()
		notifyCache.cache[cacheKey] = true
		notifyCache.Unlock()
		return nh
	}

	return nil
}

// GetNotifyHistoryExists 只检查记录是否存在，不返回完整对象
func GetNotifyHistoryExists(chatId int64, url string) bool {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%d_%s", chatId, url)

	// 先检查缓存
	notifyCache.RLock()
	exists, found := notifyCache.cache[cacheKey]
	notifyCache.RUnlock()

	if found && exists {
		return true
	}

	// 缓存未命中，查询数据库
	var count int64
	db.Model(&NotifyHistory{}).
		Select("1").
		Where("chat_id = ? AND url = ?", chatId, url).
		Limit(1).
		Count(&count)

	// 更新缓存
	if count > 0 {
		notifyCache.Lock()
		notifyCache.cache[cacheKey] = true
		notifyCache.Unlock()
		return true
	}

	return false
}

// GetNotifyHistoryBatch 批量查询通知历史
func GetNotifyHistoryBatch(chatId int64, urls []string) map[string]bool {
	result := make(map[string]bool)

	// 初始化所有 URL 为不存在
	for _, url := range urls {
		result[url] = false
	}

	if len(urls) == 0 {
		return result
	}

	// 先从缓存中查找
	uncachedUrls := make([]string, 0, len(urls))

	notifyCache.RLock()
	for _, url := range urls {
		cacheKey := fmt.Sprintf("%d_%s", chatId, url)
		exists, found := notifyCache.cache[cacheKey]
		if found && exists {
			result[url] = true
		} else {
			uncachedUrls = append(uncachedUrls, url)
		}
	}
	notifyCache.RUnlock()

	// 如果所有 URL 都在缓存中找到，直接返回
	if len(uncachedUrls) == 0 {
		return result
	}

	// 批量查询未缓存的 URL
	var records []NotifyHistory
	db.Where("chat_id = ? AND url IN ?", chatId, uncachedUrls).Find(&records)

	// 更新结果和缓存
	notifyCache.Lock()
	for _, record := range records {
		result[record.Url] = true
		cacheKey := fmt.Sprintf("%d_%s", chatId, record.Url)
		notifyCache.cache[cacheKey] = true
	}
	notifyCache.Unlock()

	return result
}

// AddNotifyHistoryBatch 批量添加通知历史
func AddNotifyHistoryBatch(histories []*NotifyHistory) error {
	if len(histories) == 0 {
		return nil
	}

	// 开始事务
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 批量插入
	if err := tx.Create(&histories).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	err := tx.Commit().Error

	// 更新缓存
	if err == nil {
		notifyCache.Lock()
		for _, nh := range histories {
			cacheKey := fmt.Sprintf("%d_%s", nh.ChatId, nh.Url)
			notifyCache.cache[cacheKey] = true
		}
		notifyCache.Unlock()
	}

	return err
}

func GetNotifyCountByDateTime(start, end time.Time) int64 {
	var count int64
	db.Model(&NotifyHistory{}).Where("created_at >= ? and created_at<?", start, end).Count(&count)
	return count
}
