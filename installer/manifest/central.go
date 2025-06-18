package manifest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/renderer"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CentralGenerator struct{}

func (g CentralGenerator) Name() string {
	return "Central"
}

func (g CentralGenerator) Exportable() bool {
	return true
}

func (g CentralGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	adminPass, err := genTlsSecret("admin-pass", m.CA, func(fileMap map[string][]byte) error {
		fileMap["password"] = []byte("letmein")
		return nil
	})
	if err != nil {
		return nil, err
	}

	htpasswd, err := genTlsSecret("central-htpasswd", m.CA, func(fileMap map[string][]byte) error {
		htpasswdBytes, err := renderer.CreateHtpasswd("letmein")
		if err != nil {
			return err
		}

		fileMap["htpasswd"] = htpasswdBytes
		return nil
	})
	if err != nil {
		return nil, err
	}

	tlsSecret, err := genTlsSecret("central-tls", m.CA, func(fileMap map[string][]byte) error {
		if err := certgen.IssueCentralCert(fileMap, m.CA, mtls.WithNamespace(m.Config.Namespace)); err != nil {
			return fmt.Errorf("issuing central service certificate: %w\n", err)
		}

		jwtKey, err := certgen.GenerateJWTSigningKey()
		if err != nil {
			return fmt.Errorf("generating JWT signing key: %w\n", err)
		}

		certgen.AddJWTSigningKeyToFileMap(fileMap, jwtKey)

		fileMap["ca-key.pem"] = m.CA.KeyPEM()
		return nil
	})
	if err != nil {
		return nil, err
	}

	svc := genService("central", []v1.ServicePort{{
		Name:       "https",
		Port:       443,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromString("api"),
	}})

	return []Resource{
		genServiceAccount("central"),
		adminPass,
		htpasswd,
		tlsSecret,
		genRole("view-gcp-cloud-credentials", []rbacv1.PolicyRule{{
			APIGroups:     []string{""},
			Verbs:         []string{"list", "watch", "get"},
			Resources:     []string{"secrets"},
			ResourceNames: []string{"gcp-cloud-credentials"},
		}}),
		genRoleBinding("central", "view-gcp-cloud-credentials", m.Config.Namespace),
		genRoleBinding("central", "use-nonroot-v2-scc", m.Config.Namespace),
		g.createCentralEndpointsConfig(),
		g.createExternalDBConfig(),
		g.createCentralConfig(),
		svc,
		g.createCentralDeployment(m),
	}, nil
}

func (g *CentralGenerator) createCentralConfig() Resource {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"central-config.yaml": `maintenance:
  safeMode: false # When set to true, Central will sleep forever on the next restart
  compaction:
    enabled: true
    bucketFillFraction: .5 # This controls how densely to compact the buckets. Usually not advised to modify
    freeFractionThreshold: 0.75 # This is the threshold for free bytes / total bytes after which compaction will occur
  forceRollbackVersion: none # This is the config and target rollback version after upgrade complete.`,
		},
	}
	cm.SetName("central-config")
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return Resource{
		Object:       &cm,
		Name:         cm.Name,
		IsUpdateable: true,
	}
}

func (g *CentralGenerator) createExternalDBConfig() Resource {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"central-external-db.yaml": `centralDB:
   external: false
   source: >
     host=central-db
     port=5432
     user=postgres
     sslmode=verify-ca
     sslrootcert=/run/secrets/stackrox.io/certs/ca.pem
     statement_timeout=1.2e+06
     pool_min_conns=10
     pool_max_conns=90
     client_encoding=UTF8`,
		},
	}
	cm.SetName("central-external-db")
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return Resource{
		Object:       &cm,
		Name:         "central-external-db",
		IsUpdateable: true,
	}
}

func (g *CentralGenerator) createCentralEndpointsConfig() Resource {
	cm := v1.ConfigMap{
		Data: map[string]string{
			"endpoints.yaml": "",
		},
	}
	cm.SetName("central-endpoints")
	cm.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return Resource{
		Object:       &cm,
		Name:         cm.Name,
		IsUpdateable: true,
	}
}

