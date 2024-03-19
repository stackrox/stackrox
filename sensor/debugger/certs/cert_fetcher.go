package certs

import (
	"context"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	certEnvName         = "ROX_MTLS_CA_FILE"
	sensorCertEnvName   = "ROX_MTLS_CERT_FILE"
	sensorKeyEnvName    = "ROX_MTLS_KEY_FILE"
	helmConfigEnvName   = "ROX_HELM_CONFIG_FILE_OVERRIDE"
	clusterNameEnvName  = "ROX_HELM_CLUSTER_NAME_FILE_OVERRIDE"
	helmConfigFPEnvName = "ROX_HELM_CLUSTER_CONFIG_FP"
)

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

// WithCertNames specifies the secret containing the certificates and a map of certificate names to retrieve and write.
func WithCertNames(secretName string, certNames map[string]string) OptionFunc {
	return func(fetcher *CertificateFetcher) {
		fetcher.secretName = secretName
		fetcher.certNames = certNames
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

// NewCertificateFetcher returns a new CertificateFetcher
func NewCertificateFetcher(k8s client.Interface, opts ...OptionFunc) *CertificateFetcher {
	fetcher := &CertificateFetcher{
		k8s:        k8s,
		outputDir:  "tmp/",
		namespace:  "stackrox",
		secretName: "sensor-tls",
		certNames: map[string]string{
			certEnvName:       "ca.pem",
			sensorCertEnvName: "sensor-cert.pem",
			sensorKeyEnvName:  "sensor-key.pem",
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
	secretName                      string
	certNames                       map[string]string
	helmClusterConfigSecretName     string
	helmEffectiveClusterSecretName  string
	helmClusterConfigFileName       string
	helmClusterConfigOutputFileName string
	clusterNameField                string
	clusterNameOutputFileName       string
}

// FetchCertificatesAndSetEnvironment retrieves the certificates/configuration and writes it.
func (f *CertificateFetcher) FetchCertificatesAndSetEnvironment() error {
	if err := os.MkdirAll(path.Dir(f.outputDir), os.ModePerm); err != nil {
		return errors.Errorf("could not create directory %s: %v", f.outputDir, err)
	}
	if f.secretName != "" {
		certSlice := toSlice(f.certNames)
		if err := f.getSecretAndWrite(f.secretName, f.namespace, certSlice, certSlice); err != nil {
			return err
		}
		if err := f.setSensorCertsEnv(); err != nil {
			return err
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

func (f *CertificateFetcher) setSensorCertsEnv() error {
	for k, v := range f.certNames {
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
		return err
	}
	if err := yaml.Unmarshal(yamlFile, &clusterC); err != nil {
		return err
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
