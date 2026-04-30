package migratetooperator

import (
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TransformToCentral detects the configuration from the given source and generates
// a Central custom resource. It returns the CR and a list of warnings for the caller
// to emit.
func TransformToCentral(src Source) (*platform.Central, []string, error) {
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

	if err := setCentralDBSpec(src, cr); err != nil {
		return nil, nil, err
	}

	centralDep, err := src.Deployment("central")
	if err != nil {
		return nil, nil, errors.Wrap(err, "retrieving central Deployment")
	}
	if centralDep == nil {
		return nil, nil, errors.New("central Deployment not found")
	}

	setCentralMonitoring(centralDep, cr)
	if err := setCentralExposure(src, cr); err != nil {
		return nil, nil, err
	}
	setCentralDefaultTLS(src, cr)
	setCentralDeclarativeConfig(centralDep, cr)
	setCentralTelemetry(centralDep, cr)
	setCentralPlaintextEndpoints(centralDep, cr)
	setCentralOfflineMode(centralDep, cr)

	if detectCustomImages(centralDep) {
		warnings = append(warnings, "Detected non-default container images. "+
			"The operator does not support image overrides in the Central CR. "+
			"Configure RELATED_IMAGE_* environment variables on the operator Deployment instead.")
	}

	return cr, warnings, nil
}

func setCentralDBSpec(src Source, cr *platform.Central) error {
	dep, err := src.Deployment("central-db")
	if err != nil {
		return errors.Wrap(err, "retrieving central-db Deployment")
	}
	if dep == nil {
		return errors.New("central-db Deployment not found")
	}

	db := &platform.CentralDBSpec{}
	diskVolume := findVolume(dep, "disk")
	if diskVolume == nil {
		return errors.New("central-db Deployment has no volume named \"disk\"")
	}
	switch {
	case diskVolume.PersistentVolumeClaim != nil:
		// Only claimName is set. Size and storageClassName are intentionally
		// omitted: the PVC already exists on the cluster, and the operator
		// rejects these fields for pre-existing ("BYO") PVCs.
		db.Persistence = &platform.DBPersistence{
			PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
				ClaimName: pointers.String(diskVolume.PersistentVolumeClaim.ClaimName),
			},
		}
	case diskVolume.HostPath != nil:
		db.Persistence = &platform.DBPersistence{
			HostPath: &platform.HostPathSpec{
				Path: pointers.String(diskVolume.HostPath.Path),
			},
		}
	default:
		return errors.New("central-db Deployment \"disk\" volume is neither a PVC nor a hostPath")
	}
	if ns := dep.Spec.Template.Spec.NodeSelector; len(ns) > 0 {
		db.NodeSelector = ns
	}
	cr.Spec.Central = &platform.CentralComponentSpec{DB: db}
	return nil
}

func setCentralMonitoring(centralDep *appsv1.Deployment, cr *platform.Central) {
	isOpenShift := envVarIsTrue(centralDep, "ROX_ENABLE_OPENSHIFT_AUTH")
	if isOpenShift && !envVarIsTrue(centralDep, "ROX_ENABLE_SECURE_METRICS") {
		cr.Spec.Monitoring = &platform.GlobalMonitoring{
			OpenShiftMonitoring: &platform.OpenShiftMonitoring{
				Enabled: pointers.Bool(false),
			},
		}
	}
}

func setCentralExposure(src Source, cr *platform.Central) error {
	svc, err := src.Service("central-loadbalancer")
	if err != nil {
		return errors.Wrap(err, "checking for central-loadbalancer Service")
	}
	route, err := src.Route("central")
	if err != nil {
		return errors.Wrap(err, "checking for central Route")
	}
	if svc != nil || route != nil {
		exposure := &platform.Exposure{}
		if svc != nil {
			switch svc.Spec.Type {
			case corev1.ServiceTypeLoadBalancer:
				exposure.LoadBalancer = &platform.ExposureLoadBalancer{Enabled: pointers.Bool(true)}
			case corev1.ServiceTypeNodePort:
				exposure.NodePort = &platform.ExposureNodePort{Enabled: pointers.Bool(true)}
			}
		}
		if route != nil {
			exposure.Route = &platform.ExposureRoute{Enabled: pointers.Bool(true)}
		}
		cr.Spec.Central.Exposure = exposure
	}
	return nil
}

func setCentralDefaultTLS(src Source, cr *platform.Central) {
	tlsSecret, _ := src.Secret("central-default-tls-cert")
	if tlsSecret != nil {
		cr.Spec.Central.DefaultTLSSecret = &platform.LocalSecretReference{
			Name: "central-default-tls-cert",
		}
	}
}

func setCentralDeclarativeConfig(centralDep *appsv1.Deployment, cr *platform.Central) {
	declConfigMaps, declSecrets := detectDeclarativeConfig(centralDep)
	if len(declConfigMaps) > 0 || len(declSecrets) > 0 {
		dc := &platform.DeclarativeConfiguration{}
		for _, name := range declConfigMaps {
			dc.ConfigMaps = append(dc.ConfigMaps, platform.LocalConfigMapReference{Name: name})
		}
		for _, name := range declSecrets {
			dc.Secrets = append(dc.Secrets, platform.LocalSecretReference{Name: name})
		}
		cr.Spec.Central.DeclarativeConfiguration = dc
	}
}

func setCentralTelemetry(centralDep *appsv1.Deployment, cr *platform.Central) {
	if envVarValue(centralDep, "ROX_TELEMETRY_STORAGE_KEY_V1") == "DISABLED" {
		cr.Spec.Central.Telemetry = &platform.Telemetry{
			Enabled: pointers.Bool(false),
		}
	}
}

func setCentralPlaintextEndpoints(centralDep *appsv1.Deployment, cr *platform.Central) {
	if pe := envVarValue(centralDep, "ROX_PLAINTEXT_ENDPOINTS"); pe != "" {
		if cr.Spec.Customize == nil {
			cr.Spec.Customize = &platform.CustomizeSpec{}
		}
		cr.Spec.Customize.EnvVars = append(cr.Spec.Customize.EnvVars, corev1.EnvVar{
			Name:  "ROX_PLAINTEXT_ENDPOINTS",
			Value: pe,
		})
	}
}

func setCentralOfflineMode(centralDep *appsv1.Deployment, cr *platform.Central) {
	if envVarValue(centralDep, "ROX_OFFLINE_MODE") == "true" {
		cr.Spec.Egress = &platform.Egress{
			ConnectivityPolicy: platform.ConnectivityOffline.Pointer(),
		}
	}
}
