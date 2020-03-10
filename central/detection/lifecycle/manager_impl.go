package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/alertmanager"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/detection/lifecycle/metrics"
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processwhitelist"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/process/filter"
	processWhitelistPkg "github.com/stackrox/rox/pkg/processwhitelist"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/time/rate"
)

var (
	lifecycleMgrCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert, resources.Deployment, resources.Image, resources.Indicator, resources.Policy, resources.ProcessWhitelist)))
)

type processWhitelistKey struct {
	deploymentID  string
	containerName string
	clusterID     string
	namespace     string
}

type managerImpl struct {
	reprocessor        reprocessor.Loop
	enricher           enrichment.Enricher
	riskManager        riskManager.Manager
	runtimeDetector    runtime.Detector
	deploytimeDetector deploytime.Detector
	alertManager       alertmanager.AlertManager

	deploymentDataStore     deploymentDatastore.DataStore
	processesDataStore      processIndicatorDatastore.DataStore
	whitelists              whitelistDataStore.DataStore
	imageDataStore          imageDatastore.DataStore
	deletedDeploymentsCache expiringcache.Cache
	processFilter           filter.Filter

	queuedIndicators map[string]*storage.ProcessIndicator

	indicatorQueueLock   sync.Mutex
	flushProcessingLock  concurrency.TransparentMutex
	indicatorRateLimiter *rate.Limiter
	indicatorFlushTicker *time.Ticker

	policyAlertsLock *concurrency.KeyedMutex
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

	deploymentIDs, err := m.deploymentDataStore.GetDeploymentIDs()
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

func indicatorToWhitelistKey(indicator *storage.ProcessIndicator) processWhitelistKey {
	return processWhitelistKey{
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

	// Map copiedQueue to slice
	indicatorSlice := make([]*storage.ProcessIndicator, 0, len(copiedQueue))
	for id, indicator := range copiedQueue {
		if deleted, _ := m.deletedDeploymentsCache.Get(indicator.GetDeploymentId()).(bool); deleted {
			delete(copiedQueue, id)
			continue
		}
		indicatorSlice = append(indicatorSlice, indicator)
	}

	// Index the process indicators in batch
	if err := m.processesDataStore.AddProcessIndicators(lifecycleMgrCtx, indicatorSlice...); err != nil {
		log.Errorf("Error adding process indicators: %v", err)
	}

	// Group the processes into particular whitelist segments
	whitelistMap := make(map[processWhitelistKey][]*storage.ProcessIndicator)
	for _, indicator := range indicatorSlice {
		key := indicatorToWhitelistKey(indicator)
		whitelistMap[key] = append(whitelistMap[key], indicator)
	}

	for key, indicators := range whitelistMap {
		if _, err := m.checkAndUpdateWhitelist(key, indicators); err != nil {
			log.Errorf("error checking and updating whitelist for %+v: %v", key, err)
		}
	}
}

func (m *managerImpl) addToQueue(indicator *storage.ProcessIndicator) {
	m.indicatorQueueLock.Lock()
	defer m.indicatorQueueLock.Unlock()

	m.queuedIndicators[indicator.GetId()] = indicator
}

func (m *managerImpl) checkAndUpdateWhitelist(whitelistKey processWhitelistKey, indicators []*storage.ProcessIndicator) (bool, error) {
	key := &storage.ProcessWhitelistKey{
		DeploymentId:  whitelistKey.deploymentID,
		ContainerName: whitelistKey.containerName,
		ClusterId:     whitelistKey.clusterID,
		Namespace:     whitelistKey.namespace,
	}

	// TODO joseph what to do if whitelist doesn't exist?  Always create for now?
	whitelist, exists, err := m.whitelists.GetProcessWhitelist(lifecycleMgrCtx, key)
	if err != nil {
		return false, err
	}

	existingProcess := set.NewStringSet()
	for _, element := range whitelist.GetElements() {
		existingProcess.Add(element.GetElement().GetProcessName())
	}

	var elements []*storage.WhitelistItem
	var hasNonStartupProcess bool
	for _, indicator := range indicators {
		if !processwhitelist.IsStartupProcess(indicator) {
			hasNonStartupProcess = true
		}
		whitelistItem := processWhitelistPkg.WhitelistItemFromProcess(indicator)
		if !existingProcess.Add(whitelistItem) {
			continue
		}
		insertableElement := &storage.WhitelistItem{Item: &storage.WhitelistItem_ProcessName{ProcessName: whitelistItem}}
		elements = append(elements, insertableElement)
	}
	if len(elements) == 0 {
		return false, nil
	}
	if !exists {
		_, err = m.whitelists.UpsertProcessWhitelist(lifecycleMgrCtx, key, elements, true)
		return false, err
	}

	userWhitelist := processwhitelist.IsUserLocked(whitelist)
	roxWhitelist := processwhitelist.IsRoxLocked(whitelist) && hasNonStartupProcess
	if userWhitelist || roxWhitelist {
		// We already checked if it's in the whitelist and it is not, so reprocess risk to mark the results are suspicious if necessary
		m.reprocessor.ReprocessRiskForDeployments(whitelistKey.deploymentID)
		return userWhitelist, nil
	}
	_, err = m.whitelists.UpdateProcessWhitelistElements(lifecycleMgrCtx, key, elements, nil, true)
	return userWhitelist, err
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

func (m *managerImpl) HandleAlerts(deploymentID string, alerts []*storage.Alert, stage storage.LifecycleStage) error {
	defer m.reprocessor.ReprocessRiskForDeployments(deploymentID)

	if _, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, alerts,
		alertmanager.WithLifecycleStage(stage), alertmanager.WithDeploymentIDs(deploymentID)); err != nil {
		return err
	}

	return nil
}

func (m *managerImpl) UpsertPolicy(policy *storage.Policy) error {
	var presentAlerts []*storage.Alert

	m.policyAlertsLock.Lock(policy.GetId())
	defer m.policyAlertsLock.Unlock(policy.GetId())
	// Add policy to set.
	if policies.AppliesAtDeployTime(policy) {
		if err := m.deploytimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to deploy time detector", policy.GetName())
		}
		deployTimeAlerts, err := m.deploytimeDetector.AlertsForPolicy(policy.GetId())
		if err != nil {
			return errors.Wrapf(err, "error generating deploy-time alerts for policy %s", policy.GetName())
		}
		presentAlerts = append(presentAlerts, deployTimeAlerts...)
	} else {
		err := m.deploytimeDetector.PolicySet().RemovePolicy(policy.GetId())
		if err != nil {
			return errors.Wrapf(err, "removing policy %s from deploy time detector", policy.GetName())
		}
	}

	if policies.AppliesAtRunTime(policy) {
		if err := m.runtimeDetector.PolicySet().UpsertPolicy(policy); err != nil {
			return errors.Wrapf(err, "adding policy %s to runtime detector", policy.GetName())
		}
		runTimeAlerts, err := m.runtimeDetector.AlertsForPolicy(policy.GetId())
		if err != nil {
			return errors.Wrapf(err, "error generating runtime alerts for policy %s", policy.GetName())
		}
		presentAlerts = append(presentAlerts, runTimeAlerts...)
	} else {
		err := m.runtimeDetector.PolicySet().RemovePolicy(policy.GetId())
		if err != nil {
			return errors.Wrapf(err, "removing policy %s from runtime detector", policy.GetName())
		}
	}

	// Perform notifications and update DB.
	modifiedDeployments, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, presentAlerts, alertmanager.WithPolicyID(policy.GetId()))
	if err != nil {
		return err
	}
	if modifiedDeployments.Cardinality() > 0 {
		defer m.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
	}
	return nil
}

func (m *managerImpl) DeploymentRemoved(deployment *storage.Deployment) error {
	_, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, nil, alertmanager.WithDeploymentIDs(deployment.GetId()))
	return err
}

func (m *managerImpl) RemovePolicy(policyID string) error {
	m.policyAlertsLock.Lock(policyID)
	defer m.policyAlertsLock.Unlock(policyID)
	if err := m.deploytimeDetector.PolicySet().RemovePolicy(policyID); err != nil {
		return err
	}
	if err := m.runtimeDetector.PolicySet().RemovePolicy(policyID); err != nil {
		return err
	}
	modifiedDeployments, err := m.alertManager.AlertAndNotify(lifecycleMgrCtx, nil, alertmanager.WithPolicyID(policyID))
	if err != nil {
		return err
	}
	if modifiedDeployments.Cardinality() > 0 {
		m.reprocessor.ReprocessRiskForDeployments(modifiedDeployments.AsSlice()...)
	}
	return nil
}
