package cache

import (
	"sync"
	"time"
)

var (
	cacheTime = 3 * time.Minute // 十分钟
)

type (
	Key                 comparable
	Cache[K Key, V any] struct {
		dict sync.Map

		// 缓存时间
		cacheTime time.Duration
	}

	valueInterface struct {
		v          any
		activeTime time.Time
	}
)

func newValueInterface(v any) *valueInterface {
	return &valueInterface{
		v:          v,
		activeTime: time.Now(),
	}
}

func New[K Key, V any](cacheTime time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		dict:      sync.Map{},
		cacheTime: cacheTime,
	}
	if cacheTime > 0 {
		go c.check()
	}

	return c
}

func (c *Cache[K, V]) check() {
	ticker := time.NewTicker(time.Minute * 15)
	for range ticker.C {
		c.dict.Range(func(k, v any) bool {
			ojb := v.(*valueInterface)
			if ojb.activeTime.Add(c.cacheTime).Before(time.Now()) {
				c.dict.Delete(k)
			}
			return true
		})
	}
}

/*
通过k获取缓存的v
*/
func (c *Cache[K, V]) Get(k K) (V, bool) {
	ojb, ok := c.dict.Load(k)
	if !ok {
		var v V
		return v, false
	}
	ojb.(*valueInterface).activeTime = time.Now()

	return ojb.(*valueInterface).v.(V), ok
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	ojb := newValueInterface(v)
	c.dict.Store(k, ojb)
	return true
}

func (c *Cache[K, V]) Del(k K) {
	c.dict.Delete(k)
}
