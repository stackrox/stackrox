package migratetooperator

import (
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type storageType int

const (
	storagePVC storageType = iota
	storageHostPath
)

type storageConfig struct {
	Type         storageType
	PVCName      string
	HostPath     string
	NodeSelector map[string]string
}

type monitoringConfig struct {
	IsOpenShift                bool
	OpenShiftMonitoringEnabled bool
}

type exposureConfig struct {
	LoadBalancerEnabled bool
	NodePortEnabled     bool
	RouteEnabled        bool
}

type detectedConfig struct {
	Storage           storageConfig
	Monitoring        monitoringConfig
	Exposure          exposureConfig
	OfflineMode       bool
	TelemetryDisabled bool
}

func detect(src source) (*detectedConfig, error) {
	storage, err := detectStorage(src)
	if err != nil {
		return nil, err
	}
	centralDep, err := src.CentralDeployment()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving central Deployment")
	}
	exposure, err := detectExposure(src)
	if err != nil {
		return nil, err
	}

	return &detectedConfig{
		Storage:           *storage,
		Monitoring:        detectMonitoring(centralDep),
		Exposure:          *exposure,
		OfflineMode:       envVarValue(centralDep, "ROX_OFFLINE_MODE") == "true",
		TelemetryDisabled: envVarValue(centralDep, "ROX_TELEMETRY_STORAGE_KEY_V1") == "DISABLED",
	}, nil
}

func detectStorage(src source) (*storageConfig, error) {
	dep, err := src.CentralDBDeployment()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving central-db Deployment")
	}

	var diskVolume *corev1.Volume
	for i := range dep.Spec.Template.Spec.Volumes {
		if dep.Spec.Template.Spec.Volumes[i].Name == "disk" {
			diskVolume = &dep.Spec.Template.Spec.Volumes[i]
			break
		}
	}
	if diskVolume == nil {
		return nil, errors.New("central-db Deployment has no volume named \"disk\"")
	}

	var cfg *storageConfig
	switch {
	case diskVolume.PersistentVolumeClaim != nil:
		cfg = &storageConfig{
			Type:    storagePVC,
			PVCName: diskVolume.PersistentVolumeClaim.ClaimName,
		}
	case diskVolume.HostPath != nil:
		cfg = &storageConfig{
			Type:     storageHostPath,
			HostPath: diskVolume.HostPath.Path,
		}
	default:
		return nil, errors.New("central-db Deployment \"disk\" volume is neither a PVC nor a hostPath")
	}

	if ns := dep.Spec.Template.Spec.NodeSelector; len(ns) > 0 {
		cfg.NodeSelector = ns
	}
	return cfg, nil
}

func detectMonitoring(dep *appsv1.Deployment) monitoringConfig {
	isOpenShift := hasEnvVar(dep, "ROX_ENABLE_OPENSHIFT_AUTH")
	return monitoringConfig{
		IsOpenShift:                isOpenShift,
		OpenShiftMonitoringEnabled: isOpenShift && hasEnvVar(dep, "ROX_ENABLE_SECURE_METRICS"),
	}
}

func detectExposure(src source) (*exposureConfig, error) {
	cfg := &exposureConfig{}

	found, data, err := src.ResourceByKindAndName("Service", "central-loadbalancer")
	if err != nil {
		return nil, errors.Wrap(err, "checking for central-loadbalancer Service")
	}
	if found {
		spec, _ := data["spec"].(map[string]interface{})
		svcType, _ := spec["type"].(string)
		switch svcType {
		case "LoadBalancer":
			cfg.LoadBalancerEnabled = true
		case "NodePort":
			cfg.NodePortEnabled = true
		}
	}

	routeFound, _, err := src.ResourceByKindAndName("Route", "central")
	if err != nil {
		return nil, errors.Wrap(err, "checking for central Route")
	}
	cfg.RouteEnabled = routeFound

	return cfg, nil
}

func hasEnvVar(dep *appsv1.Deployment, name string) bool {
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, env := range c.Env {
			if env.Name == name {
				return true
			}
		}
	}
	return false
}

func envVarValue(dep *appsv1.Deployment, name string) string {
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, env := range c.Env {
			if env.Name == name {
				return env.Value
			}
		}
	}
	return ""
}
