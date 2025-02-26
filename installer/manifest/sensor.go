package manifest

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (m *manifestGenerator) applySensor(ctx context.Context) error {
	err := m.createServiceAccount(ctx, "sensor")
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create sensor service account: %w\n", err)
	}
	log.Info("Created sensor service account")

	if err := m.createClusterRoleBinding(ctx, "sensor", "cluster-admin"); err != nil {
		return fmt.Errorf("Failed to create central service account: %w\n", err)
	}

	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create TLS secret: %w\n", err)
	}

	if err := m.applyHelmConfig(ctx); err != nil {
		return err
	}

	if err := m.applySensorDeployment(ctx); err != nil {
		return err
	}

	return m.applyService(ctx, "sensor", []v1.ServicePort{{
		Name:       "https",
		Port:       443,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromString("api"),
	}})
}

func (m *manifestGenerator) applyHelmConfig(ctx context.Context) error {
	config := fmt.Sprintf(`clusterName: local
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
    null`, m.Config.Images.Stackrox, m.Config.Images.Stackrox, m.Config.Namespace)

	crsSecret := v1.Secret{
		Data: map[string][]byte{"config.yaml": []byte(config)},
	}
	crsSecret.SetName("helm-cluster-config")
	_, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Create(ctx, &crsSecret, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Info("helm-cluster-config secret already exists")
			return nil
		} else {
			return errors.Wrap(err, "Failed to create helm-cluster-config secret")
		}
	}
	log.Info("Created helm-cluster-config secret")
	return nil
}

func (m *manifestGenerator) applySensorDeployment(ctx context.Context) error {
	trueVar := true
	envVars := []v1.EnvVar{{
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
						Image:           m.Config.Images.Stackrox,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"/stackrox/kubernetes"},
						Args:            []string{"ensure-service-certificates"},
						Env:             envVars,
					}, {
						Name:            "init-tls-certs",
						Image:           m.Config.Images.Stackrox,
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
						Image:           m.Config.Images.Stackrox,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"/stackrox/kubernetes"},
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

	_, err := m.Client.AppsV1().Deployments(m.Config.Namespace).Create(ctx, &deployment, metav1.CreateOptions{})

	if k8serrors.IsAlreadyExists(err) {
		_, err = m.Client.AppsV1().Deployments(m.Config.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		log.Info("Updated sensor deployment")
	} else {
		log.Info("Created sensor deployment")
	}

	return err
}
