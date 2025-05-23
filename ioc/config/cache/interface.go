package cache

import (
	"context"
	"time"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/gocache"
	"github.com/infraboard/mcube/v2/ioc/config/redis"
)

const (
	AppName = "cache"
)

type PROVIDER string

const (
	PROVIDER_GO_CACHE = gocache.AppName
	PROVIDER_REDIS    = redis.AppName
)

func C() Cache {
	return Get().c
}

func Get() *cache {
	obj := ioc.Config().Get(AppName)
	if obj == nil {
		return defaultConfig
	}
	return obj.(*cache)
}

type Cache interface {
	Set(ctx context.Context, key string, value any, options ...SetOption) error
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Get(ctx context.Context, key string, value any) error
	Exist(ctx context.Context, key string) error
	Del(ctx context.Context, keys ...string) error
}

func WithExpiration(expiration int64) SetOption {
	return func(o *options) {
		o.expiration = expiration
	}
}

type SetOption func(*options)

func newOptions(defaultTTL int64, opts ...SetOption) *options {
	options := &options{expiration: defaultTTL}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

type options struct {
	// 过期时间, 单位秒
	expiration int64
}

func (m *options) GetTTL() time.Duration {
	return time.Duration(m.expiration) * time.Second
}
