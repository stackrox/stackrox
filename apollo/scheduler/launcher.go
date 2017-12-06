package scheduler

import (
	"time"

	"sync"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("scheduler")
)

// DockerBenchScheduler schedules the docker benchmark.
type DockerBenchScheduler struct {
	NextScheduled time.Time
	Interval      time.Duration
	Enabled       bool

	ticker    *time.Ticker
	done      chan struct{}
	started   bool
	stateLock sync.Mutex
}

// NewDockerBenchScheduler returns a new scheduler.
func NewDockerBenchScheduler() *DockerBenchScheduler {
	return &DockerBenchScheduler{}
}

// Disable stops the scheduler.
func (d *DockerBenchScheduler) Disable() {
	d.stateLock.Lock()

	d.NextScheduled = time.Unix(0, 0)
	d.Interval = 0
	d.Enabled = false

	d.stateLock.Unlock()
}

// Enable starts the scheduler and takes in a duration at which to run the interval.
func (d *DockerBenchScheduler) Enable(duration time.Duration) {
	d.stateLock.Lock()

	if d.started {
		d.ticker.Stop()
		d.ticker = time.NewTicker(duration)
	} else {
		d.ticker = time.NewTicker(duration)
		go d.Start()
	}

	d.NextScheduled = time.Now().Add(duration)
	d.Interval = duration
	d.Enabled = true

	d.stateLock.Unlock()
}

// Trigger causes a scan to be scheduled immediately.
func (d *DockerBenchScheduler) Trigger() {
	d.stateLock.Lock()

	d.NextScheduled = time.Now()

	d.stateLock.Unlock()
}

// Start runs the scheduler
func (d *DockerBenchScheduler) Start() {
	for {
		select {
		case <-d.ticker.C:
			d.stateLock.Lock()
			d.NextScheduled = time.Now()
			d.stateLock.Unlock()
		case <-d.done:
			d.stateLock.Lock()
			d.started = false
			d.stateLock.Unlock()
			return
		}
	}
}
