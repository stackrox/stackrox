package connection

import (
	"container/heap"
	"math"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/ratelimit"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	boostInitSyncRateLimit  = 50
	maxTopCandidateClusters = 3
	rateOverPeriod          = 10 * time.Minute
	minimumRatePeriod       = 10 * time.Second
)

// Inspiration is taken from EARRRL:
// https://blog.jnbrymn.com/2021/03/18/estimated-average-recent-request-rate-limiter.html
//
// A shorter period will cause older rates to diminish more rapidly, while
// a longer period will result in older rates retaining greater importance.
// The tick is set to 1s. The minimum period is 10s.
type clusterMsgRate struct {
	mutex sync.Mutex

	index     int
	clusterID string

	halfPeriod float64
	lastTime   int64
	rate       float64
	acc        float64
}

func (cmr *clusterMsgRate) recvMsg() {
	cmr.mutex.Lock()
	defer cmr.mutex.Unlock()

	now := time.Now().Unix()

	deltaT := cmr.lastTime - now
	if deltaT < 0 {
		cmr.acc *= math.Exp(float64(deltaT) / cmr.halfPeriod)
	}
	cmr.acc++
	cmr.lastTime = now

	cmr.rate = cmr.acc / cmr.halfPeriod
}

func newClusterMsgRate(clusterID string, period time.Duration) *clusterMsgRate {
	if period < minimumRatePeriod {
		period = minimumRatePeriod
	}
	halfPeriod := period.Seconds() / 2.0

	return &clusterMsgRate{
		index:      -1,
		clusterID:  clusterID,
		halfPeriod: halfPeriod,
		lastTime:   time.Now().Unix(),
		rate:       0.0,
	}
}

// Maintains a list of the leading consumer clusters, which serves as
// a reference for identifying potential candidates for rate limiting.
type clusterMsgRateHeap []*clusterMsgRate

func newClusterMsgRateHeap() *clusterMsgRateHeap {
	return &clusterMsgRateHeap{}
}

func (h *clusterMsgRateHeap) Len() int {
	return len(*h)
}

func (h *clusterMsgRateHeap) Less(i, j int) bool {
	return (*h)[i].rate < (*h)[j].rate
}

func (h *clusterMsgRateHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
	(*h)[i].index = i
	(*h)[j].index = j
}

func (h *clusterMsgRateHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*clusterMsgRate)
	item.index = n
	*h = append(*h, item)
}

func (h *clusterMsgRateHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]

	return item
}

type rateLimitManager struct {
	mutex sync.Mutex

	maxSensors      int
	initSyncSensors set.StringSet

	msgRateLimiter         ratelimit.RateLimiter
	clusterMsgRates        map[string]*clusterMsgRate
	clusterMsgRatesHeap    *clusterMsgRateHeap
	clusterLimitCandidates set.StringSet
}

// newRateLimitManager creates an rateLimitManager with max sensors
// retrieved from env variable, ensuring it is non-negative.
func newRateLimitManager() *rateLimitManager {
	maxSensors := env.CentralMaxInitSyncSensors.IntegerSetting()
	if maxSensors < 0 {
		log.Panicf("Negative number is not allowed for max init sync sensors. Check env variable: %q", env.CentralMaxInitSyncSensors.EnvVar())
	}

	eventRateLimit := env.CentralSensorMaxEventsPerSecond.IntegerSetting()
	if eventRateLimit < 0 {
		log.Panicf("Negative number is not allowed for rate limit of sensors events. Check env variable: %q", env.CentralSensorMaxEventsPerSecond.EnvVar())
	}

	// Use MaxInt for unlimited max init sync sensors.
	if maxSensors == 0 {
		maxSensors = math.MaxInt
	}

	return &rateLimitManager{
		maxSensors:      maxSensors,
		initSyncSensors: set.NewStringSet(),

		msgRateLimiter:         ratelimit.NewRateLimiter(eventRateLimit, env.CentralSensorMaxEventsThrottleDuration.DurationSetting()),
		clusterMsgRates:        make(map[string]*clusterMsgRate),
		clusterMsgRatesHeap:    newClusterMsgRateHeap(),
		clusterLimitCandidates: set.NewStringSet(),
	}
}

func (m *rateLimitManager) AddInitSync(clusterID string) bool {
	if m == nil {
		return true
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.initSyncSensors) >= m.maxSensors {
		return false
	}

	if m.initSyncSensors.Add(clusterID) && m.msgRateLimiter != nil {
		m.msgRateLimiter.IncreaseLimit(boostInitSyncRateLimit)
	}

	return true
}

func (m *rateLimitManager) RemoveInitSync(clusterID string) {
	if m == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.initSyncSensors.Remove(clusterID) && m.msgRateLimiter != nil {
		m.msgRateLimiter.DecreaseLimit(boostInitSyncRateLimit)
	}
}

func (m *rateLimitManager) getClusterRate(clusterID string) *clusterMsgRate {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	clusterRate, found := m.clusterMsgRates[clusterID]
	if !found {
		clusterRate = newClusterMsgRate(clusterID, rateOverPeriod)
		m.clusterMsgRates[clusterID] = clusterRate
	}

	return clusterRate
}

func (m *rateLimitManager) updateClusterRateHeap(clusterRate *clusterMsgRate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.clusterLimitCandidates.Contains(clusterRate.clusterID) {
		heap.Fix(m.clusterMsgRatesHeap, clusterRate.index)
	} else {
		heap.Push(m.clusterMsgRatesHeap, clusterRate)
		m.clusterLimitCandidates.Add(clusterRate.clusterID)

		if m.clusterMsgRatesHeap.Len() > maxTopCandidateClusters {
			droppedCandidate := heap.Pop(m.clusterMsgRatesHeap).(*clusterMsgRate)
			m.clusterLimitCandidates.Remove(droppedCandidate.clusterID)
		}
	}
}

func (m *rateLimitManager) throttleMsg(clusterID string) bool {
	log.Warnf("Throttling messages from cluster %q.", clusterID)

	return m.msgRateLimiter.Limit()
}

func (m *rateLimitManager) LimitClusterMsg(clusterID string) bool {
	if m == nil || m.msgRateLimiter == nil {
		return false
	}

	clusterRate := m.getClusterRate(clusterID)
	clusterRate.recvMsg()
	m.updateClusterRateHeap(clusterRate)

	// When the global limit is reached. If we are processing a message from
	// a cluster that is a candidate for limiting, we initially apply message
	// throttling. If this throttling doesn't help the situation, we will
	// return false, which should terminate the connection to the cluster.
	return m.msgRateLimiter.LimitNoThrottle() && m.clusterLimitCandidates.Contains(clusterID) && m.throttleMsg(clusterID)
}

func (m *rateLimitManager) RemoveMsgRateCluster(clusterID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	clusterRate := m.getClusterRate(clusterID)
	delete(m.clusterMsgRates, clusterID)
	m.clusterLimitCandidates.Remove(clusterID)

	if clusterRate.index != -1 {
		heap.Remove(m.clusterMsgRatesHeap, clusterRate.index)
	}
}
