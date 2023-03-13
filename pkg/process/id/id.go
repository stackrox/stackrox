package id

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// This will impact scale negatively if you change this. Please only change this if you know what you're doing
	processIDNamespace = uuid.FromStringOrPanic("801fcce1-56d3-48bd-b1ac-c41fdc6c3d94")
)

// GetIndicatorIDFromParts sets the indicator ID based on the stable namespace
func GetIndicatorIDFromParts(podId string, containerName string, execFilePath string, name string, args string) string {
	id := uuid.NewV5(processIDNamespace,
		fmt.Sprintf("%s %s %s %s %s", podId, containerName,
			execFilePath, name, args)).String()

	return id
}

// SetIndicatorID sets the indicator ID based on the stable namespace
func SetIndicatorID(indicator *storage.ProcessIndicator) {
//	id := uuid.NewV5(processIDNamespace,
//		fmt.Sprintf("%s %s %s %s %s", indicator.GetPodId(), indicator.GetContainerName(),
//			indicator.GetSignal().GetExecFilePath(), indicator.GetSignal().GetName(), indicator.GetSignal().GetArgs())).String()
	id := GetIndicatorIDFromParts(indicator.GetPodId(), indicator.GetContainerName(),
			indicator.GetSignal().GetExecFilePath(), indicator.GetSignal().GetName(), indicator.GetSignal().GetArgs())

	indicator.Id = id
}

func GetIndicatorIDFromProcessIndicatorUniqueKey(uniqueKey *storage.ProcessIndicatorUniqueKey) string {
	return GetIndicatorIDFromParts(uniqueKey.GetPodId(), uniqueKey.GetContainerName(),
		uniqueKey.GetProcessExecFilePath(), uniqueKey.GetProcessName(), uniqueKey.GetProcessArgs())
}
