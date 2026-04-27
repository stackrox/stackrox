package migratetooperator

import (
	"strings"

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
	typ          storageType
	pvcName      string
	hostPath     string
	nodeSelector map[string]string
}

type monitoringConfig struct {
	isOpenShift                bool
	openShiftMonitoringEnabled bool
}

type exposureConfig struct {
	loadBalancerEnabled bool
	nodePortEnabled     bool
	routeEnabled        bool
}

type centralConfig struct {
	storage               storageConfig
	monitoring            monitoringConfig
	exposure              exposureConfig
	offlineMode           bool
	telemetryDisabled     bool
	defaultTLSSecretName  string
	declarativeConfigMaps []string
	declarativeSecrets    []string
	plaintextEndpoints    string
	customImages          bool
}

func detectCentral(src Source) (*centralConfig, error) {
	storage, err := detectStorage(src)
	if err != nil {
		return nil, err
	}
	centralDep, err := src.Deployment("central")
	if err != nil {
		return nil, errors.Wrap(err, "retrieving central Deployment")
	}
	if centralDep == nil {
		return nil, errors.New("central Deployment not found")
	}
	exposure, err := detectExposure(src)
	if err != nil {
		return nil, err
	}

	var defaultTLSSecretName string
	if tlsSecret, tlsErr := src.Secret("central-default-tls-cert"); tlsErr != nil {
		return nil, errors.Wrap(tlsErr, "checking for default TLS cert Secret")
	} else if tlsSecret != nil {
		defaultTLSSecretName = "central-default-tls-cert"
	}

	declConfigMaps, declSecrets := detectDeclarativeConfig(centralDep)

	return &centralConfig{
		storage:               *storage,
		monitoring:            detectMonitoring(centralDep),
		exposure:              *exposure,
		offlineMode:           envVarValue(centralDep, "ROX_OFFLINE_MODE") == "true",
		telemetryDisabled:     envVarValue(centralDep, "ROX_TELEMETRY_STORAGE_KEY_V1") == "DISABLED",
		defaultTLSSecretName:  defaultTLSSecretName,
		declarativeConfigMaps: declConfigMaps,
		declarativeSecrets:    declSecrets,
		plaintextEndpoints:    envVarValue(centralDep, "ROX_PLAINTEXT_ENDPOINTS"),
		customImages:          detectCustomImages(centralDep),
	}, nil
}

func detectStorage(src Source) (*storageConfig, error) {
	dep, err := src.Deployment("central-db")
	if err != nil {
		return nil, errors.Wrap(err, "retrieving central-db Deployment")
	}
	if dep == nil {
		return nil, errors.New("central-db Deployment not found")
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
			typ:     storagePVC,
			pvcName: diskVolume.PersistentVolumeClaim.ClaimName,
		}
	case diskVolume.HostPath != nil:
		cfg = &storageConfig{
			typ:      storageHostPath,
			hostPath: diskVolume.HostPath.Path,
		}
	default:
		return nil, errors.New("central-db Deployment \"disk\" volume is neither a PVC nor a hostPath")
	}

	if ns := dep.Spec.Template.Spec.NodeSelector; len(ns) > 0 {
		cfg.nodeSelector = ns
	}
	return cfg, nil
}

func detectMonitoring(dep *appsv1.Deployment) monitoringConfig {
	isOpenShift := envVarIsTrue(dep, "ROX_ENABLE_OPENSHIFT_AUTH")
	return monitoringConfig{
		isOpenShift:                isOpenShift,
		openShiftMonitoringEnabled: isOpenShift && envVarIsTrue(dep, "ROX_ENABLE_SECURE_METRICS"),
	}
}

func detectExposure(src Source) (*exposureConfig, error) {
	cfg := &exposureConfig{}

	svc, err := src.Service("central-loadbalancer")
	if err != nil {
		return nil, errors.Wrap(err, "checking for central-loadbalancer Service")
	}
	if svc != nil {
		switch svc.Spec.Type {
		case corev1.ServiceTypeLoadBalancer:
			cfg.loadBalancerEnabled = true
		case corev1.ServiceTypeNodePort:
			cfg.nodePortEnabled = true
		}
	}

	route, err := src.Route("central")
	if err != nil {
		return nil, errors.Wrap(err, "checking for central Route")
	}
	cfg.routeEnabled = route != nil

	return cfg, nil
}

const declarativeConfigMountPrefix = "/run/stackrox.io/declarative-configuration/"

func detectDeclarativeConfig(dep *appsv1.Deployment) (configMaps []string, secrets []string) {
	declVolumes := make(map[string]bool)
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, vm := range c.VolumeMounts {
			if strings.HasPrefix(vm.MountPath, declarativeConfigMountPrefix) {
				declVolumes[vm.Name] = true
			}
		}
	}
	for _, v := range dep.Spec.Template.Spec.Volumes {
		if !declVolumes[v.Name] {
			continue
		}
		if v.ConfigMap != nil {
			configMaps = append(configMaps, v.ConfigMap.Name)
		}
		if v.Secret != nil {
			secrets = append(secrets, v.Secret.SecretName)
		}
	}
	return configMaps, secrets
}

var defaultImageRegistries = []string{
	"registry.redhat.io/advanced-cluster-security/",
	"quay.io/stackrox-io/",
	"quay.io/rhacs-eng/",
}

func detectCustomImages(dep *appsv1.Deployment) bool {
	for _, c := range dep.Spec.Template.Spec.Containers {
		if !isDefaultImage(c.Image) {
			return true
		}
	}
	return false
}

func isDefaultImage(image string) bool {
	for _, prefix := range defaultImageRegistries {
		if strings.HasPrefix(image, prefix) {
			return true
		}
	}
	return false
}

func envVarIsTrue(dep *appsv1.Deployment, name string) bool {
	return strings.EqualFold(envVarValue(dep, name), "true")
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
