//go:build test_e2e

package tests

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stretchr/testify/suite"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configControllerNs               = namespaces.StackRox
	configControllerProxyNs          = "qa-config-controller-ca" // Must match the certificate X509v3 Subject Alternative Name
	configControllerDeployment       = "config-controller"
	configControllerContainer        = "manager"
	configControllerCentralEndpoint  = "ROX_CENTRAL_ENDPOINT"
	configControllerProxySecretName  = "quay" //nolint:gosec // G101
	configControllerAdditionalCAName = "additional-ca"
)

// Embed certificates from config-controller-ca directory.
// These certs have SANs for *.qa-config-controller-ca namespace.
//
//go:embed "config-controller-ca/root-ca.crt"
var configControllerAdditionalCA []byte

//go:embed "config-controller-ca/server.crt"
var configControllerProxyCert []byte

//go:embed "config-controller-ca/server.key"
var configControllerProxyKey []byte

func TestConfigControllerAdditionalCA(t *testing.T) {
	suite.Run(t, new(ConfigControllerCASuite))
}

type ConfigControllerCASuite struct {
	KubernetesSuite
	ctx                     context.Context
	cleanupCtx              context.Context
	cancel                  func()
	originalCentralEndpoint string
	hadAdditionalCASecret   bool
}

func (s *ConfigControllerCASuite) SetupSuite() {
	s.KubernetesSuite.SetupSuite()
	s.ctx, s.cleanupCtx, s.cancel = testContexts(s.T(), "TestConfigControllerAdditionalCA", 10*time.Minute)

	// Check if config-controller is running before test
	s.waitUntilK8sDeploymentReady(s.ctx, configControllerNs, configControllerDeployment)

	// Save original central endpoint (if any)
	s.logf("Gathering original central endpoint value from config-controller...")
	var err error
	s.originalCentralEndpoint, err = s.getDeploymentEnvVal(s.ctx, configControllerNs, configControllerDeployment, configControllerContainer, configControllerCentralEndpoint)
	requireNoErrorOrEnvVarNotFound(s.T(), err)
	s.logf("Original value is %q. (Will restore this value on cleanup.)", s.originalCentralEndpoint)

	// Check if additional-ca secret already exists
	_, err = s.k8s.CoreV1().Secrets(configControllerNs).Get(s.ctx, configControllerAdditionalCAName, metaV1.GetOptions{})
	s.hadAdditionalCASecret = err == nil

	s.setupProxy()
	s.createAdditionalCASecret()
	s.restartConfigController()
}

func (s *ConfigControllerCASuite) TearDownSuite() {
	s.cleanupProxy(s.cleanupCtx)
	s.cleanupAdditionalCASecret(s.cleanupCtx)

	// Restore original central endpoint
	if s.originalCentralEndpoint != "" {
		_ = s.mustSetDeploymentEnvVal(s.cleanupCtx, configControllerNs, configControllerDeployment, configControllerContainer, configControllerCentralEndpoint, s.originalCentralEndpoint)
	} else {
		s.mustDeleteDeploymentEnvVar(s.cleanupCtx, configControllerNs, configControllerDeployment, configControllerCentralEndpoint)
	}

	// Restart to restore normal operation
	s.restartConfigController()
	s.waitUntilK8sDeploymentReady(s.cleanupCtx, configControllerNs, configControllerDeployment)

	s.cancel()
}

func (s *ConfigControllerCASuite) TestConfigControllerConnectsThroughProxyWithAdditionalCA() {
	// The proxy is deployed in qa-config-controller-ca namespace using certs
	// with SAN for *.qa-config-controller-ca
	const (
		proxyServiceName = "nginx-proxy"
		proxyEndpoint    = proxyServiceName + "." + configControllerProxyNs + ":443"
	)

	s.logf("Pointing config-controller at the proxy...")
	patchedDeploy := s.mustSetDeploymentEnvVal(s.ctx, configControllerNs, configControllerDeployment, configControllerContainer, configControllerCentralEndpoint, proxyEndpoint)
	s.waitUntilK8sDeploymentGenerationReady(s.ctx, configControllerNs, configControllerDeployment, patchedDeploy.GetGeneration())
	s.logf("Config-controller will now attempt connecting via the nginx proxy.")

	// Wait for config-controller to successfully connect through the proxy
	// The manager logs "Starting manager" when it successfully initializes
	s.waitUntilLog(s.ctx, configControllerNs,
		map[string]string{"app": "config-controller"},
		configControllerContainer,
		"show successful initialization",
		containsLineMatching(regexp.MustCompile(`(Starting manager|Reconciler started|successfully connected)`)),
	)

	// Verify config-controller is healthy (readiness probe passes)
	s.waitUntilK8sDeploymentReady(s.ctx, configControllerNs, configControllerDeployment)
	s.logf("Config-controller successfully connected through proxy with additional CA")
}

