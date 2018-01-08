// Package benchmarks handles receiving and relaying benchmarks to Apollo.
package benchmarks

import (
	"context"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/hashicorp/golang-lru"
)

const (
	cacheSize      = 1000
	interval       = 2 * time.Second
	requestTimeout = 3 * time.Second
)

// A Relayer sends received benchmark payloads onto central Apollo.
type Relayer interface {
	Start()
	Stop()
	Accept(payload *v1.BenchmarkResult)
}

// An LRURelayer sends received benchmark payloads onto central Apollo.
// If the relay is not successful at first, payloads are cached and will
// be retried until new ones exceed the cache size.
type LRURelayer struct {
	cache *lru.Cache

	client v1.BenchmarkResultsServiceClient

	clusterID string

	tick  *time.Ticker
	stopC chan struct{}

	logger *logging.Logger
}

// NewLRURelayer creates a new LRURelayer, which must then be started.
func NewLRURelayer(client v1.BenchmarkResultsServiceClient, clusterID string) *LRURelayer {
	cache, err := lru.New(cacheSize)
	if err != nil {
		// This only happens in extreme cases (at this time, for invalid size only).
		panic(err)
	}
	return &LRURelayer{
		cache:     cache,
		client:    client,
		clusterID: clusterID,
		tick:      time.NewTicker(interval),
		stopC:     make(chan struct{}),
		logger:    logging.New("relayer"),
	}
}

// Start starts the periodic retry task for cached payloads.
func (r *LRURelayer) Start() {
	for {
		select {
		case <-r.tick.C:
			r.run()
		case <-r.stopC:
			r.tick.Stop()
			return
		}
	}
}

// Stop stops any ongoing processing inside the LRURelayer.
func (r *LRURelayer) Stop() {
	r.stopC <- struct{}{}
}

func payloadKey(payload *v1.BenchmarkResult) string {
	return strings.Join([]string{payload.GetScanId(), payload.GetHost(), payload.GetStartTime().String()}, "-")
}

// Accept accepts a new payload, tries to relay it, and caches if unsuccessful.
func (r *LRURelayer) Accept(payload *v1.BenchmarkResult) {
	payload.ClusterId = r.clusterID
	err := r.relay(payload)
	if err != nil {
		r.logger.Warnf("Couldn't send %s: %s", payloadKey(payload), err)
		r.cache.Add(payloadKey(payload), payload)
	}
}

func (r *LRURelayer) run() {
	for _, k := range r.cache.Keys() {
		obj, ok := r.cache.Peek(k)
		if !ok {
			// Must have been evicted. Nothing more we can do about that.
			continue
		}
		err := r.relay(obj.(*v1.BenchmarkResult))
		if err != nil {
			r.logger.Warnf("Couldn't retry %s: %s", k, err)
		} else {
			r.cache.Remove(k)
		}
	}
}

func (r *LRURelayer) relay(payload *v1.BenchmarkResult) error {
	r.logger.Infof("Relaying payload %s", payloadKey(payload))
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := r.client.PostBenchmarkResult(ctx, payload)
	return err
}
