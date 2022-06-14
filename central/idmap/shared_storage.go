package idmap

import (
	"sync/atomic"
	"unsafe"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// sharedIDMapStorage is a concurrency-safe, copy-on-write storage for an IDMap that is optimized in such a way that
// subsequent writes do not require a copy if no read occurred in between them.
type sharedIDMapStorage struct {
	updateMutex sync.Mutex
	shared      unsafe.Pointer
	readOnly    unsafe.Pointer
}

// newSharedIDMapStorage creates a new shared storage for an ID map.
func newSharedIDMapStorage() *sharedIDMapStorage {
	return &sharedIDMapStorage{
		shared: unsafe.Pointer(NewIDMap()),
	}
}

func (s *sharedIDMapStorage) Update(updater func(m *IDMap) bool) {
	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	sharedInstance := (*IDMap)(atomic.LoadPointer(&s.shared))
	if sharedInstance == nil {
		// If we have no shared instance, clone the current read-only instance.
		sharedInstance = (*IDMap)(atomic.LoadPointer(&s.readOnly)).Clone()
		atomic.StorePointer(&s.shared, unsafe.Pointer(sharedInstance))
	}

	if !updater(sharedInstance) {
		return // no update
	}

	atomic.StorePointer(&s.readOnly, nil) // invalidate read-only instance
}

func (s *sharedIDMapStorage) Get() *IDMap {
	m := (*IDMap)(atomic.LoadPointer(&s.readOnly))
	if m != nil {
		return m
	}

	s.updateMutex.Lock()
	defer s.updateMutex.Unlock()

	// Another `Get()` call has already ensured the existence of a read-only instance.
	m = (*IDMap)(atomic.LoadPointer(&s.readOnly))
	if m != nil {
		return m
	}

	// Claim the current shared instance as the read-only instance.
	m = (*IDMap)(atomic.LoadPointer(&s.shared))
	atomic.StorePointer(&s.readOnly, unsafe.Pointer(m))
	atomic.StorePointer(&s.shared, nil)

	return m
}

func (s *sharedIDMapStorage) OnNamespaceAdd(nss ...*storage.NamespaceMetadata) {
	if len(nss) == 0 {
		return
	}
	s.Update(func(m *IDMap) bool {
		for _, ns := range nss {
			newNSInfo := &NamespaceInfo{
				Name:        ns.GetName(),
				ID:          ns.GetId(),
				ClusterName: ns.GetClusterName(),
				ClusterID:   ns.GetClusterId(),
			}
			nsInfo := m.byNamespaceID[ns.GetId()]
			if nsInfo != nil && *nsInfo == *newNSInfo {
				return false
			}
			m.byNamespaceID[ns.GetId()] = newNSInfo
		}
		return true
	})
}

func (s *sharedIDMapStorage) OnNamespaceRemove(nsIDs ...string) {
	if len(nsIDs) == 0 {
		return
	}
	s.Update(func(m *IDMap) bool {
		prevLen := len(m.byNamespaceID)
		for _, nsID := range nsIDs {
			delete(m.byNamespaceID, nsID)
		}
		return prevLen > len(m.byNamespaceID)
	})
}
