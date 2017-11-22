package scheduler

import (
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/orchestrators/types"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("scheduler")
)

const (
	cleanupDuration = 10 * time.Minute
	retries         = 5
)

// DockerBenchScheduler schedules the docker benchmark
type DockerBenchScheduler struct {
	ticker       *time.Ticker
	orchestrator types.Orchestrator

	started bool
	done    chan struct{}

	NextScheduled time.Time
	Interval      time.Duration
	Enabled       bool
}

// NewDockerBenchScheduler returns a new scheduler
func NewDockerBenchScheduler(orchestrator types.Orchestrator) *DockerBenchScheduler {
	return &DockerBenchScheduler{
		orchestrator: orchestrator,
		done:         make(chan struct{}),
	}
}

// Disable stops the scheduler
func (d *DockerBenchScheduler) Disable() {
	d.ticker.Stop()
	d.done <- struct{}{}

	d.NextScheduled = time.Unix(0, 0)
	d.Interval = 0
	d.Enabled = false
}

// Enable starts the scheduler and takes in a duration at which to run the interval
func (d *DockerBenchScheduler) Enable(duration time.Duration) {
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
}

func (d *DockerBenchScheduler) removeService(delay time.Duration, id string) {
	time.Sleep(delay)
	for i := 1; i < retries+1; i++ {
		if err := d.orchestrator.Kill(id); err != nil {
			log.Errorf("Error removing benchmark service %v: %+v", id, err)
		} else {
			return
		}
		time.Sleep(time.Duration(i) * 2 * time.Second)
	}
	log.Error("Timed out trying to remove benchmark service")
}

// Launch triggers a run of the benchmark immediately
func (d *DockerBenchScheduler) Launch() error {
	// TODO(cgorman) parameterize the tag for docker-bench-bootstrap as well as ROX_APOLLO_ENDPOINT
	service := types.SystemService{
		Envs:   []string{"ROX_APOLLO_ENDPOINT=localhost:8080"},
		Image:  "stackrox/docker-bench-bootstrap:latest",
		Mounts: []string{"/var/run/docker.sock:/var/run/docker.sock"},
		Global: true,
	}
	id, err := d.orchestrator.Launch(service)
	if err != nil {
		log.Error(err)
	} else {
		go d.removeService(cleanupDuration, id)
	}
	return err
}

// Start runs the scheduler
func (d *DockerBenchScheduler) Start() {
	for {
		select {
		case <-d.ticker.C:
			log.Infof("Launching docker bench")
			// launch docker bench bootstrap
			if err := d.Launch(); err != nil {
				log.Error(err)
			}
		case <-d.done:
			d.started = false
			return
		}
	}
}
