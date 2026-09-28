package fake

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/events"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var fileActivityDirs = []string{
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

func (w *WorkloadManager) manageFileActivity(ctx context.Context) {
	defer w.wg.Done()
	w.sanitizeFileActivityParams()

	ticker := time.NewTicker(w.workload.FileActivityWorkload.ActivityInterval)
	defer ticker.Stop()
	paths := generateFileActivityPaths(w.workload.FileActivityWorkload.NumPaths)

	var nodeNames []string

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if !w.servicesInitialized.IsDone() {
			continue
		}

		if w.pubSubDispatcher == nil {
			continue
		}

		if len(nodeNames) == 0 {
			nodeNames = w.listNodeNames(ctx)
			if len(nodeNames) == 0 {
				continue
			}
		}

		for range w.workload.FileActivityWorkload.BatchSize {
			hostname := nodeNames[rand.Intn(len(nodeNames))]
			activity := w.generateFileActivity(paths, hostname)
			if activity == nil {
				continue
			}

			event := &events.FakeFileActivityEvent{
				Activity: activity,
			}

			if err := w.pubSubDispatcher.Publish(event); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Fatalf("Failed to publish fake file activity: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

func (w *WorkloadManager) listNodeNames(ctx context.Context) []string {
	nodeResp, err := w.fakeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("error listing nodes for file activity: %v", err)
		return nil
	}
	names := make([]string, 0, len(nodeResp.Items))
	for _, node := range nodeResp.Items {
		names = append(names, node.Name)
	}
	return names
}

func (w *WorkloadManager) generateFileActivity(paths []string, hostname string) *sensorAPI.FileActivity {
	path := paths[rand.Intn(len(paths))]
	now := timestamppb.Now()

	var process *sensorAPI.ProcessSignal
	if rand.Intn(100) >= w.workload.FileActivityWorkload.NodeEventPercent {
		// Container event - try to get from pool or generate with container ID
		containerID, ok := w.containerPool.randomElem()
		if ok {
			process = getRandomSensorProcess(containerID, w.processPool)
		}
	}

	if process == nil {
		// Node event (or no container available) - generate process with empty container ID
		process = getRandomNodeProcess()
	}

	base := &sensorAPI.FileActivityBase{
		Path:     path,
		HostPath: "/host" + path,
	}

	activity := &sensorAPI.FileActivity{
		Timestamp: now,
		Process:   process,
		Hostname:  hostname,
	}

	switch rand.Intn(6) {
	case 0:
		activity.File = &sensorAPI.FileActivity_Open{
			Open: &sensorAPI.FileOpen{Activity: base},
		}
	case 1:
		activity.File = &sensorAPI.FileActivity_Creation{
			Creation: &sensorAPI.FileCreation{Activity: base},
		}
	case 2:
		activity.File = &sensorAPI.FileActivity_Unlink{
			Unlink: &sensorAPI.FileUnlink{Activity: base},
		}
	case 3:
		newPath := paths[rand.Intn(len(paths))]
		activity.File = &sensorAPI.FileActivity_Rename{
			Rename: &sensorAPI.FileRename{
				Old: base,
				New: &sensorAPI.FileActivityBase{
					Path:     newPath,
					HostPath: "/host" + newPath,
				},
			},
		}
	case 4:
		activity.File = &sensorAPI.FileActivity_Permission{
			Permission: &sensorAPI.FilePermissionChange{
				Activity: base,
				Mode:     0644,
			},
		}
	case 5:
		activity.File = &sensorAPI.FileActivity_Ownership{
			Ownership: &sensorAPI.FileOwnershipChange{
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

func (w *WorkloadManager) sanitizeFileActivityParams() {
	fa := &w.workload.FileActivityWorkload
	if fa.NumPaths <= 0 {
		defaultNumPaths := 50
		log.Infof("FileActivityWorkload: numPaths=%d is invalid, defaulting to %d", fa.NumPaths, defaultNumPaths)
		fa.NumPaths = defaultNumPaths
	}
	if fa.BatchSize <= 0 {
		defaultBatchSize := 1
		log.Infof("FileActivityWorkload: batchSize=%d is invalid, defaulting to %d", fa.BatchSize, defaultBatchSize)
		fa.BatchSize = defaultBatchSize
	}
	if fa.NodeEventPercent < 0 || fa.NodeEventPercent > 100 {
		defaultNodeEventPercent := 50
		log.Infof("FileActivityWorkload: nodeEventPercent=%d is invalid, defaulting to %d", fa.NodeEventPercent, defaultNodeEventPercent)
		fa.NodeEventPercent = defaultNodeEventPercent
	}
	if fa.ActivityInterval <= 0 {
		defaultActivityInterval := 50 * time.Millisecond
		log.Infof("FileActivityWorkload: activityInterval=%s is invalid, defaulting to %s", fa.ActivityInterval, defaultActivityInterval)
		fa.ActivityInterval = defaultActivityInterval
	}
}

func generateFileActivityPaths(n int) []string {
	paths := make([]string, 0, n)
	for range n {
		dir := fileActivityDirs[rand.Intn(len(fileActivityDirs))]
		paths = append(paths, fmt.Sprintf("%s/file-%s.conf", dir, uuid.NewV4().String()[:8]))
	}
	return paths
}
