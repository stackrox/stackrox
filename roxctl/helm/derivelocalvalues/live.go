package derivelocalvalues

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/namespaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	// Activate Auth Providers for client-go.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

/// Retrieve Kubernetes Object Definitions from a running cluster.

type liveK8sObjectDescription struct {
	client    dynamic.Interface
	namespace string
}

func (k liveK8sObjectDescription) get(ctx context.Context, kind string, name string) (*unstructured.Unstructured, error) {
	var gvr *schema.GroupVersionResource

	switch strings.ToLower(kind) {
	case "deployment":
		gvr = &schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		}
	case "secret":
		gvr = &schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		}
	case "service":
		gvr = &schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "services",
		}
	case "configmap":
		gvr = &schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "configmaps",
		}
	case "hpa":
		gvr = &schema.GroupVersionResource{
			Group:    "autoscaling",
			Version:  "v1",
			Resource: "horizontalpodautoscalers",
		}
	default:
		// This means that `deriveLocalValues` tries to lookup K8s resources, which are not yet contained
		// in the above list and need to be added.
		panic(fmt.Sprintf("Unknown resource kind %q", kind))
	}

	resClient := k.client.Resource(*gvr)

	resp, err := resClient.
		Namespace(k.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving resource %s/%s", kind, name)
	}

	return resp, nil
}

func newLiveK8sObjectDescription() (*liveK8sObjectDescription, error) {
	config, err := loadKubeCtlConfig()
	if err != nil {
		return nil, err
	}

	// create the clientset
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating dynamic client set for accessing Kubernetes cluster")
	}
	return &liveK8sObjectDescription{client: client, namespace: namespaces.StackRox}, nil
}

func loadKubeCtlConfig() (*rest.Config, error) {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading default Kubernetes client config")
	}

	clientConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	return clientConfig, errors.Wrap(err, "could not load new default Kubernetes client config")
}
