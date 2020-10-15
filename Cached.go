package utility

import (
	"github.com/ssgo/log"
	"github.com/ssgo/redis"
	"time"
)

type Cached struct {
	Starter
	name          string
	cacheMaker    func(uint64, uint64)
	versionGetter func() uint64
	cd            *Coordinator
	redis         *redis.Redis
	logger        *log.Logger
	running       bool
	stopChan      chan bool
}

func NewCached(name string, interval time.Duration, cacheMaker func(minVersion uint64, maxVersion uint64), versionGetter func() uint64, redis *redis.Redis, logger *log.Logger) *Cached {
	cached := Cached{
		Starter: Starter{
			Interval: interval,
		},
		name:          name,
		cacheMaker:    cacheMaker,
		versionGetter: versionGetter,
		cd:            NewCoordinator(name, interval, redis),
		redis:         redis,
		logger:        logger,
		running:       false,
		stopChan:      nil,
	}
	cached.Starter.Work = cached.work
	return &cached
}

func (cached *Cached) work() {
	if cached.cd.GetLock(cached.logger) {
		require := true
		oldVersion := uint64(0)
		newVersion := uint64(0)
		if cached.versionGetter != nil {
			oldVersion = cached.redis.GET("_CACHED_VRR_" + cached.name).Uint64()
			newVersion = cached.versionGetter()
			if newVersion == oldVersion {
				require = false
			} else {
				cached.redis.SET("_CACHED_VRR_"+cached.name, newVersion)
			}
		}

		if require {
			cached.cacheMaker(oldVersion, newVersion)
		}
	}
}
