package complianceoperator

import (
	"time"

	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watchers/utils"
)

var (
	log = logging.LoggerForModule()
)

// NewComplianceOperatorRulesWatcher creates a Compliance Operator RulesWatcher.
func NewComplianceOperatorRulesWatcher(cli client.Interface) *RulesWatcher {
	return &RulesWatcher{
		cli: cli,
	}
}

// RulesWatcher is responsible to watch if the CRD for Compliance Operator Rules becomes available in the cluster.
type RulesWatcher struct {
	cli client.Interface
}

// Watch for the availability of the Compliance Operator Rules CRD and calls stopCallback if it becomes available.
func (w *RulesWatcher) Watch(stop *concurrency.Signal, stopCallback func(string)) {
	featureDisabledMsg := "Sensor will not watch for the presence of Compliance CRDs. If the Compliance Operator is deployed in the cluster, Sensor will require a manual restart"
	if env.ComplianceCRDsWatchTimer.DurationSetting() == 0 {
		log.Warnf("%s is set to zero", env.ComplianceCRDsWatchTimer.EnvVar())
		log.Info(featureDisabledMsg)
		return
	}
	if stopCallback == nil {
		log.Warn("The stop callback is nil")
		log.Info(featureDisabledMsg)
		return
	}
	log.Infof("Starting the Compliance CRDs watcher with interval %v", env.ComplianceCRDsWatchTimer.DurationSetting())
	ticker := time.NewTicker(env.ComplianceCRDsWatchTimer.DurationSetting())
	for {
		select {
		case <-stop.Done():
			return
		case <-ticker.C:
			if resourceList, err := utils.ServerResourcesForGroup(w.cli, complianceoperator.GetGroupVersion().String()); err != nil {
				continue
			} else if utils.ResourceExists(resourceList, complianceoperator.ComplianceCheckResult.Name) {
				stopCallback("Compliance Operator CRDs detected. Gracefully restarting sensor...")
				return
			}
		}
	}
}
