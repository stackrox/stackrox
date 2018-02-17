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
	log = logging.LoggerForModule()
)

const (
	updateInterval = 15 * time.Second
)

// A Client checks for new scanner integrations.
type Client struct {
	updateTicker *time.Ticker

	scanners []ImageScanner
	lock     sync.RWMutex

	clusterID       string
	centralEndpoint string

	done chan struct{}
}

// NewScannersClient returns a new client of the scanners API
func NewScannersClient(centralEndpoint string, clusterID string) *Client {
	return &Client{
		updateTicker:    time.NewTicker(updateInterval),
		clusterID:       clusterID,
		centralEndpoint: centralEndpoint,
		done:            make(chan struct{}),
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
	conn, err := clientconn.GRPCConnection(c.centralEndpoint)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := v1.NewScannerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := cli.GetScanners(ctx, &v1.GetScannersRequest{Cluster: c.clusterID})
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
