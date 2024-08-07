//go:build test_e2e

package tests

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stretchr/testify/suite"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	s                  = namespaces.StackRox // for brevity
	proxyNs            = "qa-tls-challenge"  // Must match the additionalCA X509v3 Subject Alternative Name
	sensorDeployment   = "sensor"
	sensorContainer    = "sensor"
	centralEndpointVar = "ROX_CENTRAL_ENDPOINT"
)

//go:embed "bad-ca/root.crt"
var additionalCA []byte

//go:embed "bad-ca/nginx-loadbalancer.qa-tls-challenge.crt"
var leafCert []byte

//go:embed "bad-ca/nginx-loadbalancer.qa-tls-challenge.key"
var leafKey []byte

func TestTLSChallenge(t *testing.T) {
	suite.Run(t, new(TLSChallengeSuite))
}

type TLSChallengeSuite struct {
	KubernetesSuite
	ctx                     context.Context
	cleanupCtx              context.Context
	cancel                  func()
	originalCentralEndpoint string
}

func (ts *TLSChallengeSuite) SetupSuite() {
	ts.KubernetesSuite.SetupSuite()
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(ts.T(), "TestTLSChallenge", 20*time.Minute)

	// Check sanity before test.
	waitUntilCentralSensorConnectionIs(ts.T(), ts.ctx, storage.ClusterHealthStatus_HEALTHY)

	ts.logf("Gathering original central endpoint value from sensor...")
	ts.originalCentralEndpoint = ts.getDeploymentEnvVal(ts.ctx, s, sensorDeployment, sensorContainer, centralEndpointVar)
	ts.logf("Original value is %q. (Will restore this value on cleanup.)", ts.originalCentralEndpoint)

	ts.setupProxy(proxyNs, ts.originalCentralEndpoint)
}

func (ts *TLSChallengeSuite) TearDownSuite() {
	ts.cleanupProxy(ts.cleanupCtx, proxyNs)
	if ts.originalCentralEndpoint != "" {
		ts.setDeploymentEnvVal(ts.cleanupCtx, s, sensorDeployment, sensorContainer, centralEndpointVar, ts.originalCentralEndpoint)
	}
	// Check sanity after test.
	waitUntilCentralSensorConnectionIs(ts.T(), ts.cleanupCtx, storage.ClusterHealthStatus_HEALTHY)
	ts.cancel()
}

func (ts *TLSChallengeSuite) TestTLSChallenge() {
	const (
		proxyServiceName = "nginx-loadbalancer"
		proxyEndpoint    = proxyServiceName + "." + proxyNs + ":443"
	)

	ts.logf("Pointing sensor at the proxy...")
	ts.setDeploymentEnvVal(ts.ctx, s, sensorDeployment, sensorContainer, centralEndpointVar, proxyEndpoint)
	ts.logf("Sensor will now attempt connecting via the nginx proxy.")

	ts.waitUntilLog(ts.ctx, s, map[string]string{"app": "sensor"}, sensorContainer, "contain info about successful connection",
		containsLineMatching(regexp.MustCompile("Info: Add central CA cert with CommonName: 'Custom Root'")),
		containsLineMatching(regexp.MustCompile("Info: Connecting to Central server "+proxyEndpoint)),
		containsLineMatching(regexp.MustCompile("Info: Established connection to Central.")),
		containsLineMatching(regexp.MustCompile("Info: Communication with central started.")),
	)
	waitUntilCentralSensorConnectionIs(ts.T(), ts.ctx, storage.ClusterHealthStatus_HEALTHY)
}

func (ts *TLSChallengeSuite) setupProxy(proxyNs string, centralEndpoint string) {
	name := "nginx-loadbalancer"
	nginxLabels := map[string]string{"app": "nginx"}
	nginxTLSSecretName := "nginx-tls-conf" //nolint:gosec // G101
	nginxConfigName := "nginx-proxy-conf"
	ts.logf("Setting up nginx proxy in namespace %q...", proxyNs)
	ts.createProxyNamespace(proxyNs)
	ts.installImagePullSecret(proxyNs)
	ts.createProxyTLSSecret(proxyNs, nginxTLSSecretName)
	ts.createProxyConfigMap(proxyNs, centralEndpoint, nginxConfigName)
	ts.createService(ts.ctx, proxyNs, name, nginxLabels, map[int32]int32{443: 8443})
	ts.createProxyDeployment(proxyNs, name, nginxLabels, nginxConfigName, nginxTLSSecretName)
	ts.logf("Nginx proxy is now set up in namespace %q.", proxyNs)
}

