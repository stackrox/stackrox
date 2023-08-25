package notifiers

import (
	"bytes"
	"sort"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
)

const (
	sizeBuffer = 50
)

func cleanProcessIndicator(process *storage.ProcessIndicator) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            process.GetId(),
		ContainerName: process.GetContainerName(),
		PodId:         process.GetPodId(),
		Signal:        process.GetSignal(),
	}
}

type mapPair struct {
	key  string
	size int
}

func filterMap(m map[string]string, maxSize int, currSize *int) {
	if *currSize < maxSize {
		return
	}
	// Sort by longest key-value pair first to try and remove the fewest number of entries
	pairs := make([]mapPair, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, mapPair{key: k, size: len(k) + len(v)})
	}
	// reverse sort largest size to smallest
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].size > pairs[j].size })

	for _, p := range pairs {
		*currSize -= p.size
		delete(m, p.key)

		if *currSize < maxSize {
			return
		}
	}
}

func filterDeploymentMaps(deployment *storage.Alert_Deployment, maxSize int, currSize *int) {
	filterMap(deployment.GetAnnotations(), maxSize, currSize)
	filterMap(deployment.GetLabels(), maxSize, currSize)
}

func filterProcesses(processes []*storage.ProcessIndicator, maxSize int, currSize *int) []*storage.ProcessIndicator {
	if *currSize < maxSize {
		return processes
	}

	marshaler := new(jsonpb.Marshaler)
	// Clean Process first then prune
	for _, p := range processes {
		cleanProcessIndicator(p)
	}

	for i := len(processes) - 1; i >= 0; i-- {
		var data bytes.Buffer
		if err := marshaler.Marshal(&data, processes[i]); err != nil {
			log.Error(err)
		}
		*currSize -= data.Len()
		if *currSize < maxSize {
			return processes[:i]
		}
	}
	return nil
}

func filterViolations(violations []*storage.Alert_Violation, maxSize int, currSize *int) []*storage.Alert_Violation {
	if *currSize < maxSize {
		return violations
	}
	marshaler := new(jsonpb.Marshaler)

	for i := len(violations) - 1; i >= 0; i-- {
		var data bytes.Buffer
		if err := marshaler.Marshal(&data, violations[i]); err != nil {
			log.Error(err)
		}
		*currSize -= data.Len()
		if *currSize < maxSize {
			return violations[:i]
		}
	}
	return nil
}

// PruneAlert takes in an alert and max size and ensures that the resultant alert is < maxSize
func PruneAlert(alert *storage.Alert, maxSize int) {
	maxSize -= sizeBuffer

	// Get current size and then determine how to trim more in terms of violations
	var data bytes.Buffer
	marshaler := new(jsonpb.Marshaler)
	if err := marshaler.Marshal(&data, alert); err != nil {
		log.Error(err)
	}

	currSize := data.Len()
	filterDeploymentMaps(alert.GetDeployment(), maxSize, &currSize)

	if alert.ProcessViolation != nil {
		alert.ProcessViolation.Processes = filterProcesses(alert.GetProcessViolation().GetProcesses(), maxSize, &currSize)
	}
	alert.Violations = filterViolations(alert.GetViolations(), maxSize, &currSize)
}
