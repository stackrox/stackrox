package defaults

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common"
)

var staticDefaults = platform.SecuredClusterSpec{
	AdmissionControl: &platform.AdmissionControlComponentSpec{
		Bypass:        new(platform.BypassBreakGlassAnnotation),
		FailurePolicy: new(platform.FailurePolicyIgnore),
		Replicas:      new(int32(3)),
	},
	PerNode: &platform.PerNodeSpec{
		Collector: &platform.CollectorContainerSpec{
			Collection: platform.CollectionCOREBPF.Pointer(),
		},
		TaintToleration: platform.TaintTolerate.Pointer(),
		FileActivityMonitoring: &platform.FAMContainerSpec{
			Mode: platform.FileActivityMonitoringDisabled.Pointer(),
		},
	},
	AuditLogs: &platform.AuditLogsSpec{
		Collection: platform.AuditLogsCollectionAuto.Pointer(),
	},
	ProcessBaselines: &platform.ProcessBaselinesSpec{
		AutoLock: platform.ProcessBaselinesAutoLockModeDisabled.Pointer(),
	},
	ScannerV4: &platform.LocalScannerV4ComponentSpec{
		// ScannerComponent field is set using a dedicated defaulting flow.
		Indexer: &platform.ScannerV4Component{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: new(platform.ScannerAutoScalingEnabled),
				Replicas:    new(int32(3)),
				MinReplicas: new(int32(2)),
				MaxReplicas: new(int32(5)),
			},
		},
		DB: &platform.ScannerV4DB{
			Persistence: &platform.ScannerV4Persistence{
				PersistentVolumeClaim: &platform.ScannerV4PersistentVolumeClaim{
					ClaimName: new("scanner-v4-db"),
				},
			},
		},
	},
	Monitoring: &platform.GlobalMonitoring{
		OpenShiftMonitoring: &platform.OpenShiftMonitoring{
			Enabled: new(true),
		},
	},
	Network: &platform.GlobalNetworkSpec{
		Policies: new(platform.NetworkPoliciesEnabled),
	},
	Customize: &platform.CustomizeSpec{
		DeploymentDefaults: &platform.DeploymentDefaultsSpec{
			PinToNodes: new(platform.PinToNodesNone),
		},
	},
	ProcessIndicators: &platform.ProcessIndicatorsSpec{
		Persistence:        platform.ProcessIndicatorConfigEnabled.Pointer(),
		ExcludeOpenshiftNs: platform.ProcessIndicatorConfigDisabled.Pointer(),
	},
	VirtualMachines: &platform.VirtualMachinesSpec{
		Scraper: &platform.VirtualMachinesScraperSpec{
			Concurrency:       new(int32(20)),
			MaxResponseSizeKB: new(int32(16384)),
			PollInterval:      new("4h"),
		},
	},
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
