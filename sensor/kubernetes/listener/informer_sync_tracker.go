package listener

import (
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	sensorMetrics "github.com/stackrox/rox/sensor/common/metrics"
)

// Informer names used for sync tracking. Every handle() call must use one of these constants.
// Keep this list sorted alphabetically by constant name.
const (
	informerClusterOperators              = "ClusterOperators"
	informerClusterRoleBindings           = "ClusterRoleBindings"
	informerClusterRoles                  = "ClusterRoles"
	informerComplianceCheckResults        = "ComplianceCheckResults"
	informerComplianceCustomRules         = "ComplianceCustomRules"
	informerComplianceProfiles            = "ComplianceProfiles"
	informerComplianceRemediations        = "ComplianceRemediations"
	informerComplianceRules               = "ComplianceRules"
	informerComplianceScanSettingBindings = "ComplianceScanSettingBindings"
	informerComplianceScans               = "ComplianceScans"
	informerComplianceSuites              = "ComplianceSuites"
	informerComplianceTailoredProfiles    = "ComplianceTailoredProfiles"
	informerCronJobs                      = "CronJobs"
	informerDaemonSets                    = "DaemonSets"
	informerDeploymentConfigs             = "DeploymentConfigs"
	informerDeployments                   = "Deployments"
	informerImageContentSourcePolicies    = "ImageContentSourcePolicies"
	informerImageDigestMirrorSets         = "ImageDigestMirrorSets"
	informerImageTagMirrorSets            = "ImageTagMirrorSets"
	informerJobs                          = "Jobs"
	informerNamespaces                    = "Namespaces"
	informerNetworkPolicies               = "NetworkPolicies"
	informerNodes                         = "Nodes"
	informerPodCache                      = "PodCache"
	informerPods                          = "Pods"
	informerReplicaSets                   = "ReplicaSets"
	informerReplicationControllers        = "ReplicationControllers"
	informerRoleBindings                  = "RoleBindings"
	informerRoles                         = "Roles"
	informerRoutes                        = "Routes"
	informerSecrets                       = "Secrets"
	informerServiceAccounts               = "ServiceAccounts"
	informerServices                      = "Services"
	informerStatefulSets                  = "StatefulSets"
	informerVirtualMachineInstances       = "VirtualMachineInstances"
	informerVirtualMachines               = "VirtualMachines"
)

// noRegistrationsTimeout is the duration to wait for any informer to be registered before logging a warning.
// In normal operation, Sensor should have multiple informers registered shortly after connecting to the API server.
// If no informers are registered after 1m, it is likely that Sensor is running without connection to k8s API server.
const noRegistrationsTimeout = 60 * time.Second

type informerState struct {
	registeredAt time.Time
	syncedAt     time.Time
}

func (i *informerState) isPending() bool {
	return i.syncedAt.IsZero()
}

// informerSyncTracker monitors individual informer sync progress and periodically logs
// which informers have not yet synced. All methods are nil-safe: if the tracker is nil,
// calls are no-ops. This allows callers to pass a nil tracker when the feature is disabled.
type informerSyncTracker struct {
	mu           sync.Mutex
	informers    map[string]*informerState
	warnInterval time.Duration
	stopper      concurrency.Stopper
}

// newInformerSyncTracker creates a tracker and starts a background goroutine that
// periodically reports on informers that have not synced.
//
// The tracker uses shared global Prometheus metrics (InformersRegisteredCurrent,
// InformersPendingCurrent, informerSyncDurationMs) which are reset on creation.
// Only one tracker instance should exist at a time. If multiple concurrent
// trackers are ever needed, the metrics must be refactored to be per-instance.
func newInformerSyncTracker(warnInterval time.Duration) *informerSyncTracker {
	// Do not reset in `stop`, as it may be called shortly after sensor startup.
	// Reseting here ensures that the metrics are current for the most recent informer sync
	// (e.g., after offline->online transition).
	sensorMetrics.InformersRegisteredCurrent.Set(0)
	sensorMetrics.InformersPendingCurrent.Set(0)
	sensorMetrics.ResetInformerSyncDuration()

	t := &informerSyncTracker{
		informers:    make(map[string]*informerState),
		warnInterval: warnInterval,
		stopper:      concurrency.NewStopper(),
	}

	go t.run()

	return t
}

// register adds an informer to the tracker as pending.
// The informer is expected to be synced shortly after connection to the API server.
// It is called when an informer handler is set up, before the informer starts syncing.
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

	if inf, ok := t.informers[name]; ok && inf != nil && inf.isPending() {
		inf.syncedAt = time.Now()
		sensorMetrics.InformersPendingCurrent.Dec()
		sensorMetrics.ObserveInformerSyncDuration(name, inf.syncedAt.Sub(inf.registeredAt))
	}
}

// stop requests the background goroutine to shut down and waits for it to exit.
func (t *informerSyncTracker) stop() {
	if t == nil {
		return
	}
	t.stopper.Client().Stop()
	<-t.stopper.Client().Stopped().Done()
	log.Info("Informer sync tracker stopped")
}

func (t *informerSyncTracker) run() {
	defer t.stopper.Flow().ReportStopped()

	pendingInformerTicker := time.NewTicker(t.warnInterval)
	defer pendingInformerTicker.Stop()

	noRegistrations := time.NewTimer(noRegistrationsTimeout)
	defer noRegistrations.Stop()

	for {
		select {
		case <-t.stopper.Flow().StopRequested():
			t.getState().log()
			return
		// If no informers are registered after the timeout, exit the sync tracker, but let Sensor continue operation.
		case <-noRegistrations.C:
			state := t.getState()
			if len(state.synced) == 0 && len(state.pending) == 0 {
				log.Warnf("No informers registered after %s, exiting sync tracker. "+
					"Sensor will continue operation in this state, "+
					"but the data from this secured cluster are not being processed. ", noRegistrationsTimeout.String())
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
		if inf == nil {
			log.Warnf("Informer %q has nil tracker state; skipping from sync progress", name)
			continue
		}
		if inf.isPending() {
			state.pending = append(state.pending, pendingInformer{
				name:    name,
				pending: now.Sub(inf.registeredAt),
			})
			continue
		}
		state.synced = append(state.synced, name)
	}
	return state
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

// log prints the log message for the current state of informer sync progress.
func (s syncState) log() {
	total := len(s.synced) + len(s.pending)
	if total == 0 {
		log.Infof("No informers registered yet")
		return
	}
	if len(s.pending) == 0 {
		log.Infof("All %d informers have synced successfully", total)
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
