package interactivesessions

import (
	"sync"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// defaultTTL is the fallback retention for interactive sessions whose
	// close event is never observed, e.g. when a streaming session ends
	// abruptly. Streaming exec/attach sessions have no lease renewal, and
	// real-world sessions (oc debug) routinely last far longer than a few
	// minutes, so the fallback is generous. In the majority of cases a close
	// event (ResponseComplete/Panic) destroys the session promptly via the
	// closed-session grace period below, and the TTL never comes into play.
	defaultTTL = 6 * time.Hour
	// closedSessionGracePeriod is how long a closed session stays matchable
	// after its closing event (e.g. ResponseComplete) is observed. It covers
	// file events that are still in the pipeline when the session closes.
	closedSessionGracePeriod = 30 * time.Second
	maxSessions              = 1024
	responseStarted          = "ResponseStarted"
)

type key struct {
	namespace string
	pod       string
	container string
}

type session struct {
	event     *storage.KubernetesEvent
	expiresAt time.Time
}

// Tracker retains interactive exec and attach sessions for attributing
// container file activity to the Kubernetes user behind the session. At
// most one session is retained per container: a newly opened session
// replaces any session already tracked for the same container (newest
// wins). A session whose close event (ResponseComplete/Panic) is observed
// is destroyed after a short grace period; sessions whose close event is
// never seen fall back to the TTL (6 hours by default).
type Tracker struct {
	mutex       sync.Mutex
	sessions    map[key]map[string]session
	ttl         time.Duration
	grace       time.Duration
	maxSessions int
	now         func() time.Time
}

func New() *Tracker {
	return newTracker(defaultTTL, closedSessionGracePeriod, maxSessions, time.Now)
}

func newTracker(ttl, grace time.Duration, maxSessions int, now func() time.Time) *Tracker {
	return &Tracker{
		sessions:    make(map[key]map[string]session),
		ttl:         ttl,
		grace:       grace,
		maxSessions: maxSessions,
		now:         now,
	}
}

func (t *Tracker) Observe(events *sensor.AuditEvents) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.pruneExpiredLocked()
	for _, event := range events.GetEvents() {
		if !isInteractive(event) {
			continue
		}
		key := eventKey(event)
		if key.container == "" {
			continue
		}
		if event.GetAuditStage() == responseStarted {
			t.addLocked(key, event)
			continue
		}
		t.closeLocked(key, event.GetId())
	}
}

// Match returns the session retained for the container, if any. With
// newest-session-wins replacement at most one session is retained per
// container; requiring exactly one is a defensive invariant. A session
// closed within its grace period counts just like an active one.
func (t *Tracker) Match(namespace, pod, container string) (*storage.KubernetesEvent, bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.pruneExpiredLocked()
	sessions := t.sessions[key{namespace: namespace, pod: pod, container: container}]
	if len(sessions) != 1 {
		return nil, false
	}
	for _, session := range sessions {
		return session.event, true
	}
	return nil, false
}

// addLocked tracks the session opened by a ResponseStarted event. When a
// new session opens on a container that already has a session, the existing
// session is dropped and the newest session wins. Retaining multiple
// concurrent sessions per container and disambiguating attribution between
// them is a potential future enhancement.
func (t *Tracker) addLocked(key key, event *storage.KubernetesEvent) {
	if len(t.sessions[key]) == 0 && t.sessionCountLocked() >= t.maxSessions {
		// The cap is only enforced for sessions on new keys; replacing a
		// session on an existing key does not grow the count.
		t.evictOldestLocked()
	}
	// Replacing the inner map drops any session previously tracked for the
	// key: a duplicate ResponseStarted for the same audit ID refreshes the
	// session, and a different audit ID replaces it.
	t.sessions[key] = map[string]session{
		event.GetId(): {event: event, expiresAt: t.now().Add(t.ttl)},
	}
}

// evictOldestLocked deletes the session with the smallest expiresAt across
// all keys; it is a no-op when no sessions are tracked. Evicting the
// earliest-expiring entry preferentially removes closed/grace-expiring
// sessions and keeps a new session from ever being blocked: with a 6 hour
// fallback TTL, the previous policy of dropping new sessions at the cap
// could silently block attribution for hours on exec-heavy clusters.
func (t *Tracker) evictOldestLocked() {
	var oldestKey key
	var oldestID string
	var oldestExpiry time.Time
	found := false
	for key, sessions := range t.sessions {
		for auditID, session := range sessions {
			if !found || session.expiresAt.Before(oldestExpiry) {
				found = true
				oldestKey, oldestID, oldestExpiry = key, auditID, session.expiresAt
			}
		}
	}
	if !found {
		return
	}
	delete(t.sessions[oldestKey], oldestID)
	if len(t.sessions[oldestKey]) == 0 {
		delete(t.sessions, oldestKey)
	}
}

// closeLocked retains a closed session for the grace period instead of the
// full TTL, so file events still in the pipeline keep their attribution.
func (t *Tracker) closeLocked(key key, auditID string) {
	session, ok := t.sessions[key][auditID]
	if !ok {
		return
	}
	session.expiresAt = t.now().Add(t.grace)
	t.sessions[key][auditID] = session
}

func (t *Tracker) sessionCountLocked() int {
	count := 0
	for _, sessions := range t.sessions {
		count += len(sessions)
	}
	return count
}

func (t *Tracker) pruneExpiredLocked() {
	now := t.now()
	for key, sessions := range t.sessions {
		for auditID, session := range sessions {
			if !session.expiresAt.After(now) {
				delete(sessions, auditID)
			}
		}
		if len(sessions) == 0 {
			delete(t.sessions, key)
		}
	}
}

func isInteractive(event *storage.KubernetesEvent) bool {
	return event.GetObject().GetResource() == storage.KubernetesEvent_Object_PODS_EXEC ||
		event.GetObject().GetResource() == storage.KubernetesEvent_Object_PODS_ATTACH
}

func eventKey(event *storage.KubernetesEvent) key {
	container := ""
	if args := event.GetPodExecArgs(); args != nil {
		container = args.GetContainer()
	}
	if args := event.GetPodAttachArgs(); args != nil {
		container = args.GetContainer()
	}
	return key{namespace: event.GetObject().GetNamespace(), pod: event.GetObject().GetName(), container: container}
}
