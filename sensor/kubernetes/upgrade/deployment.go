package upgrade

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/namespaces"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func createDeployment(trigger *central.SensorUpgradeTrigger, serviceAccountName string) *v1beta1.Deployment {
	deployment := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgraderDeploymentName,
			Namespace: namespaces.StackRox,
			Labels: map[string]string{
				"app":             upgraderDeploymentName,
				processIDLabelKey: trigger.GetUpgradeProcessId(),
			},
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &([]int32{1})[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":             upgraderDeploymentName,
					processIDLabelKey: trigger.GetUpgradeProcessId(),
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespaces.StackRox,
					Labels: map[string]string{
						"app":             upgraderDeploymentName,
						processIDLabelKey: trigger.GetUpgradeProcessId(),
					},
				},
				Spec: v1.PodSpec{
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
					},
					Containers: []v1.Container{
						{
							Name:    "upgrader",
							Image:   trigger.GetImage(),
							Command: trigger.GetCommand(),
							Env:     toK8sEnvVars(trigger.GetEnvVars()),
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "sensor-tls-volume",
									ReadOnly:  true,
									MountPath: "/run/secrets/stackrox.io/certs",
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

	return deployment
}
