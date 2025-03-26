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

type ScannerV4DBGenerator struct{}

func (g ScannerV4DBGenerator) Name() string {
	return "Scanner V4 DB"
}

func (g ScannerV4DBGenerator) Exportable() bool {
	return true
}

func (g ScannerV4DBGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	dbPass, err := genTlsSecret("scanner-v4-db-password", m.CA, func(fileMap map[string][]byte) error {
		fileMap["password"] = []byte("letmein")
		return nil
	})

	if err != nil {
		return []Resource{}, err
	}

	dbTls, err := genTlsSecret("scanner-v4-db-tls", m.CA, func(fileMap map[string][]byte) error {
		if err := certgen.IssueOtherServiceCerts(fileMap, m.CA, []mtls.Subject{mtls.ScannerV4DBSubject}, mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner DB certificate: %w\n", err)
		}
		return nil
	})

	svc := genService("scanner-v4-db", []v1.ServicePort{{
		Name:       "tcp-db",
		Port:       5432,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(5432),
	}})

	return []Resource{
		genServiceAccount("scanner-v4-db"),
		genRoleBinding("scanner-v4-db", "use-nonroot-v2-scc", m.Config.Namespace),
		dbPass,
		dbTls,
		svc,
		g.genScannerV4DBConfigs(),
		g.genScannerV4DBDeployment(m),
	}, nil
}

func (g *ScannerV4DBGenerator) genScannerV4DBConfigs() Resource {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"pg_hba.conf": `# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             all                                     scram-sha-256
# IPv4 local connections:
host    all             all             127.0.0.1/32            scram-sha-256
# IPv6 local connections:
host    all             all             ::1/128                 scram-sha-256
# Allow replication connections from localhost, by a user with the
# replication privilege.
local   replication     all                                     reject
host    replication     all             127.0.0.1/32            reject
host    replication     all             ::1/128                 reject

### STACKROX MODIFIED
# Reject all non ssl connections from IPs
hostnossl  all       all   0.0.0.0/0     reject
hostnossl  all       all   ::0/0         reject

# Accept connections from ssl with password
hostssl    all       all   0.0.0.0/0     scram-sha-256
hostssl    all       all   ::0/0         scram-sha-256
###`,
			"postgresql.conf": `#------------------------------------------------------------------------------
# FILE LOCATIONS
#------------------------------------------------------------------------------

hba_file = '/etc/stackrox.d/config/pg_hba.conf'

#------------------------------------------------------------------------------
# CONNECTIONS AND AUTHENTICATION
#------------------------------------------------------------------------------

# - Connection Settings -

listen_addresses = '*'
max_connections = 500

# - Authentication -

password_encryption = 'scram-sha-256'

# - SSL -

ssl = on
ssl_ca_file = '/run/secrets/stackrox.io/certs/root.crt'
ssl_cert_file = '/run/secrets/stackrox.io/certs/server.crt'
ssl_key_file = '/run/secrets/stackrox.io/certs/server.key'

#------------------------------------------------------------------------------
# RESOURCE USAGE (except WAL)
#------------------------------------------------------------------------------

# - Memory -

# Keep this in sync with the shared-memory volume in the
# templates/02-scanner-v4-07-db-deployment.yaml
shared_buffers = 750MB
work_mem = 16MB
maintenance_work_mem = 128MB
dynamic_shared_memory_type = posix

#------------------------------------------------------------------------------
# WRITE-AHEAD LOG
#------------------------------------------------------------------------------

# - Checkpoints -

max_wal_size = 3GB
min_wal_size = 80MB

#------------------------------------------------------------------------------
# REPORTING AND LOGGING
#------------------------------------------------------------------------------

# - What to Log -

log_timezone = 'Etc/UTC'

#------------------------------------------------------------------------------
# CLIENT CONNECTION DEFAULTS
#------------------------------------------------------------------------------

# - Locale and Formatting -

datestyle = 'iso, mdy'
timezone = 'Etc/UTC'

lc_messages = 'en_US.utf8'
lc_monetary = 'en_US.utf8'
lc_numeric = 'en_US.utf8'
lc_time = 'en_US.utf8'

default_text_search_config = 'pg_catalog.english'

# - Shared Library Preloading -

shared_preload_libraries = 'pg_stat_statements'`,
		},
	}

	cm.SetName("scanner-v4-db-config")
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return Resource{
		Object:       &cm,
		Name:         cm.Name,
		IsUpdateable: true,
	}
}

func (g *ScannerV4DBGenerator) genScannerV4DBDeployment(m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "scanner-v4-db",
				},
			},
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "scanner-v4-db",
					},
				},
				Spec: v1.PodSpec{
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: &PostgresUser,
					},
					ServiceAccountName: "scanner-v4",
					Containers: []v1.Container{{
						Name:            "db",
						Image:           m.Config.Images.ScannerV4DB,
						SecurityContext: RestrictedSecurityContext(PostgresUser),
						Ports: []v1.ContainerPort{{
							Name:          "tcp-postgresql",
							ContainerPort: 5432,
							Protocol:      v1.ProtocolTCP,
						}},
						Env: []v1.EnvVar{
							{
								Name:  "POSTGRES_HOST_AUTH_METHOD",
								Value: "scram-sha-256",
							},
							{
								Name:  "PGDATA",
								Value: "/var/lib/postgresql/data/pgdata",
							},
						},
					}},
					InitContainers: []v1.Container{{
						Name:            "init-db",
						Image:           m.Config.Images.ScannerV4DB,
						SecurityContext: RestrictedSecurityContext(PostgresUser),
						Command:         []string{"init-entrypoint.sh"},
						Env: []v1.EnvVar{
							{
								Name:  "PGDATA",
								Value: "/var/lib/postgresql/data/pgdata",
							},
							{
								Name:  "POSTGRES_HOST_AUTH_METHOD",
								Value: "scram-sha-256",
							},
							{
								Name:  "POSTGRES_PASSWORD_FILE",
								Value: "/run/secrets/stackrox.io/secrets/password",
							},
							{
								Name:  "SCANNER_DB_INIT_BUNDLE_ENABLED",
								Value: "true",
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
			Name:      "scanner-v4-db-config",
			MountPath: "/etc/stackrox.d/config",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						DefaultMode:          &ReadOnlyMode,
						LocalObjectReference: v1.LocalObjectReference{Name: "scanner-v4-db-config"},
					},
				},
			},
		},
		{
			Name:      "scanner-v4-db-tls-volume",
			MountPath: "/run/secrets/stackrox.io/certs",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "scanner-v4-db-tls",
						Items: []v1.KeyToPath{
							{
								Key:  "scanner-v4-db-cert.pem",
								Path: "server.crt",
							},
							{
								Key:  "scanner-v4-db-key.pem",
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
					SecretName: "scanner-v4-db-password",
				},
			},
		},
	}

	dbPasswd.Apply(&deployment.Spec.Template.Spec.InitContainers[0], &deployment.Spec.Template.Spec)

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
		v.Apply(&deployment.Spec.Template.Spec.InitContainers[0], nil)
	}

	deployment.SetName("scanner-v4-db")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, ScannerV4DBGenerator{})
}
