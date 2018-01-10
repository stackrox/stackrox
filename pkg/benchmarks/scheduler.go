package benchmarks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"github.com/deckarep/golang-set"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

var (
	log = logging.New("scheduler")
)

const (
	cleanupTimeout = 1 * time.Minute
	retries        = 5
	updateInterval = 30 * time.Second
	// triggerTimespan is how long we should check for unfired triggers
	triggerTimespan = 5 * time.Minute
)

type benchmarkRun struct {
	benchmarkName string
	scanID        string
}

type scheduleMetadata struct {
	*v1.BenchmarkSchedule
	NextScanTime time.Time
}

// SchedulerClient schedules the docker benchmark
type SchedulerClient struct {
	updateTicker   *time.Ticker
	scheduleTicker *time.Ticker
	orchestrator   orchestrators.Orchestrator

	advertisedEndpoint string
	apolloEndpoint     string
	cluster            string
	image              string

	started bool
	done    chan struct{}

	schedules map[string]*scheduleMetadata
	triggers  map[string]*v1.BenchmarkTrigger

	clientConn              *grpc.ClientConn
	scheduleClient          v1.BenchmarkScheduleServiceClient
	benchmarkResultsClient  v1.BenchmarkResultsServiceClient
	benchmarkTriggersClient v1.BenchmarkTriggerServiceClient
	benchmarkClient         v1.BenchmarkServiceClient

	benchmarkChan chan benchmarkRun
}

// NewSchedulerClient returns a new scheduler
func NewSchedulerClient(orchestrator orchestrators.Orchestrator, apolloEndpoint, advertisedEndpoint, image string, cluster string) (*SchedulerClient, error) {
	conn, err := clientconn.GRPCConnection(apolloEndpoint)
	if err != nil {
		return nil, err
	}
	return &SchedulerClient{
		updateTicker:       time.NewTicker(updateInterval),
		orchestrator:       orchestrator,
		done:               make(chan struct{}),
		cluster:            cluster,
		apolloEndpoint:     apolloEndpoint,
		advertisedEndpoint: advertisedEndpoint,
		image:              image,

		schedules: make(map[string]*scheduleMetadata),
		triggers:  make(map[string]*v1.BenchmarkTrigger),

		clientConn:              conn,
		scheduleClient:          v1.NewBenchmarkScheduleServiceClient(conn),
		benchmarkResultsClient:  v1.NewBenchmarkResultsServiceClient(conn),
		benchmarkTriggersClient: v1.NewBenchmarkTriggerServiceClient(conn),
		benchmarkClient:         v1.NewBenchmarkServiceClient(conn),

		benchmarkChan: make(chan benchmarkRun, 512),
	}, nil
}

func grpcContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
}

func (s *SchedulerClient) getSchedules() ([]*v1.BenchmarkSchedule, error) {
	ctx, cancel := grpcContext()
	defer cancel()
	scheduleResp, err := s.scheduleClient.GetBenchmarkSchedules(ctx, &v1.GetBenchmarkSchedulesRequest{
		Cluster: s.cluster,
	})
	if err != nil {
		return nil, fmt.Errorf("Error checking schedule: %s", err)
	}
	return scheduleResp.Schedules, nil
}

func (s *SchedulerClient) getBenchmarkResults(scanID string) ([]*v1.BenchmarkResult, error) {
	ctx, cancel := grpcContext()
	defer cancel()
	log.Infof("Fetching benchmark results for scan: %v", scanID)
	benchmarkResults, err := s.benchmarkResultsClient.GetBenchmarkResults(ctx, &v1.GetBenchmarkResultsRequest{
		ScanId:   scanID,
		Clusters: []string{s.cluster},
	})
	if err != nil {
		return nil, fmt.Errorf("error checking results: %s", err)
	}
	return benchmarkResults.Benchmarks, nil
}

func (s *SchedulerClient) getTriggers() ([]*v1.BenchmarkTrigger, error) {
	ctx, cancel := grpcContext()
	defer cancel()

	ts := ptypes.TimestampNow()
	ts.Seconds -= int64(triggerTimespan.Seconds())
	triggerResp, err := s.benchmarkTriggersClient.GetTriggers(ctx, &v1.GetBenchmarkTriggersRequest{
		Clusters: []string{s.cluster},
		FromTime: ts,
	})
	if err != nil {
		return nil, err
	}
	return triggerResp.Triggers, err
}

