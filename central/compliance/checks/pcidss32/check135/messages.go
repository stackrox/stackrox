package check135

import (
	"fmt"
)

const interpretationText = `StackRox has visibility into the ports and protocols enabled by containers in the environment.
Connections cannot be established without bidirectional communication. UDP protocol is unidirectional.
Therefore, a deployment can be considered compliant if none of its exposed ports are using UDP.`

func passText() string {
	return fmt.Sprintf("Deployment does not use UDP")
}

func failText() string {
	return fmt.Sprintf("Deployment uses UDP, which allows data exchange without an established connection")
}
