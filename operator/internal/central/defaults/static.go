package defaults

import (
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common"
	"k8s.io/utils/ptr"
)

var staticDefaults = platform.CentralSpec{
	Central: &platform.CentralComponentSpec{
		NotifierSecretsEncryption: &platform.NotifierSecretsEncryption{
			Enabled: new(false),
		},
		DB: &platform.CentralDBSpec{
			// Persistence is taken care of by CentralDBPersistenceDefaultingFlow
			ConnectionPoolSize: &platform.DBConnectionPoolSize{
				MinConnections: new(int32(10)),
				MaxConnections: new(int32(90)),
			},
		},
		Exposure: &platform.Exposure{
			LoadBalancer: &platform.ExposureLoadBalancer{
				Enabled: new(false),
				Port:    new(int32(443)),
			},
			NodePort: &platform.ExposureNodePort{
				Enabled: new(false),
			},
			Route: &platform.ExposureRoute{
				Enabled: new(false),
				Reencrypt: &platform.ExposureRouteReencrypt{
					Enabled: new(false),
				},
			},
		},
		Telemetry: &platform.Telemetry{
			Enabled: new(true),
		},
	},
	Scanner: &platform.ScannerComponentSpec{
		Analyzer: &platform.ScannerAnalyzerComponent{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
				Replicas:    new(int32(3)),
				MinReplicas: new(int32(2)),
				MaxReplicas: new(int32(5)),
			},
		},
	},
	ScannerV4: &platform.ScannerV4Spec{
		// ScannerComponent field is set using a dedicated defaulting flow.
		Indexer: &platform.ScannerV4Component{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
				Replicas:    new(int32(3)),
				MinReplicas: new(int32(2)),
				MaxReplicas: new(int32(5)),
			},
		},
		Matcher: &platform.ScannerV4Component{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
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
	Egress: &platform.Egress{
		ConnectivityPolicy: platform.ConnectivityOnline.Pointer(),
	},
	Monitoring: &platform.GlobalMonitoring{
		OpenShiftMonitoring: &platform.OpenShiftMonitoring{
			Enabled: new(true),
		},
	},
	Network: &platform.GlobalNetworkSpec{
		Policies: ptr.To(platform.NetworkPoliciesEnabled),
	},
	ConfigAsCode: &platform.ConfigAsCodeSpec{
		ComponentPolicy: ptr.To(platform.ConfigAsCodeComponentEnabled),
	},
	Customize: &platform.CustomizeSpec{
		DeploymentDefaults: &platform.DeploymentDefaultsSpec{
			PinToNodes: ptr.To(platform.PinToNodesNone),
		},
	},
}

var CentralStaticDefaults = CentralDefaultingFlow{
	Name: "central-static-defaults",
	DefaultingFunc: func(_ logr.Logger, _ *platform.CentralStatus, _ map[string]string, _ *platform.CentralSpec, defaults *platform.CentralSpec) error {
		if !reflect.DeepEqual(defaults, &platform.CentralSpec{}) {
			return fmt.Errorf("supplied central's .Default is not empty: %s", common.MarshalToSingleLine(defaults))
		}
		staticDefaults.DeepCopyInto(defaults)
		return nil
	},
}
