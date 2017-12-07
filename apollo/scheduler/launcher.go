package scheduler

import (
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
)

var (
	log = logging.New("scheduler")
)

// DockerBenchScheduler schedules the docker benchmark.
type DockerBenchScheduler struct {
	Interval      time.Duration
	Enabled       bool
	CurrentScanID string

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

	d.CurrentScanID = ""
	d.Interval = 0
	d.Enabled = false

	d.stateLock.Unlock()
}

// Enable starts the scheduler and takes in a duration at which to run the interval.
func (d *DockerBenchScheduler) Enable(duration time.Duration) {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	if d.started {
		d.ticker.Stop()
		d.ticker = time.NewTicker(duration)
	} else {
		d.ticker = time.NewTicker(duration)
		go d.Start()
	}

	d.CurrentScanID = uuid.NewV4().String()
	d.Interval = duration
	d.Enabled = true
}

// Trigger causes a scan to be scheduled immediately.
func (d *DockerBenchScheduler) Trigger() {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	d.CurrentScanID = uuid.NewV4().String()
}

// Start runs the scheduler
func (d *DockerBenchScheduler) Start() {
	for {
		select {
		case <-d.ticker.C:
			d.stateLock.Lock()
			d.CurrentScanID = uuid.NewV4().String()
			d.stateLock.Unlock()
		case <-d.done:
			d.stateLock.Lock()
			d.started = false
			d.stateLock.Unlock()
			return
		}
	}
}
