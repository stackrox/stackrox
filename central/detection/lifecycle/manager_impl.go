package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/lifecycle/metrics"
	"github.com/stackrox/rox/central/detection/runtime"
	centralMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processbaseline"
	baselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/process/filter"
	processBaselinePkg "github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/time/rate"
)

var (
	lifecycleMgrCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert, resources.Deployment, resources.Image, resources.Indicator, resources.Policy, resources.ProcessWhitelist, resources.Namespace)))
)

type processBaselineKey struct {
	deploymentID  string
	containerName string
	clusterID     string
	namespace     string
}

type managerImpl struct {
	reprocessor        reprocessor.Loop
	runtimeDetector    runtime.Detector
	deploytimeDetector deploytime.Detector
	alertManager       alertmanager.AlertManager

	deploymentDataStore     deploymentDatastore.DataStore
	processesDataStore      processIndicatorDatastore.DataStore
	baselines               baselineDataStore.DataStore
	deletedDeploymentsCache expiringcache.Cache
	processFilter           filter.Filter

	queuedIndicators map[string]*storage.ProcessIndicator

	indicatorQueueLock   sync.Mutex
	flushProcessingLock  concurrency.TransparentMutex
	indicatorRateLimiter *rate.Limiter
	indicatorFlushTicker *time.Ticker

	policyAlertsLock          sync.RWMutex
	removedOrDisabledPolicies set.StringSet

	processAggregator aggregator.ProcessAggregator
}

func (m *managerImpl) copyAndResetIndicatorQueue() map[string]*storage.ProcessIndicator {
	m.indicatorQueueLock.Lock()
	defer m.indicatorQueueLock.Unlock()
	if len(m.queuedIndicators) == 0 {
		return nil
	}
	copiedMap := m.queuedIndicators
	m.queuedIndicators = make(map[string]*storage.ProcessIndicator)

	return copiedMap
}

func (m *managerImpl) buildIndicatorFilter() {
	ctx := sac.WithAllAccess(context.Background())
	var processesToRemove []string

	deploymentIDs, err := m.deploymentDataStore.GetDeploymentIDs(ctx)
	if err != nil {
		utils.Should(errors.Wrap(err, "error getting deployment IDs"))
		return
	}

	deploymentIDSet := set.NewStringSet(deploymentIDs...)

	err = m.processesDataStore.WalkAll(ctx, func(pi *storage.ProcessIndicator) error {
		if !deploymentIDSet.Contains(pi.GetDeploymentId()) {
			// Don't remove as these processes will be removed by GC
			// but don't add to the filter
			return nil
		}
		if !m.processFilter.Add(pi) {
			processesToRemove = append(processesToRemove, pi.GetId())
		}
		return nil
	})
	if err != nil {
		utils.Should(errors.Wrap(err, "error building indicator filter"))
	}

	log.Infof("Cleaning up %d processes as a part of building process filter", len(processesToRemove))
	if err := m.processesDataStore.RemoveProcessIndicators(ctx, processesToRemove); err != nil {
		utils.Should(errors.Wrap(err, "error removing process indicators"))
	}
	log.Infof("Successfully cleaned up those %d processes", len(processesToRemove))
}

func (m *managerImpl) flushQueuePeriodically() {
	defer m.indicatorFlushTicker.Stop()
	for range m.indicatorFlushTicker.C {
		m.flushIndicatorQueue()
	}
}

func indicatorToBaselineKey(indicator *storage.ProcessIndicator) processBaselineKey {
	return processBaselineKey{
		deploymentID:  indicator.GetDeploymentId(),
		containerName: indicator.GetContainerName(),
		clusterID:     indicator.GetClusterId(),
		namespace:     indicator.GetNamespace(),
	}
}

