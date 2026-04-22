package migratetooperator

import (
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func generateCR(config *detectedConfig) *platform.Central {
	cr := &platform.Central{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "Central",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "stackrox-central-services",
		},
	}

	db := &platform.CentralDBSpec{}
	switch config.Storage.Type {
	case storagePVC:
		// Only claimName is set. Size and storageClassName are intentionally
		// omitted: the PVC already exists on the cluster, and the operator
		// rejects these fields for pre-existing ("BYO") PVCs.
		db.Persistence = &platform.DBPersistence{
			PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
				ClaimName: pointers.String(config.Storage.PVCName),
			},
		}
	case storageHostPath:
		db.Persistence = &platform.DBPersistence{
			HostPath: &platform.HostPathSpec{
				Path: pointers.String(config.Storage.HostPath),
			},
		}
	}
	if len(config.Storage.NodeSelector) > 0 {
		db.NodeSelector = config.Storage.NodeSelector
	}
	cr.Spec.Central = &platform.CentralComponentSpec{DB: db}

	exp := config.Exposure
	if exp.LoadBalancerEnabled || exp.NodePortEnabled || exp.RouteEnabled {
		exposure := &platform.Exposure{}
		if exp.LoadBalancerEnabled {
			exposure.LoadBalancer = &platform.ExposureLoadBalancer{Enabled: pointers.Bool(true)}
		}
		if exp.NodePortEnabled {
			exposure.NodePort = &platform.ExposureNodePort{Enabled: pointers.Bool(true)}
		}
		if exp.RouteEnabled {
			exposure.Route = &platform.ExposureRoute{Enabled: pointers.Bool(true)}
		}
		cr.Spec.Central.Exposure = exposure
	}

	if config.Monitoring.IsOpenShift && !config.Monitoring.OpenShiftMonitoringEnabled {
		cr.Spec.Monitoring = &platform.GlobalMonitoring{
			OpenShiftMonitoring: &platform.OpenShiftMonitoring{
				Enabled: pointers.Bool(false),
			},
		}
	}

	if config.DefaultTLSSecretName != "" {
		cr.Spec.Central.DefaultTLSSecret = &platform.LocalSecretReference{
			Name: config.DefaultTLSSecretName,
		}
	}

	if len(config.DeclarativeConfigMaps) > 0 || len(config.DeclarativeSecrets) > 0 {
		dc := &platform.DeclarativeConfiguration{}
		for _, name := range config.DeclarativeConfigMaps {
			dc.ConfigMaps = append(dc.ConfigMaps, platform.LocalConfigMapReference{Name: name})
		}
		for _, name := range config.DeclarativeSecrets {
			dc.Secrets = append(dc.Secrets, platform.LocalSecretReference{Name: name})
		}
		cr.Spec.Central.DeclarativeConfiguration = dc
	}

	if config.TelemetryDisabled {
		cr.Spec.Central.Telemetry = &platform.Telemetry{
			Enabled: pointers.Bool(false),
		}
	}

	if config.PlaintextEndpoints != "" {
		if cr.Spec.Customize == nil {
			cr.Spec.Customize = &platform.CustomizeSpec{}
		}
		cr.Spec.Customize.EnvVars = append(cr.Spec.Customize.EnvVars, corev1.EnvVar{
			Name:  "ROX_PLAINTEXT_ENDPOINTS",
			Value: config.PlaintextEndpoints,
		})
	}

	if config.OfflineMode {
		cr.Spec.Egress = &platform.Egress{
			ConnectivityPolicy: platform.ConnectivityOffline.Pointer(),
		}
	}

	return cr
}

// marshalCR serializes only the spec portion of the Central CR, avoiding
// empty status fields that the operator types include.
func marshalCR(cr *platform.Central) ([]byte, error) {
	obj := struct {
		APIVersion string               `json:"apiVersion"`
		Kind       string               `json:"kind"`
		Metadata   map[string]string    `json:"metadata"`
		Spec       platform.CentralSpec `json:"spec"`
	}{
		APIVersion: cr.APIVersion,
		Kind:       cr.Kind,
		Metadata:   map[string]string{"name": cr.Name},
		Spec:       cr.Spec,
	}
	out, err := yaml.Marshal(obj)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling to YAML")
	}
	return out, nil
}
