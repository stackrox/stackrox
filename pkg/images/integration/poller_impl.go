package integration

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/grpc"
)

type pollerImpl struct {
	centralConn *grpc.ClientConn

	clusterID string
	onUpdate  func([]*v1.ImageIntegration) error

	updateTicker *time.Ticker
	stop         chan struct{}
	stopped      chan struct{}
}

// Start starts polling.
func (c *pollerImpl) Start() {
	// push to the stopped channel when stopped.
	defer func() { c.stopped <- struct{}{} }()

	// Run until stopped.
	for {
		select {
		case <-c.updateTicker.C:
			c.doUpdate()
		case <-c.stop:
			return
		}
	}
}

// Stop stops polling.
func (c *pollerImpl) Stop() {
	// Send stop signal.
	c.stop <- struct{}{}

	// Wait until stopped.
	<-c.stopped
}

// doUpdate is run every cycle, creating a connection to the image integration service, making a request
// for this cluster, and executing the poll function with the results.
func (c *pollerImpl) doUpdate() {
	cli := v1.NewImageIntegrationServiceClient(c.centralConn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cli.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{Cluster: c.clusterID})
	if err != nil {
		log.Errorf("Error checking integrations: %s", err)
		return
	}

	if err = c.onUpdate(resp.GetIntegrations()); err != nil {
		log.Errorf("error on poll: ", err)
	}
}
