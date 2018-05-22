package sources

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

const (
	updateInterval = 15 * time.Second
)

// A Client checks for new registry integrations.
type Client struct {
	updateTicker *time.Ticker

	integrations []*ImageIntegration
	lock         sync.RWMutex

	conn *grpc.ClientConn

	clusterID       string
	centralEndpoint string

	done chan struct{}
}

// NewImageIntegrationsClient returns a new client of the integrations API
func NewImageIntegrationsClient(conn *grpc.ClientConn, clusterID string) *Client {
	return &Client{
		updateTicker: time.NewTicker(updateInterval),
		clusterID:    clusterID,
		conn:         conn,
		done:         make(chan struct{}),
	}
}

// Start runs the client
func (c *Client) Start() {
	for {
		select {
		case <-c.updateTicker.C:
			c.doUpdate()
		case <-c.done:
			return
		}
	}
}

func (c *Client) doUpdate() {
	cli := v1.NewImageIntegrationServiceClient(c.conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := cli.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{Cluster: c.clusterID})
	if err != nil {
		log.Errorf("Error checking integrations: %s", err)
		return
	}
	c.replaceImageIntegrations(resp)
}
func (c *Client) replaceImageIntegrations(resp *v1.GetImageIntegrationsResponse) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.integrations = nil
	for _, integration := range resp.GetIntegrations() {
		s, err := NewImageIntegration(integration)
		if err != nil {
			log.Errorf("Could not instantiate integration %v: %s", integration, err)
			continue
		}
		c.integrations = append(c.integrations, s)
	}
}

// Stop stops polling for new integrations.
func (c *Client) Stop() {
	c.done <- struct{}{}
}

// Integrations returns the currently-defined set of image integrations.
func (c *Client) Integrations() []*ImageIntegration {
	if c == nil {
		return nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	integrations := make([]*ImageIntegration, len(c.integrations))
	for i, s := range c.integrations {
		integrations[i] = s
	}
	return integrations
}
