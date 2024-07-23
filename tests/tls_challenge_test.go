//go:build test_e2e

package tests

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

//go:embed "bad-ca/root.crt"
var additionalCA []byte

//go:embed "bad-ca/nginx-loadbalancer.qa-tls-challenge.crt"
var leafCert []byte

//go:embed "bad-ca/nginx-loadbalancer.qa-tls-challenge.key"
var leafKey []byte

func TestTLSChallenge(t *testing.T) {
	const (
		s                  = namespaces.StackRox // for brevity
		sensorDeployment   = "sensor"
		sensorContainer    = "sensor"
		centralEndpointVar = "ROX_CENTRAL_ENDPOINT"
		proxyServiceName   = "nginx-loadbalancer"
		proxyNs            = "qa-tls-challenge" // Must match the additionalCA X509v3 Subject Alternative Name
		proxyEndpoint      = proxyServiceName + "." + proxyNs + ":443"
	)

	ctx, cleanupCtx, cancel := testContexts(t, "TestTLSChallenge", 20*time.Minute)
	defer cancel()

	// Check sanity before and after test.
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	defer waitUntilCentralSensorConnectionIs(t, cleanupCtx, storage.ClusterHealthStatus_HEALTHY)

	k8s := createK8sClient(t)

	logf(t, "Gathering original central endpoint value from sensor...")
	originalCentralEndpoint := getDeploymentEnvVal(t, ctx, k8s, s, sensorDeployment, sensorContainer, centralEndpointVar)
	logf(t, "Original value is %q. (Will restore this value on cleanup.)", originalCentralEndpoint)
	defer setDeploymentEnvVal(t, cleanupCtx, k8s, s, sensorDeployment, sensorContainer, centralEndpointVar, originalCentralEndpoint)

	setupProxy(t, ctx, k8s, proxyNs, originalCentralEndpoint)
	defer cleanupProxy(t, cleanupCtx, k8s, proxyNs)

	logf(t, "Pointing sensor at the proxy...")
	setDeploymentEnvVal(t, ctx, k8s, s, sensorDeployment, sensorContainer, centralEndpointVar, proxyEndpoint)
	logf(t, "Sensor will now attempt connecting via the nginx proxy.")

	waitUntilLog(t, ctx, k8s, s, map[string]string{"app": "sensor"}, "sensor", "contain info about successful connection",
		containsLineMatching(regexp.MustCompile("Info: Add central CA cert with CommonName: 'Custom Root'")),
		containsLineMatching(regexp.MustCompile("Info: Connecting to Central server "+proxyEndpoint)),
		containsLineMatching(regexp.MustCompile("Info: Established connection to Central.")),
		containsLineMatching(regexp.MustCompile("Info: Communication with central started.")),
	)
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
}

func setupProxy(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string, centralEndpoint string) {
	name := "nginx-loadbalancer"
	nginxLabels := map[string]string{"app": "nginx"}
	nginxTLSSecretName := "nginx-tls-conf"
	nginxConfigName := "nginx-proxy-conf"
	logf(t, "Setting up nginx proxy in namespace %q...", proxyNs)
	createProxyNamespace(t, ctx, k8s, proxyNs)
	installImagePullSecret(t, ctx, k8s, proxyNs)
	createProxyTLSSecret(t, ctx, k8s, proxyNs, nginxTLSSecretName)
	createProxyConfigMap(t, ctx, k8s, proxyNs, centralEndpoint, nginxConfigName)
	createService(t, ctx, k8s, proxyNs, name, nginxLabels, map[int32]int32{443: 8443})
	createProxyDeployment(t, ctx, k8s, proxyNs, name, nginxLabels, nginxConfigName, nginxTLSSecretName)
	logf(t, "Nginx proxy is now set up in namespace %q.", proxyNs)
}

func createProxyNamespace(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string) {
	_, err := k8s.CoreV1().Namespaces().Create(ctx, &v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: proxyNs}}, metaV1.CreateOptions{})
	if apiErrors.IsAlreadyExists(err) {
		return
	}
	require.NoError(t, err, "cannot create proxy namespace %q", proxyNs)
}

func installImagePullSecret(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string) {
	auth := fmt.Sprintf("%s:%s", os.Getenv("REGISTRY_USERNAME"), os.Getenv("REGISTRY_PASSWORD"))
	b64Auth := []byte(base64.StdEncoding.EncodeToString([]byte(auth)))
	data := map[string][]byte{
		v1.DockerConfigJsonKey: []byte(fmt.Sprintf(`{"auths":{%q:{"auth":%q}}}`, "https://quay.io", b64Auth)),
	}
	ensureSecretExists(t, ctx, k8s, proxyNs, "quay", v1.SecretTypeDockerConfigJson, data)

	patch := []byte(`{"imagePullSecrets":[{"name":"quay"}]}`)
	_, err := k8s.CoreV1().ServiceAccounts(proxyNs).Patch(ctx, namespaces.Default, types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
	require.NoError(t, err, "cannot patch service account %q in namespace %q", "default", proxyNs)
}

func createProxyTLSSecret(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string, nginxTLSSecretName string) {
	var certChain []byte
	certChain = append(certChain, leafCert...)
	certChain = append(certChain, additionalCA...)
	ensureSecretExists(t, ctx, k8s, proxyNs, nginxTLSSecretName, v1.SecretTypeTLS, map[string][]byte{
		v1.TLSCertKey:       certChain,
		v1.TLSPrivateKeyKey: leafKey,
	})
}

func createProxyConfigMap(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string, centralEndpoint string, nginxConfigName string) {
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
	ensureConfigMapExists(t, ctx, k8s, proxyNs, nginxConfigName, map[string]string{
		"nginx-proxy-grpc-tls.conf": fmt.Sprintf(nginxConfigTmpl, centralEndpoint),
	})
}

func createProxyDeployment(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string, name string, nginxLabels map[string]string, nginxConfigName string, nginxTLSSecretName string) {
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
	_, err := k8s.AppsV1().Deployments(proxyNs).Create(ctx, d, metaV1.CreateOptions{})
	require.NoError(t, err, "cannot create deployment %q in namespace %q", name, proxyNs)
}

func cleanupProxy(t *testing.T, ctx context.Context, k8s kubernetes.Interface, proxyNs string) {
	if t.Failed() {
		logf(t, "Test failed. Collecting k8s artifacts before cleanup.")
		collectLogs(t, namespaces.StackRox, "tls-challenge-failure")
		collectLogs(t, proxyNs, "tls-challenge-failure")
	}
	logf(t, "Cleaning up nginx proxy in namespace %q...", proxyNs)
	err := k8s.CoreV1().Namespaces().Delete(ctx, proxyNs, metaV1.DeleteOptions{})
	if apiErrors.IsNotFound(err) {
		return
	}
	require.NoError(t, err, "cannot delete proxy namespace %q", proxyNs)
}
