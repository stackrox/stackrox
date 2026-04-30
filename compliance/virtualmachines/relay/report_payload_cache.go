package relay

import (
	"container/list"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/sync"
)

const upsertExpiredEvictionPerInsert = 16

// reportPayloadCache stores VM reports keyed by resource ID with bounded capacity.
// Eviction removes the entry whose updatedAt is oldest (least recently updated), independent of read access.
//
// TTL behavior is enforced only by cleanup paths: Upsert performs a bounded front sweep of
// expired entries before insert/update, and SweepExpired removes expired entries on demand.
// Get does not enforce TTL and returns any entry that is currently present. This type does not
// start any background sweeper; callers may invoke SweepExpired periodically (for example the VM
// relay) to remove expired entries proactively. Remove deletes an entry on demand and does not
// consult TTL.
//
// Capacity: when maxSlots is non-positive, new keys are never cached (Upsert for an unknown resourceID
// is a no-op). Existing entries for that resource ID are unchanged by this constructor parameter alone;
// use maxSlots > 0 to store new keys.
type reportPayloadCache struct {
	maxSlots int
	ttl      time.Duration

	mu      sync.Mutex
	byID    map[string]*list.Element // resourceID -> LRU (least recently updated) list element (Value is *reportPayloadEntry)
	lruList *list.List               // front = LRU (oldest updatedAt), back = MRU (newest updatedAt)
}

type reportPayloadEntry struct {
	resourceID     string
	report         *v1.VMReport
	updatedAt      time.Time
	firstUpdatedAt time.Time
}

// payloadEviction records a cache entry removal event and the entry residency at removal time.
type payloadEviction struct {
	resourceID string
	residency  time.Duration // duration since the most recent payload update (now.Sub(updatedAt))
	lifetime   time.Duration // duration since the first payload insert for this entry (now.Sub(firstUpdatedAt))
}

// newReportPayloadCache creates a cache with the given slot limit and TTL duration. See reportPayloadCache
// for lazy TTL semantics and maxSlots <= 0 behavior.
func newReportPayloadCache(maxSlots int, ttl time.Duration) *reportPayloadCache {
	return &reportPayloadCache{
		maxSlots: maxSlots,
		ttl:      ttl,
		byID:     make(map[string]*list.Element),
		lruList:  list.New(),
	}
}

// Upsert inserts or updates a report. Identical payloads (EqualVT) are a no-op and do not refresh updatedAt or recency.
// When a new key is inserted at capacity, the LRU entry is evicted first; that eviction is returned for metrics (residency
// is measured at eviction time relative to the evicted entry's updatedAt).
func (c *reportPayloadCache) Upsert(resourceID string, report *v1.VMReport, now time.Time) []payloadEviction {
	c.mu.Lock()
	defer c.mu.Unlock()

	evictions := c.evictExpiredFromFrontNoLock(now, upsertExpiredEvictionPerInsert)

	if elem, ok := c.byID[resourceID]; ok {
		ent := elem.Value.(*reportPayloadEntry)
		if ent.report.EqualVT(report) {
			return evictions
		}
		ent.report = report.CloneVT()
		ent.updatedAt = now
		c.lruList.MoveToBack(elem)
		return evictions
	}

	if c.maxSlots <= 0 {
		return evictions
	}
	if len(c.byID) >= c.maxSlots {
		if ev, ok := c.evictLRUNoLock(now); ok {
			evictions = append(evictions, ev)
		}
	}

	ent := &reportPayloadEntry{
		resourceID:     resourceID,
		report:         report.CloneVT(),
		updatedAt:      now,
		firstUpdatedAt: now,
	}
	elem := c.lruList.PushBack(ent)
	c.byID[resourceID] = elem
	return evictions
}

// evictLRUNoLock removes the least-recently-updated entry and returns its eviction metadata.
// The caller must hold c.mu.
func (c *reportPayloadCache) evictLRUNoLock(now time.Time) (payloadEviction, bool) {
	front := c.lruList.Front()
	if front == nil {
		return payloadEviction{}, false
	}
	ent := front.Value.(*reportPayloadEntry)
	ev := evictionForEntry(ent, now)
	c.removeElementNoLock(front)
	return ev, true
}

// evictExpiredFromFrontNoLock removes up to budget expired entries from the LRU front.
// A non-positive budget disables this opportunistic sweep. Because the list is ordered by
// updatedAt (oldest first), eviction stops at the first non-expired entry.
// The caller must hold c.mu.
func (c *reportPayloadCache) evictExpiredFromFrontNoLock(now time.Time, budget int) []payloadEviction {
	if budget <= 0 {
		return nil
	}
	out := make([]payloadEviction, 0, min(budget, len(c.byID)))
	for range budget {
		front := c.lruList.Front()
		if front == nil {
			break
		}
		ent := front.Value.(*reportPayloadEntry)
		if now.Before(ent.updatedAt.Add(c.ttl)) {
			break
		}
		out = append(out, evictionForEntry(ent, now))
		c.removeElementNoLock(front)
	}
	return out
}

// removeElementNoLock removes the given list element and its byID index entry.
// The caller must hold c.mu.
func (c *reportPayloadCache) removeElementNoLock(elem *list.Element) {
	ent := elem.Value.(*reportPayloadEntry)
	delete(c.byID, ent.resourceID)
	c.lruList.Remove(elem)
}

// Get returns the cached report reference if present. Get does not enforce TTL and does not
// modify cache contents. Reads do not affect eviction order by updatedAt. Callers must treat
// returned reports as read-only.
func (c *reportPayloadCache) Get(resourceID string, _ time.Time) (report *v1.VMReport, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.byID[resourceID]
	if !exists {
		return nil, false
	}
	ent := elem.Value.(*reportPayloadEntry)
	return ent.report, true
}

// Remove deletes the entry for resourceID if present, regardless of TTL. now is used only to compute
// removal durations. The bool reports whether an entry was removed.
func (c *reportPayloadCache) Remove(resourceID string, now time.Time) (payloadEviction, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.byID[resourceID]
	if !ok {
		return payloadEviction{}, false
	}
	ent := elem.Value.(*reportPayloadEntry)
	ev := evictionForEntry(ent, now)
	c.removeElementNoLock(elem)
	return ev, true
}

// SweepExpired removes every entry whose updatedAt is older than TTL relative to now.
// It returns eviction records in LRU order (oldest updatedAt first among those removed).
func (c *reportPayloadCache) SweepExpired(now time.Time) []payloadEviction {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.evictExpiredFromFrontNoLock(now, len(c.byID))
}

// Len returns the number of entries currently stored.
func (c *reportPayloadCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.byID)
}

// Capacity returns the configured maxSlots (see newReportPayloadCache).
func (c *reportPayloadCache) Capacity() int {
	return c.maxSlots
}

func evictionForEntry(ent *reportPayloadEntry, now time.Time) payloadEviction {
	return payloadEviction{
		resourceID: ent.resourceID,
		residency:  now.Sub(ent.updatedAt),
		lifetime:   now.Sub(ent.firstUpdatedAt),
	}
}
