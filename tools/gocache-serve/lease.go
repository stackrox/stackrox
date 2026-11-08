package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LeaseManager implements advisory compilation leases so concurrent CI
// runners can divide compilation work instead of duplicating it:
//
//	Runner A: miss X → acquire lease-X → Go compiles X → upload X → release
//	Runner B: miss X → lease-X held    → wait & recheck → hit (A delivered)
//
// The lease is advisory, never authoritative: every failure mode (holder
// crash, release failure, steal race, network error) degrades to redundant
// compilation — today's behavior — never to a stall.
//
// Lease blobs are named lease-{actionID} with a JSON body {runner, ts}.
// A lease older than TTL is stale and may be stolen by overwriting.
type LeaseManager struct {
	backend  Backend
	ttl      time.Duration
	runnerID string

	mu   sync.Mutex
	held map[string]bool // actionIDs whose leases we hold
}

func NewLeaseManager(backend Backend, ttl time.Duration) *LeaseManager {
	host, _ := os.Hostname()
	runnerID := fmt.Sprintf("%s-%d-%d", host, os.Getpid(), time.Now().UnixNano())
	return &LeaseManager{
		backend:  backend,
		ttl:      ttl,
		runnerID: runnerID,
		held:     make(map[string]bool),
	}
}

func leaseKey(actionID string) string { return "lease-" + actionID }

// Holds reports whether this process currently holds the lease for actionID.
// Used to route the entry's upload to the priority lane.
func (lm *LeaseManager) Holds(actionID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.held[actionID]
}

type leaseBody struct {
	Runner string `json:"runner"`
	TS     int64  `json:"ts"` // unix millis
}

// Acquire tries to become the compiler for actionID.
// Returns true if we hold the lease (acquired fresh, already held, or
// stolen from a stale holder) — caller returns Miss so Go compiles.
// Returns false if another runner holds a fresh lease — caller should
// wait/recheck for the entry.
func (lm *LeaseManager) Acquire(actionID string) bool {
	lm.mu.Lock()
	if lm.held[actionID] {
		lm.mu.Unlock()
		return true // already ours (duplicate get from Go)
	}
	lm.mu.Unlock()

	body, _ := json.Marshal(leaseBody{Runner: lm.runnerID, TS: time.Now().UnixMilli()})
	ok, err := lm.backend.PutIfNotExist(leaseKey(actionID), body)
	if err != nil {
		// Network error: can't tell who's leader — compile (safe redundancy).
		log.Printf("gocache-serve: lease acquire error for %s: %v (compiling)", shortID(actionID), err)
		return true
	}
	if ok {
		lm.mu.Lock()
		lm.held[actionID] = true
		lm.mu.Unlock()
		return true
	}

	// Someone holds it. Check staleness — if the holder died or is hung,
	// steal the lease after TTL rather than waiting forever.
	if lm.isStale(actionID) {
		body, _ := json.Marshal(leaseBody{Runner: lm.runnerID, TS: time.Now().UnixMilli()})
		if err := lm.backend.Put(leaseKey(actionID), body); err == nil {
			lm.mu.Lock()
			lm.held[actionID] = true
			lm.mu.Unlock()
			log.Printf("gocache-serve: stole stale lease for %s", shortID(actionID))
			return true
		}
		// Steal failed (raced another stealer, or network) — wait instead.
	}
	return false
}

// isStale reads the lease blob and reports whether its timestamp is
// older than the TTL. Conservative: any read error counts as not stale
// (we wait rather than risk stealing a live lease).
func (lm *LeaseManager) isStale(actionID string) bool {
	data, err := lm.backend.Get(leaseKey(actionID))
	if err != nil || data == nil {
		return false
	}
	var lb leaseBody
	if err := json.Unmarshal(data, &lb); err != nil {
		return false
	}
	return time.Since(time.UnixMilli(lb.TS)) > lm.ttl
}

// HoldersStillFresh reports whether another runner's lease for actionID is
// still within TTL. Used by the wait loop to decide when to give up and
// compile anyway (bounded wait beats unbounded stall).
func (lm *LeaseManager) HolderIsFresh(actionID string) bool {
	data, err := lm.backend.Get(leaseKey(actionID))
	if err != nil || data == nil {
		return false
	}
	var lb leaseBody
	if err := json.Unmarshal(data, &lb); err != nil {
		return false
	}
	return time.Since(time.UnixMilli(lb.TS)) <= lm.ttl
}

// Release drops our lease for actionID. Called after the entry upload
// succeeds. Best-effort: a failed delete just costs followers their
// remaining wait, and the lease expires via TTL regardless.
func (lm *LeaseManager) Release(actionID string) {
	lm.mu.Lock()
	ours := lm.held[actionID]
	delete(lm.held, actionID)
	lm.mu.Unlock()
	if !ours {
		return
	}
	if err := lm.backend.DeleteKey(leaseKey(actionID)); err != nil {
		log.Printf("gocache-serve: lease release for %s failed (TTL will expire it): %v",
			shortID(actionID), err)
	}
}
