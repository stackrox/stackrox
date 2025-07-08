package manifest

import (
	"context"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CollectorGenerator struct{}

func (g CollectorGenerator) Name() string {
	return "Collector"
}

func (g CollectorGenerator) Exportable() bool {
	return true
}

func (g CollectorGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	return []Resource{
		genServiceAccount("collector"),
		genRoleBinding("central", "use-nonroot-v2-scc", m.Config.Namespace),
		g.genDaemonSet(ctx, m),
	}, nil
}

func (g CollectorGenerator) genDaemonSet(ctx context.Context, m *manifestGenerator) Resource {
	trueBool := true
	hostToContainer := v1.MountPropagationHostToContainer
	ds := apps.DaemonSet{
		Spec: apps.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "collector",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "collector",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "collector",
					Containers: []v1.Container{{
						Name:            "collector",
						Image:           m.Config.Images.Collector,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"collector"},
						Ports: []v1.ContainerPort{{
							Name:          "monitoring",
							ContainerPort: 9090,
							Protocol:      v1.ProtocolTCP,
						}},
						SecurityContext: &v1.SecurityContext{
							Capabilities: &v1.Capabilities{
								Drop: []v1.Capability{
									v1.Capability("NET_RAW"),
								},
							},
							Privileged:             &trueBool,
							ReadOnlyRootFilesystem: &trueBool,
						},
						Env: []v1.EnvVar{{
							Name:  "COLLECTOR_CONFIG",
							Value: `{"tlsConfig":{"caCertPath":"/var/run/secrets/stackrox.io/certs/ca.pem","clientCertPath":"/var/run/secrets/stackrox.io/certs/cert.pem","clientKeyPath":"/var/run/secrets/stackrox.io/certs/key.pem"}}`,
						}, {
							Name:  "COLLECTION_METHOD",
							Value: "CORE_BPF",
						}, {
							Name:  "GRPC_SERVER",
							Value: "sensor:443",
						}, {
							Name:  "SNI_HOSTNAME",
							Value: "sensor.stackrox.svc",
						}, {
							Name:  "ROX_COLLECTOR_RUNTIME_FILTERS_ENABLED",
							Value: "true",
						}},
						VolumeMounts: []v1.VolumeMount{{
							Name:             "proc-ro",
							MountPath:        "/host/proc",
							ReadOnly:         true,
							MountPropagation: &hostToContainer,
						}, {
							Name:             "etc-ro",
							MountPath:        "/host/etc",
							ReadOnly:         true,
							MountPropagation: &hostToContainer,
						}, {
							Name:             "usr-lib-ro",
							MountPath:        "/host/usr/lib",
							ReadOnly:         true,
							MountPropagation: &hostToContainer,
						}, {
							Name:             "sys-ro",
							MountPath:        "/host/sys",
							ReadOnly:         true,
							MountPropagation: &hostToContainer,
						}, {
							Name:             "dev-ro",
							MountPath:        "/host/dev",
							ReadOnly:         true,
							MountPropagation: &hostToContainer,
						}, {
							Name:      "certs",
							MountPath: "/run/secrets/stackrox.io/certs",
							ReadOnly:  true,
						}, {
							Name:      "collector-config",
							MountPath: "/etc/stackrox",
							ReadOnly:  true,
						}},
					}},
					Volumes: []v1.Volume{{
						Name: "proc-ro",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/proc",
							},
						},
					}, {
						Name: "etc-ro",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/etc",
							},
						},
					}, {
						Name: "usr-lib-ro",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/usr/lib",
							},
						},
					}, {
						Name: "sys-ro",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/sys",
							},
						},
					}, {
						Name: "dev-ro",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/dev",
							},
						},
					}, {
						Name: "collector-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								Optional:    &trueBool,
								DefaultMode: &ReadOnlyMode,
								LocalObjectReference: v1.LocalObjectReference{
									Name: "collector-config",
								},
							},
						},
					}, {
						Name: "certs",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{
								DefaultMode: &ReadOnlyMode,
								SecretName:  "tls-cert-collector",
								Optional:    &trueBool,
							},
						},
					}},
				},
			},
		},
	}

	ds.SetName("collector")
	ds.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("DaemonSet"))

	return Resource{
		Object:       &ds,
		Name:         ds.Name,
		IsUpdateable: true,
	}
}

func init() {
	securedCluster = append(securedCluster, CollectorGenerator{})
}
