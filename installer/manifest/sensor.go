package manifest

import (
	"context"
	"fmt"
	"strconv"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type SensorGenerator struct{}

func (g SensorGenerator) Name() string {
	return "Sensor"
}

func (g SensorGenerator) Exportable() bool {
	return true
}

func (g SensorGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	return []Resource{
		genServiceAccount("sensor"),
		genClusterRoleBinding("sensor", "cluster-admin", m.Config.Namespace),
		g.applyHelmConfig(m),
		g.applySensorDeployment(m),
		genService("sensor", []v1.ServicePort{{
			Name:       "https",
			Port:       443,
			Protocol:   v1.ProtocolTCP,
			TargetPort: intstr.FromString("api"),
		}}),
	}, nil
}

func (g SensorGenerator) applyHelmConfig(m *manifestGenerator) Resource {
	config := fmt.Sprintf(`clusterName: remote
managedBy: MANAGER_TYPE_HELM_CHART
clusterConfig:
  staticConfig:
    type: KUBERNETES_CLUSTER
    mainImage: %s
    collectorImage: %s
    centralApiEndpoint: central.%s.svc:443
    collectionMethod: CORE_BPF
    admissionController: false
    admissionControllerUpdates: false
    admissionControllerEvents: true
    tolerationsConfig:
      disabled: false
    slimCollector: false
  dynamicConfig:
    disableAuditLogs: true
    admissionControllerConfig:
      enabled: false
      timeoutSeconds: 10
      scanInline: false
      disableBypass: false
      enforceOnUpdates: false
    registryOverride:
  configFingerprint: fingerprint
  clusterLabels:
    null`, m.Config.Images.Sensor, m.Config.Images.Collector, m.Config.Namespace)

	sensorConfig := v1.Secret{
		StringData: map[string]string{"config.yaml": config},
	}
	sensorConfig.SetName("helm-cluster-config")
	sensorConfig.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))
	return Resource{
		Object:       &sensorConfig,
		Name:         sensorConfig.Name,
		IsUpdateable: true,
	}
}

func (g SensorGenerator) applySensorDeployment(m *manifestGenerator) Resource {
	trueVar := true
	envVars := []v1.EnvVar{{
		Name:  "ROX_HOTRELOAD",
		Value: strconv.FormatBool(m.Config.DevMode),
	}, {
		Name:  "ROX_DEVELOPMENT_BUILD",
		Value: strconv.FormatBool(m.Config.DevMode),
	}, {
		Name:  "ROX_CENTRAL_ENDPOINT",
		Value: fmt.Sprintf("central.%s.svc:443", m.Config.Namespace),
	}, {
		Name:  "ROX_HELM_CLUSTER_CONFIG_FP",
		Value: "fingerprint",
	}, {
		Name:  "ROX_CRS_FILE",
		Value: "/run/secrets/stackrox.io/crs/crs",
	}, {
		Name: "POD_NAMESPACE",
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	}, {
		Name: "POD_NAME",
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	}, {
		Name: "ROX_SENSOR_SERVICE_CERT",
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				Key: "cert.pem",
				LocalObjectReference: v1.LocalObjectReference{
					Name: "tls-cert-sensor",
				},
				Optional: &trueVar,
			},
		},
	}}
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sensor",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "sensor",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "sensor",
					InitContainers: []v1.Container{{
						Name:            "crs",
						Image:           m.Config.Images.Sensor,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"/stackrox/kubernetes"},
						Args:            []string{"ensure-service-certificates"},
						Env:             envVars,
					}, {
						Name:            "init-tls-certs",
						Image:           m.Config.Images.Sensor,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"/stackrox/init-tls-certs"},
						Args: []string{
							"--legacy=/run/secrets/stackrox.io/certs-legacy/",
							"--new=/run/secrets/stackrox.io/certs-new/",
							"--destination=/run/secrets/stackrox.io/certs/",
						},
					}},
					Containers: []v1.Container{{
						Name:            "sensor",
						Image:           m.Config.Images.Sensor,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"sh", "-c", "while true; do /stackrox/kubernetes; done"},
						Ports: []v1.ContainerPort{{
							Name:          "api",
							ContainerPort: 8443,
							Protocol:      v1.ProtocolTCP,
						}},
						Env: envVars,
					}},
				},
			},
		},
	}

	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "varlog",
			MountPath: "/var/log/stackrox/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "certs",
			MountPath: "/run/secrets/stackrox.io/certs/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "cache",
			MountPath: "/var/cache/stackrox",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "sensor-etc-ssl-volume",
			MountPath: "/etc/ssl",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "sensor-etc-pki-volume",
			MountPath: "/etc/pki/ca-trust",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "certs-new",
			MountPath: "/run/secrets/stackrox.io/certs-new/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "tls-cert-sensor",
						Optional:    &trueVar,
					},
				},
			},
		},
		{
			Name:      "crs",
			MountPath: "/run/secrets/stackrox.io/crs/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "cluster-registration-secret",
						Optional:   &trueVar,
					},
				},
			},
		},
		{
			Name:      "helm-cluster-config",
			MountPath: "/run/secrets/stackrox.io/helm-cluster-config/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "helm-cluster-config",
						Optional:   &trueVar,
					},
				},
			},
		},
		{
			Name:      "additional-ca-volume",
			MountPath: "/usr/local/share/ca-certificates/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "additional-ca-sensor",
						Optional:   &trueVar,
					},
				},
			},
		},
	}

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
		v.Apply(&deployment.Spec.Template.Spec.InitContainers[0], nil)
		v.Apply(&deployment.Spec.Template.Spec.InitContainers[1], nil)
	}

	deployment.SetName("sensor")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	securedCluster = append(securedCluster, SensorGenerator{})
}
