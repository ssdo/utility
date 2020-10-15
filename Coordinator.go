package utility

import (
	"github.com/ssgo/log"
	"github.com/ssgo/redis"
	"github.com/ssgo/u"
	"time"
)

// 多个服务节点之间竞争一个授权，一个获得授权其他人不再获得

type Coordinator struct {
	name     string
	interval time.Duration
	redis    *redis.Redis
}

func NewCoordinator(name string, interval time.Duration, redis *redis.Redis) *Coordinator {
	return &Coordinator{
		name:     name,
		interval: interval,
		redis:    redis,
	}
}

// 每个时间周期访问一次，尝试获得授权
func (cd *Coordinator) GetLock(logger *log.Logger) bool {
	rd := cd.redis.CopyByLogger(logger)

	// 计算从开始时间到现在经历的第几个时间周期
	timeValue := (time.Now().UnixNano()/int64(time.Millisecond) - timeValueStart) / int64(cd.interval/time.Millisecond)
	key := "_COORDINATOR_"+u.String(timeValue)

	if rd.INCR(key) != 1 {
		// 不是第一个访问的节点
		return false
	}

	// 第一个获得的节点设置key的过期时间
	rd.EXPIRE(key, int(cd.interval/time.Second))
	return true
}