// Need to see if we have launched a trigger before
func (s *SchedulerClient) initializeTriggers() {
	triggers, err := s.getTriggers()
	if err != nil {
		log.Error(err)
		return
	}
	for _, trigger := range triggers {
		triggered, err := ptypes.Timestamp(trigger.GetTime())
		if err != nil {
			log.Errorf("Could not convert triggered time %v to golang type", trigger.GetTime())
			continue
		}
		scanID := uniqueScanID(triggered, trigger.GetName(), "triggered")
		results, err := s.getBenchmarkResults(scanID)
		if err != nil {
			log.Errorf("Error getting benchmark results for scan %v", scanID)
			continue
		}
		if len(results) != 0 {
			s.triggers[trigger.Name] = trigger
		}
	}
}

func (s *SchedulerClient) removeService(id string) {
	for i := 1; i < retries+1; i++ {
		if err := s.orchestrator.Kill(id); err != nil {
			log.Errorf("Error removing benchmark service %v: %+v", id, err)
		} else {
			return
		}
		time.Sleep(time.Duration(i) * 2 * time.Second)
	}
	log.Error("Timed out trying to remove benchmark service")
}

func (s *SchedulerClient) waitForBenchmarkToFinish(serviceName string) {
	timeout := time.NewTimer(cleanupTimeout)
	ticker := time.NewTicker(15 * time.Second)

	client, err := docker.NewClient()
	if err != nil {
		log.Error(err)
		// default to timeout
		ticker.Stop()
	}

LOOP:
	for {
		select {
		case <-ticker.C:
			ctx, cancel := docker.TimeoutContext()
			defer cancel()
			f := filters.NewArgs()
			f.Add("name", serviceName)

			tasks, err := client.TaskList(ctx, types.TaskListOptions{Filters: f})
			if err != nil {
				log.Error(err)
				continue
			}
			if len(tasks) == 0 {
				continue
			}
			numNotFinished := len(tasks)
			for _, task := range tasks {
				switch task.Status.State {
				case swarm.TaskStateComplete, swarm.TaskStateShutdown, swarm.TaskStateFailed, swarm.TaskStateRejected:
					numNotFinished--
				}
			}
			if numNotFinished == 0 {
				log.Infof("All tasks are complete")
				break LOOP
			}
		case <-timeout.C:
			break LOOP

		}

	}
	s.removeService(serviceName)
}

// Launch triggers a run of the benchmark immediately.
// The stateLock must be held by the caller until this function returns.
func (s *SchedulerClient) Launch(scanID string, benchmark *v1.Benchmark) error {
	name := "benchmark_bootstrap_" + strings.Replace(benchmark.Name, " ", "_", -1)
	service := orchestrators.SystemService{
		Name: name,
		Envs: []string{
			fmt.Sprintf("%s=%s", env.Image.EnvVar(), s.image),
			fmt.Sprintf("%s=%s", env.AdvertisedEndpoint.EnvVar(), env.AdvertisedEndpoint.Setting()),
			fmt.Sprintf("%s=%s", env.ScanID.EnvVar(), scanID),
			fmt.Sprintf("%s=%s", env.Checks.EnvVar(), strings.Join(benchmark.Checks, ",")),
			fmt.Sprintf("%s=%s", env.BenchmarkName.EnvVar(), benchmark.Name),
		},
		Image:   s.image,
		Mounts:  []string{"/var/run/docker.sock:/var/run/docker.sock"},
		Global:  true,
		Command: []string{"benchmark-bootstrap"},
	}
	_, err := s.orchestrator.Launch(service)
	if err != nil {
		return err
	}
	s.waitForBenchmarkToFinish(name)
	return nil
}

func nextScheduledTime(schedule *v1.BenchmarkSchedule) (time.Time, error) {
	startTime, err := ptypes.Timestamp(schedule.GetStartTime())
	if err != nil {
		return startTime, err
	}
	nextDate := startTime
	for nextDate.Before(time.Now()) {
		nextDate = nextDate.AddDate(0, 0, int(schedule.IntervalDays))
	}
	return nextDate, nil
}

