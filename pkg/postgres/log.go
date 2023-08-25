package postgres

import (
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/atomic"
	"golang.org/x/time/rate"
)

var (
	slowQueryThreshold      = env.SlowQueryThreshold.DurationSetting()
	slowQueryLogRate        = rate.Every(30 * time.Minute)
	slowQueryLock           sync.Mutex
	slowQueryRateLimiterMap = make(map[string]*slowQueryEntry)

	log = logging.LoggerForModule()
)

type slowQueryEntry struct {
	limiter  *rate.Limiter
	numTimes *atomic.Int32
}

func getRateLimiter(sql string) *slowQueryEntry {
	slowQueryLock.Lock()
	defer slowQueryLock.Unlock()
	if slowQuery, ok := slowQueryRateLimiterMap[sql]; ok {
		slowQuery.numTimes.Add(1)
		return slowQuery
	}
	slowQuery := &slowQueryEntry{
		limiter:  rate.NewLimiter(slowQueryLogRate, 1),
		numTimes: atomic.NewInt32(1),
	}
	slowQueryRateLimiterMap[sql] = slowQuery
	return slowQuery
}

func logSlowQuery(start time.Time, sql string) {
	took := time.Since(start)
	if took < slowQueryThreshold {
		return
	}
	slowQuery := getRateLimiter(sql)
	if slowQuery.limiter.Allow() {
		log.Warnf("Slow query detected %d times. Most recent took %0.2f seconds: %s", slowQuery.numTimes.Load(), took.Seconds(), sql)
	}
}
