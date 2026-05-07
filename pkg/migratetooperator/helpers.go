package migratetooperator

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func findVolume(dep *appsv1.Deployment, name string) *corev1.Volume {
	for i := range dep.Spec.Template.Spec.Volumes {
		if dep.Spec.Template.Spec.Volumes[i].Name == name {
			return &dep.Spec.Template.Spec.Volumes[i]
		}
	}
	return nil
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
