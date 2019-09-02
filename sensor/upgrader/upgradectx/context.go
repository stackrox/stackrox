package upgradectx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/config"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi"
	openAPIValidation "k8s.io/kubernetes/pkg/kubectl/cmd/util/openapi/validation"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

var (
	log = logging.LoggerForModule()
)

// UpgradeContext provides a unified interface for interacting with the environment (e.g., the K8s API server) in the
// upgrade process.
type UpgradeContext struct {
	ctx context.Context

	config config.UpgraderConfig

	scheme            *runtime.Scheme
	codecs            serializer.CodecFactory
	resources         map[schema.GroupVersionKind]*resources.Metadata
	clientSet         *kubernetes.Clientset
	dynamicClientPool dynamic.ClientPool
	schemaValidator   validation.Schema

	httpClient *http.Client
}

// Create creates a new upgrader context from the given config.
func Create(ctx context.Context, config *config.UpgraderConfig) (*UpgradeContext, error) {
	// Ensure that the context lifetime has an effect.
	restConfigShallowCopy := *config.K8sRESTConfig
	oldWrapTransport := restConfigShallowCopy.WrapTransport
	restConfigShallowCopy.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if oldWrapTransport != nil {
			rt = oldWrapTransport(rt)
		}
		return httputil.ContextBoundRoundTripper(ctx, rt)
	}

	k8sClientSet, err := kubernetes.NewForConfig(&restConfigShallowCopy)
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes API clients")
	}
	dynamicClientPool := dynamic.NewDynamicClientPool(&restConfigShallowCopy)

	resourceMap, err := resources.GetAvailableResources(k8sClientSet.Discovery(), common.OrderedBundleResourceTypes)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving Kubernetes resources from server")
	}

	log.Infof("Server supports %d out of %d relevant resource types", len(resourceMap), len(common.OrderedBundleResourceTypes))
	for _, gvk := range common.OrderedBundleResourceTypes {
		if _, ok := resourceMap[gvk]; ok {
			log.Infof("Resource type %s is SUPPORTED", gvk)
		} else {
			log.Infof("Resource type %s is NOT SUPPORTED", gvk)
		}
	}

	openAPIDoc, err := k8sClientSet.Discovery().OpenAPISchema()
	if err != nil {
		return nil, errors.Wrap(err, "retrieving OpenAPI schema document from server")
	}
	if err := common.PatchOpenAPISchema(openAPIDoc); err != nil {
		return nil, errors.Wrap(err, "patching OpenAPI schema")
	}
	openAPIResources, err := openapi.NewOpenAPIData(openAPIDoc)
	if err != nil {
		return nil, errors.Wrap(err, "parsing OpenAPI schema document into resources")
	}
	schemaValidator := openAPIValidation.NewSchemaValidation(openAPIResources)

	schm := scheme.Scheme

	c := &UpgradeContext{
		ctx:               ctx,
		config:            *config,
		scheme:            schm,
		codecs:            serializer.NewCodecFactory(schm),
		resources:         resourceMap,
		clientSet:         k8sClientSet,
		dynamicClientPool: dynamicClientPool,
		schemaValidator: validation.ConjunctiveSchema{
			schemaValidator,
			yamlValidator{jsonValidator: validation.NoDoubleKeySchema{}},
		},
	}

	if config.CentralEndpoint != "" {
		tlsConf, err := clientconn.TLSConfig(mtls.CentralSubject, clientconn.TLSConfigOptions{
			UseClientCert: true,
		})
		if err != nil {
			return nil, errors.Wrap(err, "instantiating TLS config")
		}
		tlsConf.NextProtos = nil // no HTTP/2 or pure GRPC!
		c.httpClient = &http.Client{
			Transport: httputil.ContextBoundRoundTripper(ctx, &http.Transport{
				TLSClientConfig: tlsConf,
			}),
		}
	}

	return c, nil
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
	return fallbackDecoder{c.codecs.UniversalDeserializer(), yamlDecoder{jsonDecoder: unstructured.UnstructuredJSONScheme}}
}

// IsProcessStateObject checks if the given object belongs to the state of this upgrade process.
func (c *UpgradeContext) IsProcessStateObject(obj metav1.Object) bool {
	return obj.GetLabels()[common.UpgradeProcessIDLabelKey] == c.config.ProcessID
}

