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

var staticDefaults = platform.CentralSpec{
	Central: &platform.CentralComponentSpec{
		NotifierSecretsEncryption: &platform.NotifierSecretsEncryption{
			Enabled: ptr.To(false),
		},
		DB: &platform.CentralDBSpec{
			Persistence: &platform.DBPersistence{
				PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
					ClaimName: ptr.To("central-db"),
				},
			},
			ConnectionPoolSize: &platform.DBConnectionPoolSize{
				MinConnections: ptr.To(int32(10)),
				MaxConnections: ptr.To(int32(90)),
			},
		},
		Exposure: &platform.Exposure{
			LoadBalancer: &platform.ExposureLoadBalancer{
				Enabled: ptr.To(false),
				Port:    ptr.To(int32(443)),
			},
			NodePort: &platform.ExposureNodePort{
				Enabled: ptr.To(false),
			},
			Route: &platform.ExposureRoute{
				Enabled: ptr.To(false),
				Reencrypt: &platform.ExposureRouteReencrypt{
					Enabled: ptr.To(false),
				},
			},
		},
		Telemetry: &platform.Telemetry{
			Enabled: ptr.To(true),
		},
	},
	Scanner: &platform.ScannerComponentSpec{
		Analyzer: &platform.ScannerAnalyzerComponent{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
				Replicas:    ptr.To(int32(3)),
				MinReplicas: ptr.To(int32(2)),
				MaxReplicas: ptr.To(int32(5)),
			},
		},
	},
	ScannerV4: &platform.ScannerV4Spec{
		// ScannerComponent field is set using a dedicated defaulting flow.
		Indexer: &platform.ScannerV4Component{
			Scaling: &platform.ScannerComponentScaling{
				AutoScaling: ptr.To(platform.ScannerAutoScalingEnabled),
				Replicas:    ptr.To(int32(3)),
				MinReplicas: ptr.To(int32(2)),
				MaxReplicas: ptr.To(int32(5)),
			},
		},
		Matcher: &platform.ScannerV4Component{
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
	Egress: &platform.Egress{
		ConnectivityPolicy: platform.ConnectivityOnline.Pointer(),
	},
	Monitoring: &platform.GlobalMonitoring{
		OpenShiftMonitoring: &platform.OpenShiftMonitoring{
			Enabled: ptr.To(true),
		},
	},
	Network: &platform.GlobalNetworkSpec{
		Policies: ptr.To(platform.NetworkPoliciesEnabled),
	},
	ConfigAsCode: &platform.ConfigAsCodeSpec{
		ComponentPolicy: ptr.To(platform.ConfigAsCodeComponentEnabled),
	},
}

var CentralStaticDefaults = defaulting.CentralDefaultingFlow{
	Name: "central-static-defaults",
	DefaultingFunc: func(_ logr.Logger, _ *platform.CentralStatus, _ map[string]string, _ *platform.CentralSpec, defaults *platform.CentralSpec) error {
		if !reflect.DeepEqual(defaults, &platform.CentralSpec{}) {
			return fmt.Errorf("supplied central's .Default is not empty: %s", common.MarshalToSingleLine(defaults))
		}
		staticDefaults.DeepCopyInto(defaults)
		return nil
	},
}
