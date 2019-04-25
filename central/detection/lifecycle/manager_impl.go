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
	"github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/time/rate"
)

type indicatorWithInjector struct {
	indicator           *storage.ProcessIndicator
	msgToSensorInjector common.MessageInjector
}

type managerImpl struct {
	reprocessor        reprocessor.Loop
	enricher           enrichment.Enricher
	riskManager        riskManager.Manager
	runtimeDetector    runtime.Detector
	deploytimeDetector deploytime.Detector
	alertManager       alertmanager.AlertManager

	deploymentDataStore deploymentDatastore.DataStore
	processesDataStore  processIndicatorDatastore.DataStore
	imageDataStore      imageDatastore.DataStore

	queuedIndicators map[string]indicatorWithInjector

	queueLock           sync.Mutex
	flushProcessingLock concurrency.TransparentMutex

	limiter *rate.Limiter
	ticker  *time.Ticker
}

func (m *managerImpl) copyAndResetIndicatorQueue() map[string]indicatorWithInjector {
	m.queueLock.Lock()
	defer m.queueLock.Unlock()
	if len(m.queuedIndicators) == 0 {
		return nil
	}
	copiedMap := m.queuedIndicators
	m.queuedIndicators = make(map[string]indicatorWithInjector)

	return copiedMap
}

func (m *managerImpl) flushQueuePeriodically() {
	defer m.ticker.Stop()
	for range m.ticker.C {
		m.flushIndicatorQueue()
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
	for _, i := range copiedQueue {
		indicatorSlice = append(indicatorSlice, i.indicator)
	}

	// Index the process indicators in batch
	if err := m.processesDataStore.AddProcessIndicators(indicatorSlice...); err != nil {
		log.Errorf("Error adding process indicators: %v", err)
	}

	deploymentIDs := uniqueDeploymentIDs(copiedQueue)
	newAlerts, err := m.runtimeDetector.AlertsForDeployments(deploymentIDs...)
	if err != nil {
		log.Errorf("Failed to compute runtime alerts: %s", err)
		return
	}

	modified, err := m.alertManager.AlertAndNotify(newAlerts, alertmanager.WithLifecycleStage(storage.LifecycleStage_RUNTIME), alertmanager.WithDeploymentIDs(deploymentIDs...))
	if err != nil {
		log.Errorf("Couldn't alert and notify: %s", err)
	} else if modified {
		defer m.reprocessor.ReprocessRiskForDeployments(deploymentIDs...)
	}

	containersSet := containersToKill(newAlerts, copiedQueue)
	for _, indicatorInfo := range containersSet {
		info := indicatorInfo.indicator
		deployment, exists, err := m.deploymentDataStore.GetDeployment(info.GetDeploymentId())
		if err != nil {
			log.Errorf("Couldn't enforce on deployment %s: failed to retrieve: %s", info.GetDeploymentId(), err)
			continue
		}
		if !exists {
			log.Errorf("Couldn't enforce on deployment %s: not found in store", info.GetDeploymentId())
			continue
		}
		enforcementAction := createEnforcementAction(deployment, info.GetSignal().GetContainerId())
		if enforcementAction == nil {
			log.Errorf("Couldn't enforce on container %s, not found in deployment %s/%s", info.GetSignal().GetContainerId(),
				deployment.GetNamespace(), deployment.GetName())
			continue
		}
		err = indicatorInfo.msgToSensorInjector.InjectMessage(context.Background(), &central.MsgToSensor{
			Msg: &central.MsgToSensor_Enforcement{
				Enforcement: enforcementAction,
			},
		})
		if err != nil {
			log.Errorf("Failed to inject enforcement action %s: %v", proto.MarshalTextString(enforcementAction), err)
		}
	}
}

func (m *managerImpl) addToQueue(indicator *storage.ProcessIndicator, injector common.MessageInjector) {
	m.queueLock.Lock()
	defer m.queueLock.Unlock()

	m.queuedIndicators[indicator.GetId()] = indicatorWithInjector{
		indicator:           indicator,
		msgToSensorInjector: injector,
	}
}

func (m *managerImpl) IndicatorAdded(indicator *storage.ProcessIndicator, injector common.MessageInjector) error {
	if indicator.GetId() == "" {
		return fmt.Errorf("invalid indicator received: %s, id was empty", proto.MarshalTextString(indicator))
	}

	m.addToQueue(indicator, injector)

	if m.limiter.Allow() {
		go m.flushIndicatorQueue()
	}
	return nil
}

func (m *managerImpl) DeploymentUpdated(deployment *storage.Deployment) (string, storage.EnforcementAction, error) {
	// Attempt to enrich the image before detection.
	updatedImages, updated, err := m.enricher.EnrichDeployment(enricher.EnrichmentContext{NoExternalMetadata: false}, deployment)
	if err != nil {
		log.Errorf("Error enriching deployment %s: %s", deployment.GetName(), err)
	}
	if updated {
		for _, i := range updatedImages {
			if err := m.imageDataStore.UpsertImage(i); err != nil {
				log.Errorf("Error persisting image %s: %s", i.GetName().GetFullName(), err)
			}
		}
		if err := m.deploymentDataStore.UpdateDeployment(deployment); err != nil {
			log.Errorf("Error persisting deployment %s: %s", deployment.GetName(), err)
		}
	}

	// Update risk after processing.
	defer m.riskManager.ReprocessDeploymentRisk(deployment)

	presentAlerts, err := m.deploytimeDetector.AlertsForDeployment(deployment)
	if err != nil {
		return "", storage.EnforcementAction_UNSET_ENFORCEMENT, errors.Wrap(err, "fetching deploy time alerts")
	}

	if _, err := m.alertManager.AlertAndNotify(presentAlerts,
		alertmanager.WithLifecycleStage(storage.LifecycleStage_DEPLOY), alertmanager.WithDeploymentIDs(deployment.GetId())); err != nil {
		return "", storage.EnforcementAction_UNSET_ENFORCEMENT, err
	}

	// Generate enforcement actions based on the currently generated alerts.
	alertToEnforce, enforcementAction := determineEnforcement(presentAlerts)
	return alertToEnforce, enforcementAction, nil
}

func (m *managerImpl) UpsertPolicy(policy *storage.Policy) error {
	// Asynchronously update all deployments' risk after processing.
	defer m.reprocessor.ReprocessRisk()

	var presentAlerts []*storage.Alert

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
	_, err := m.alertManager.AlertAndNotify(presentAlerts, alertmanager.WithPolicyID(policy.GetId()))
	return err
}

func (m *managerImpl) DeploymentRemoved(deployment *storage.Deployment) error {
	_, err := m.alertManager.AlertAndNotify(nil, alertmanager.WithDeploymentIDs(deployment.GetId()))
	return err
}

func (m *managerImpl) RemovePolicy(policyID string) error {
	if err := m.deploytimeDetector.PolicySet().RemovePolicy(policyID); err != nil {
		return err
	}
	if err := m.runtimeDetector.PolicySet().RemovePolicy(policyID); err != nil {
		return err
	}
	_, err := m.alertManager.AlertAndNotify(nil, alertmanager.WithPolicyID(policyID))
	return err
}

// determineEnforcement returns the alert and its enforcement action to use from the input list (if any have enforcement).
func determineEnforcement(alerts []*storage.Alert) (alertID string, action storage.EnforcementAction) {
	for _, alert := range alerts {
		if alert.GetEnforcement().GetAction() == storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			return alert.GetId(), storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
		}

		if alert.GetEnforcement().GetAction() != storage.EnforcementAction_UNSET_ENFORCEMENT {
			alertID = alert.GetId()
			action = alert.GetEnforcement().GetAction()
		}
	}
	return
}