// AnnotateProcessStateObject enriches the given object with labels and annotations that allow identifying it as an
// object belonging to this upgrade process. It should only be used on objects that constitute upgrade process state,
// not on the upgraded resources itself.
func (c *UpgradeContext) AnnotateProcessStateObject(obj metav1.Object) {
	lbls := obj.GetLabels()
	if lbls == nil {
		lbls = make(map[string]string)
	}
	lbls[common.UpgradeProcessIDLabelKey] = c.config.ProcessID
	obj.SetLabels(lbls)

	if c.config.Owner != nil {
		ownerRefs := obj.GetOwnerReferences()
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion: c.config.Owner.GVK.GroupVersion().String(),
			Kind:       c.config.Owner.GVK.Kind,
			Name:       c.config.Owner.Name,
		})
		obj.SetOwnerReferences(ownerRefs)
	}
}

// ClusterID returns the ID of this cluster.
func (c *UpgradeContext) ClusterID() string {
	return c.config.ClusterID
}

// DoHTTPRequest performs an HTTP request. If the URL in req is relative, the central endpoint is filled in as the host,
// using the HTTPS scheme by default.
func (c *UpgradeContext) DoHTTPRequest(req *http.Request) (*http.Response, error) {
	if c.httpClient == nil {
		return nil, errors.New("no HTTP client configured")
	}

	if req.URL.Scheme == "" {
		req.URL.Scheme = "https"
	}
	if req.URL.Host == "" {
		req.URL.Host = c.config.CentralEndpoint
	}

	return c.httpClient.Do(req)
}

// Validator returns the schema validator to be used.
func (c *UpgradeContext) Validator() validation.Schema {
	return c.schemaValidator
}

// ParseAndValidateObject parses and validates (against the server's OpenAPI schema) a serialized Kubernetes object.
func (c *UpgradeContext) ParseAndValidateObject(data []byte) (k8sobjects.Object, error) {
	obj, _, err := c.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	if err := c.schemaValidator.ValidateBytes(data); err != nil {
		return nil, errors.Wrap(err, "schema validation failed")
	}
	k8sObj, _ := obj.(k8sobjects.Object)
	if k8sObj == nil {
		return nil, errors.Errorf("object of kind %v is not a Kubernetes API object", obj.GetObjectKind().GroupVersionKind())
	}
	return k8sObj, nil
}

func (c *UpgradeContext) unpackList(listObj runtime.Object) ([]k8sobjects.Object, error) {
	objs, ok := unpackListReflect(listObj)
	if ok {
		return objs, nil
	}

	log.Infof("Could not unpack list of kind %v using reflection", listObj.GetObjectKind().GroupVersionKind())

	var list unstructured.UnstructuredList
	if err := c.scheme.Convert(listObj, &list, nil); err != nil {
		return nil, errors.Wrapf(err, "converting object of kind %v to a generic list", listObj.GetObjectKind().GroupVersionKind())
	}

	objs = make([]k8sobjects.Object, 0, len(list.Items))
	for _, item := range list.Items {
		objs = append(objs, &item)
	}
	return objs, nil
}

// Owner returns the owning object of this upgrader instance, if any.
func (c *UpgradeContext) Owner() *k8sobjects.ObjectRef {
	return c.config.Owner
}

// ListCurrentObjects returns all Kubernetes objects that are relevant for the upgrade process.
func (c *UpgradeContext) ListCurrentObjects() ([]k8sobjects.Object, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", common.UpgradeResourceLabelKey, common.UpgradeResourceLabelValue),
	}

	var result []k8sobjects.Object

	for _, resourceType := range c.resources {
		resourceClient, err := c.DynamicClientForResource(resourceType, common.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "obtaining dynamic client for resource %v", resourceType)
		}
		listObj, err := resourceClient.List(listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "listing relevant objects of type %v", resourceType)
		}

		objs, err := c.unpackList(listObj)
		if err != nil {
			return nil, errors.Wrapf(err, "unpacking list of objects of type %v", resourceType)
		}
		result = append(result, objs...)
	}

	return result, nil
}
