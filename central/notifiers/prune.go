package notifiers

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func cleanProcessIndicator(process *storage.ProcessIndicator) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            process.GetId(),
		ContainerName: process.GetContainerName(),
		PodId:         process.GetPodId(),
		Signal:        process.GetSignal(),
	}
}

func filterToRequiredFields(alert *storage.Alert) {
	deployment := &storage.Deployment{
		Id:          alert.GetDeployment().GetId(),
		Name:        alert.GetDeployment().GetName(),
		Type:        alert.GetDeployment().GetType(),
		ClusterName: alert.GetDeployment().GetClusterName(),
		Namespace:   alert.GetDeployment().GetNamespace(),
	}

	for _, c := range alert.GetDeployment().GetContainers() {
		deployment.Containers = append(deployment.Containers, &storage.Container{
			Id:   c.GetId(),
			Name: c.GetName(),
			Image: &storage.ContainerImage{
				Id:   c.GetImage().GetId(),
				Name: c.GetImage().GetName(),
			},
		})
	}
	alert.Deployment = deployment
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
	filterToRequiredFields(alert)

	// Get current size and then determine how to trim more in terms of violations
	var data bytes.Buffer
	marshaler := new(jsonpb.Marshaler)
	if err := marshaler.Marshal(&data, alert); err != nil {
		log.Error(err)
	}

	currSize := data.Len()
	if currSize < maxSize {
		return
	}

	if alert.ProcessViolation != nil {
		alert.ProcessViolation.Processes = filterProcesses(alert.GetProcessViolation().GetProcesses(), maxSize, &currSize)
	}
	alert.Violations = filterViolations(alert.GetViolations(), maxSize, &currSize)
}