func (m *managerImpl) flushIndicatorQueue() {
	// This is a potentially long-running operation, and we don't want to have a pile of goroutines queueing up on
	// this lock.
	if !m.flushProcessingLock.MaybeLock() {
		return
	}
	defer m.flushProcessingLock.Unlock()

	copiedQueue := m.copyAndResetIndicatorQueue()
	if len(copiedQueue) == 0 {
		return
	}

	defer centralMetrics.SetFunctionSegmentDuration(time.Now(), "FlushingIndicatorQueue")

	// Map copiedQueue to slice
	indicatorSlice := make([]*storage.ProcessIndicator, 0, len(copiedQueue))
	for _, indicator := range copiedQueue {
		if deleted, _ := m.deletedDeploymentsCache.Get(indicator.GetDeploymentId()).(bool); deleted {
			continue
		}
		indicatorSlice = append(indicatorSlice, indicator)
	}

	// Index the process indicators in batch
	if err := m.processesDataStore.AddProcessIndicators(lifecycleMgrCtx, indicatorSlice...); err != nil {
		log.Errorf("Error adding process indicators: %v", err)
	}

	if features.ActiveVulnManagement.Enabled() {
		m.processAggregator.Add(indicatorSlice)
	}

	defer centralMetrics.SetFunctionSegmentDuration(time.Now(), "CheckAndUpdateBaseline")

	// Group the processes into particular baseline segments
	baselineMap := make(map[processBaselineKey][]*storage.ProcessIndicator)
	for _, indicator := range indicatorSlice {
		key := indicatorToBaselineKey(indicator)
		baselineMap[key] = append(baselineMap[key], indicator)
	}

	for key, indicators := range baselineMap {
		if _, err := m.checkAndUpdateBaseline(key, indicators); err != nil {
			log.Errorf("error checking and updating baseline for %+v: %v", key, err)
		}
	}
}

func (m *managerImpl) addToQueue(indicator *storage.ProcessIndicator) {
	m.indicatorQueueLock.Lock()
	defer m.indicatorQueueLock.Unlock()

	m.queuedIndicators[indicator.GetId()] = indicator
}

func (m *managerImpl) checkAndUpdateBaseline(baselineKey processBaselineKey, indicators []*storage.ProcessIndicator) (bool, error) {
	key := &storage.ProcessBaselineKey{
		DeploymentId:  baselineKey.deploymentID,
		ContainerName: baselineKey.containerName,
		ClusterId:     baselineKey.clusterID,
		Namespace:     baselineKey.namespace,
	}

	// TODO joseph what to do if exclusions ("baseline" in the old non-inclusive language) doesn't exist?  Always create for now?
	baseline, exists, err := m.baselines.GetProcessBaseline(lifecycleMgrCtx, key)
	if err != nil {
		return false, err
	}

	existingProcess := set.NewStringSet()
	for _, element := range baseline.GetElements() {
		existingProcess.Add(element.GetElement().GetProcessName())
	}

	var elements []*storage.BaselineItem
	var hasNonStartupProcess bool
	for _, indicator := range indicators {
		if !processbaseline.IsStartupProcess(indicator) {
			hasNonStartupProcess = true
		}
		baselineItem := processBaselinePkg.BaselineItemFromProcess(indicator)
		if !existingProcess.Add(baselineItem) {
			continue
		}
		insertableElement := &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: baselineItem}}
		elements = append(elements, insertableElement)
	}
	if len(elements) == 0 {
		return false, nil
	}
	if !exists {
		_, err = m.baselines.UpsertProcessBaseline(lifecycleMgrCtx, key, elements, true)
		return false, err
	}

	userBaseline := processbaseline.IsUserLocked(baseline)
	roxBaseline := processbaseline.IsRoxLocked(baseline) && hasNonStartupProcess
	if userBaseline || roxBaseline {
		// We already checked if it's in the baseline and it is not, so reprocess risk to mark the results are suspicious if necessary
		m.reprocessor.ReprocessRiskForDeployments(baselineKey.deploymentID)
		return userBaseline, nil
	}
	_, err = m.baselines.UpdateProcessBaselineElements(lifecycleMgrCtx, key, elements, nil, true)
	return userBaseline, err
}

func (m *managerImpl) IndicatorAdded(indicator *storage.ProcessIndicator) error {
	if indicator.GetId() == "" {
		return fmt.Errorf("invalid indicator received: %s, id was empty", proto.MarshalTextString(indicator))
	}

	// Evaluate filter before even adding to the queue
	if !m.processFilter.Add(indicator) {
		metrics.ProcessFilterCounterInc("NotAdded")
		return nil
	}
	metrics.ProcessFilterCounterInc("Added")
	m.addToQueue(indicator)

	if m.indicatorRateLimiter.Allow() {
		go m.flushIndicatorQueue()
	}
	return nil
}

