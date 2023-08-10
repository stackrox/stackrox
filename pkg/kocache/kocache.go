package kocache

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

const (
	objMemLimit  = 1024 * 1024      // 1MB, most probes are ~50kb
	objHardLimit = 10 * 1024 * 1024 // 10MB

	cleanupThreshold = 10               // never clean up if there are not more than this number of objects in the cache.
	errorCleanupAge  = 30 * time.Second // clean up error entries after this time
	cleanupAge       = 5 * time.Minute  // clean up objects that are at least this old
	cleanupInterval  = 1 * time.Minute
)

var (
	errNotFound = errors.New("not found")
)

// options controls the behavior of the kernel object cache.
type options struct {
	ObjMemLimit  int
	ObjHardLimit int

	CleanupThreshold int
	CleanupAge       time.Duration
	ErrorCleanUpAge  time.Duration
	CleanupInterval  time.Duration

	StartOnline bool

	ModifyRequest func(*http.Request)
}

// StartOffline sets the initial state of cache to offline.
// Cache in the offline state will not attempt to reach Central.
func StartOffline() func(o *options) {
	return func(o *options) {
		o.StartOnline = false
	}
}

// defaultsOptions provides default set of options
func applyDefaults(o *options) *options {
	if o.ObjMemLimit == 0 {
		o.ObjMemLimit = objMemLimit
	}
	if o.ObjHardLimit == 0 {
		o.ObjHardLimit = objHardLimit
	}
	if o.CleanupThreshold == 0 {
		o.CleanupThreshold = cleanupThreshold
	}
	if o.CleanupAge == 0 {
		o.CleanupAge = cleanupAge
	}
	if o.ErrorCleanUpAge == 0 {
		o.ErrorCleanUpAge = errorCleanupAge
	}
	if o.CleanupInterval == 0 {
		o.CleanupInterval = cleanupInterval
	}
	o.StartOnline = true
	return o
}

type koCache struct {
	opts *options

	entries      map[string]*entry
	entriesMutex sync.Mutex

	upstreamClient  *http.Client
	upstreamBaseURL string

	centralReady atomic.Bool
}

// New returns a new kernel object cache whose lifetime is bound by the given context, using the given
// HTTP client and base URL for upstream requests.
func New(ctx context.Context, upstreamClient *http.Client, upstreamBaseURL string, opts ...func(o *options)) *koCache {
	opt := applyDefaults(&options{})
	for _, option := range opts {
		option(opt)
	}
	cache := &koCache{
		opts:            opt,
		entries:         make(map[string]*entry),
		upstreamClient:  upstreamClient,
		upstreamBaseURL: strings.TrimSuffix(upstreamBaseURL, "/"),
		centralReady:    atomic.Bool{},
	}
	cache.centralReady.Store(opt.StartOnline)

	go cache.cleanupLoop(ctx)
	return cache
}

func (c *koCache) GoOnline() {
	c.centralReady.Store(true)
}

func (c *koCache) GoOffline() {
	c.centralReady.Store(false)
}

func (c *koCache) getOrAddEntry(path string) (*entry, error) {
	c.entriesMutex.Lock()
	defer c.entriesMutex.Unlock()

	e := c.entries[path]
	if e != nil {
		if err, ok := e.done.Error(); ok && err != nil {
			// Clean out error entries proactively, don't wait for the cleanup loop.
			if time.Since(e.CreationTime()) > c.opts.ErrorCleanUpAge {
				delete(c.entries, path)
				e = nil
			}
		}
	}

	if e == nil {
		if !c.centralReady.Load() {
			return nil, errors.New("central is currently unreachable")
		} else if c.entries == nil {
			return nil, errors.New("kernel object cache is shutting down")
		} else {
			e = newEntry()
			c.entries[path] = e
			go e.Populate(c.upstreamClient, fmt.Sprintf("%s/%s", c.upstreamBaseURL, path), c.opts)
		}
	}
	e.AcquireRef()
	return e, nil
}

func (c *koCache) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(c.opts.CleanupInterval)
	defer ticker.Stop()

	stop := false
	for !stop {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-ctx.Done():
			stop = true
		}
	}

	c.entriesMutex.Lock()
	defer c.entriesMutex.Unlock()

	for _, e := range c.entries {
		go e.Destroy()
	}
	c.entries = nil
}

func (c *koCache) cleanup() {
	c.entriesMutex.Lock()
	defer c.entriesMutex.Unlock()

	toClean := len(c.entries) - c.opts.CleanupThreshold
	if toClean <= 0 {
		return
	}

	type candidate struct {
		path string
		e    *entry
	}

	var candidates []candidate
	for path, e := range c.entries {
		if e.IsError() {
			// Delete error entries right away if they are too old.
			if time.Since(e.LastAccess()) > c.opts.ErrorCleanUpAge {
				delete(c.entries, path)
			}
			continue
		}

		if time.Since(e.LastAccess()) > c.opts.CleanupAge {
			candidates = append(candidates, candidate{
				path: path,
				e:    e,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].e.LastAccess().Before(candidates[j].e.LastAccess())
	})

	if len(candidates) > toClean {
		candidates = candidates[:toClean]
	}

	for _, cand := range candidates {
		delete(c.entries, cand.path)
		go cand.e.Destroy()
	}
}
