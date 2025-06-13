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

type CentralDBGenerator struct{}

func (g CentralDBGenerator) Name() string {
	return "Central DB"
}

func (g CentralDBGenerator) Exportable() bool {
	return true
}

func (g CentralDBGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	sa := genServiceAccount("central-db")

	svc := genService("central-db", []v1.ServicePort{{
		Name:       "tcp-db",
		Port:       5432,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromString("postgresql"),
	}})

	tlsSecret, err := genTlsSecret("central-db-tls", m.CA, func(fileMap map[string][]byte) error {
		subjects := []mtls.Subject{mtls.CentralDBSubject}
		if err := certgen.IssueOtherServiceCerts(fileMap, m.CA, subjects, mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing central service certificate: %w\n", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	passwordSecret, err := genTlsSecret("central-db-password", m.CA, func(fileMap map[string][]byte) error {
		fileMap["password"] = []byte("letmein")
		return nil
	})
	if err != nil {
		return nil, err
	}

	return []Resource{
		sa,
		tlsSecret,
		passwordSecret,
		g.createCentralDbConfig(),
		g.createCentralDbDeployment(m),
		svc,
	}, nil
}

func (g CentralDBGenerator) createCentralDbConfig() Resource {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"pg_hba.conf": `local   all             all                                     scram-sha-256
host    all             all             127.0.0.1/32            scram-sha-256
host    all             all             ::1/128                 scram-sha-256
local   replication     all                                     trust
host    replication     all             127.0.0.1/32            trust
host    replication     all             ::1/128                 trust

hostnossl  all       all   0.0.0.0/0     reject
hostnossl  all       all   ::0/0         reject

hostssl    all       all   0.0.0.0/0     scram-sha-256
hostssl    all       all   ::0/0         scram-sha-256`,
			"postgresql.conf": `hba_file = '/etc/stackrox.d/config/pg_hba.conf'
listen_addresses = '*'
max_connections = 200
password_encryption = scram-sha-256

ssl = on
ssl_ca_file = '/run/secrets/stackrox.io/certs/root.crt'
ssl_cert_file = '/run/secrets/stackrox.io/certs/server.crt'
ssl_key_file = '/run/secrets/stackrox.io/certs/server.key'

shared_buffers = 2GB
work_mem = 40MB
maintenance_work_mem = 512MB
effective_cache_size = 4GB

dynamic_shared_memory_type = posix
max_wal_size = 5GB
min_wal_size = 80MB

log_timezone = 'Etc/UTC'
datestyle = 'iso, mdy'
timezone = 'Etc/UTC'
lc_messages = 'en_US.utf8'
lc_monetary = 'en_US.utf8'
lc_numeric = 'en_US.utf8'
lc_time = 'en_US.utf8'

default_text_search_config = 'pg_catalog.english'
shared_preload_libraries = 'pg_stat_statements'`,
		},
	}
	cm.SetName("central-db-config")
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return Resource{
		Object:       &cm,
		Name:         cm.Name,
		IsUpdateable: true,
	}

}

func (g CentralDBGenerator) createCentralDbDeployment(m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "central-db",
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "central-db",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "central-db",
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: &PostgresUser,
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []v1.Container{{
						Name:            "central-db",
						Image:           m.Config.Images.CentralDB,
						SecurityContext: RestrictedSecurityContext(PostgresUser),
						Ports: []v1.ContainerPort{{
							Name:          "postgresql",
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
						Image:           m.Config.Images.CentralDB,
						Command:         []string{"init-entrypoint.sh"},
						SecurityContext: RestrictedSecurityContext(PostgresUser),
						Env: []v1.EnvVar{
							{
								Name:  "PGDATA",
								Value: "/var/lib/postgresql/data/pgdata",
							},
						},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "disk",
								MountPath: "/var/lib/postgresql/data",
							},
						},
					}},
				},
			},
		},
	}
	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "config-volume",
			MountPath: "/etc/stackrox.d/config/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "central-db-config",
						},
					},
				},
			},
		},
		{
			Name:      "disk",
			MountPath: "/var/lib/postgresql/data",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
				// VolumeSource: v1.VolumeSource{
				// 	PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				// 		ClaimName: "central-db",
				// 	},
				// },
			},
		},
		{
			Name:      "central-db-tls-volume",
			MountPath: "/run/secrets/stackrox.io/certs",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "central-db-tls",
						Items: []v1.KeyToPath{
							{
								Key:  "central-db-cert.pem",
								Path: "server.crt",
							},
							{
								Key:  "central-db-key.pem",
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
		Name:      "central-db-password",
		MountPath: "/run/secrets/stackrox.io/secrets",
		Volume: v1.Volume{
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "central-db-password",
				},
			},
		},
	}

	dbPasswd.Apply(&deployment.Spec.Template.Spec.InitContainers[0], &deployment.Spec.Template.Spec)

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
	}

	deployment.SetName("central-db")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, CentralDBGenerator{})
}
