package db

type FeedConfig struct {
	ID      uint   `gorm:"primaryKey,autoIncrement" json:"id"`
	Name    string `gorm:"not null" json:"name"`
	FeedUrl string `gorm:"not null" json:"feedUrl"`
	FeedId  string `gorm:"not null;uniqueIndex" json:"feedId"`
}

func (f FeedConfig) TableName() string {
	return "feed_config"
}

func ListAllFeedConfig() []FeedConfig {
	var feedConfigs []FeedConfig
	db.Find(&feedConfigs)
	return feedConfigs
}

func GetFeedConfigWithFeedId(feedId string) FeedConfig {
	var feedConfig FeedConfig
	db.Where("feed_id = ?", feedId).First(&feedConfig)
	return feedConfig
}

func AddOrUpdateFeed(config FeedConfig) {
	var exists = GetFeedConfigWithFeedId(config.FeedId)
	if exists.ID > 0 {
		//// 根据 `struct` 更新属性，只会更新非零值的字段
		//db.Model(&user).Updates(User{Name: "hello", Age: 18, Active: false})
		db.Model(&FeedConfig{}).Where("id = ?", exists.ID).Updates(FeedConfig{
			Name:    config.Name,
			FeedUrl: config.FeedUrl,
			FeedId:  config.FeedId,
		})
		return
	}
	db.Create(&config)
}
