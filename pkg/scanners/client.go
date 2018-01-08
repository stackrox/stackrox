package scanners

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("scanners")
)

const (
	updateInterval = 15 * time.Second
)

// A Client checks for new scanner integrations.
type Client struct {
	updateTicker *time.Ticker

	scanners []ImageScanner
	lock     sync.RWMutex

	clusterID      string
	apolloEndpoint string

	done chan struct{}
}

// NewScannersClient returns a new client of the scanners API
func NewScannersClient(apolloEndpoint string, clusterID string) *Client {
	return &Client{
		updateTicker:   time.NewTicker(updateInterval),
		clusterID:      clusterID,
		apolloEndpoint: apolloEndpoint,
		done:           make(chan struct{}),
	}
}

// Start runs the client
func (c *Client) Start() {
	conn, err := clientconn.GRPCConnection(c.apolloEndpoint)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := v1.NewScannerServiceClient(conn)
	for {
		select {
		case <-c.updateTicker.C:
			c.doUpdate(cli)
		case <-c.done:
			return
		}
	}
}

func (c *Client) doUpdate(cli v1.ScannerServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := cli.GetScanners(ctx, &v1.GetScannersRequest{RequestorIsAgent: true, Cluster: c.clusterID})
	if err != nil {
		log.Errorf("Error checking scanners: %s", err)
		return
	}
	c.replaceScanners(resp)
}
func (c *Client) replaceScanners(resp *v1.GetScannersResponse) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, scanner := range resp.GetScanners() {
		s, err := CreateScanner(scanner)
		if err != nil {
			log.Errorf("Could not instantiate scanner %v: %s", scanner, err)
			continue
		}
		c.scanners = append(c.scanners, s)
	}
}

// Stop stops polling for new scanners.
func (c *Client) Stop() {
	c.done <- struct{}{}
}

// Scanners returns the currently-defined set of image scanners.
func (c *Client) Scanners() []ImageScanner {
	if c == nil {
		return nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	scanners := make([]ImageScanner, len(c.scanners))
	for i, s := range c.scanners {
		scanners[i] = s
	}
	return scanners
}