func (s *SchedulerClient) updateTriggers() {
	triggers, err := s.getTriggers()
	if err != nil {
		log.Error()
		return
	}
	for _, trigger := range triggers {
		key := trigger.GetTime().String()
		if _, ok := s.triggers[key]; !ok {
			t, err := ptypes.Timestamp(trigger.GetTime())
			if err != nil {
				log.Error(err)
				continue
			}
			scanID := uniqueScanID(t, trigger.GetName(), "triggered")
			log.Infof("Adding %v to the benchmark queue", scanID)
			s.benchmarkChan <- benchmarkRun{scanID: scanID, benchmarkName: trigger.GetName()}
			s.triggers[key] = trigger
		}
	}
}

func (s *SchedulerClient) updateSchedules() {
	schedules, err := s.getSchedules()
	if err != nil {
		log.Error(err)
		return
	}
	currentSchedules := mapset.NewSet()
	for _, schedule := range schedules {
		oldSchedule, exists := s.schedules[schedule.Name]
		// If the schedule doesn't exist or has been updated then start scheduling for it
		if !exists || protoconv.CompareProtoTimestamps(schedule.LastUpdated, oldSchedule.LastUpdated) != 0 {
			nextTime, err := nextScheduledTime(schedule)
			if err != nil {
				log.Error(err)
				continue
			}
			s.schedules[schedule.Name] = &scheduleMetadata{
				BenchmarkSchedule: schedule,
				NextScanTime:      nextTime,
			}
		}
		currentSchedules.Add(schedule.Name)
	}

	for name := range s.schedules {
		if !currentSchedules.Contains(name) {
			delete(s.schedules, name)
		}
	}
	// Run through the schedules and run their benchmarks if they have expired
	now := time.Now()
	for benchmarkName, scheduleMetadata := range s.schedules {
		nextScanTime := scheduleMetadata.NextScanTime
		if nextScanTime.Before(now) {
			scanID := uniqueScanID(nextScanTime, benchmarkName, "scheduled")
			// Add benchmark to the queue to be scheduled
			log.Infof("Adding %v to the benchmark queue", scanID)
			s.benchmarkChan <- benchmarkRun{scanID: scanID, benchmarkName: benchmarkName}

			// Update the benchmark time to be triggered on the next scan time
			scheduleMetadata.NextScanTime, err = nextScheduledTime(scheduleMetadata.BenchmarkSchedule)
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (s *SchedulerClient) launchBenchmark(scanID, benchmarkName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	benchmark, err := s.benchmarkClient.GetBenchmark(ctx, &v1.GetBenchmarkRequest{Name: benchmarkName})
	if err != nil {
		return err
	}
	if err := s.Launch(scanID, benchmark); err != nil {
		return fmt.Errorf("Error launching benchmark: %s", err)
	}
	return nil
}

func uniqueScanID(t time.Time, benchmarkName, triggerType string) string {
	return fmt.Sprintf("%v %d-%02d-%02d %02d:%02d:00 %v", benchmarkName,
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), triggerType)
}

// Start runs the scheduler
func (s *SchedulerClient) Start() {
	// Initialize triggers that have results from this agent
	s.initializeTriggers()
	for {
		select {
		case <-s.updateTicker.C:
			// Update the schedules and schedule any that need to be run
			s.updateSchedules()
			// Update the triggers and schedule any ones that need to be run
			s.updateTriggers()
		case run := <-s.benchmarkChan:
			log.Infof("Launching benchmark %v for scan id '%s'", run.benchmarkName, run.scanID)
			if err := s.launchBenchmark(run.scanID, run.benchmarkName); err != nil {
				log.Errorf("Error launching benchmark %v with scan id '%v': %+v", run.benchmarkName, run.scanID, err)
			}
		case <-s.done:
			s.started = false
			return
		}
	}
}

// Stop stops the scheduler client from triggering any more jobs.
func (s *SchedulerClient) Stop() {
	s.clientConn.Close()
	s.done <- struct{}{}

	// TODO(cg): Also stop any launched benchmark.
}
