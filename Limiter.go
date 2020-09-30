package utility

import (
	"fmt"
	"github.com/ssgo/log"
	"github.com/ssgo/redis"
	"math"
	"time"
)

// 单位时间内进行频次限制

type Limiter struct {
	keyPrefix  string
	durationMs int64
	times      int64
	redis      *redis.Redis
}

var timeValueStart int64 = 1577808000000

func SetStartTimeValue(msTimeValue int64) {
	timeValueStart = msTimeValue
}

// 创建一个限制器，指定一个名称来区分，设置好多长时间内允许多少次，传入一个 redis 连接池用于存储临时计数器
func NewLimiter(name string, duration time.Duration, times int, rd *redis.Redis) *Limiter {
	if rd == nil {
		rd = redis.GetRedis("default", nil)
	}
	return &Limiter{
		keyPrefix:  fmt.Sprint("_LIMIT_", name, "_"),
		durationMs: int64(duration / time.Millisecond),
		times:      int64(times),
		redis:      rd,
	}
}

// 检查一次指定 key，累加后如果在指定时间内的限额范围则返回true
func (limiter *Limiter) Check(key string, logger *log.Logger) bool {
	if logger == nil {
		logger = log.DefaultLogger
	}
	rd := limiter.redis.CopyByLogger(logger)

	// 产生时间数据
	timeValue := time.Now().UnixNano() - timeValueStart            // 起始时间到现在的毫秒值
	timeStep := limiter.durationMs / 10                            // 分10段的每一段时间大小
	timeTag := timeValue / timeStep                                // 最后一段时间值
	currentKey := fmt.Sprint(limiter.keyPrefix, key, "_", timeTag) // 最后一段的完整key

	// 更新计时器
	times := rd.INCR(currentKey)
	if times == 1 {
		// 第一次使用，设置过期时间
		rd.EXPIRE(currentKey, int(math.Ceil(float64(limiter.durationMs)/1000)))
	}

	// 往前查找9个之前的值
	prevKeys := make([]string, 9)
	for i := 0; i < 9; i++ {
		prevKeys[i] = fmt.Sprint(limiter.keyPrefix, key, "_", timeTag-int64(i+1))
	}

	for _, prevResult := range rd.MGET(prevKeys...) {
		//fmt.Println(" =>", i, prevKeys[i], prevResult.Int64())
		times += prevResult.Int64()
	}

	//fmt.Println(currentKey, times)
	if times <= limiter.times {
		return true
	} else {
		logger.Warning("limited", "keyPrefix", limiter.keyPrefix, "key", key, "timeTag", timeTag, "times", times, "limitTimes", limiter.times)
		return false
	}
}