func (s *ConfigControllerCASuite) setupProxy() {
	// Deploy nginx proxy in qa-config-controller-ca namespace using certs
	// with SANs for *.qa-config-controller-ca
	proxyNs := configControllerProxyNs
	name := "nginx-proxy"
	nginxLabels := map[string]string{"app": "nginx-proxy"}
	nginxTLSSecretName := "nginx-tls-conf" //nolint:gosec // G101
	nginxConfigName := "nginx-proxy-conf"

	s.logf("Setting up nginx proxy in namespace %q...", proxyNs)
	s.createProxyNamespace(proxyNs)
	s.installProxyImagePullSecret(proxyNs)
	s.createProxyTLSSecret(proxyNs, nginxTLSSecretName)
	s.createProxyConfigMap(proxyNs, nginxConfigName)
	s.createService(s.ctx, proxyNs, name, nginxLabels, map[int32]int32{443: 8443})
	s.createProxyDeployment(proxyNs, name, nginxLabels, nginxConfigName, nginxTLSSecretName)
	s.waitUntilK8sDeploymentReady(s.ctx, proxyNs, name)
	s.logf("Nginx proxy is now set up in namespace %q.", proxyNs)
}

func (s *ConfigControllerCASuite) createProxyNamespace(proxyNs string) {
	_, err := s.k8s.CoreV1().Namespaces().Create(s.ctx, &v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: proxyNs}}, metaV1.CreateOptions{})
	if apiErrors.IsAlreadyExists(err) {
		return
	}
	s.Require().NoError(err, "cannot create proxy namespace %q", proxyNs)
}

func (s *ConfigControllerCASuite) installProxyImagePullSecret(proxyNs string) {
	configBytes, err := json.Marshal(config.DockerConfigJSON{
		Auths: map[string]config.DockerConfigEntry{
			"https://quay.io": {
				Username: mustGetEnv(s.T(), "REGISTRY_USERNAME"),
				Password: mustGetEnv(s.T(), "REGISTRY_PASSWORD"),
			},
		},
	})
	s.Require().NoError(err, "cannot serialize docker config for image pull secret %q in namespace %q", configControllerProxySecretName, proxyNs)
	s.ensureSecretExists(s.ctx, proxyNs, configControllerProxySecretName, v1.SecretTypeDockerConfigJson, map[string][]byte{v1.DockerConfigJsonKey: configBytes})
}

func (s *ConfigControllerCASuite) createProxyTLSSecret(proxyNs, nginxTLSSecretName string) {
	var certChain []byte
	certChain = append(certChain, configControllerProxyCert...)
	certChain = append(certChain, configControllerAdditionalCA...)
	s.ensureSecretExists(s.ctx, proxyNs, nginxTLSSecretName, v1.SecretTypeTLS, map[string][]byte{
		v1.TLSCertKey:       certChain,
		v1.TLSPrivateKeyKey: configControllerProxyKey,
	})
}

func (s *ConfigControllerCASuite) createProxyConfigMap(proxyNs, nginxConfigName string) {
	// The proxy forwards gRPC traffic to the real Central
	centralEndpoint := fmt.Sprintf("central.%s.svc:443", configControllerNs)
	const nginxConfigTmpl = `
server {
    listen 8443 ssl http2;

    ssl_certificate     /run/secrets/tls/tls.crt;
    ssl_certificate_key /run/secrets/tls/tls.key;

    proxy_temp_path       /tmp/nginx_proxy_temp;
    client_body_temp_path /tmp/nginx_client_temp;
    fastcgi_temp_path     /tmp/nginx_fastcgi;
    uwsgi_temp_path       /tmp/nginx_uwsgi;
    scgi_temp_path        /tmp/nginx_scgi;

    location / {
        client_max_body_size 50M;
        grpc_pass grpcs://%s;
        grpc_ssl_verify off;
	}
}
`
	s.ensureConfigMapExists(s.ctx, proxyNs, nginxConfigName, map[string]string{
		"nginx-proxy-grpc-tls.conf": fmt.Sprintf(nginxConfigTmpl, centralEndpoint),
	})
}

