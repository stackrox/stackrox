package manifest

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ScannerV4Generator struct{}

func (g ScannerV4Generator) Name() string {
	return "Scanner V4"
}

func (g ScannerV4Generator) Exportable() bool {
	return true
}

func (g ScannerV4Generator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	matcherTls, err := genTlsSecret("scanner-v4-matcher-tls", m.CA, func(fileMap map[string][]byte) error {
		if err := certgen.IssueServiceCert(fileMap, m.CA, mtls.ScannerV4MatcherSubject, "", mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner-v4-matcher certificate: %w\n", err)
		}
		return nil
	})

	if err != nil {
		return []Resource{}, err
	}

	indexerTls, err := genTlsSecret("scanner-v4-indexer-tls", m.CA, func(fileMap map[string][]byte) error {
		if err := certgen.IssueServiceCert(fileMap, m.CA, mtls.ScannerV4IndexerSubject, "", mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner-v4-indexer certificate: %w\n", err)
		}
		return nil
	})

	if err != nil {
		return []Resource{}, err
	}

	ports := []v1.ServicePort{{
		Name:       "grpcs-scanner",
		Port:       8443,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(8443),
	}, {
		Name:       "https-scanner",
		Port:       8080,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(8080),
	}}

	indexerCm, matcherCm := g.genScannerV4Configs(m)

	return []Resource{
		genServiceAccount("scanner-v4"),
		genRoleBinding("scanner-v4", "use-nonroot-v2-scc", m.Config.Namespace),
		matcherTls,
		indexerTls,
		indexerCm,
		matcherCm,
		genService("scanner-v4-matcher", ports),
		genService("scanner-v4-indexer", ports),
		g.genScannerV4Deployment("matcher", int32(2), m),
		g.genScannerV4Deployment("indexer", int32(3), m),
	}, nil
}

func (g *ScannerV4Generator) genScannerV4Configs(m *manifestGenerator) (Resource, Resource) {
	dbConfig := fmt.Sprintf(`
    conn_string: >
      host=scanner-v4-db.%s
      port=5432
      sslrootcert=/run/secrets/stackrox.io/certs/ca.pem
      user=postgres
      sslmode=verify-full
      pool_min_conns=5
      pool_max_conns=40
      client_encoding=UTF8
    password_file: /run/secrets/stackrox.io/secrets/password`, m.Config.Namespace)

	indexerCm := v1.ConfigMap{
		Data: map[string]string{
			"config.yaml": fmt.Sprintf(`# Configuration file for Scanner v4 Indexer.
stackrox_services: true
indexer:
  enable: true
  database:
%s
  get_layer_timeout: 1m
  repository_to_cpe_url: https://central/api/extensions/scannerdefinitions?file=repo2cpe
  name_to_repos_url: https://central/api/extensions/scannerdefinitions?file=name2repos
  repository_to_cpe_file: /repo2cpe/repository-to-cpe.json
  name_to_repos_file: /repo2cpe/container-name-repos-map.json
matcher:
  enable: false
log_level: "INFO"
grpc_listen_addr: 0.0.0.0:8443
http_listen_addr: 0.0.0.0:9443
proxy:
  config_dir: /run/secrets/stackrox.io/proxy-config
  config_file: config.yaml`, dbConfig),
		},
	}

	matcherCm := v1.ConfigMap{
		Data: map[string]string{
			"config.yaml": fmt.Sprintf(`# Configuration file for Scanner v4 Matcher.
stackrox_services: true
indexer:
  enable: false
matcher:
  enable: true
  database:
%s
  vulnerabilities_url: https://central.%s.svc/api/extensions/scannerdefinitions?version=ROX_VULNERABILITY_VERSION
  indexer_addr: scanner-v4-indexer.%s.svc:8443
log_level: "INFO"
grpc_listen_addr: 0.0.0.0:8443
http_listen_addr: 0.0.0.0:9443
proxy:
  config_dir: /run/secrets/stackrox.io/proxy-config
  config_file: config.yaml`, dbConfig, m.Config.Namespace, m.Config.Namespace),
		},
	}

	indexerCm.SetName("scanner-v4-indexer-config")
	indexerCm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	matcherCm.SetName("scanner-v4-matcher-config")
	matcherCm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))

	return Resource{
			Object:       &indexerCm,
			Name:         indexerCm.Name,
			IsUpdateable: true,
		}, Resource{
			Object:       &matcherCm,
			Name:         matcherCm.Name,
			IsUpdateable: true,
		}
}

func (g *ScannerV4Generator) genScannerV4Deployment(name string, replicaCount int32, m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fmt.Sprintf("scanner-v4-%s", name),
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Replicas: &replicaCount,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("scanner-v4-%s", name),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "scanner-v4",
					InitContainers: []v1.Container{{
						Name:            "add-additional-cas",
						Image:           m.Config.Images.ScannerV4,
						ImagePullPolicy: v1.PullAlways,
						Command: []string{
							"sh",
							"-c",
							"restore-all-dir-contents && import-additional-cas",
						},
					}},
					Containers: []v1.Container{{
						Name:  name,
						Image: m.Config.Images.ScannerV4,
						Command: []string{
							"/stackrox/scanner-v4",
							"--conf=/etc/scanner/config.yaml",
						},
						Ports: []v1.ContainerPort{{
							Name:          "https",
							ContainerPort: 8080,
							Protocol:      v1.ProtocolTCP,
						}, {
							Name:          "grpc",
							ContainerPort: 8443,
							Protocol:      v1.ProtocolTCP,
						}},
						Env: []v1.EnvVar{
							{
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
								Name:  "PGSSLROOTCERT",
								Value: "/run/secrets/stackrox.io/certs/ca.pem",
							},
						},
					}},
				},
			},
		},
	}

	trueBool := true
	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "scanner-etc-ssl-volume",
			MountPath: "/etc/ssl",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "scanner-etc-pki-volume",
			MountPath: "/etc/pki/ca-trust",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "vuln-db-temp",
			MountPath: "/var/lib/stackrox",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
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
						DefaultMode: &ReadOnlyMode,
						SecretName:  "additional-ca",
						Optional:    &trueBool,
					},
				},
			},
		},
		{
			Name:      "scanner-config-volume",
			MountPath: "/etc/scanner",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: fmt.Sprintf("scanner-v4-%s-config", name),
						},
					},
				},
			},
		},
		{
			Name:      "scanner-tls-volume",
			MountPath: "/run/secrets/stackrox.io/certs/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  fmt.Sprintf("scanner-v4-%s-tls", name),
					},
				},
			},
		},
		{
			Name:      "proxy-config-volume",
			MountPath: "/run/secrets/stackrox.io/proxy-config/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "proxy-config",
						Optional:   &trueBool,
					},
				},
			},
		},
		{
			Name:      "scanner-db-password",
			MountPath: "/run/secrets/stackrox.io/secrets",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "scanner-db-password",
					},
				},
			},
		},
	}

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
		v.Apply(&deployment.Spec.Template.Spec.InitContainers[0], nil)
	}

	deployment.SetName(fmt.Sprintf("scanner-v4-%s", name))
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))
	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, ScannerV4Generator{})
}
