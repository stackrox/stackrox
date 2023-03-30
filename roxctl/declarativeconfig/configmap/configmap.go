package configmap

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// Flag names exposing the config map name and the namespace where the configmap resides.
const (
	ConfigMapFlag = "config-map"
	NamespaceFlag = "namespace"
)

// WriteToConfigMap writes bytes containing a YAML string to a config map under a specific key.
// The Kubernetes context will be inferred from either the $KUBECONFIG variable or the $HOME/.kube/config path.
func WriteToConfigMap(ctx context.Context, configMap, namespace, key string, yaml []byte) error {
	client, namespaceFromConfig, err := createK8SClient(namespace)
	if err != nil {
		return err
	}

	cm, err := client.CoreV1().ConfigMaps(namespaceFromConfig).Get(ctx, configMap, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "retrieving config map %s", configMap)
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[key] = string(yaml)

	if _, err := client.CoreV1().ConfigMaps(namespaceFromConfig).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "updating config map %s", configMap)
	}
	return nil
}

// ReadFromConfigMap will read all keys from a config map and return them in byte format.
// The Kubernetes context will be inferred from either the $KUBECONFIG variable or the $HOME/.kube/config path.
// Note: the binary data within the config map will be skipped, as the output shall be UTF-8 compatible.
func ReadFromConfigMap(ctx context.Context, configMap, namespace string) ([][]byte, error) {
	client, namespaceFromConfig, err := createK8SClient(namespace)
	if err != nil {
		return nil, err
	}

	cm, err := client.CoreV1().ConfigMaps(namespaceFromConfig).Get(ctx, configMap, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving config map %s", configMap)
	}
	contents := make([][]byte, 0, len(cm.Data))
	for _, data := range cm.Data {
		contents = append(contents, []byte(data))
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

// ReadConfigMapFlags reads the values of the ConfigMapFlag and NamespaceFlag from the cobra.Command, returning any
// error that may occur.
func ReadConfigMapFlags(cmd *cobra.Command) (string, string, error) {
	configMap, err := cmd.Flags().GetString(ConfigMapFlag)
	if err != nil {
		return "", "", errox.InvariantViolation.Newf("retrieving value for flag %s", ConfigMapFlag).CausedBy(err)
	}

	namespace, err := cmd.Flags().GetString(NamespaceFlag)
	if err != nil {
		return "", "", errox.InvariantViolation.Newf("retrieving value for flag %s", NamespaceFlag).CausedBy(err)
	}

	return configMap, namespace, nil
}
