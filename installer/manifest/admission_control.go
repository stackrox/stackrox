package manifest

import (
	"context"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type AdmissionControlGenerator struct{}

func (g AdmissionControlGenerator) Name() string {
	return "AdmissionControl"
}

func (g AdmissionControlGenerator) Exportable() bool {
	return true
}

// Priority returns a high priority (100) to ensure admission control is applied last
// since it includes validating webhooks that can break subsequent resource creation.
func (g AdmissionControlGenerator) Priority() int {
	return 100
}

func (g AdmissionControlGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	resources := []Resource{
		genServiceAccount("admission-control"),
		g.applyAdmissionControlDeployment(m),
		g.applyAdmissionControlService(),
		g.applyValidatingWebhookConfiguration(m),
		g.applyAdmissionControlRBAC(),
	}

	if m.Config.ApplyNetworkPolicies {
		resources = append(resources, g.applyAdmissionControlNetworkPolicy())
	}

	return resources, nil
}

func (g AdmissionControlGenerator) applyAdmissionControlRBAC() Resource {
	rules := []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"secrets", "configmaps"},
		Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
	}, {
		APIGroups: []string{""},
		Resources: []string{"pods"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		APIGroups: []string{"apps"},
		Resources: []string{"deployments", "replicasets", "daemonsets", "statefulsets"},
		Verbs:     []string{"get", "list", "watch"},
	}, {
		APIGroups: []string{"batch"},
		Resources: []string{"jobs", "cronjobs"},
		Verbs:     []string{"get", "list", "watch"},
	}}

	return genRole("admission-control", rules)
}

func (g AdmissionControlGenerator) applyAdmissionControlNetworkPolicy() Resource {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admission-control",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "admission-control",
				},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 8443},
						},
					},
				},
			},
		},
	}

	policy.SetGroupVersionKind(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"))

	return Resource{
		Object:       policy,
		Name:         policy.Name,
		IsUpdateable: true,
	}
}

func (g AdmissionControlGenerator) applyAdmissionControlDeployment(m *manifestGenerator) Resource {
	trueVar := true
	falseVar := false

	envVars := []v1.EnvVar{
		{
			Name: "ROX_MEMLIMIT",
			ValueFrom: &v1.EnvVarSource{
				ResourceFieldRef: &v1.ResourceFieldSelector{
					Resource: "limits.memory",
				},
			},
		},
		{
			Name: "GOMAXPROCS",
			ValueFrom: &v1.EnvVarSource{
				ResourceFieldRef: &v1.ResourceFieldSelector{
					Resource: "limits.cpu",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "ROX_SENSOR_ENDPOINT",
			Value: "sensor:443",
		},
	}

	deployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admission-control",
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "admission-control",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "admission-control",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "admission-control",
					InitContainers: []v1.Container{{
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
						Name:            "admission-control",
						Image:           m.Config.Images.AdmissionControl,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"/stackrox/admission-control"},
						Ports: []v1.ContainerPort{{
							Name:          "webhook",
							ContainerPort: 8443,
							Protocol:      v1.ProtocolTCP,
						}},
						ReadinessProbe: &v1.Probe{
							ProbeHandler: v1.ProbeHandler{
								HTTPGet: &v1.HTTPGetAction{
									Path:   "/ready",
									Port:   intstr.FromInt(8443),
									Scheme: v1.URISchemeHTTPS,
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       5,
							FailureThreshold:    1,
						},
						SecurityContext: &v1.SecurityContext{
							ReadOnlyRootFilesystem:   &trueVar,
							AllowPrivilegeEscalation: &falseVar,
						},
						Env: envVars,
					}},
				},
			},
		},
	}

	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "config",
			MountPath: "/run/config/stackrox.io/admission-control/config/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "admission-control",
						},
						Optional: &trueVar,
					},
				},
			},
		}, {
			Name:      "config-store",
			MountPath: "/var/lib/stackrox/admission-control/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		}, {
			Name:      "ca",
			MountPath: "/run/secrets/stackrox.io/ca/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "additional-ca",
					},
				},
			},
		}, {
			Name:      "certs",
			MountPath: "/run/secrets/stackrox.io/certs/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		}, {
			Name:      "certs-new",
			MountPath: "/run/secrets/stackrox.io/certs-new/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "tls-cert-admission-control",
						Optional:    &trueVar,
					},
				},
			},
		}, {
			Name:      "ssl",
			MountPath: "/etc/ssl",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		}, {
			Name:      "pki",
			MountPath: "/etc/pki/ca-trust/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		}, {
			Name:      "additional-cas",
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
	}

	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func (g AdmissionControlGenerator) applyAdmissionControlService() Resource {
	return genService("admission-control", []v1.ServicePort{{
		Name:       "https",
		Port:       443,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromString("webhook"),
	}})
}

func (g AdmissionControlGenerator) applyValidatingWebhookConfiguration(m *manifestGenerator) Resource {
	failurePolicy := admissionregistrationv1.Fail
	sideEffects := admissionregistrationv1.SideEffectClassNoneOnDryRun
	timeout := int32(12) // 10 + 2 seconds buffer

	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "stackrox",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    "policyeval.stackrox.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				SideEffects:             &sideEffects,
				TimeoutSeconds:          &timeout,
				FailurePolicy:           &failurePolicy,
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      "admission-control",
						Namespace: m.Config.Namespace,
						Path:      stringPtr("/validate"),
					},
					CABundle: GetCertificateManager().GetCACertificate(),
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"*"},
							Resources: []string{
								"pods",
								"deployments",
								"deployments/scale",
								"replicasets",
								"replicationcontrollers",
								"statefulsets",
								"daemonsets",
								"cronjobs",
								"jobs",
								"deploymentconfigs",
							},
						},
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "namespace.metadata.stackrox.io/name",
							Operator: metav1.LabelSelectorOpNotIn,
							Values: []string{
								m.Config.Namespace,
								"kube-system",
								"kube-public",
								"istio-system",
							},
						},
					},
				},
			},
			{
				Name:                    "k8sevents.stackrox.io",
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				SideEffects:             &sideEffects,
				TimeoutSeconds:          &timeout,
				FailurePolicy:           &failurePolicy,
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      "admission-control",
						Namespace: m.Config.Namespace,
						Path:      stringPtr("/events"),
					},
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Connect,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"*"},
							Resources: []string{
								"pods",
								"pods/exec",
								"pods/portforward",
							},
						},
					},
				},
			},
		},
	}

	webhook.SetGroupVersionKind(admissionregistrationv1.SchemeGroupVersion.WithKind("ValidatingWebhookConfiguration"))

	return Resource{
		Object:        webhook,
		Name:          webhook.Name,
		IsUpdateable:  true,
		ClusterScoped: true,
	}
}

func stringPtr(s string) *string {
	return &s
}

func init() {
	securedCluster = append(securedCluster, AdmissionControlGenerator{})
}
