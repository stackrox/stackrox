package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"github.com/golang/protobuf/ptypes/empty"
)

var (
	log = logging.New("scheduler")
)

const (
	cleanupDuration = 10 * time.Minute
	retries         = 5

	updateInterval = 15 * time.Second
)

// BenchmarkSchedulerClient schedules the docker benchmark
type BenchmarkSchedulerClient struct {
	updateTicker *time.Ticker
	orchestrator orchestrators.Orchestrator

	endpoint string

	started bool
	done    chan struct{}

	scanActive bool
	lastScanID string

	stateLock sync.Mutex
}

// NewBenchmarkSchedulerClient returns a new scheduler
func NewBenchmarkSchedulerClient(orchestrator orchestrators.Orchestrator, apolloEndpoint string) *BenchmarkSchedulerClient {
	return &BenchmarkSchedulerClient{
		updateTicker: time.NewTicker(updateInterval),
		orchestrator: orchestrator,
		done:         make(chan struct{}),
		endpoint:     apolloEndpoint,
	}
}

func (d *BenchmarkSchedulerClient) removeService(delay time.Duration, id string) {
	defer func() {
		d.stateLock.Lock()
		d.scanActive = false
		d.stateLock.Unlock()
	}()

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

// Launch triggers a run of the benchmark immediately.
// The stateLock must be held by the caller until this function returns.
func (d *BenchmarkSchedulerClient) Launch() error {
	d.scanActive = true
	// TODO(cgorman) parametrize the tag for docker-bench-bootstrap
	service := orchestrators.SystemService{
		Envs: []string{
			fmt.Sprintf("ROX_APOLLO_ENDPOINT=%s", d.endpoint),
			fmt.Sprintf("ROX_APOLLO_SCAN_ID=%s", d.lastScanID),
		},
		Image:  "stackrox/docker-bench-bootstrap:latest",
		Mounts: []string{"/var/run/docker.sock:/var/run/docker.sock"},
		Global: true,
	}
	id, err := d.orchestrator.Launch(service)
	if err != nil {
		log.Error(err)
		d.scanActive = false
		return err
	}
	go d.removeService(cleanupDuration, id)
	return nil
}

// Start runs the scheduler
func (d *BenchmarkSchedulerClient) Start() {
	conn, err := clientconn.GRPCConnection(d.endpoint)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := v1.NewBenchmarkServiceClient(conn)
	for {
		select {
		case <-d.updateTicker.C:
			log.Infof("Checking Docker bench schedule")
			schedule, err := cli.GetBenchmarkSchedule(context.Background(), &empty.Empty{})
			if err != nil {
				log.Errorf("Error checking schedule: %s", err)
				continue
			}
			if schedule == nil {
				log.Errorf("Schedule was nil")
				continue
			}
			d.stateLock.Lock()
			if d.scanActive {
				d.stateLock.Unlock()
				continue
			}
			id := schedule.GetCurrentScanId()
			if d.lastScanID != id && len(id) > 0 {
				log.Infof("Launching Docker bench for scan %s", id)
				d.lastScanID = id
				if err := d.Launch(); err != nil {
					log.Errorf("Error launching benchmark: %s", err)
				}
			}
			d.stateLock.Unlock()
		case <-d.done:
			d.started = false
			return
		}
	}
}

// Stop stops the scheduler client from triggering any more jobs.
func (d *BenchmarkSchedulerClient) Stop() {
	d.done <- struct{}{}

	// TODO(cg): Also stop any launched benchmark.
}
