package integration

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

const (
	updateInterval = 15 * time.Second
)

// Poller provides an interface for starting and stopping background polling for image integrations.
type Poller interface {
	Run()
	Stop()
}

// NewPoller returns a new poller that updates the set.
func NewPoller(is Set, centralConn *grpc.ClientConn, clusterID string) Poller {
	// Everytime we poll the image integration service, replace all of our integrations with the result.
	poller := newPoller(centralConn, clusterID, func(integrations []*v1.ImageIntegration) error {
		is.Clear()
		errList := errorhelpers.NewErrorList("polling integrations")
		for _, ii := range integrations {
			if err := is.UpdateImageIntegration(ii); err != nil {
				errList.AddError(err)
			}
		}
		return errList.ToError()
	})

	return poller
}

// NewImageIntegrationsPoller returns a new instance of a Poller using the given connection and
// cluster id, and runs the given function on every poll cycle.
func newPoller(centralConn *grpc.ClientConn, clusterID string, onUpdate func([]*v1.ImageIntegration) error) Poller {
	return &pollerImpl{
		centralConn: centralConn,
		clusterID:   clusterID,
		onUpdate:    onUpdate,

		updateTicker: time.NewTicker(updateInterval),
		stop:         concurrency.NewSignal(),
		stopped:      concurrency.NewSignal(),
	}
}
