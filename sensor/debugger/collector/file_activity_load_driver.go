package collector

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LoadDriverConfig configures the file activity load driver.
type LoadDriverConfig struct {
	// EventsPerSecond is the target event rate. 0 means unlimited (burst mode).
	EventsPerSecond int
	// NumUniquePaths is the number of distinct file paths to generate events for.
	NumUniquePaths int
	// Hostname is the hostname to set on generated events.
	Hostname string
	// ContainerID is the container ID to set on generated process signals.
	// Empty means generate node-level events (no deployment enrichment).
	ContainerID string
	// OperationWeights controls the relative frequency of each operation type.
	// Keys: "open", "create", "unlink", "rename", "chmod", "chown"
	// Omitted keys default to 0. If all are 0, defaults to equal distribution.
	OperationWeights map[string]int
}

// DefaultLoadDriverConfig returns a sensible default configuration.
func DefaultLoadDriverConfig() LoadDriverConfig {
	return LoadDriverConfig{
		EventsPerSecond: 100,
		NumUniquePaths:  50,
		Hostname:        "fake-collector",
		ContainerID:     "",
		OperationWeights: map[string]int{
			"open":   40,
			"create": 20,
			"unlink": 10,
			"rename": 10,
			"chmod":  10,
			"chown":  10,
		},
	}
}

// LoadDriverStats tracks load driver execution statistics.
type LoadDriverStats struct {
	EventsSent int64
	StartTime  time.Time
	EndTime    time.Time
	Errors     int64
}

// ActualRate returns the achieved events per second.
func (s LoadDriverStats) ActualRate() float64 {
	elapsed := s.EndTime.Sub(s.StartTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(s.EventsSent) / elapsed
}

// FileActivityLoadDriver generates synthetic file activity events and sends them
// through a FakeCollector to stress-test the Sensor file activity pipeline.
type FileActivityLoadDriver struct {
	config    LoadDriverConfig
	collector *FakeCollector
	stopper   concurrency.Stopper
	paths     []string
	opPicker  weightedPicker
}

// NewFileActivityLoadDriver creates a load driver attached to the given fake collector.
func NewFileActivityLoadDriver(collector *FakeCollector, config LoadDriverConfig) *FileActivityLoadDriver {
	paths := generatePaths(config.NumUniquePaths)
	picker := buildWeightedPicker(config.OperationWeights)

	return &FileActivityLoadDriver{
		config:    config,
		collector: collector,
		stopper:   concurrency.NewStopper(),
		paths:     paths,
		opPicker:  picker,
	}
}

// Run starts generating events and blocks until the stopper is triggered.
// Returns statistics about the run.
func (d *FileActivityLoadDriver) Run() LoadDriverStats {
	stats := LoadDriverStats{StartTime: time.Now()}
	defer func() { stats.EndTime = time.Now() }()

	if d.config.EventsPerSecond <= 0 {
		d.runBurst(&stats)
	} else {
		d.runRateLimited(&stats)
	}
	return stats
}

// Stop signals the load driver to stop.
func (d *FileActivityLoadDriver) Stop() {
	d.stopper.Client().Stop()
}

func (d *FileActivityLoadDriver) runRateLimited(stats *LoadDriverStats) {
	interval := time.Second / time.Duration(d.config.EventsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopper.Flow().StopRequested():
			return
		case <-ticker.C:
			d.sendEvent(stats)
		}
	}
}

func (d *FileActivityLoadDriver) runBurst(stats *LoadDriverStats) {
	for {
		select {
		case <-d.stopper.Flow().StopRequested():
			return
		default:
			d.sendEvent(stats)
		}
	}
}

func (d *FileActivityLoadDriver) sendEvent(stats *LoadDriverStats) {
	msg := d.generateEvent()
	d.collector.SendFakeFileActivity(msg)
	stats.EventsSent++

	if stats.EventsSent%10000 == 0 {
		elapsed := time.Since(stats.StartTime).Seconds()
		log.Infof("Load driver: sent %d events (%.1f events/sec)", stats.EventsSent, float64(stats.EventsSent)/elapsed)
	}
}

func (d *FileActivityLoadDriver) generateEvent() *sensor.FileActivity {
	path := d.paths[rand.Intn(len(d.paths))]
	op := d.opPicker.pick()
	now := timestamppb.Now()

	activity := &sensor.FileActivity{
		Timestamp: now,
		Process:   d.generateProcess(),
		Hostname:  d.config.Hostname,
	}

	base := &sensor.FileActivityBase{
		Path:     path,
		HostPath: "/host" + path,
	}

	switch op {
	case "open":
		activity.File = &sensor.FileActivity_Open{
			Open: &sensor.FileOpen{Activity: base},
		}
	case "create":
		activity.File = &sensor.FileActivity_Creation{
			Creation: &sensor.FileCreation{Activity: base},
		}
	case "unlink":
		activity.File = &sensor.FileActivity_Unlink{
			Unlink: &sensor.FileUnlink{Activity: base},
		}
	case "rename":
		newPath := d.paths[rand.Intn(len(d.paths))]
		activity.File = &sensor.FileActivity_Rename{
			Rename: &sensor.FileRename{
				Old: base,
				New: &sensor.FileActivityBase{
					Path:     newPath,
					HostPath: "/host" + newPath,
				},
			},
		}
	case "chmod":
		activity.File = &sensor.FileActivity_Permission{
			Permission: &sensor.FilePermissionChange{
				Activity: base,
				Mode:     0644,
			},
		}
	case "chown":
		activity.File = &sensor.FileActivity_Ownership{
			Ownership: &sensor.FileOwnershipChange{
				Activity: base,
				Uid:      1000,
				Gid:      1000,
				Username: "testuser",
				Group:    "testgroup",
			},
		}
	}

	return activity
}

func (d *FileActivityLoadDriver) generateProcess() *sensor.ProcessSignal {
	return &sensor.ProcessSignal{
		Id:           uuid.NewV4().String(),
		ContainerId:  d.config.ContainerID,
		Name:         "test-process",
		Args:         "--flag value",
		ExecFilePath: "/usr/bin/test-process",
		Pid:          uint32(rand.Intn(65535) + 1),
		Uid:          1000,
		Gid:          1000,
	}
}

func generatePaths(n int) []string {
	dirs := []string{
		"/etc/security",
		"/etc/pam.d",
		"/etc/ssh",
		"/var/log",
		"/var/run",
		"/tmp",
		"/etc/kubernetes",
		"/etc/cni",
		"/etc/sysconfig",
		"/etc/audit",
	}

	paths := make([]string, 0, n)
	for i := range n {
		dir := dirs[i%len(dirs)]
		paths = append(paths, fmt.Sprintf("%s/file-%04d.conf", dir, i))
	}
	return paths
}

type weightedPicker struct {
	ops     []string
	weights []int
	total   int
}

func buildWeightedPicker(weights map[string]int) weightedPicker {
	allOps := []string{"open", "create", "unlink", "rename", "chmod", "chown"}

	hasWeights := false
	for _, w := range weights {
		if w > 0 {
			hasWeights = true
			break
		}
	}

	p := weightedPicker{}
	for _, op := range allOps {
		w := 1
		if hasWeights {
			w = weights[op]
		}
		if w > 0 {
			p.ops = append(p.ops, op)
			p.weights = append(p.weights, w)
			p.total += w
		}
	}
	return p
}

func (p *weightedPicker) pick() string {
	r := rand.Intn(p.total)
	for i, w := range p.weights {
		r -= w
		if r < 0 {
			return p.ops[i]
		}
	}
	return p.ops[len(p.ops)-1]
}
