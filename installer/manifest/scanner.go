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

type ScannerGenerator struct{}

func (g ScannerGenerator) Name() string {
	return "Scanner V2"
}

func (g ScannerGenerator) Exportable() bool {
	return true
}

func (g ScannerGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	scannerTls, err := genTlsSecret("scanner-tls", m.CA, func(fileMap map[string][]byte) error {
		if err := certgen.IssueScannerCerts(fileMap, m.CA, mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner service certificate: %w\n", err)
		}
		return nil
	})

	if err != nil {
		return []Resource{}, err
	}

	svc := genService("scanner", []v1.ServicePort{{
		Name:       "grpcs-scanner",
		Port:       8443,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(8443),
	}, {
		Name:       "https-scanner",
		Port:       8080,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(8080),
	}})

	return []Resource{
		genServiceAccount("scanner"),
		genRoleBinding("scanner", "use-nonroot-v2-scc", m.Config.Namespace),
		scannerTls,
		svc,
		g.genScannerConfig(m),
		g.genScannerDeployment(m),
	}, nil
}

func (g *ScannerGenerator) genScannerConfig(m *manifestGenerator) Resource {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"config.yaml": fmt.Sprintf(`# Configuration file for scanner.
scanner:
  centralEndpoint: https://central.%s.svc
  sensorEndpoint: https://sensor.%s.svc
  database:
    # Database driver
    type: pgsql
    options:
      # PostgreSQL Connection string
      # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
      source: host=scanner-db.%s.svc port=5432 user=postgres sslmode=verify-full statement_timeout=60000

      # Number of elements kept in the cache
      # Values unlikely to change (e.g. namespaces) are cached in order to save prevent needless roundtrips to the database.
      cachesize: 16384

  api:
    httpsPort: 8080
    grpcPort: 8443

  updater:
    # Frequency with which the scanner will poll for vulnerability updates.
    interval: 5m

  logLevel: INFO

  # The scanner intentionally avoids extracting or analyzing any files
  # larger than the following default sizes to prevent DoS attacks.
  # Leave these commented to use a reasonable default.

  # The max size of files in images that are extracted.
  # Increasing this number increases memory pressure.
  # maxExtractableFileSizeMB: 200
  # The max size of ELF executable files that are analyzed.
  # Increasing this number may increase disk pressure.
  # maxELFExecutableFileSizeMB: 800
  # The max size of image file reader buffer. Image file data beyond this limit are overflowed to temporary files on disk.
  # maxImageFileReaderBufferSizeMB: 100

  exposeMonitoring: false`, m.Config.Namespace, m.Config.Namespace, m.Config.Namespace),
		},
	}
	cm.SetName("scanner-config")
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return Resource{
		Object:       &cm,
		Name:         cm.Name,
		IsUpdateable: true,
	}
}

func (g *ScannerGenerator) genScannerDeployment(m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "scanner",
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "scanner",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "scanner",
					Containers: []v1.Container{{
						Name:    "scanner",
						Image:   m.Config.Images.Scanner,
						Command: []string{"/stackrox/scanner-v2"},
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
							Name: "scanner-config",
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
						SecretName:  "scanner-tls",
						Items: []v1.KeyToPath{
							{
								Key:  "scanner-cert.pem",
								Path: "cert.pem",
							},
							{
								Key:  "scanner-key.pem",
								Path: "key.pem",
							},
							{
								Key:  "scanner-db-cert.pem",
								Path: "scanner-db-cert.pem",
							},
							{
								Key:  "scanner-db-key.pem",
								Path: "scanner-db-key.pem",
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
		// v.Apply(&deployment.Spec.Template.Spec.InitContainers[0], nil)
	}

	deployment.SetName("scanner")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, ScannerGenerator{})
}
