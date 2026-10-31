package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/mtls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultCertsSecret    = "central-tls"
	defaultCertsNamespace = "stackrox"
)

// certsSecretFileMap maps Kubernetes secret data keys to the file names
// expected by the mtls package. The boolean indicates whether the key is
// required for mTLS authentication.
var certsSecretFileMap = map[string]struct {
	fileName string
	required bool
}{
	"ca.pem":     {mtls.CACertFileName, true},
	"ca-key.pem": {mtls.CAKeyFileName, false},
	"cert.pem":   {mtls.ServiceCertFileName, true},
	"key.pem":    {mtls.ServiceKeyFileName, true},
}

// parseCertsSecret parses a secret reference into namespace and name.
// Accepted formats: "namespace/name" or "name" (namespace defaults to stackrox).
func parseCertsSecret(ref string) (namespace, name string, err error) {
	if ref == "" {
		return "", "", fmt.Errorf("empty secret reference")
	}
	if i := strings.IndexByte(ref, '/'); i >= 0 {
		namespace, name = ref[:i], ref[i+1:]
		if namespace == "" || name == "" {
			return "", "", fmt.Errorf("invalid secret reference %q: expected [namespace/]name", ref)
		}
		return namespace, name, nil
	}
	return defaultCertsNamespace, ref, nil
}

// loadCertsFromSecret fetches TLS certificate data from a Kubernetes secret
// and writes it to a temporary directory. The mtls package requires file paths,
// so in-memory injection is not possible without modifying shared infrastructure.
func loadCertsFromSecret(ctx context.Context, secretRef string) (certsDir string, cleanup func(), err error) {
	namespace, name, err := parseCertsSecret(secretRef)
	if err != nil {
		return "", nil, err
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return "", nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	restConfig.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("fetching secret %s/%s: %w", namespace, name, err)
	}

	tmpDir, err := os.MkdirTemp("", "scannerctl-certs-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp directory: %w", err)
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	var missing []string
	for secretKey, entry := range certsSecretFileMap {
		data, ok := secret.Data[secretKey]
		if !ok {
			if entry.required {
				missing = append(missing, secretKey)
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(tmpDir, entry.fileName), data, 0o600); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("writing %s: %w", entry.fileName, err)
		}
	}
	if len(missing) > 0 {
		cleanup()
		return "", nil, fmt.Errorf("secret %s/%s is missing required keys: %v", namespace, name, missing)
	}

	return tmpDir, cleanup, nil
}
