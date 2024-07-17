package cache

import "time"

type cacheItem struct {
	key       string
	value     interface{}
	lifeSpan  time.Duration //存储时长
	createdAt time.Time
}

func NewCacheItem(key string, value interface{}, lifeSpan time.Duration) *cacheItem {
	return &cacheItem{
		key:       key,
		value:     value,
		lifeSpan:  lifeSpan,
		createdAt: time.Now(),
	}
}

func (c *cacheItem) LifeSpan() time.Duration {
	return c.lifeSpan
}

func (c *cacheItem) CreatedAt() time.Time {
	return c.createdAt
}

func (c *cacheItem) Key() string {
	return c.key
}

func (c *cacheItem) Value() interface{} {
	return c.value
}
