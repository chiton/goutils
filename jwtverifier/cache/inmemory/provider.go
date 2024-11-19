package inmemory

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

type Provider struct {
	cache  *cache.Cache
	lookup func(string) (interface{}, error)
	mutex  *sync.Mutex
}

func (c *Provider) Get(key string) (interface{}, error) {
	// return straight away
	if value, found := c.cache.Get(key); found {
		return value, nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if value, found := c.cache.Get(key); found {
		return value, nil
	}

	value, err := c.lookup(key)
	if err != nil {
		return nil, err
	}

	c.cache.SetDefault(key, value)
	return value, nil
}

// NewProvider creates a new in-memory cache.
func NewProvider(lookup func(string) (interface{}, error), expiration, cleanupInterval time.Duration) (*Provider, error) {
	return &Provider{
		cache:  cache.New(expiration, cleanupInterval),
		lookup: lookup,
		mutex:  &sync.Mutex{},
	}, nil
}
