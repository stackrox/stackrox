package fake

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/uuid"
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
	if w.workload.FileActivityWorkload.ActivityInterval == 0 {
		return
	}

	ticker := time.NewTicker(w.workload.FileActivityWorkload.ActivityInterval)
	defer ticker.Stop()

	numPaths := w.workload.FileActivityWorkload.NumPaths
	if numPaths == 0 {
		numPaths = 50
	}
	paths := generateFileActivityPaths(numPaths)

	batchSize := w.workload.FileActivityWorkload.BatchSize
	if batchSize == 0 {
		batchSize = 1
	}

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

		if w.fileActivityChan == nil {
			continue
		}

		if len(nodeNames) == 0 {
			nodeResp, err := w.fakeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Errorf("error listing nodes for file activity: %v", err)
				continue
			}
			for _, node := range nodeResp.Items {
				nodeNames = append(nodeNames, node.Name)
			}
			if len(nodeNames) == 0 {
				continue
			}
		}

		for range batchSize {
			hostname := nodeNames[rand.Intn(len(nodeNames))]
			activity := w.generateFileActivity(paths, hostname)
			if activity == nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case w.fileActivityChan <- activity:
			}
		}
	}
}

func (w *WorkloadManager) generateFileActivity(paths []string, hostname string) *sensorAPI.FileActivity {
	path := paths[rand.Intn(len(paths))]
	now := timestamppb.Now()

	nodePercent := w.workload.FileActivityWorkload.NodeEventPercent
	if nodePercent == 0 {
		nodePercent = 50
	}

	var containerID string
	if rand.Intn(100) >= nodePercent {
		cid, ok := w.containerPool.randomElem()
		if ok {
			containerID = cid
		}
	}

	process := &sensorAPI.ProcessSignal{
		Id:           uuid.NewV4().String(),
		ContainerId:  containerID,
		Name:         "test-process",
		Args:         "--flag value",
		ExecFilePath: "/usr/bin/test-process",
		Pid:          uint32(rand.Intn(65535) + 1),
		Uid:          1000,
		Gid:          1000,
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

func generateFileActivityPaths(n int) []string {
	paths := make([]string, 0, n)
	for i := range n {
		dir := fileActivityDirs[i%len(fileActivityDirs)]
		paths = append(paths, fmt.Sprintf("%s/file-%04d.conf", dir, i))
	}
	return paths
}
