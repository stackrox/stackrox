package certs

import (
	"context"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"go.yaml.in/yaml/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	certEnvName         = "ROX_MTLS_CA_FILE"
	sensorCertEnvName   = "ROX_MTLS_CERT_FILE"
	sensorKeyEnvName    = "ROX_MTLS_KEY_FILE"
	helmConfigEnvName   = "ROX_HELM_CONFIG_FILE_OVERRIDE"
	clusterNameEnvName  = "ROX_HELM_CLUSTER_NAME_FILE_OVERRIDE"
	helmConfigFPEnvName = "ROX_HELM_CLUSTER_CONFIG_FP"

	// DefaultNamespace is the standard namespace for StackRox/RHACS deployments
	DefaultNamespace = "stackrox"
)

// CertConfig represents a single certificate source configuration.
// It specifies the Kubernetes secret name and the mapping of environment
// variable names to certificate file names within that secret.
type CertConfig struct {
	SecretName string
	CertNames  map[string]string
}

// OptionFunc provides options for the CertificateFetcher.
type OptionFunc func(*CertificateFetcher)

// WithOutputDir specifies the output directory.
func WithOutputDir(outputDir string) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		fetcher.outputDir = outputDir
	}
}

// WithNamespace specifies the namespace from which to retrieve the secrets.
func WithNamespace(namespace string) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		fetcher.namespace = namespace
	}
}

// WithCertConfig specifies one or more certificate configurations to use.
// When provided, it replaces the default fallback behavior with custom configs.
func WithCertConfig(configs ...CertConfig) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		if len(configs) > 0 {
			fetcher.certConfigs = configs
		}
	}
}

// WithHelmConfig specifies the secret containing the Helm configuration, the name of the file within the secret, and the output file name.
func WithHelmConfig(helmConfigSecretName string, helmConfigFile string, helmConfigOutputFileName string) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		fetcher.helmClusterConfigSecretName = helmConfigSecretName
		fetcher.helmClusterConfigFileName = helmConfigFile
		fetcher.helmClusterConfigOutputFileName = helmConfigOutputFileName
	}
}

// WithClusterName specifies the secret containing the cluster information, the name of the filed within the secret, and the output file name.
func WithClusterName(clusterNameSecretName string, clusterNameField string, clusterNameOutputFileName string) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		fetcher.helmEffectiveClusterSecretName = clusterNameSecretName
		fetcher.clusterNameField = clusterNameField
		fetcher.clusterNameOutputFileName = clusterNameOutputFileName
	}
}

// WithSetEnvFunc specifies the function to set the environment variables.
func WithSetEnvFunc(fn func(string, string) error) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		fetcher.setEnvFn = fn
	}
}

// NewCertificateFetcher returns a new CertificateFetcher with default certificate
// fallback behavior. By default, it attempts to fetch certificates from multiple
// sources in priority order:
//  1. tls-cert-sensor (current location)
//  2. sensor-tls (legacy location for backward compatibility)
//
// The fetcher stops at the first successful fetch. Use WithCertConfig() to override
// default configs and specify custom certificate sources.
func NewCertificateFetcher(k8s client.Interface, opts ...OptionFunc) *CertificateFetcher {
	fetcher := &CertificateFetcher{
		k8s:       k8s,
		outputDir: "tmp/",
		namespace: DefaultNamespace,
		certConfigs: []CertConfig{
			// Primary: current location
			{
				SecretName: "tls-cert-sensor",
				CertNames: map[string]string{
					certEnvName:       "ca.pem",
					sensorCertEnvName: "cert.pem",
					sensorKeyEnvName:  "key.pem",
				},
			},
			// Fallback: legacy location for backward compatibility
			{
				SecretName: "sensor-tls",
				CertNames: map[string]string{
					certEnvName:       "ca.pem",
					sensorCertEnvName: "sensor-cert.pem",
					sensorKeyEnvName:  "sensor-key.pem",
				},
			},
		},
		helmClusterConfigSecretName:     "helm-cluster-config",
		helmClusterConfigFileName:       "config.yaml",
		helmClusterConfigOutputFileName: "helm-config.yaml",
		helmEffectiveClusterSecretName:  "helm-effective-cluster-name",
		clusterNameField:                "cluster-name",
		clusterNameOutputFileName:       "helm-name.yaml",
		setEnvFn:                        os.Setenv,
	}
	for _, o := range opts {
		o(fetcher)
	}
	return fetcher
}

// CertificateFetcher retrieves kubernetes secrets containing certificates/configurations and writes them locally.
type CertificateFetcher struct {
	k8s                             client.Interface
	setEnvFn                        func(string, string) error
	outputDir                       string
	namespace                       string
	certConfigs                     []CertConfig
	helmClusterConfigSecretName     string
	helmEffectiveClusterSecretName  string
	helmClusterConfigFileName       string
	helmClusterConfigOutputFileName string
	clusterNameField                string
	clusterNameOutputFileName       string
}

