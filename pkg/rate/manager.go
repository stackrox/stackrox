package rate

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type managerImpl struct {
	stopC           concurrency.ReadOnlyErrorSignal
	recordsC        chan struct{}
	rateLimit       int
	limitReached    bool
	onLimitExceeded func(int)
	onLimitMissed   func(int)
}

func NewManager(
	recordC chan struct{},
	stopC concurrency.ReadOnlyErrorSignal,
	rateTime time.Duration,
	rateLimit int,
	onLimitExceeded func(int),
	onLimitMissed func(int),
) *managerImpl {
	if recordC == nil {
		return nil
	}
	ret := &managerImpl{
		stopC:           stopC,
		recordsC:        recordC,
		rateLimit:       rateLimit,
		onLimitExceeded: functionWrapper(onLimitExceeded),
		onLimitMissed:   functionWrapper(onLimitMissed),
	}
	ticker := time.NewTicker(rateTime)
	go ret.run(ticker.C)
	return ret
}

func functionWrapper(fn func(int)) func(int) {
	return func(num int) {
		if fn != nil {
			fn(num)
		}
	}
}

func (m *managerImpl) Record() {
	if m == nil {
		return
	}
	select {
	case m.recordsC <- struct{}{}:
	case <-m.stopC.Done():
	}
}

func (m *managerImpl) handleTick(numDropped int) {
	if numDropped >= m.rateLimit && !m.limitReached {
		m.onLimitExceeded(numDropped)
		m.limitReached = true
		return
	}
	if numDropped < m.rateLimit && m.limitReached {
		m.onLimitMissed(numDropped)
		m.limitReached = false
	}
}

func (m *managerImpl) run(ticker <-chan time.Time) {
	numDropped := 0
	for {
		select {
		case <-m.stopC.Done():
			return
		case <-ticker:
			m.handleTick(numDropped)
			numDropped = 0
		case _, ok := <-m.recordsC:
			if !ok {
				return
			}
			numDropped++
		}
	}
}
