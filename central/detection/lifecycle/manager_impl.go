package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/activecomponent/updater/aggregator"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/queue"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/lifecycle/metrics"
	"github.com/stackrox/rox/central/detection/runtime"
	centralMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processbaseline"
	baselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/process/filter"
	processBaselinePkg "github.com/stackrox/rox/pkg/processbaseline"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/time/rate"
)

var (
	lifecycleMgrCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert, resources.Deployment, resources.Image,
				resources.DeploymentExtension, resources.WorkflowAdministration, resources.Namespace)))

	genDuration = env.BaselineGenerationDuration.DurationSetting()
)

type processBaselineKey struct {
	deploymentID  string
	containerName string
	clusterID     string
	namespace     string
}

type managerImpl struct {
	reprocessor reprocessor.Loop

	buildTimeDetector  buildtime.Detector
	runtimeDetector    runtime.Detector
	deployTimeDetector deploytime.Detector

	alertManager alertmanager.AlertManager

	deploymentDataStore     deploymentDatastore.DataStore
	processesDataStore      processIndicatorDatastore.DataStore
	baselines               baselineDataStore.DataStore
	deletedDeploymentsCache expiringcache.Cache
	processFilter           filter.Filter

	queuedIndicators           map[string]*storage.ProcessIndicator
	deploymentObservationQueue queue.DeploymentObservationQueue

	indicatorQueueLock   sync.Mutex
	flushProcessingLock  concurrency.TransparentMutex
	indicatorRateLimiter *rate.Limiter
	indicatorFlushTicker *time.Ticker
	baselineFlushTicker  *time.Ticker

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
	deploymentIDs, err := m.deploymentDataStore.GetDeploymentIDs(ctx)
	if err != nil {
		utils.Should(errors.Wrap(err, "error getting deployment IDs"))
		return
	}

	var processesToRemove []string
	walkFn := func() error {
		deploymentIDSet := set.NewStringSet(deploymentIDs...)
		processesToRemove = processesToRemove[:0]
		return m.processesDataStore.WalkAll(ctx, func(pi *storage.ProcessIndicator) error {
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
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
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

func (m *managerImpl) flushBaselineQueuePeriodically() {
	defer m.baselineFlushTicker.Stop()
	for range m.baselineFlushTicker.C {
		m.flushBaselineQueue()
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

func (m *managerImpl) flushBaselineQueue() {
	for {
		// ObservationEnd is in the future so we have nothing to do at this time
		head := m.deploymentObservationQueue.Peek()
		if head == nil || protoutils.After(head.ObservationEnd, types.TimestampNow()) {
			return
		}

		// Grab the first deployment to baseline.
		// NOTE:  This is the only place from which Pull is called.
		deployment := m.deploymentObservationQueue.Pull()

		m.addBaseline(deployment.DeploymentID)
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
	defer centralMetrics.ModifyProcessQueueLength(-len(copiedQueue))

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

	now := time.Now()
	m.processAggregator.Add(indicatorSlice)
	centralMetrics.SetFunctionSegmentDuration(now, "AddProcessToAggregator")

	defer centralMetrics.SetFunctionSegmentDuration(time.Now(), "CheckAndUpdateBaseline")

	m.buildMapAndCheckBaseline(indicatorSlice)
}

func (m *managerImpl) addToIndicatorQueue(indicator *storage.ProcessIndicator) {
	m.indicatorQueueLock.Lock()
	defer m.indicatorQueueLock.Unlock()

	previousSize := len(m.queuedIndicators)
	m.queuedIndicators[indicator.GetId()] = indicator
	if len(m.queuedIndicators) != previousSize {
		centralMetrics.ModifyProcessQueueLength(1)
	}
}

func (m *managerImpl) addBaseline(deploymentID string) {
	defer centralMetrics.SetFunctionSegmentDuration(time.Now(), "AddBaseline")

	// Simply use search to find the process indicators for the deployment
	indicatorSlice, _ := m.processesDataStore.SearchRawProcessIndicators(
		lifecycleMgrCtx,
		search.NewQueryBuilder().
			AddExactMatches(search.DeploymentID, deploymentID).
			ProtoQuery(),
	)

	m.buildMapAndCheckBaseline(indicatorSlice)
}

func (m *managerImpl) buildMapAndCheckBaseline(indicatorSlice []*storage.ProcessIndicator) {
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

	// If the baseline does not exist AND this deployment is in the observation period, we
	// need not process further at this time.
	if !exists && m.deploymentObservationQueue.InObservation(key.GetDeploymentId()) {
		return false, nil
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
		_, err = m.baselines.UpsertProcessBaseline(lifecycleMgrCtx, key, elements, true, true)
		return false, err
	}

	userBaseline := processbaseline.IsUserLocked(baseline)
	roxBaseline := processbaseline.IsRoxLocked(baseline) && hasNonStartupProcess
	if userBaseline || roxBaseline {
		// We already checked if it's in the baseline and it is not, so reprocess risk to mark the results are suspicious if necessary
		m.reprocessor.ReprocessRiskForDeployments(baselineKey.deploymentID)
	} else {
		// So we have a baseline, but not locked.  Now we need to add these elements to the unlocked baseline
		_, err = m.baselines.UpdateProcessBaselineElements(lifecycleMgrCtx, key, elements, nil, true)
	}

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

	observationEnd, _ := types.TimestampProto(time.Now().Add(genDuration))
	m.deploymentObservationQueue.Push(&queue.DeploymentObservation{DeploymentID: indicator.GetDeploymentId(), InObservation: true, ObservationEnd: observationEnd})

	m.addToIndicatorQueue(indicator)

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

	// Split the alerts into unique groups so that we can do targeted lookups of alerts that need to be merged.
	// Based on the current Sensor logic, this should only ever result in a single group as the alert results are
	// multiple policy evaluations against the same audit event which only ever references a single resource type and name.
	type alertKey struct {
		namespace    string
		resourceName string
		resourceType storage.Alert_Resource_ResourceType
	}
	alertGroups := make(map[alertKey][]*storage.Alert)
	for _, alert := range alerts {
		key := alertKey{
			namespace:    alert.GetNamespace(),
			resourceName: alert.GetResource().GetName(),
			resourceType: alert.GetResource().GetResourceType(),
		}
		alertGroups[key] = append(alertGroups[key], alert)
	}
	for key, alerts := range alertGroups {
		opts := []alertmanager.AlertFilterOption{
			alertmanager.WithLifecycleStage(stage),
			// Use cluster id and namespace name to align with sac filters
			alertmanager.WithClusterID(clusterID),
			alertmanager.WithNamespace(key.namespace),
			alertmanager.WithResource(key.resourceName, key.resourceType),
		}
		log.Infof("Got resource alerts")
		for _, a := range alerts {
			log.Infof("[Res Alert] Id %s -  %+v at %+v", a.GetId(), protoutils.NewWrapper(a.GetResource()).String(), protoutils.NewWrapper(a.GetTime()))
		}
		if _, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, alerts, opts...); err != nil {
			return err
		}
	}
	return nil
}

func (m *managerImpl) UpsertPolicy(policy *storage.Policy) error {
	m.policyAlertsLock.Lock()
	defer m.policyAlertsLock.Unlock()
	// Add policy to set.
	if policies.AppliesAtBuildTime(policy) {
		if err := m.buildTimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to build time detector", policy.GetName())
		}
	} else {
		m.buildTimeDetector.PolicySet().RemovePolicy(policy.GetId())
	}

	if policies.AppliesAtDeployTime(policy) {
		if err := m.deployTimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to deploy time detector", policy.GetName())
		}
	} else {
		m.deployTimeDetector.PolicySet().RemovePolicy(policy.GetId())
	}

	if policies.AppliesAtRunTime(policy) {
		if err := m.runtimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to runtime detector", policy.GetName())
		}
		// Perform notifications and update DB.
		modifiedDeployments, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, nil, alertmanager.WithPolicyID(policy.GetId()))
		if err != nil {
			return err
		}
		if modifiedDeployments.Cardinality() > 0 {
			defer m.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
		}

	} else {
		m.runtimeDetector.PolicySet().RemovePolicy(policy.GetId())
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

	m.deploymentObservationQueue.RemoveDeployment(deploymentID)

	return err
}

func (m *managerImpl) RemoveDeploymentFromObservation(deploymentID string) {
	m.deploymentObservationQueue.RemoveFromObservation(deploymentID)
}

func (m *managerImpl) RemovePolicy(policyID string) error {
	m.policyAlertsLock.Lock()
	defer m.policyAlertsLock.Unlock()

	m.buildTimeDetector.PolicySet().RemovePolicy(policyID)

	m.deployTimeDetector.PolicySet().RemovePolicy(policyID)

	numRuntimePolicies := len(m.runtimeDetector.PolicySet().GetCompiledPolicies())
	m.runtimeDetector.PolicySet().RemovePolicy(policyID)
	runtimePolicyRemoved := numRuntimePolicies-len(m.runtimeDetector.PolicySet().GetCompiledPolicies()) > 0

	m.removedOrDisabledPolicies.Add(policyID)

	// Runtime alerts need to be explicitly marked resolved as their updates are not synced from sensors
	if runtimePolicyRemoved {
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