func (s *ConfigControllerCASuite) createProxyDeployment(proxyNs, name string, nginxLabels map[string]string, nginxConfigName, nginxTLSSecretName string) {
	d := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   name,
			Labels: nginxLabels,
		},
		Spec: appsV1.DeploymentSpec{
			Selector: &metaV1.LabelSelector{
				MatchLabels: nginxLabels,
			},
			MinReadySeconds: 15,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: nginxLabels,
				},
				Spec: v1.PodSpec{
					ImagePullSecrets: []v1.LocalObjectReference{
						{Name: configControllerProxySecretName},
					},
					Containers: []v1.Container{
						{
							Image: "quay.io/rhacs-eng/qa-multi-arch:nginx-1-17-1",
							Name:  "nginx-proxy",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/etc/nginx/conf.d/",
								},
								{
									Name:      "tls",
									ReadOnly:  true,
									MountPath: "/run/secrets/tls",
								},
								{
									Name:      "varrun",
									MountPath: "/var/run",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: nginxConfigName,
									},
								},
							},
						},
						{
							Name: "tls",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: nginxTLSSecretName,
								},
							},
						},
						{
							Name: "varrun",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	_, err := s.k8s.AppsV1().Deployments(proxyNs).Create(s.ctx, d, metaV1.CreateOptions{})
	if apiErrors.IsAlreadyExists(err) {
		return
	}
	s.Require().NoError(err, "cannot create deployment %q in namespace %q", name, proxyNs)
}

func (s *ConfigControllerCASuite) createAdditionalCASecret() {
	s.logf("Creating additional-ca secret in namespace %q...", configControllerNs)
	s.ensureSecretExists(s.ctx, configControllerNs, configControllerAdditionalCAName, v1.SecretTypeOpaque, map[string][]byte{
		"custom-ca.crt": configControllerAdditionalCA,
	})
}

func (s *ConfigControllerCASuite) restartConfigController() {
	s.logf("Restarting config-controller to pick up additional CA...")
	err := s.k8s.CoreV1().Pods(configControllerNs).DeleteCollection(s.ctx,
		metaV1.DeleteOptions{},
		metaV1.ListOptions{LabelSelector: "app=config-controller"})
	s.Require().NoError(err, "cannot delete config-controller pods")

	// Wait for new pod to be ready
	s.waitUntilK8sDeploymentReady(s.ctx, configControllerNs, configControllerDeployment)
}

func (s *ConfigControllerCASuite) cleanupProxy(ctx context.Context) {
	proxyNs := configControllerProxyNs
	if s.T().Failed() {
		s.logf("Test failed. Collecting k8s artifacts before cleanup.")
		collectLogs(s.T(), configControllerNs, "config-controller-ca-failure")
		collectLogs(s.T(), proxyNs, "config-controller-ca-failure")
	}
	s.logf("Cleaning up nginx proxy in namespace %q...", proxyNs)
	err := s.k8s.CoreV1().Namespaces().Delete(ctx, proxyNs, metaV1.DeleteOptions{})
	if apiErrors.IsNotFound(err) {
		return
	}
	s.Require().NoError(err, "cannot delete proxy namespace %q", proxyNs)
}

func (s *ConfigControllerCASuite) cleanupAdditionalCASecret(ctx context.Context) {
	// Only delete the secret if it didn't exist before the test
	if s.hadAdditionalCASecret {
		s.logf("Keeping existing additional-ca secret in namespace %q", configControllerNs)
		return
	}
	s.logf("Cleaning up additional-ca secret in namespace %q...", configControllerNs)
	err := s.k8s.CoreV1().Secrets(configControllerNs).Delete(ctx, configControllerAdditionalCAName, metaV1.DeleteOptions{})
	if apiErrors.IsNotFound(err) {
		return
	}
	s.Require().NoError(err, "cannot delete additional-ca secret in namespace %q", configControllerNs)
}