func uniqueDeploymentIDs(indicatorsToInfo map[string]indicatorWithInjector) []string {
	m := set.NewStringSet()
	for _, infoWithInjector := range indicatorsToInfo {
		deploymentID := infoWithInjector.indicator.GetDeploymentId()
		if deploymentID == "" {
			continue
		}
		m.Add(deploymentID)
	}
	return m.AsSlice()
}

func containersToKill(alerts []*storage.Alert, indicatorsToInfo map[string]indicatorWithInjector) map[string]indicatorWithInjector {
	containersSet := make(map[string]indicatorWithInjector)

	for _, alert := range alerts {
		if alert.GetEnforcement().GetAction() != storage.EnforcementAction_KILL_POD_ENFORCEMENT {
			continue
		}
		for _, singleIndicator := range alert.GetProcessViolation().GetProcesses() {
			if infoWithInjector, ok := indicatorsToInfo[singleIndicator.GetId()]; ok {
				containersSet[infoWithInjector.indicator.GetSignal().GetContainerId()] = infoWithInjector
			}
		}
	}

	return containersSet
}

func createEnforcementAction(deployment *storage.Deployment, containerID string) *central.SensorEnforcement {
	containers := deployment.GetContainers()
	for _, container := range containers {
		for _, instance := range container.GetInstances() {
			if len(instance.GetInstanceId().GetId()) < 12 {
				continue
			}
			if containerID == instance.GetInstanceId().GetId()[:12] {
				resource := &central.SensorEnforcement_ContainerInstance{
					ContainerInstance: &central.ContainerInstanceEnforcement{
						ContainerInstanceId: instance.GetInstanceId().GetId(),
						PodId:               instance.GetContainingPodId(),
					},
				}
				return &central.SensorEnforcement{
					Enforcement: storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					Resource:    resource,
				}
			}
		}
	}
	return nil
}
