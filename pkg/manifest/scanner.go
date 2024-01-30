package manifest

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (m ManifestGenerator) applyScanner(ctx context.Context) error {
	if err := m.createScannerConfig(ctx); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create central config: %w\n", err)
	}
	log.Info("Created central config")

	if err := m.createScannerTlsSecrets(ctx); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create TLS secret: %w\n", err)
	}
	log.Info("Created scanner TLS secrets")

	if err := m.applyScannerDbDeployment(ctx); err != nil {
		return err
	}

	if err := m.applyScannerDeployment(ctx); err != nil {
		return err
	}

	if err := m.applyScannerServices(ctx); err != nil {
		return err
	}

	return nil
}

func (m ManifestGenerator) createScannerConfig(ctx context.Context) error {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"config.yaml": `# Configuration file for scanner.
scanner:
  centralEndpoint: https://central.stackrox.svc
  sensorEndpoint: https://sensor.stackrox.svc
  database:
    # Database driver
    type: pgsql
    options:
      # PostgreSQL Connection string
      # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
      source: host=scanner-db.stackrox.svc port=5432 user=postgres sslmode=verify-full statement_timeout=60000

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

  exposeMonitoring: false`,
		},
	}
	cm.SetName("scanner-config")
	_, err := m.Client.CoreV1().ConfigMaps(m.Namespace).Create(ctx, &cm, metav1.CreateOptions{})

	return err
}

func (m ManifestGenerator) createScannerTlsSecrets(ctx context.Context) error {
	var secret v1.Secret
	var err error

	apply := func() error {
		_, err = m.Client.CoreV1().Secrets(m.Namespace).Create(ctx, &secret, metav1.CreateOptions{})

		if errors.IsAlreadyExists(err) {
			_, err = m.Client.CoreV1().Secrets(m.Namespace).Update(ctx, &secret, metav1.UpdateOptions{})
			log.Infof("Updated secret %s", secret.GetName())
		} else {
			log.Infof("Created secret %s", secret.GetName())
		}

		return err
	}

	// scanner

	fileMap := make(types.SecretDataMap)
	certgen.AddCAToFileMap(fileMap, m.CA)
	if err := certgen.IssueScannerCerts(fileMap, m.CA, mtls.WithNamespace(m.Namespace)); err != nil {
		return fmt.Errorf("issuing central service certificate: %w\n", err)
	}

	secret = v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "scanner-tls",
		},
		Data: fileMap,
	}

	if err := apply(); err != nil {
		return err
	}

	// password

	secret = v1.Secret{
		StringData: map[string]string{
			"password": "letmein",
		},
	}
	secret.SetName("scanner-db-password")

	if err := apply(); err != nil {
		return err
	}

	// scanner-db

	fileMap = make(types.SecretDataMap)
	certgen.AddCAToFileMap(fileMap, m.CA)

	if err := certgen.IssueOtherServiceCerts(fileMap, m.CA, []mtls.Subject{mtls.ScannerDBSubject}, mtls.WithNamespace(m.Namespace)); err != nil {
		return fmt.Errorf("issuing scanner DB certificate: %w\n", err)
	}

	secret = v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "scanner-db-tls",
		},
		Data: fileMap,
	}

	if err := apply(); err != nil {
		return err
	}

	return nil
}

func (m ManifestGenerator) applyScannerDbDeployment(ctx context.Context) error {
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
					Containers: []v1.Container{{
						Name:  "db",
						Image: "quay.io/stackrox-io/scanner-db:4.3.4",
						SecurityContext: &v1.SecurityContext{
							RunAsUser:  &PostgresUser,
							RunAsGroup: &PostgresUser,
						},
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
						Name:  "init-db",
						Image: "quay.io/stackrox-io/scanner-db:4.3.4",
						SecurityContext: &v1.SecurityContext{
							RunAsUser:  &PostgresUser,
							RunAsGroup: &PostgresUser,
						},
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
	_, err := m.Client.AppsV1().Deployments(m.Namespace).Create(ctx, &deployment, metav1.CreateOptions{})

	if errors.IsAlreadyExists(err) {
		_, err = m.Client.AppsV1().Deployments(m.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		log.Info("Updated central deployment")
	} else {
		log.Info("Created central deployment")
	}

	return err
}

func (m ManifestGenerator) applyScannerDeployment(ctx context.Context) error {
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
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: &ScannerUser,
					},
					Containers: []v1.Container{{
						Name:    "scanner",
						Image:   "quay.io/stackrox-io/scanner:4.3.4",
						Command: []string{"/entrypoint.sh"},
						SecurityContext: &v1.SecurityContext{
							RunAsUser:  &ScannerUser,
							RunAsGroup: &ScannerUser,
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
	}

	deployment.SetName("scanner")

	_, err := m.Client.AppsV1().Deployments(m.Namespace).Create(ctx, &deployment, metav1.CreateOptions{})

	if errors.IsAlreadyExists(err) {
		_, err = m.Client.AppsV1().Deployments(m.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		log.Info("Updated scanner deployment")
	} else {
		log.Info("Created scanner deployment")
	}

	return err
}

func (m ManifestGenerator) applyScannerServices(ctx context.Context) error {
	// scanner

	svc := v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "scanner",
			},
			Ports: []v1.ServicePort{{
				Name:       "grpcs-scanner",
				Port:       8443,
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt(8443),
			}, {
				Name:       "https-scanner",
				Port:       8080,
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}

	svc.SetName("scanner")

	_, err := m.Client.CoreV1().Services(m.Namespace).Create(ctx, &svc, metav1.CreateOptions{})

	if errors.IsAlreadyExists(err) {
		_, err = m.Client.CoreV1().Services(m.Namespace).Update(ctx, &svc, metav1.UpdateOptions{})
		log.Info("Updated scanner service")
	} else {
		log.Info("Created scanner service")
	}

	if err != nil {
		return err
	}

	// scanner-db

	svc = v1.Service{
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "scanner-db",
			},
			Ports: []v1.ServicePort{{
				Name:       "tcp-db",
				Port:       5432,
				Protocol:   v1.ProtocolTCP,
				TargetPort: intstr.FromInt(5432),
			}},
		},
	}

	svc.SetName("scanner-db")

	_, err = m.Client.CoreV1().Services(m.Namespace).Create(ctx, &svc, metav1.CreateOptions{})

	if errors.IsAlreadyExists(err) {
		_, err = m.Client.CoreV1().Services(m.Namespace).Update(ctx, &svc, metav1.UpdateOptions{})
		log.Info("Updated scanner-db service")
	} else {
		log.Info("Created scanner-db service")
	}

	return err
}