// FetchCertificatesAndSetEnvironment retrieves the certificates/configuration and writes it.
// It attempts to fetch certificates from configured sources in order, stopping at the first success.
func (f *CertificateFetcher) FetchCertificatesAndSetEnvironment() error {
	if err := os.MkdirAll(path.Dir(f.outputDir), os.ModePerm); err != nil {
		return errors.Errorf("could not create directory %s: %v", f.outputDir, err)
	}

	// Try each certificate configuration in order
	if len(f.certConfigs) > 0 {
		var fetchErrors []error

		for i, config := range f.certConfigs {
			certSlice := toSlice(config.CertNames)
			err := f.getSecretAndWrite(config.SecretName, f.namespace, certSlice, certSlice)

			if err == nil {
				// Success - set environment and continue with other secrets
				if err := f.setSensorCertsEnv(config.CertNames); err != nil {
					return err
				}
				break // Stop trying other configs
			}

			// Store error from this attempt
			fetchErrors = append(fetchErrors,
				errors.Wrapf(err, "attempt %d: failed to fetch from secret %s", i+1, config.SecretName))
		}

		// If all attempts failed, return aggregate error
		if len(fetchErrors) == len(f.certConfigs) {
			return errors.Errorf("failed to fetch certificates from any source: %v", fetchErrors)
		}
	}

	if f.helmClusterConfigSecretName != "" {
		if err := f.getSecretAndWrite(f.helmClusterConfigSecretName, f.namespace, []string{f.helmClusterConfigFileName}, []string{f.helmClusterConfigOutputFileName}); err != nil {
			return err
		}
		if err := f.setHelmConfigEnv(); err != nil {
			return err
		}
	}
	if f.helmEffectiveClusterSecretName != "" {
		if err := f.getSecretAndWrite(f.helmEffectiveClusterSecretName, f.namespace, []string{f.clusterNameField}, []string{f.clusterNameOutputFileName}); err != nil {
			return err
		}
		if err := f.setClusterNameEnv(); err != nil {
			return err
		}
	}
	return nil
}

func (f *CertificateFetcher) setSensorCertsEnv(certNames map[string]string) error {
	for k, v := range certNames {
		if err := f.setEnvFn(k, path.Join(f.outputDir, v)); err != nil {
			return err
		}
	}
	return nil
}

func (f *CertificateFetcher) setHelmConfigEnv() error {
	type helmConfig struct {
		ClusterName   string `yaml:"clusterName"`
		ClusterConfig struct {
			FingerPrint string `yaml:"configFingerprint"`
		} `yaml:"clusterConfig"`
	}
	var clusterC helmConfig
	yamlFile, err := os.ReadFile(path.Join(f.outputDir, f.helmClusterConfigOutputFileName))
	if err != nil {
		return errors.Wrapf(err, "reading helm cluster config file %s", f.helmClusterConfigOutputFileName)
	}
	if err := yaml.Unmarshal(yamlFile, &clusterC); err != nil {
		return errors.Wrap(err, "unmarshaling helm cluster config")
	}

	if err := f.setEnvFn(helmConfigFPEnvName, clusterC.ClusterConfig.FingerPrint); err != nil {
		return err
	}
	if err := f.setEnvFn(helmConfigEnvName, path.Join(f.outputDir, f.helmClusterConfigOutputFileName)); err != nil {
		return err
	}
	return nil
}

func (f *CertificateFetcher) setClusterNameEnv() error {
	if err := f.setEnvFn(clusterNameEnvName, path.Join(f.outputDir, f.clusterNameOutputFileName)); err != nil {
		return err
	}
	return nil
}

func (f *CertificateFetcher) getSecretAndWrite(secretName string, namespace string, toFetch []string, toWrite []string) error {
	secret, err := f.k8s.Kubernetes().CoreV1().Secrets(namespace).Get(context.Background(), secretName, v1.GetOptions{})
	if err != nil {
		return errors.Errorf("could not retrieve secret %s from namespace %s: %v", secretName, namespace, err)
	}
	if len(toFetch) != len(toWrite) {
		return errors.New("the length of files to fetch should be the same as the files to write")
	}
	for i, fetchedFile := range toFetch {
		data, ok := secret.Data[fetchedFile]
		if !ok {
			return errors.Errorf("%s not found in the secret %s", fetchedFile, secretName)
		}
		if err := f.writeFile(toWrite[i], data); err != nil {
			return err
		}
	}
	return nil
}

func (f *CertificateFetcher) writeFile(fname string, data []byte) error {
	file, err := os.OpenFile(path.Join(f.outputDir, fname), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Errorf("could not create file %s: %v", fname, err)
	}
	if _, err := file.Write(data); err != nil {
		return errors.Errorf("could not write file %s: %v", fname, err)
	}
	if err := file.Close(); err != nil {
		return errors.Errorf("could not close file %s: %v", fname, err)
	}
	return nil
}

func toSlice(m map[string]string) []string {
	ret := make([]string, len(m))
	i := 0
	for _, v := range m {
		ret[i] = v
		i++
	}
	return ret
}
