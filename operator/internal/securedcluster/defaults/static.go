package defaults

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common"
	"k8s.io/utils/ptr"
)

var staticDefaults = platform.SecuredClusterSpec{
	AdmissionControl: &platform.AdmissionControlComponentSpec{
		Bypass:        ptr.To(platform.BypassBreakGlassAnnotation),
		FailurePolicy: ptr.To(platform.FailurePolicyIgnore),
		Replicas:      ptr.To(int32(3)),
	},
	PerNode: &platform.PerNodeSpec{
		Collector: &platform.CollectorContainerSpec{
			Collection: platform.CollectionCOREBPF.Pointer(),
		},
		TaintToleration: platform.TaintTolerate.Pointer(),
	},
	AuditLogs: &platform.AuditLogsSpec{
		Collection: platform.AuditLogsCollectionAuto.Pointer(),
	},
	Scanner: &platform.LocalScannerComponentSpec{
		ScannerComponent: platform.LocalScannerComponentAutoSense.Pointer(),
		Analyzer: &platform.ScannerAnalyzerComponent{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
				Replicas:    ptr.To(int32(3)),
				MinReplicas: ptr.To(int32(2)),
				MaxReplicas: ptr.To(int32(5)),
			},
		},
	},
	ScannerV4: &platform.LocalScannerV4ComponentSpec{
		// ScannerComponent field is set using a dedicated defaulting flow.
		Indexer: &platform.ScannerV4Component{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
				Replicas:    ptr.To(int32(3)),
				MinReplicas: ptr.To(int32(2)),
				MaxReplicas: ptr.To(int32(5)),
			},
		},
		DB: &platform.ScannerV4DB{
			Persistence: &platform.ScannerV4Persistence{
				PersistentVolumeClaim: &platform.ScannerV4PersistentVolumeClaim{
					ClaimName: ptr.To("scanner-v4-db"),
				},
			},
		},
	},
	Monitoring: &platform.GlobalMonitoring{
		OpenShiftMonitoring: &platform.OpenShiftMonitoring{
			Enabled: ptr.To(true),
		},
	},
	Network: &platform.GlobalNetworkSpec{
		Policies: ptr.To(platform.NetworkPoliciesEnabled),
	},
	ProcessBaselines: nil,
}

var SecuredClusterStaticDefaults = SecuredClusterDefaultingFlow{
	Name: "secured-cluster-static-defaults",
	DefaultingFunc: func(_ logr.Logger, _ *platform.SecuredClusterStatus, _ map[string]string, _ *platform.SecuredClusterSpec, defaults *platform.SecuredClusterSpec) error {
		if !reflect.DeepEqual(defaults, &platform.SecuredClusterSpec{}) {
			return fmt.Errorf("supplied secured cluster's .Default is not empty: %s", common.MarshalToSingleLine(defaults))
		}
		staticDefaults.DeepCopyInto(defaults)
		return nil
	},
}
