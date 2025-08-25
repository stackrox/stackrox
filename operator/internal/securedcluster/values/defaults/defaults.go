package defaults

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"k8s.io/utils/ptr"
)

var staticDefaults = platform.SecuredClusterSpec{
	ClusterName:     "",
	ClusterLabels:   nil,
	CentralEndpoint: "",
	Sensor:          nil,
	AdmissionControl: &platform.AdmissionControlComponentSpec{
		Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
		FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
		Replicas:      ptr.To(int32(3)),
	},
	PerNode: &platform.PerNodeSpec{
		Collector:       nil,
		Compliance:      nil,
		NodeInventory:   nil,
		TaintToleration: platform.TaintTolerate.Pointer(),
		HostAliases:     nil,
	},
	AuditLogs:        nil,
	Scanner:          nil,
	ScannerV4:        nil,
	TLS:              nil,
	ImagePullSecrets: nil,
	Customize:        nil,
	Misc:             nil,
	Overlays:         nil,
	Monitoring:       nil,
	RegistryOverride: "",
	Network:          nil,
}

var SecuredClusterStaticDefaults = defaulting.SecuredClusterDefaultingFlow{
	Name: "secured-cluster-static-defaults",
	DefaultingFunc: func(_ logr.Logger, _ *platform.SecuredClusterStatus, _ map[string]string, _ *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
		if !reflect.DeepEqual(defaults, &platform.SecuredClusterSpec{}) {
			return fmt.Errorf("supplied secured cluster's .Default is not empty: %s", common.MarshalToSingleLine(defaults))
		}
		staticDefaults.DeepCopyInto(defaults)
		return nil
	},
}