func (ts *TLSChallengeSuite) createProxyNamespace(proxyNs string) {
	_, err := ts.k8s.CoreV1().Namespaces().Create(ts.ctx, &v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: proxyNs}}, metaV1.CreateOptions{})
	if apiErrors.IsAlreadyExists(err) {
		return
	}
	ts.Require().NoError(err, "cannot create proxy namespace %q", proxyNs)
}

func (ts *TLSChallengeSuite) installImagePullSecret(proxyNs string) {
	configBytes, err := json.Marshal(config.DockerConfigJSON{
		Auths: map[string]config.DockerConfigEntry{
			"https://quay.io": {
				Username: os.Getenv("REGISTRY_USERNAME"),
				Password: os.Getenv("REGISTRY_PASSWORD"),
			},
		},
	})
	secretName := "quay"
	ts.Require().NoError(err, "cannot serialize docker config for image pull secret %q in namespace %q", secretName, proxyNs)
	ts.ensureSecretExists(ts.ctx, proxyNs, secretName, v1.SecretTypeDockerConfigJson, map[string][]byte{v1.DockerConfigJsonKey: configBytes})
	patch := []byte(fmt.Sprintf(`{"imagePullSecrets":[{"name":%q}]}`, secretName))
	_, err = ts.k8s.CoreV1().ServiceAccounts(proxyNs).Patch(ts.ctx, "default", types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
	ts.Require().NoError(err, "cannot patch service account %q in namespace %q", "default", proxyNs)
}

func (ts *TLSChallengeSuite) createProxyTLSSecret(proxyNs string, nginxTLSSecretName string) {
	var certChain []byte
	certChain = append(certChain, leafCert...)
	certChain = append(certChain, additionalCA...)
	ts.ensureSecretExists(ts.ctx, proxyNs, nginxTLSSecretName, v1.SecretTypeTLS, map[string][]byte{
		v1.TLSCertKey:       certChain,
		v1.TLSPrivateKeyKey: leafKey,
	})
}

func (ts *TLSChallengeSuite) createProxyConfigMap(proxyNs string, centralEndpoint string, nginxConfigName string) {
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
	ts.ensureConfigMapExists(ts.ctx, proxyNs, nginxConfigName, map[string]string{
		"nginx-proxy-grpc-tls.conf": fmt.Sprintf(nginxConfigTmpl, centralEndpoint),
	})
}

func (ts *TLSChallengeSuite) createProxyDeployment(proxyNs string, name string, nginxLabels map[string]string, nginxConfigName string, nginxTLSSecretName string) {
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
					Containers: []v1.Container{
						{
							Image: "quay.io/rhacs-eng/qa-multi-arch:nginx-1-17-1",
							Name:  "nginx-loadbalancer",
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
	_, err := ts.k8s.AppsV1().Deployments(proxyNs).Create(ts.ctx, d, metaV1.CreateOptions{})
	ts.Require().NoError(err, "cannot create deployment %q in namespace %q", name, proxyNs)
}

func (ts *TLSChallengeSuite) cleanupProxy(ctx context.Context, proxyNs string) {
	if ts.T().Failed() {
		ts.logf("Test failed. Collecting k8s artifacts before cleanup.")
		collectLogs(ts.T(), namespaces.StackRox, "tls-challenge-failure")
		collectLogs(ts.T(), proxyNs, "tls-challenge-failure")
	}
	ts.logf("Cleaning up nginx proxy in namespace %q...", proxyNs)
	err := ts.k8s.CoreV1().Namespaces().Delete(ctx, proxyNs, metaV1.DeleteOptions{})
	if apiErrors.IsNotFound(err) {
		return
	}
	ts.Require().NoError(err, "cannot delete proxy namespace %q", proxyNs)
}
