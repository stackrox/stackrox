package interactivesessions

import (
	"sync"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
)

const (
	defaultTTL      = 5 * time.Minute
	maxSessions     = 1024
	responseStarted = "ResponseStarted"
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

// Tracker retains active exec and attach sessions that can be unambiguously
// associated with a container file activity.
type Tracker struct {
	mutex    sync.Mutex
	sessions map[key]map[string]session
	ttl      time.Duration
	now      func() time.Time
}

func New() *Tracker {
	return newTracker(defaultTTL, time.Now)
}

func newTracker(ttl time.Duration, now func() time.Time) *Tracker {
	return &Tracker{sessions: make(map[key]map[string]session), ttl: ttl, now: now}
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
		if sessions := t.sessions[key]; sessions != nil {
			delete(sessions, event.GetId())
			if len(sessions) == 0 {
				delete(t.sessions, key)
			}
		}
	}
}

// Match returns an active session only when exactly one session owns the container.
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

func (t *Tracker) addLocked(key key, event *storage.KubernetesEvent) {
	if t.sessions[key][event.GetId()].event == nil && t.sessionCountLocked() >= maxSessions {
		return
	}
	if t.sessions[key] == nil {
		t.sessions[key] = make(map[string]session)
	}
	t.sessions[key][event.GetId()] = session{event: event, expiresAt: t.now().Add(t.ttl)}
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