func (g *CentralGenerator) createCentralDeployment(m *manifestGenerator) Resource {
	deployment := apps.Deployment{
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "central",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "central",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "central",
					InitContainers: []v1.Container{{
						Name:            "add-additional-cas",
						Image:           m.Config.Images.Central,
						ImagePullPolicy: v1.PullAlways,
						Command: []string{
							"sh",
							"-c",
							"restore-all-dir-contents && import-additional-cas",
						},
					}, {
						Name:            "migrator",
						Image:           m.Config.Images.Central,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"/stackrox/bin/migrator"},
					}},
					Containers: []v1.Container{{
						Name:            "central",
						Image:           m.Config.Images.Central,
						ImagePullPolicy: v1.PullAlways,
						Command:         []string{"sh", "-c", "while true; do /stackrox/central; done"},
						Ports: []v1.ContainerPort{{
							Name:          "api",
							ContainerPort: 8443,
							Protocol:      v1.ProtocolTCP,
						}},
						Env: []v1.EnvVar{
							{
								Name:  "ROX_HOTRELOAD",
								Value: strconv.FormatBool(m.Config.DevMode),
							}, {
								Name:  "ROX_DEVELOPMENT_BUILD",
								Value: strconv.FormatBool(m.Config.DevMode),
							}, {
								Name: "POD_NAMESPACE",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							}, {
								Name: "ROX_NAMESPACE",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
					}, {
						Name:            "nodejs",
						Image:           m.Config.Images.Central,
						ImagePullPolicy: v1.PullAlways,
						Command: []string{
							"sh",
							"-c",
							"cd /ui; npm run start",
						},
					}},
				},
			},
		},
	}

	if m.Config.ScannerV4 {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "ROX_SCANNER_V4",
			Value: "true",
		})
	}

	trueBool := true
	volumeMounts := []VolumeDefAndMount{
		{
			Name:      "varlog",
			MountPath: "/var/log/stackrox/",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "central-tmp-volume",
			MountPath: "/tmp",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "central-etc-ssl-volume",
			MountPath: "/etc/ssl",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "central-etc-pki-volume",
			MountPath: "/etc/pki/ca-trust",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "central-certs-volume",
			MountPath: "/run/secrets/stackrox.io/certs/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						DefaultMode: &ReadOnlyMode,
						SecretName:  "central-tls",
					},
				},
			},
		},
		{
			Name:      "central-default-tls-cert-volume",
			MountPath: "/run/secrets/stackrox.io/default-tls-cert/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "central-default-tls-cert",
						Optional:   &trueBool,
					},
				},
			},
		},
		{
			Name:      "central-htpasswd-volume",
			MountPath: "/run/secrets/stackrox.io/htpasswd/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "central-htpasswd",
						Optional:   &trueBool,
					},
				},
			},
		},
		{
			Name:      "central-jwt-volume",
			MountPath: "/run/secrets/stackrox.io/jwt/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "central-tls",
						Items: []v1.KeyToPath{
							{
								Key:  "jwt-key.pem",
								Path: "jwt-key.pem",
							},
						},
					},
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
						SecretName: "additional-ca",
						Optional:   &trueBool,
					},
				},
			},
		},
		{
			Name:      "central-license-volume",
			MountPath: "/run/secrets/stackrox.io/central-license/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "central-license",
						Optional:   &trueBool,
					},
				},
			},
		},
		{
			Name:      "central-config-volume",
			MountPath: "/etc/stackrox",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "central-config",
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
			Name:      "endpoints-config-volume",
			MountPath: "/etc/stackrox.d/endpoints/",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "central-endpoints",
						},
					},
				},
			},
		},
		{
			Name:      "central-db-password",
			MountPath: "/run/secrets/stackrox.io/db-password",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: "central-db-password",
					},
				},
			},
		},
		{
			Name:      "stackrox-db",
			MountPath: "/var/lib/stackrox",
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			Name:      "central-external-db-volume",
			MountPath: "/etc/ext-db",
			ReadOnly:  true,
			Volume: v1.Volume{
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						DefaultMode: &ReadOnlyMode,
						LocalObjectReference: v1.LocalObjectReference{
							Name: "central-external-db",
						},
					},
				},
			},
		},
	}

	for _, v := range volumeMounts {
		v.Apply(&deployment.Spec.Template.Spec.Containers[0], &deployment.Spec.Template.Spec)
		v.Apply(&deployment.Spec.Template.Spec.InitContainers[0], nil)
		v.Apply(&deployment.Spec.Template.Spec.InitContainers[1], nil)
	}

	deployment.SetName("central")
	deployment.SetGroupVersionKind(apps.SchemeGroupVersion.WithKind("Deployment"))

	return Resource{
		Object:       &deployment,
		Name:         deployment.Name,
		IsUpdateable: true,
	}
}

func init() {
	central = append(central, CentralGenerator{})
}
