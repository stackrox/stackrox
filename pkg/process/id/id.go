package id

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/uuid"
)

var (
	// This will impact scale negatively if you change this. Please only change this if you know what you're doing
	processIDNamespace = uuid.FromStringOrPanic("801fcce1-56d3-48bd-b1ac-c41fdc6c3d94")
)

// SetIndicatorID sets the indicator ID based on the stable namespace
func SetIndicatorID(indicator *storage.ProcessIndicator) {
	id := uuid.NewV5(processIDNamespace,
		fmt.Sprintf("%s %s %s %s %s", indicator.GetPodId(), indicator.GetContainerName(),
			indicator.GetSignal().GetExecFilePath(), indicator.GetSignal().GetName(), indicator.GetSignal().GetArgs())).String()
	indicator.Id = id
}
