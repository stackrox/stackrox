package manager

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/hash/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	flushInterval = env.HashFlushInterval.DurationSetting()
)

var (
	log = logging.LoggerForModule()
)

// Manager is a hash manager that provides access to cluster-based dedupers and persists
// the hashes into the database
type Manager interface {
	Start(ctx context.Context)

	GetDeduper(ctx context.Context, clusterID string) Deduper
	Delete(ctx context.Context, clusterID string) error
}

// NewManager instantiates a Manager
func NewManager(datastore datastore.Datastore) Manager {
	return &managerImpl{
		datastore: datastore,
		dedupers:  make(map[string]Deduper),
	}
}

type managerImpl struct {
	datastore datastore.Datastore

	dedupersLock sync.RWMutex
	dedupers     map[string]Deduper
}

func (m *managerImpl) flushHashes(ctx context.Context) {
	// Get clusters first to flush hashes one at a time.
	clusters := concurrency.WithLock1(&m.dedupersLock, func() []string {
		clusters := make([]string, 0, len(m.dedupers))
		for clusterID := range m.dedupers {
			clusters = append(clusters, clusterID)
		}
		return clusters
	})
	for _, cluster := range clusters {
		deduper, ok := concurrency.WithLock2(&m.dedupersLock, func() (Deduper, bool) {
			deduper, ok := m.dedupers[cluster]
			return deduper, ok
		})
		if !ok {
			continue
		}
		hash := &storage.Hash{
			ClusterId: cluster,
			Hashes:    deduper.GetSuccessfulHashes(),
		}
		if err := m.datastore.UpsertHash(ctx, hash); err != nil {
			log.Errorf("flushing hashes: %v", err)
		}
		dedupingHashSizeGauge.With(prometheus.Labels{"cluster": hash.GetClusterId()}).Set(float64(len(hash.GetHashes())))
	}
}

func (m *managerImpl) Start(ctx context.Context) {
	if flushInterval == 0 {
		return
	}
	t := time.NewTicker(flushInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.flushHashes(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (m *managerImpl) getDeduper(clusterID string) (Deduper, bool) {
	m.dedupersLock.RLock()
	defer m.dedupersLock.RUnlock()

	d, ok := m.dedupers[clusterID]
	return d, ok
}

func (m *managerImpl) GetDeduper(ctx context.Context, clusterID string) Deduper {
	d, ok := m.getDeduper(clusterID)
	if ok {
		return d
	}
	hash, exists, err := m.datastore.GetHashes(ctx, clusterID)
	if err != nil {
		log.Errorf("could not get hashes from database for cluster %q: %v", clusterID, err)
	}
	if !exists {
		d = NewDeduper(make(map[string]uint64))
	} else {
		d = NewDeduper(hash.GetHashes())
	}

	m.dedupersLock.Lock()
	defer m.dedupersLock.Unlock()

	existingDeduper, ok := m.dedupers[clusterID]
	if ok {
		return existingDeduper
	}
	m.dedupers[clusterID] = d
	return d
}

func (m *managerImpl) Delete(ctx context.Context, clusterID string) error {
	concurrency.WithLock(&m.dedupersLock, func() {
		delete(m.dedupers, clusterID)
	})

	return m.datastore.DeleteHashes(ctx, clusterID)
}
