package migratetooperator

import (
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Transform detects the configuration from the given source and generates a Central
// custom resource. It returns the CR and a list of warnings for the caller to emit.
func TransformToCentral(src Source) (*platform.Central, []string, error) {
	config, err := detectCentral(src)
	if err != nil {
		return nil, nil, err
	}
	cr, warnings := generateCentral(config)
	return cr, warnings, nil
}

func generateCentral(config *centralConfig) (*platform.Central, []string) {
	var warnings []string

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
	switch config.storage.typ {
	case storagePVC:
		// Only claimName is set. Size and storageClassName are intentionally
		// omitted: the PVC already exists on the cluster, and the operator
		// rejects these fields for pre-existing ("BYO") PVCs.
		db.Persistence = &platform.DBPersistence{
			PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
				ClaimName: pointers.String(config.storage.pvcName),
			},
		}
	case storageHostPath:
		db.Persistence = &platform.DBPersistence{
			HostPath: &platform.HostPathSpec{
				Path: pointers.String(config.storage.hostPath),
			},
		}
	}
	if len(config.storage.nodeSelector) > 0 {
		db.NodeSelector = config.storage.nodeSelector
	}
	cr.Spec.Central = &platform.CentralComponentSpec{DB: db}

	exp := config.exposure
	if exp.loadBalancerEnabled || exp.nodePortEnabled || exp.routeEnabled {
		exposure := &platform.Exposure{}
		if exp.loadBalancerEnabled {
			exposure.LoadBalancer = &platform.ExposureLoadBalancer{Enabled: pointers.Bool(true)}
		}
		if exp.nodePortEnabled {
			exposure.NodePort = &platform.ExposureNodePort{Enabled: pointers.Bool(true)}
		}
		if exp.routeEnabled {
			exposure.Route = &platform.ExposureRoute{Enabled: pointers.Bool(true)}
		}
		cr.Spec.Central.Exposure = exposure
	}

	if config.monitoring.isOpenShift && !config.monitoring.openShiftMonitoringEnabled {
		cr.Spec.Monitoring = &platform.GlobalMonitoring{
			OpenShiftMonitoring: &platform.OpenShiftMonitoring{
				Enabled: pointers.Bool(false),
			},
		}
	}

	if config.defaultTLSSecretName != "" {
		cr.Spec.Central.DefaultTLSSecret = &platform.LocalSecretReference{
			Name: config.defaultTLSSecretName,
		}
	}

	if len(config.declarativeConfigMaps) > 0 || len(config.declarativeSecrets) > 0 {
		dc := &platform.DeclarativeConfiguration{}
		for _, name := range config.declarativeConfigMaps {
			dc.ConfigMaps = append(dc.ConfigMaps, platform.LocalConfigMapReference{Name: name})
		}
		for _, name := range config.declarativeSecrets {
			dc.Secrets = append(dc.Secrets, platform.LocalSecretReference{Name: name})
		}
		cr.Spec.Central.DeclarativeConfiguration = dc
	}

	if config.telemetryDisabled {
		cr.Spec.Central.Telemetry = &platform.Telemetry{
			Enabled: pointers.Bool(false),
		}
	}

	if config.plaintextEndpoints != "" {
		if cr.Spec.Customize == nil {
			cr.Spec.Customize = &platform.CustomizeSpec{}
		}
		cr.Spec.Customize.EnvVars = append(cr.Spec.Customize.EnvVars, corev1.EnvVar{
			Name:  "ROX_PLAINTEXT_ENDPOINTS",
			Value: config.plaintextEndpoints,
		})
	}

	if config.offlineMode {
		cr.Spec.Egress = &platform.Egress{
			ConnectivityPolicy: platform.ConnectivityOffline.Pointer(),
		}
	}

	if config.customImages {
		warnings = append(warnings, "Detected non-default container images. "+
			"The operator does not support image overrides in the Central CR. "+
			"Configure RELATED_IMAGE_* environment variables on the operator Deployment instead.")
	}

	return cr, warnings
}