func (m *managerImpl) filterOutDisabledPolicies(alerts *[]*storage.Alert) {
	if alerts == nil {
		return
	}
	filteredAlerts := (*alerts)[:0]

	m.policyAlertsLock.RLock()
	defer m.policyAlertsLock.RUnlock()
	for _, a := range *alerts {
		if m.removedOrDisabledPolicies.Contains(a.GetPolicy().GetId()) {
			continue
		}
		filteredAlerts = append(filteredAlerts, a)
	}
	*alerts = filteredAlerts
}

// HandleDeploymentAlerts handles the lifecycle of the provided alerts (including alerting, merging, etc) all of which belong to the specified deployment
func (m *managerImpl) HandleDeploymentAlerts(deploymentID string, alerts []*storage.Alert, stage storage.LifecycleStage) error {
	defer m.reprocessor.ReprocessRiskForDeployments(deploymentID)

	m.filterOutDisabledPolicies(&alerts)
	if len(alerts) == 0 && stage == storage.LifecycleStage_RUNTIME {
		return nil
	}
	if _, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, alerts,
		alertmanager.WithLifecycleStage(stage), alertmanager.WithDeploymentID(deploymentID, false)); err != nil {
		return err
	}

	return nil
}

// HandleResourceAlerts handles the lifecycle of the provided alerts (including alerting, merging, etc) all of which belong to the specified resource
func (m *managerImpl) HandleResourceAlerts(clusterID string, alerts []*storage.Alert, stage storage.LifecycleStage) error {
	m.filterOutDisabledPolicies(&alerts)
	if len(alerts) == 0 && stage == storage.LifecycleStage_RUNTIME {
		return nil
	}
	// These alerts are all for a single cluster but may belong to any number of namespaces or resource types (except deployment)
	// Ideally search filters should be for lifecycle stage && (namespace1 || namespace2...) && resource_type!=DEPLOYMENT
	// But with these filters they are all ANDs
	// Therefore for now, we will have to pull all non-deployment alerts for this lifecycle stage within specified cluster.
	if _, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, alerts,
		alertmanager.WithLifecycleStage(stage), alertmanager.WithClusterID(clusterID), alertmanager.WithoutResourceType(storage.ListAlert_DEPLOYMENT)); err != nil {
		return err
	}

	return nil
}

func (m *managerImpl) UpsertPolicy(policy *storage.Policy) error {
	m.policyAlertsLock.Lock()
	defer m.policyAlertsLock.Unlock()
	// Add policy to set.
	if policies.AppliesAtDeployTime(policy) {
		if err := m.deploytimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to deploy time detector", policy.GetName())
		}
	} else {
		m.deploytimeDetector.PolicySet().RemovePolicy(policy.GetId())
	}

	if policies.AppliesAtRunTime(policy) {
		if err := m.runtimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to runtime detector", policy.GetName())
		}
	} else {
		m.runtimeDetector.PolicySet().RemovePolicy(policy.GetId())
	}

	if policies.AppliesAtRunTime(policy) {
		// Perform notifications and update DB.
		modifiedDeployments, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, nil, alertmanager.WithPolicyID(policy.GetId()))
		if err != nil {
			return err
		}
		if modifiedDeployments.Cardinality() > 0 {
			defer m.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
		}
	}
	if policy.GetDisabled() {
		m.removedOrDisabledPolicies.Add(policy.GetId())
	} else {
		m.removedOrDisabledPolicies.Remove(policy.GetId())
	}
	return nil
}

func (m *managerImpl) DeploymentRemoved(deploymentID string) error {
	_, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, nil, alertmanager.WithDeploymentID(deploymentID, true))
	return err
}

func (m *managerImpl) RemovePolicy(policyID string) error {
	m.policyAlertsLock.Lock()
	defer m.policyAlertsLock.Unlock()

	m.deploytimeDetector.PolicySet().RemovePolicy(policyID)

	numRuntimeAlerts := len(m.runtimeDetector.PolicySet().GetCompiledPolicies())
	m.runtimeDetector.PolicySet().RemovePolicy(policyID)
	runtimeAlertRemoved := numRuntimeAlerts-len(m.runtimeDetector.PolicySet().GetCompiledPolicies()) > 0

	m.removedOrDisabledPolicies.Add(policyID)

	// Runtime alerts need to be explicitly removed as their updates are not synced from sensors
	if runtimeAlertRemoved {
		modifiedDeployments, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, nil, alertmanager.WithPolicyID(policyID))
		if err != nil {
			return err
		}
		if modifiedDeployments.Cardinality() > 0 {
			m.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
		}
	}
	return nil
}
