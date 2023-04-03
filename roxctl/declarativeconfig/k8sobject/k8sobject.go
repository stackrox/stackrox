package k8sobject

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// Flag names exposing the config map, secret, and namespace.
const (
	ConfigMapFlag = "config-map"
	SecretFlag    = "secret"
	NamespaceFlag = "namespace"
)

// WriteToK8sObject writes bytes containing a YAML string to a config map / secret under a specific key.
// The Kubernetes context will be inferred from either the $KUBECONFIG variable or the $HOME/.kube/config path.
func WriteToK8sObject(ctx context.Context, configMap, secret, namespace, key string, yaml []byte) error {
	k8sCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	client, namespaceFromConfig, err := createK8SClient(namespace)
	if err != nil {
		return err
	}

	if configMap != "" {
		return writeConfigMap(k8sCtx, client, configMap, namespaceFromConfig, key, yaml)
	}
	return writeSecret(k8sCtx, client, secret, namespaceFromConfig, key, yaml)
}

func writeConfigMap(ctx context.Context, client kubernetes.Interface, configMap, namespace, key string, yaml []byte) error {
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configMap, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "retrieving config map %s/%s", namespace, configMap)
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[key] = string(yaml)

	if _, err := client.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "updating config map %s/%s", namespace, configMap)
	}
	return nil
}

func writeSecret(ctx context.Context, client kubernetes.Interface, secret, namespace, key string, yaml []byte) error {
	s, err := client.CoreV1().Secrets(namespace).Get(ctx, secret, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "retrieving secret %s/%s", namespace, secret)
	}

	if s.Data == nil {
		s.Data = map[string][]byte{}
	}
	s.Data[key] = []byte(base64.StdEncoding.EncodeToString(yaml))
	if _, err := client.CoreV1().Secrets(namespace).Update(ctx, s, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "updating secret %s/%s", namespace, secret)
	}

	return nil
}

// ReadFromK8sObject will read all keys from a config map / secret and return them in byte format.
// The Kubernetes context will be inferred from either the $KUBECONFIG variable or the $HOME/.kube/config path.
// Note: the binary data within the config map will be skipped, as the output shall be UTF-8 compatible.
func ReadFromK8sObject(ctx context.Context, configMap, secret, namespace string) ([][]byte, error) {
	k8sCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	client, namespaceFromConfig, err := createK8SClient(namespace)
	if err != nil {
		return nil, err
	}

	if configMap != "" {
		return readConfigMap(k8sCtx, client, configMap, namespaceFromConfig)
	}
	return readSecret(k8sCtx, client, secret, namespaceFromConfig)
}

func readConfigMap(ctx context.Context, client kubernetes.Interface, configMap string, namespace string) ([][]byte, error) {
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configMap, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving config map %s/%s", namespace, configMap)
	}
	contents := make([][]byte, 0, len(cm.Data))
	for _, data := range cm.Data {
		contents = append(contents, []byte(data))
	}
	return contents, nil
}

func readSecret(ctx context.Context, client kubernetes.Interface, secret string, namespace string) ([][]byte, error) {
	s, err := client.CoreV1().Secrets(namespace).Get(ctx, secret, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving secret %s/%s", namespace, secret)
	}
	contents := make([][]byte, 0, len(s.Data))
	for _, data := range s.Data {
		out, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, errors.Wrap(err, "decoding base64 input")
		}
		contents = append(contents, out)
	}
	return contents, nil
}

func createK8SClient(namespace string) (*kubernetes.Clientset, string, error) {
	rawConfig := genericclioptions.NewConfigFlags(true).ToRawKubeConfigLoader()
	cfg, err := rawConfig.ClientConfig()
	if err != nil {
		return nil, "", errors.Wrap(err, "retrieving kubeconfig")
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, "", errors.Wrap(err, "creating kubernetes client")
	}
	if namespace != "" {
		return client, namespace, nil
	}
	configNamespace, _, err := rawConfig.Namespace()
	if err != nil {
		return nil, "", errors.Wrap(err, "retrieving namespace from kube config")
	}
	return client, configNamespace, nil
}

// ReadK8sObjectFlags reads the values of the ConfigMapFlag, SecretFlag, and NamespaceFlag from the cobra.Command,
// returning any error that may occur.
func ReadK8sObjectFlags(cmd *cobra.Command) (string, string, string, error) {
	configMap, err := cmd.Flags().GetString(ConfigMapFlag)
	if err != nil {
		return "", "", "", errox.InvariantViolation.Newf("retrieving value for flag %s", ConfigMapFlag).CausedBy(err)
	}

	secret, err := cmd.Flags().GetString(SecretFlag)
	if err != nil {
		return "", "", "", errox.InvariantViolation.Newf("retrieving value for flag %s", SecretFlag).CausedBy(err)
	}

	namespace, err := cmd.Flags().GetString(NamespaceFlag)
	if err != nil {
		return "", "", "", errox.InvariantViolation.Newf("retrieving value for flag %s", NamespaceFlag).CausedBy(err)
	}

	return configMap, secret, namespace, nil
}
