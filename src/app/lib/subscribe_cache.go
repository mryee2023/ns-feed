package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"ns-rss/src/app/db"
)

var subCache *SubscribeCache

func init() {
	subCache = NewSubscribeCache(context.Background())
}

func SubCacheInstance() *SubscribeCache {
	return subCache
}

var cacheKey = "subscribe:%d"
var cacheKeyAll = "subscribe:all"

type SubscribeCache struct {
	ctx   context.Context
	cache *collection.Cache
}

func NewSubscribeCache(ctx context.Context) *SubscribeCache {
	c, _ := collection.NewCache(time.Hour)
	return &SubscribeCache{
		ctx:   ctx,
		cache: c,
	}
}

func (c *SubscribeCache) Get(chatId int64) *db.Subscribe {
	p, _ := c.cache.Take(fmt.Sprintf(cacheKey, chatId), func() (any, error) {
		return db.GetSubscribeWithChatId(chatId), nil
	})
	return p.(*db.Subscribe)
}

func (c *SubscribeCache) Set(chatId int64, s *db.Subscribe) {
	c.cache.Set(fmt.Sprintf(cacheKey, chatId), s)
}

func (c *SubscribeCache) Del(chatId int64) {
	c.cache.Del(fmt.Sprintf(cacheKey, chatId))
}

func (c *SubscribeCache) All() []*db.Subscribe {
	p, _ := c.cache.Take(cacheKeyAll, func() (any, error) {
		return db.ListSubscribes(), nil
	})
	return p.([]*db.Subscribe)
}
func (c *SubscribeCache) ReloadAll() {
	c.cache.Del(cacheKeyAll)
}
