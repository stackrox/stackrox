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

type ScannerDBGenerator struct{}

func (g ScannerDBGenerator) Name() string {
	return "Scanner V2 DB"
}

func (g ScannerDBGenerator) Exportable() bool {
	return true
}

func (g ScannerDBGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	dbPass, err := genTlsSecret("scanner-db-password", m.CA, func(fileMap map[string][]byte) error {
		fileMap["password"] = []byte("letmein")
		return nil
	})

	if err != nil {
		return []Resource{}, err
	}

	dbTls, err := genTlsSecret("scanner-db-tls", m.CA, func(fileMap map[string][]byte) error {
		if err := certgen.IssueOtherServiceCerts(fileMap, m.CA, []mtls.Subject{mtls.ScannerDBSubject}, mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner DB certificate: %w\n", err)
		}
		return nil
	})

	svc := genService("scanner-db", []v1.ServicePort{{
		Name:       "tcp-db",
		Port:       5432,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(5432),
	}})

	return []Resource{
		genServiceAccount("scanner-db"),
		genRoleBinding("scanner", "use-nonroot-v2-scc", m.Config.Namespace),
		dbPass,
		dbTls,
		svc,
		g.genScannerDbDeployment(m),
	}, nil
}

func (g *ScannerDBGenerator) genScannerDbDeployment(m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "scanner-db",
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "scanner-db",
					},
				},
				Spec: v1.PodSpec{
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: &PostgresUser,
					},
					ServiceAccountName: "scanner-db",
					Containers: []v1.Container{{
						Name:            "db",
						Image:           m.Config.Images.ScannerDB,
						SecurityContext: RestrictedSecurityContext(PostgresUser),
						Ports: []v1.ContainerPort{{
							Name:          "tcp-postgresql",
							ContainerPort: 5432,
							Protocol:      v1.ProtocolTCP,
						}},
						Env: []v1.EnvVar{
							{
								Name:  "POSTGRES_HOST_AUTH_METHOD",
								Value: "password",
							},
							{
								Name:  "PGDATA",
								Value: "/var/lib/postgresql/data/pgdata",
							},
						},
					}},
					InitContainers: []v1.Container{{
						Name:            "init-db",
						Image:           m.Config.Images.ScannerDB,
						SecurityContext: RestrictedSecurityContext(PostgresUser),
						Env: []v1.EnvVar{
							{
								Name:  "POSTGRES_PASSWORD_FILE",
								Value: "/run/secrets/stackrox.io/secrets/password",
							},
							{
								Name:  "ROX_SCANNER_DB_INIT",
								Value: "true",
							},
						},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "scanner-db-data",
								MountPath: "/var/lib/postgresql/data",
							},
							{
								Name:      "scanner-db-tls-volume",
								MountPath: "/run/secrets/stackrox.io/certs",
							},
						},
					}},
				},
			},
		},
	}
	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "scanner-db-data",
			MountPath: "/var/lib/postgresql/data",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "scanner-db-tls-volume",
			MountPath: "/run/secrets/stackrox.io/certs",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "scanner-db-tls",
						Items: []v1.KeyToPath{
							{
								Key:  "scanner-db-cert.pem",
								Path: "server.crt",
							},
							{
								Key:  "scanner-db-key.pem",
								Path: "server.key",
							},
							{
								Key:  "ca.pem",
								Path: "root.crt",
							},
						},
					},
				},
			},
		},
		{
			Name:      "shared-memory",
			MountPath: "/dev/shm",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{
						Medium:    v1.StorageMediumMemory,
						SizeLimit: &TwoGigs,
					},
				},
			},
		},
	}

	dbPasswd := VolumeDefAndMount{
		Name:      "scanner-db-password",
		MountPath: "/run/secrets/stackrox.io/secrets",
		Volume: v1.Volume{
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "scanner-db-password",
				},
			},
		},
	}

	dbPasswd.Apply(&deployment.Spec.Template.Spec.InitContainers[0], &deployment.Spec.Template.Spec)

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
	}

	deployment.SetName("scanner-db")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, ScannerDBGenerator{})
}
