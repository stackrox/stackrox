package upgradectx

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/config"
	"github.com/stackrox/rox/sensor/upgrader/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	log = logging.LoggerForModule()
)

// UpgradeContext provides a unified interface for interacting with the environment (e.g., the K8s API server) in the
// upgrade process.
type UpgradeContext struct {
	config config.UpgraderConfig

	scheme            *runtime.Scheme
	codecs            serializer.CodecFactory
	resources         map[schema.GroupVersionKind]*resources.Metadata
	clientSet         *kubernetes.Clientset
	dynamicClientPool dynamic.ClientPool
}

// Create creates a new upgrader context from the given config.
func Create(config *config.UpgraderConfig) (*UpgradeContext, error) {
	k8sClientSet, err := kubernetes.NewForConfig(config.K8sRESTConfig)
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes API clients")
	}
	dynamicClientPool := dynamic.NewDynamicClientPool(config.K8sRESTConfig)

	resourceMap, err := resources.GetAvailableResources(k8sClientSet.Discovery(), common.BundleResourceTypes)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving Kubernetes resources from server")
	}

	log.Infof("Server supports %d out of %d relevant resource types", len(resourceMap), len(common.BundleResourceTypes))
	for _, gvk := range common.BundleResourceTypes {
		if _, ok := resourceMap[gvk]; ok {
			log.Infof("Resource type %s is SUPPORTED", gvk)
		} else {
			log.Infof("Resource type %s is NOT SUPPORTED", gvk)
		}
	}

	schm := scheme.Scheme

	ctx := &UpgradeContext{
		config:            *config,
		scheme:            schm,
		codecs:            serializer.NewCodecFactory(schm),
		resources:         resourceMap,
		clientSet:         k8sClientSet,
		dynamicClientPool: dynamicClientPool,
	}

	return ctx, nil
}

// GetResourceMetadata returns the API resource metadata for the given GroupVersionKind. It returns `nil` if the server
// does not support the given resource (this is not necessarily an error, unless we are trying to create an object of
// this resource type).
func (c *UpgradeContext) GetResourceMetadata(gvk schema.GroupVersionKind) *resources.Metadata {
	return c.resources[gvk]
}

// Resources returns a slice of the metadata objects for all supported API resources.
func (c *UpgradeContext) Resources() []*resources.Metadata {
	list := make([]*resources.Metadata, 0, len(c.resources))
	for _, res := range c.resources {
		list = append(list, res)
	}
	return list
}

// ClientSet returns the Kubernetes client set.
func (c *UpgradeContext) ClientSet() *kubernetes.Clientset {
	return c.clientSet
}

// DynamicClientForResource returns a dynamic client for the given resource and namespace. If the resource is not
// namespaced, the namespace parameter is ignored.
func (c *UpgradeContext) DynamicClientForResource(resource *resources.Metadata, namespace string) (dynamic.ResourceInterface, error) {
	client, err := c.dynamicClientPool.ClientForGroupVersionKind(resource.GroupVersionKind())
	if err != nil {
		return nil, err
	}
	return client.Resource(&resource.APIResource, namespace), nil
}

// ProcessID returns the ID of the current upgrade process.
func (c *UpgradeContext) ProcessID() string {
	return c.config.ProcessID
}

// Scheme returns the Kubernetes resource scheme we are using.
func (c *UpgradeContext) Scheme() *runtime.Scheme {
	return c.scheme
}

// Codecs returns the Kubernetes resource codec factory we are using.
func (c *UpgradeContext) Codecs() *serializer.CodecFactory {
	return &c.codecs
}

// UniversalDecoder is a decoder that can be used to decode any object.
func (c *UpgradeContext) UniversalDecoder() runtime.Decoder {
	return fallbackDecoder{c.codecs.UniversalDeserializer(), unstructured.UnstructuredJSONScheme}
}

// AnnotateProcessStateObject enriches the given object with labels and annotations that allow identifying it as an
// object belonging to this upgrade process. It should only be used on objects that constitute upgrade process state,
// not on the upgraded resources itself.
func (c *UpgradeContext) AnnotateProcessStateObject(obj metav1.Object) {
	if obj.GetLabels() == nil {
		obj.SetLabels(make(map[string]string))
	}
	obj.GetLabels()[common.UpgradeProcessIDLabelKey] = c.config.ProcessID
}
