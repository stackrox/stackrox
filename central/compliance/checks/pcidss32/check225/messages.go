package check225

import (
	"fmt"
)

const interpretationText = `StackRox has visibility into the network traffic in a cluster. Ports that are open
and not receiving traffic can be said to be unused functionality. Therefore, a deployment can be considered
compliant if it does not expose any ports that do not receive traffic.`

func passText() string {
	return "Deployment has no unused exposed ports"
}

func failText(exposedAndUnused []uint32) string {
	return fmt.Sprintf("Deployment has exposed ports that are not receiving traffic: %v", exposedAndUnused)
}
