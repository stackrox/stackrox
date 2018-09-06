package sensor

import "github.com/stackrox/rox/generated/api/v1"

// Pipeline defines the way to process a signal
type Pipeline interface {
	Process(signal *v1.Signal)
}
