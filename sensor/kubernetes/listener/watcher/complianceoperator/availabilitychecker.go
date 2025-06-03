package complianceoperator

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	log = logging.LoggerForModule()
)

type availabilityChecker struct {
	gv        schema.GroupVersion
	resources []complianceoperator.APIResource
}

func apiResourceToNameGroupString(resource complianceoperator.APIResource) string {
	return fmt.Sprintf("%s.%s", resource.Name, resource.Group)
}

// NewComplianceOperatorAvailabilityChecker creates a new AvailabilityChecker
func NewComplianceOperatorAvailabilityChecker() *availabilityChecker {
	resources := []complianceoperator.APIResource{
		complianceoperator.Profile,
		complianceoperator.Rule,
		complianceoperator.ScanSetting,
		complianceoperator.ScanSettingBinding,
		complianceoperator.ComplianceScan,
		complianceoperator.ComplianceSuite,
		complianceoperator.ComplianceCheckResult,
		complianceoperator.TailoredProfile,
		complianceoperator.ComplianceRemediation,
	}
	return &availabilityChecker{
		gv:        complianceoperator.GetGroupVersion(),
		resources: resources,
	}
}

// Available returns 'true' if the Compliance Operator resources are available in the cluster
func (w *availabilityChecker) Available(client client.Interface) bool {
	var resourceList *v1.APIResourceList
	var err error
	if resourceList, err = utils.ServerResourcesForGroup(client, w.gv.String()); err != nil {
		log.Errorf("Checking API resources for group %q: %v", w.gv.String(), err)
		return false
	}
	for _, r := range w.resources {
		if !utils.ResourceExists(resourceList, r.Name, w.gv.String()) {
			return false
		}
	}
	return true
}

type crdWatcher interface {
	AddResourceToWatch(string) error
}

// AppendToCRDWatcher adds the Compliance Operator resources to the CRD watcher
func (w *availabilityChecker) AppendToCRDWatcher(watcher crdWatcher) error {
	for _, r := range w.resources {
		nameGroupString := apiResourceToNameGroupString(r)
		if err := watcher.AddResourceToWatch(nameGroupString); err != nil {
			return errors.Wrapf(err, "watching resource %q", nameGroupString)
		}
	}
	return nil
}
