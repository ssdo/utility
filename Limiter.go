package utility

import (
	"fmt"
	"github.com/ssgo/log"
	"github.com/ssgo/redis"
	"math"
	"sync"
	"time"
)

// 单位时间内进行频次限制

type Limiter struct {
	keyPrefix     string
	durationMs    int64
	times         int64
	redis         *redis.Redis
	localData     []int64
	localDataTags []int64
	localDataLock *sync.Mutex
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

// 创建一个本地限制器，指定一个名称来区分，设置好多长时间内允许多少次
func NewLocalLimiter(name string, duration time.Duration, times int) *Limiter {
	return &Limiter{
		keyPrefix:     fmt.Sprint("_LIMIT_", name, "_"),
		durationMs:    int64(duration / time.Millisecond),
		times:         int64(times),
		localData:     make([]int64, 10),
		localDataTags: make([]int64, 10),
		localDataLock: new(sync.Mutex),
	}
}

// 检查一次指定 key，累加后如果在指定时间内的限额范围则返回true
func (limiter *Limiter) Check(key string, logger *log.Logger) bool {
	if logger == nil {
		logger = log.DefaultLogger
	}

	// 产生时间数据
	timeValue := time.Now().UnixNano()/int64(time.Millisecond) - timeValueStart // 起始时间到现在的毫秒值
	timeStep := limiter.durationMs / 10                                         // 分10段的每一段时间大小
	timeTag := timeValue / timeStep                                             // 最后一段时间值
	//fmt.Println(" >", time.Now().UnixNano()/int64(time.Millisecond), timeValueStart, timeValue, limiter.durationMs, timeTag)

	// 更新计时器
	times := int64(0)
	if limiter.redis != nil {
		// 基于Redis
		rd := limiter.redis.CopyByLogger(logger)
		currentKey := fmt.Sprint(limiter.keyPrefix, key, "_", timeTag) // 最后一段的完整key
		times = rd.INCR(currentKey)
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
	} else if limiter.localData != nil {
		// 基于本地
		pos := int(timeTag % 10)
		//fmt.Println(" >", timeTag, pos)

		limiter.localDataLock.Lock()
		if timeTag != limiter.localDataTags[pos] {
			// 位置变化时讲新位置清零
			limiter.localData[pos] = 0
			limiter.localDataTags[pos] = timeTag
		}
		limiter.localData[pos] ++

		for i := int64(0); i < 9; i++ {
			p := int(timeTag-i) % 10
			if limiter.localDataTags[p] == timeTag-i {
				times += limiter.localData[p]
			}
		}
		limiter.localDataLock.Unlock()
		//fmt.Println(limiter.localData, times)
	}

	//fmt.Println(currentKey, times)
	if times <= limiter.times {
		return true
	} else {
		logger.Warning("limited", "keyPrefix", limiter.keyPrefix, "key", key, "timeTag", timeTag, "times", times, "limitTimes", limiter.times)
		return false
	}
}
