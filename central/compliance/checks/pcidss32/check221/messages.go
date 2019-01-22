package check221

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const interpretationText = `StackRox has visibility into the binaries being run in a container. A container can be said to be 
performing a single function if it invokes processes from no more than a single binary. Therefore, a
deployment can be considered compliant if all of its containers operate with no more than a single binary.`

func passText() string {
	return "Every container in Deployment has launched processes from no more than one binary"
}

func failText(container *storage.Container) string {
	return fmt.Sprintf("Container %s in Deployment is running processes from multiple binaries, indicating the container is performing multiple tasks", container.GetName())
}
