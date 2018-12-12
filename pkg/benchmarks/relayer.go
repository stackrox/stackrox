// Package benchmarks handles receiving and relaying benchmarks to StackRox Central.
package benchmarks

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
)

const (
	cacheSize      = 1000
	interval       = 2 * time.Second
	requestTimeout = 3 * time.Second
)

// A Relayer sends received benchmark payloads onto StackRox Central.
type Relayer interface {
	Start()
	Stop()
	Accept(payload *storage.BenchmarkResult)
}

// An LRURelayer sends received benchmark payloads onto StackRox Central.
// If the relay is not successful at first, payloads are cached and will
// be retried until new ones exceed the cache size.
type LRURelayer struct {
	cache *lru.Cache

	centralEndpoint string
	conn            *grpc.ClientConn

	tick    *time.Ticker
	stopSig concurrency.Signal

	logger *logging.Logger
}

// NewLRURelayer creates a new LRURelayer, which must then be started.
func NewLRURelayer(conn *grpc.ClientConn) *LRURelayer {
	cache, err := lru.New(cacheSize)
	if err != nil {
		// This only happens in extreme cases (at this time, for invalid size only).
		panic(err)
	}
	return &LRURelayer{
		conn:    conn,
		cache:   cache,
		tick:    time.NewTicker(interval),
		stopSig: concurrency.NewSignal(),
		logger:  logging.NewOrGet("relayer"),
	}
}

// Start starts the periodic retry task for cached payloads.
func (r *LRURelayer) Start() {
	for {
		select {
		case <-r.tick.C:
			r.run()
		case <-r.stopSig.Done():
			r.tick.Stop()
			return
		}
	}
}

// Stop stops any ongoing processing inside the LRURelayer.
func (r *LRURelayer) Stop() {
	r.stopSig.Signal()
}

func payloadKey(payload *storage.BenchmarkResult) string {
	return strings.Join([]string{payload.GetScanId(), payload.GetHost(), payload.GetStartTime().String()}, "-")
}

// Accept accepts a new payload, tries to relay it, and caches if unsuccessful.
func (r *LRURelayer) Accept(payload *storage.BenchmarkResult) {
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
		err := r.relay(obj.(*storage.BenchmarkResult))
		if err != nil {
			r.logger.Warnf("Couldn't retry %s: %s", k, err)
		} else {
			r.cache.Remove(k)
		}
	}
}

func (r *LRURelayer) relay(payload *storage.BenchmarkResult) error {
	cli := v1.NewBenchmarkResultsServiceClient(r.conn)

	r.logger.Infof("Relaying payload %s", payloadKey(payload))
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	_, err := cli.PostBenchmarkResult(ctx, payload)
	return err
}
