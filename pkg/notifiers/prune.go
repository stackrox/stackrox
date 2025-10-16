package notifiers

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/jsonutil"
)

const (
	sizeBuffer = 50
)

func cleanProcessIndicator(process *storage.ProcessIndicator) *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{}
	pi.SetId(process.GetId())
	pi.SetContainerName(process.GetContainerName())
	pi.SetPodId(process.GetPodId())
	pi.SetSignal(process.GetSignal())
	return pi
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
	// Clean Process first then prune
	for _, p := range processes {
		cleanProcessIndicator(p)
	}

	for i := len(processes) - 1; i >= 0; i-- {
		data, err := jsonutil.MarshalToCompactString(processes[i])
		if err != nil {
			log.Error(err)
		}
		*currSize -= len(data)
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
	for i := len(violations) - 1; i >= 0; i-- {
		data, err := jsonutil.MarshalToCompactString(violations[i])
		if err != nil {
			log.Error(err)
		}
		*currSize -= len(data)
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
	data, err := jsonutil.MarshalToCompactString(alert)
	if err != nil {
		log.Error(err)
	}

	currSize := len(data)
	filterDeploymentMaps(alert.GetDeployment(), maxSize, &currSize)

	if alert.GetProcessViolation() != nil {
		alert.GetProcessViolation().SetProcesses(filterProcesses(alert.GetProcessViolation().GetProcesses(), maxSize, &currSize))
	}
	alert.SetViolations(filterViolations(alert.GetViolations(), maxSize, &currSize))
}
