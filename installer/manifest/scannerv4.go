package manifest

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (m *manifestGenerator) applyScannerV4(ctx context.Context) error {
	err := m.createServiceAccount(ctx, "scanner-v4")
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create scanner v4 service account: %w\n", err)
	}
	log.Info("Created scanner v4 service account")

	if err := m.createScannerV4Configs(ctx); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create scanner v4 config: %w\n", err)
	}
	log.Info("Created central config")

	if err := m.createScannerV4TlsSecrets(ctx); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create TLS secret: %w\n", err)
	}
	log.Info("Created scanner TLS secrets")

	if err := m.applyScannerV4DbDeployment(ctx); err != nil {
		return err
	}

	if err := m.applyScannerV4Deployment(ctx, "matcher", int32(2)); err != nil {
		return err
	}

	if err := m.applyScannerV4Deployment(ctx, "indexer", int32(3)); err != nil {
		return err
	}

	if err := m.applyScannerV4Services(ctx); err != nil {
		return err
	}

	return nil
}

func (m *manifestGenerator) createScannerV4Configs(ctx context.Context) error {
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
	// scanner-v4-indexer-config   1      52m
	cm := v1.ConfigMap{
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

	if err := m.applyConfigMap(ctx, "scanner-v4-indexer-config", &cm); err != nil {
		return err
	}

	cm = v1.ConfigMap{
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

	if err := m.applyConfigMap(ctx, "scanner-v4-matcher-config", &cm); err != nil {
		return err
	}

	//scanner-v4-db-config        2      52m
	cm = v1.ConfigMap{
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

	return m.applyConfigMap(ctx, "scanner-v4-db-config", &cm)
}

func (m *manifestGenerator) createScannerV4TlsSecrets(ctx context.Context) error {
	err := m.applyTlsSecret(ctx, "scanner-v4-matcher-tls", func(fileMap map[string][]byte) error {
		if err := certgen.IssueServiceCert(fileMap, m.CA, mtls.ScannerV4MatcherSubject, "", mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner-v4-matcher certificate: %w\n", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = m.applyTlsSecret(ctx, "scanner-v4-indexer-tls", func(fileMap map[string][]byte) error {
		if err := certgen.IssueServiceCert(fileMap, m.CA, mtls.ScannerV4IndexerSubject, "", mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner-v4-indexer certificate: %w\n", err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = m.applyTlsSecret(ctx, "scanner-v4-db-password", func(fileMap map[string][]byte) error {
		fileMap["password"] = []byte("letmein")
		return nil
	})

	if err != nil {
		return err
	}

	err = m.applyTlsSecret(ctx, "scanner-v4-db-tls", func(fileMap map[string][]byte) error {
		if err := certgen.IssueOtherServiceCerts(fileMap, m.CA, []mtls.Subject{mtls.ScannerV4DBSubject}, mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing scanner DB certificate: %w\n", err)
		}
		return nil
	})

	return err
}

func (m *manifestGenerator) applyScannerV4DbDeployment(ctx context.Context) error {
	// apply affinity rules
	// init container - init-entrypoint.sh - try to unwrap that
	// volumes:
	// - name: disk
	// persistentVolumeClaim:
	// claimName: scanner-v4-db
	// - configMap:
	// defaultMode: 420
	// name: scanner-v4-db-config
	// name: config-volume
	// - name: certs
	// secret:
	// defaultMode: 416
	// items:
	// - key: cert.pem
	// path: server.crt
	// - key: key.pem
	// path: server.key
	// - key: ca.pem
	// path: root.crt
	// secretName: scanner-v4-db-tls
	// - emptyDir:
	// medium: Memory
	// sizeLimit: 750Mi
	// name: shared-memory
	// - name: scanner-v4-db-password
	// secret:
	// defaultMode: 420
	// secretName: scanner-v4-db-password

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
						Image:           "quay.io/stackrox-io/scanner-v4-db:4.8.x-92-g99b84f31ac-amd64",
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
						Image:           "quay.io/stackrox-io/scanner-v4-db:4.8.x-92-g99b84f31ac-amd64",
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
	_, err := m.Client.AppsV1().Deployments(m.Config.Namespace).Create(ctx, &deployment, metav1.CreateOptions{})

	if errors.IsAlreadyExists(err) {
		_, err = m.Client.AppsV1().Deployments(m.Config.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		log.Info("Updated scanner-v4 deployment")
	} else {
		log.Info("Created scanner-v4 deployment")
	}

	return err
}

func (m *manifestGenerator) applyScannerV4Deployment(ctx context.Context, name string, replicaCount int32) error {
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
						Image:           m.Config.Images.Stackrox,
						ImagePullPolicy: v1.PullAlways,
						Command: []string{
							"sh",
							"-c",
							"restore-all-dir-contents && import-additional-cas",
						},
					}},
					Containers: []v1.Container{{
						Name:  name,
						Image: m.Config.Images.Stackrox,
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

	_, err := m.Client.AppsV1().Deployments(m.Config.Namespace).Create(ctx, &deployment, metav1.CreateOptions{})

	if errors.IsAlreadyExists(err) {
		_, err = m.Client.AppsV1().Deployments(m.Config.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		log.Infof("Updated scanner-v4-%s deployment", name)
	} else {
		log.Infof("Created scanner-v4-%s deployment", name)
	}

	return err
}

func (m *manifestGenerator) applyScannerV4Services(ctx context.Context) error {
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

	if err := m.applyService(ctx, "scanner-v4-matcher", ports); err != nil {
		return err
	}

	if err := m.applyService(ctx, "scanner-v4-indexer", ports); err != nil {
		return err
	}

	return m.applyService(ctx, "scanner-v4-db", []v1.ServicePort{{
		Name:       "tcp-db",
		Port:       5432,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromInt(5432),
	}})
}
