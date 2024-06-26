package lock

import (
	"github.com/infraboard/mcube/v2/ioc/config/gocache"
	"github.com/infraboard/mcube/v2/ioc/config/redis"
)

type PROVIDER string

const (
	PROVIDER_REDIS    = redis.AppName
	PROVIDER_GO_CACHE = gocache.AppName
)
