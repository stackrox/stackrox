package upgrade

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/namespaces"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	upgraderCPURequest = resource.MustParse("50m")
	upgraderCPULimit   = resource.MustParse("500m")
	upgraderMemRequest = resource.MustParse("50Mi")
	upgraderMemLimit   = resource.MustParse("500Mi")
)

func toK8sEnvVars(triggerEnvVars []*central.SensorUpgradeTrigger_EnvVarDef) []v1.EnvVar {
	envVars := make([]v1.EnvVar, 0, len(triggerEnvVars))

	for _, tev := range triggerEnvVars {
		ev := v1.EnvVar{
			Name:  tev.GetName(),
			Value: tev.GetDefaultValue(),
		}
		if tev.GetSourceEnvVar() != "" {
			if valueFromEnv := os.Getenv(tev.GetSourceEnvVar()); valueFromEnv != "" {
				ev.Value = valueFromEnv
			}
		}
		envVars = append(envVars, ev)
	}

	return envVars
}

func (p *process) determineImage() (string, error) {
	if image := p.trigger.GetImage(); image != "" {
		return image, nil
	}

	// If the image is not specified, sensor uses the same image it's using to launch the upgrader.
	// This code path will be hit during cert rotation.
	sensorDeployment, err := p.k8sClient.AppsV1().Deployments(namespaces.StackRox).Get(p.ctx(), sensorDeploymentName, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch sensor deployment from Kube")
	}
	for _, container := range sensorDeployment.Spec.Template.Spec.Containers {
		if container.Name == sensorContainerName {
			return container.Image, nil
		}
	}
	return "", errors.New("no sensor container found in sensor deployment")
}

func (p *process) createDeployment(serviceAccountName string, sensorDeployment *appsV1.Deployment) (*appsV1.Deployment, error) {
	image, err := p.determineImage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine image")
	}
	deployment := &appsV1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgraderDeploymentName,
			Namespace: namespaces.StackRox,
			Labels: map[string]string{
				"app":             upgraderDeploymentName,
				processIDLabelKey: p.trigger.GetUpgradeProcessId(),
			},
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: &([]int32{1})[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":             upgraderDeploymentName,
					processIDLabelKey: p.trigger.GetUpgradeProcessId(),
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaces.StackRox,
					Labels: map[string]string{
						"app":             upgraderDeploymentName,
						processIDLabelKey: p.trigger.GetUpgradeProcessId(),
					},
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
								{
									Weight: 100,
									Preference: v1.NodeSelectorTerm{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "cloud.google.com/gke-preemptible",
												Operator: v1.NodeSelectorOpNotIn,
												Values:   []string{"true"},
											},
										},
									},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "sensor-tls-volume",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "sensor-tls",
									Items: []v1.KeyToPath{
										{
											Key:  "sensor-cert.pem",
											Path: "cert.pem",
										},
										{
											Key:  "sensor-key.pem",
											Path: "key.pem",
										},
										{
											Key:  "ca.pem",
											Path: "ca.pem",
										},
									},
								},
							},
						},
						{
							Name: "additional-ca-volume",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "additional-ca-sensor",
									Optional:   &[]bool{true}[0],
								},
							},
						},
						{
							Name: "etc-ssl-volume",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "etc-pki-trust-volume",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:    "upgrader",
							Image:   image,
							Command: p.trigger.GetCommand(),
							Env:     toK8sEnvVars(p.trigger.GetEnvVars()),
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    upgraderCPURequest,
									v1.ResourceMemory: upgraderMemRequest,
								},
								Limits: v1.ResourceList{
									v1.ResourceCPU:    upgraderCPULimit,
									v1.ResourceMemory: upgraderMemLimit,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "sensor-tls-volume",
									ReadOnly:  true,
									MountPath: "/run/secrets/stackrox.io/certs",
								},
								{
									Name:      "additional-ca-volume",
									ReadOnly:  true,
									MountPath: "/usr/local/share/ca-certificates/",
								},
								{
									Name:      "etc-ssl-volume",
									MountPath: "/etc/ssl/",
								},
								{
									Name:      "etc-pki-trust-volume",
									MountPath: "/etc/pki/ca-trust/",
								},
							},
						},
					},
					ImagePullSecrets: []v1.LocalObjectReference{
						{Name: "stackrox"},
					},
					ServiceAccountName: serviceAccountName,
				},
			},
		},
	}

	envVars := &deployment.Spec.Template.Spec.Containers[0].Env
	*envVars = append(*envVars, v1.EnvVar{
		Name:  "ROX_UPGRADER_OWNER",
		Value: fmt.Sprintf("%s:%s:%s/%s", deployment.Kind, deployment.APIVersion, deployment.Namespace, deployment.Name),
	})

	// These are all nil safe because they are all non-pointers
	if sensorDeployment != nil {
		deployment.Spec.Template.Spec.Tolerations = sensorDeployment.Spec.Template.Spec.Tolerations
	}

	return deployment, nil
}
