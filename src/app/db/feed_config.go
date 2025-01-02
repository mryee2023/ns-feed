package db

type FeedConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	FeedUrl string `json:"feedUrl"`
	FeedId  string `json:"feedId"`
}

func ListAllFeedConfig() []FeedConfig {
	collection := GetPB().Dao().FindCollectionByNameOrId("feed_configs")
	if collection == nil {
		return nil
	}

	records, err := GetPB().Dao().FindRecordsByExpr(collection)
	if err != nil {
		return nil
	}

	var feedConfigs []FeedConfig
	for _, record := range records {
		feedConfig := FeedConfig{
			ID:      record.Id,
			Name:    record.GetString("name"),
			FeedUrl: record.GetString("feed_url"),
			FeedId:  record.GetString("feed_id"),
		}
		feedConfigs = append(feedConfigs, feedConfig)
	}

	return feedConfigs
}

func GetFeedConfigWithFeedId(feedId string) FeedConfig {
	collection := GetPB().Dao().FindCollectionByNameOrId("feed_configs")
	if collection == nil {
		return FeedConfig{}
	}

	record, err := GetPB().Dao().FindFirstRecord(collection, "feed_id = {}", feedId)
	if err != nil {
		return FeedConfig{}
	}

	return FeedConfig{
		ID:      record.Id,
		Name:    record.GetString("name"),
		FeedUrl: record.GetString("feed_url"),
		FeedId:  record.GetString("feed_id"),
	}
}

func AddOrUpdateFeed(config FeedConfig) error {
	collection := GetPB().Dao().FindCollectionByNameOrId("feed_configs")
	if collection == nil {
		return errors.New("collection not found")
	}

	record, _ := GetPB().Dao().FindFirstRecord(collection, "feed_id = {}", config.FeedId)
	if record != nil {
		// Update existing record
		record.Set("name", config.Name)
		record.Set("feed_url", config.FeedUrl)
		return GetPB().Dao().SaveRecord(record)
	}

	// Create new record
	record = models.NewRecord(collection)
	record.Set("name", config.Name)
	record.Set("feed_url", config.FeedUrl)
	record.Set("feed_id", config.FeedId)

	return GetPB().Dao().SaveRecord(record)
}
