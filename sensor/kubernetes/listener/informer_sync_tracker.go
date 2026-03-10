package listener

import (
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	sensorMetrics "github.com/stackrox/rox/sensor/common/metrics"
)

// informer names used for sync tracking. Every handle() call must use one of these constants.
const (
	informerNamespaces                    = "Namespaces"
	informerSecrets                       = "Secrets"
	informerServiceAccounts               = "ServiceAccounts"
	informerRoles                         = "Roles"
	informerClusterRoles                  = "ClusterRoles"
	informerClusterOperators              = "ClusterOperators"
	informerImageDigestMirrorSets         = "ImageDigestMirrorSets"
	informerImageTagMirrorSets            = "ImageTagMirrorSets"
	informerImageContentSourcePolicies    = "ImageContentSourcePolicies"
	informerComplianceCheckResults        = "ComplianceCheckResults"
	informerComplianceRules               = "ComplianceRules"
	informerComplianceCustomRules         = "ComplianceCustomRules"
	informerComplianceScanSettingBindings = "ComplianceScanSettingBindings"
	informerComplianceScans               = "ComplianceScans"
	informerComplianceSuites              = "ComplianceSuites"
	informerComplianceRemediations        = "ComplianceRemediations"
	informerComplianceProfiles            = "ComplianceProfiles"
	informerComplianceTailoredProfiles    = "ComplianceTailoredProfiles"
	informerVirtualMachineInstances       = "VirtualMachineInstances"
	informerVirtualMachines               = "VirtualMachines"
	informerRoleBindings                  = "RoleBindings"
	informerClusterRoleBindings           = "ClusterRoleBindings"
	informerPodCache                      = "PodCache"
	informerNetworkPolicies               = "NetworkPolicies"
	informerNodes                         = "Nodes"
	informerServices                      = "Services"
	informerRoutes                        = "Routes"
	informerJobs                          = "Jobs"
	informerReplicaSets                   = "ReplicaSets"
	informerReplicationControllers        = "ReplicationControllers"
	informerDaemonSets                    = "DaemonSets"
	informerDeployments                   = "Deployments"
	informerStatefulSets                  = "StatefulSets"
	informerCronJobs                      = "CronJobs"
	informerDeploymentConfigs             = "DeploymentConfigs"
	informerPods                          = "Pods"
)

// noRegistrationsTimeout is the duration to wait for informers to be registered before logging a warning.
const noRegistrationsTimeout = 60 * time.Second

type syncStatus int

const (
	syncPending syncStatus = iota
	syncComplete
)

type informerState struct {
	status       syncStatus
	registeredAt time.Time
	syncedAt     time.Time
}

// informerSyncTracker monitors individual informer sync progress and periodically logs
// which informers have not yet synced. All methods are nil-safe: if the tracker is nil,
// calls are no-ops. This allows callers to pass a nil tracker when the feature is disabled.
type informerSyncTracker struct {
	mu           sync.Mutex
	informers    map[string]*informerState
	warnInterval time.Duration
	stopC        <-chan struct{}
	wg           sync.WaitGroup
}

// newInformerSyncTracker creates a tracker and starts a background goroutine that
// periodically reports on informers that have not synced.
func newInformerSyncTracker(warnInterval time.Duration, stopC <-chan struct{}) *informerSyncTracker {
	t := &informerSyncTracker{
		informers:    make(map[string]*informerState),
		warnInterval: warnInterval,
		stopC:        stopC,
	}

	t.wg.Add(1)
	go t.run()

	return t
}

// register adds an informer to the tracker as pending. It is called when an informer
// handler is set up, before the informer starts syncing.
func (t *informerSyncTracker) register(name string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.informers[name]; exists {
		log.Warnf("Informer %q registered more than once; duplicate ignored. Tracking whether this informer hangs may be inaccurate.", name)
		return
	}

	t.informers[name] = &informerState{
		status:       syncPending,
		registeredAt: time.Now(),
	}
	sensorMetrics.InformersRegisteredCurrent.Inc()
	sensorMetrics.InformersPendingCurrent.Inc()
}

// markSynced marks the named informer as synced. It is called after
// cache.WaitForCacheSync returns true for that informer.
func (t *informerSyncTracker) markSynced(name string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if inf, ok := t.informers[name]; ok && inf.status == syncPending {
		inf.status = syncComplete
		inf.syncedAt = time.Now()
		sensorMetrics.InformersPendingCurrent.Dec()
		sensorMetrics.ObserveInformerSyncDuration(name, inf.syncedAt.Sub(inf.registeredAt))
	}
}

// pendingInformer holds the name and duration of an informer that has not yet synced.
type pendingInformer struct {
	name    string
	pending time.Duration
}

// syncState holds the current state of informer sync progress.
type syncState struct {
	synced  []string
	pending []pendingInformer
}

// stop stops the background goroutine and waits for it to exit.
func (t *informerSyncTracker) stop() {
	if t == nil {
		return
	}
	t.wg.Wait()
	log.Info("Informer sync tracker stopped")
}

func (t *informerSyncTracker) run() {
	defer t.wg.Done()

	pendingInformerTicker := time.NewTicker(t.warnInterval)
	defer pendingInformerTicker.Stop()

	noRegistrations := time.NewTimer(noRegistrationsTimeout)
	defer noRegistrations.Stop()

	for {
		select {
		case <-t.stopC:
			return
		// If no informers are registered after the timeout, exit the sync tracker, but let Sensor continue operation.
		case <-noRegistrations.C:
			state := t.getState()
			if len(state.synced) == 0 && len(state.pending) == 0 {
				log.Warnf("No informers registered after %s, exiting sync tracker. Sensor continues regular operation.", noRegistrationsTimeout.String())
				return
			}
		case <-pendingInformerTicker.C:
			state := t.getState()
			for _, p := range state.pending {
				sensorMetrics.ObserveInformerSyncDuration(p.name, p.pending)
			}
			state.log()
			if len(state.synced) > 0 && len(state.pending) == 0 {
				return
			}
		}
	}
}

func (s syncState) log() {
	total := len(s.synced) + len(s.pending)
	if total == 0 {
		log.Infof("No informers registered")
		return
	}
	if len(s.pending) == 0 {
		log.Infof("No pending informers. %d informers synced.", total)
		return
	}
	var sb strings.Builder
	for i, p := range s.pending {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.name)
		sb.WriteString(" (")
		sb.WriteString(p.pending.Truncate(time.Second).String())
		sb.WriteByte(')')
	}
	log.Warnf("Informer sync progress: %d/%d synced. Pending: %s",
		len(s.synced), total, sb.String())
}

// getState returns the current sync state: which informers are synced and which are pending.
func (t *informerSyncTracker) getState() syncState {
	if t == nil {
		return syncState{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	var state syncState
	now := time.Now()
	for name, inf := range t.informers {
		switch inf.status {
		case syncComplete:
			state.synced = append(state.synced, name)
		case syncPending:
			state.pending = append(state.pending, pendingInformer{
				name:    name,
				pending: now.Sub(inf.registeredAt),
			})
		}
	}
	return state
}
