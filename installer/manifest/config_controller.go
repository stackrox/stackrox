package manifest

import (
	"context"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigControllerGenerator struct{}

func (g ConfigControllerGenerator) Name() string {
	return "Configuration Controller"
}

func (g ConfigControllerGenerator) Exportable() bool {
	return true
}

func (g ConfigControllerGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	return []Resource{
		genServiceAccount("config-controller"),
		genRole("config-controller-manager-role", []rbacv1.PolicyRule{{
			APIGroups: []string{"config.stackrox.io"},
			Resources: []string{
				"securitypolicies",
				"securitypolicies/status",
			},
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			},
		}}),
		genRoleBinding("config-controller", "config-controller-manager-role", m.Config.Namespace),
		g.genConfigControllerDeployment(m),
	}, nil
}

func (g *ConfigControllerGenerator) genConfigControllerDeployment(m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "config-controller",
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "config-controller",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "config-controller",
					Containers: []v1.Container{{
						Name:    "config-controller",
						Image:   m.Config.Images.ConfigController,
						Command: []string{"/stackrox/config-controller"},
						Env: []v1.EnvVar{
							{
								Name: "POD_NAMESPACE",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "additional-ca-volume",
			MountPath: "/run/secrets/stackrox.io/certs",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "additional-ca",
					},
				},
			},
		},
	}

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
		// v.Apply(&deployment.Spec.Template.Spec.InitContainers[0], nil)
	}

	deployment.SetName("config-controller")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, ConfigControllerGenerator{})
}
