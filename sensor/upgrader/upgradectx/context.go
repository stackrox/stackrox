package upgradectx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/config"
	"github.com/stackrox/rox/sensor/upgrader/resources"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubectl/pkg/util/openapi"
	openAPIValidation "k8s.io/kubectl/pkg/util/openapi/validation"
	"k8s.io/kubectl/pkg/validation"
)

var (
	log = logging.LoggerForModule()
)

// UpgradeContext provides a unified interface for interacting with the environment (e.g., the K8s API server) in the
// upgrade process.
type UpgradeContext struct {
	ctx context.Context

	config config.UpgraderConfig

	scheme                 *runtime.Scheme
	codecs                 serializer.CodecFactory
	resources              map[schema.GroupVersionKind]*resources.Metadata
	clientSet              kubernetes.Interface
	dynamicClientGenerator dynamic.Interface
	schemaValidator        validation.Schema

	ownerRef *metav1.OwnerReference

	centralHTTPClient *http.Client
	grpcClientConn    *grpc.ClientConn
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

	dynamicClientGenerator, err := dynamic.NewForConfig(&restConfigShallowCopy)
	if err != nil {
		return nil, errors.Wrap(err, "creating dynamic client")
	}

	expectedGVKs := make(map[schema.GroupVersionKind]struct{})
	for _, gvk := range common.OrderedBundleResourceTypes {
		expectedGVKs[gvk] = struct{}{}
	}
	for _, gvk := range common.StateResourceTypes {
		expectedGVKs[gvk] = struct{}{}
	}
	if config.Owner != nil {
		expectedGVKs[config.Owner.GVK] = struct{}{}
	}

	resourceMap, err := resources.GetAvailableResources(k8sClientSet.Discovery(), expectedGVKs)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving Kubernetes resources from server")
	}

	numBundleResources := 0
	for _, br := range common.OrderedBundleResourceTypes {
		resMD := resourceMap[br]
		if resMD != nil {
			resMD.Purpose |= resources.BundleResource
			numBundleResources++
		}
	}
	log.Infof("Server supports %d out of %d relevant bundle resource types", numBundleResources, len(common.OrderedBundleResourceTypes))

	numStateResources := 0
	for _, sr := range common.StateResourceTypes {
		resMD := resourceMap[sr]
		if resMD != nil {
			resMD.Purpose |= resources.StateResource
			numStateResources++
		}
	}
	log.Infof("Server supports %d out of %d relevant state resource types", numStateResources, len(common.StateResourceTypes))

	for _, gvk := range common.OrderedBundleResourceTypes {
		if _, ok := resourceMap[gvk]; ok {
			log.Infof("Resource type %s is SUPPORTED", gvk)
		} else {
			log.Infof("Resource type %s is NOT SUPPORTED", gvk)
		}
	}

	unversionedGKs := make(map[schema.GroupKind]schema.GroupVersionKind)
	for i := len(common.OrderedBundleResourceTypes) - 1; i >= 0; i-- {
		gvk := common.OrderedBundleResourceTypes[i]
		gk := gvk.GroupKind()
		if canonicalGVK, exists := unversionedGKs[gk]; exists {
			log.Infof("Disregarding obsolete resource type %s in favor of %s", gvk, canonicalGVK)
			delete(resourceMap, gvk)
			continue
		}
		unversionedGKs[gk] = gvk
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
		ctx:                    ctx,
		config:                 *config,
		scheme:                 schm,
		codecs:                 serializer.NewCodecFactory(schm),
		resources:              resourceMap,
		clientSet:              k8sClientSet,
		dynamicClientGenerator: dynamicClientGenerator,
		schemaValidator: validation.ConjunctiveSchema{
			schemaValidator,
			yamlValidator{jsonValidator: validation.NoDoubleKeySchema{}},
		},
	}

	if config.CentralEndpoint != "" {
		transport, err := clientconn.AuthenticatedHTTPTransport(config.CentralEndpoint, mtls.CentralSubject, nil, clientconn.UseServiceCertToken(true))
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize HTTP transport to Central")
		}
		c.centralHTTPClient = &http.Client{
			Transport: transport,
		}
		c.grpcClientConn, err = clientconn.AuthenticatedGRPCConnection(config.CentralEndpoint, mtls.CentralSubject, clientconn.UseServiceCertToken(true))
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize gRPC connection to Central")
		}
	}

	if config.Owner != nil {
		ownerRes := resourceMap[config.Owner.GVK]
		if ownerRes == nil {
			return nil, errors.Errorf("server does not support resource type of supposed owner %v", config.Owner)
		}
		ownerResourceClient := c.DynamicClientForResource(ownerRes, config.Owner.Namespace)
		ownerObj, err := ownerResourceClient.Get(ctx, config.Owner.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "could not retrieve supposed owner %v", config.Owner)
		}
		c.ownerRef = &metav1.OwnerReference{
			APIVersion: config.Owner.GVK.GroupVersion().String(),
			Kind:       config.Owner.GVK.Kind,
			Name:       config.Owner.Name,
			UID:        ownerObj.GetUID(),
		}
	}

	return c, nil
}

// Context returns a Go context valid for an upgrade process.
func (c *UpgradeContext) Context() context.Context {
	return c.ctx
}

// GetResourceMetadata returns the API resource metadata for the given GroupVersionKind and purpose. It returns `nil`
// if the server does not support the given resource (this is not necessarily an error, unless we are trying to create
// an object of this resource type), or if the purpose does not match what this resource was intended to be used for.
func (c *UpgradeContext) GetResourceMetadata(gvk schema.GroupVersionKind, purpose resources.Purpose) *resources.Metadata {
	resMD := c.resources[gvk]
	if resMD == nil {
		return nil
	}
	if resMD.Purpose&purpose != purpose {
		return nil
	}
	return resMD
}

// ClientSet returns the Kubernetes client set.
func (c *UpgradeContext) ClientSet() kubernetes.Interface {
	return c.clientSet
}

// DynamicClientForResource returns a dynamic client for the given resource and namespace. If the resource is not
// namespaced, the namespace parameter is ignored.
func (c *UpgradeContext) DynamicClientForResource(resource *resources.Metadata, namespace string) dynamic.ResourceInterface {
	r := c.dynamicClientGenerator.Resource(resource.GroupVersionResource())
	if resource.Namespaced {
		return r.Namespace(namespace)
	}
	return r
}

// DynamicClientForGVK returns a dynamic client for the given group/version/kind, given that it is a valid resource for
// the given purpose.
func (c *UpgradeContext) DynamicClientForGVK(gvk schema.GroupVersionKind, purpose resources.Purpose, namespace string) (dynamic.ResourceInterface, error) {
	resMD := c.GetResourceMetadata(gvk, purpose)
	if resMD == nil {
		return nil, errors.Errorf("the server does not support resource type %v for purpose %v", gvk, purpose)
	}
	return c.DynamicClientForResource(resMD, namespace), nil
}

// ProcessID returns the ID of the current upgrade process.
func (c *UpgradeContext) ProcessID() string {
	return c.config.ProcessID
}

// Scheme returns the Kubernetes resource scheme we are using.
func (c *UpgradeContext) Scheme() *runtime.Scheme {
	return c.scheme
}

// InCertRotationMode returns whether this is a cert rotation upgrade.
func (c *UpgradeContext) InCertRotationMode() bool {
	return c.config.InCertRotationMode
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

	if c.ownerRef != nil {
		ownerRefs := obj.GetOwnerReferences()
		ownerRefs = append(ownerRefs, *c.ownerRef)
		obj.SetOwnerReferences(ownerRefs)
	}
}

// ClusterID returns the ID of this cluster.
func (c *UpgradeContext) ClusterID() string {
	return c.config.ClusterID
}

// DoCentralHTTPRequest performs an HTTP request to central. If the URL in req is relative, the central endpoint is filled in
// as the host, using the HTTPS scheme by default.
func (c *UpgradeContext) DoCentralHTTPRequest(req *http.Request) (*http.Response, error) {
	if c.centralHTTPClient == nil {
		return nil, errors.New("no HTTP client configured")
	}

	return c.centralHTTPClient.Do(req)
}

// GetGRPCClient gets the gRPC client that can be used to make requests to Central.
func (c *UpgradeContext) GetGRPCClient() *grpc.ClientConn {
	return c.grpcClientConn
}

// Validator returns the schema validator to be used.
func (c *UpgradeContext) Validator() validation.Schema {
	return c.schemaValidator
}

// ParseAndValidateObject parses and validates (against the server's OpenAPI schema) a serialized Kubernetes object.
func (c *UpgradeContext) ParseAndValidateObject(data []byte) (k8sutil.Object, error) {
	obj, _, err := c.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	if err := c.schemaValidator.ValidateBytes(data); err != nil {
		return nil, errors.Wrap(err, "schema validation failed")
	}
	k8sObj, _ := obj.(k8sutil.Object)
	if k8sObj == nil {
		return nil, errors.Errorf("object of kind %v is not a Kubernetes API object", obj.GetObjectKind().GroupVersionKind())
	}
	return k8sObj, nil
}

func (c *UpgradeContext) unpackList(listObj runtime.Object) ([]k8sutil.Object, error) {
	objs, ok := unpackListReflect(listObj)
	if ok {
		return objs, nil
	}

	log.Infof("Could not unpack list of kind %v using reflection", listObj.GetObjectKind().GroupVersionKind())

	var list unstructured.UnstructuredList
	if err := c.scheme.Convert(listObj, &list, nil); err != nil {
		return nil, errors.Wrapf(err, "converting object of kind %v to a generic list", listObj.GetObjectKind().GroupVersionKind())
	}

	objs = make([]k8sutil.Object, 0, len(list.Items))
	for i := range list.Items {
		objs = append(objs, &list.Items[i])
	}
	return objs, nil
}

// Owner returns the owning object of this upgrader instance, if any.
func (c *UpgradeContext) Owner() *k8sobjects.ObjectRef {
	return c.config.Owner
}

// List lists all Kubernetes options of resources of the given purpose, applying the given list options.
func (c *UpgradeContext) List(resourcePurpose resources.Purpose, listOpts *metav1.ListOptions) ([]k8sutil.Object, error) {
	if listOpts == nil {
		listOpts = &metav1.ListOptions{}
	}

	var result []k8sutil.Object

	for _, resourceMD := range c.resources {
		if resourceMD.Purpose&resourcePurpose != resourcePurpose {
			continue
		}
		resourceClient := c.DynamicClientForResource(resourceMD, common.Namespace)
		listObj, err := resourceClient.List(c.ctx, *listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "listing relevant objects of type %v", resourceMD)
		}

		objs, err := c.unpackList(listObj)
		if err != nil {
			return nil, errors.Wrapf(err, "unpacking list of objects of type %v", resourceMD)
		}
		result = append(result, objs...)
	}

	return result, nil
}

// ListCurrentObjects returns all Kubernetes objects that are relevant for the upgrade process.
func (c *UpgradeContext) ListCurrentObjects() ([]k8sutil.Object, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", common.UpgradeResourceLabelKey, common.UpgradeResourceLabelValue),
	}

	objects, err := c.List(resources.BundleResource, &listOpts)
	if err != nil {
		return nil, err
	}

	if c.InCertRotationMode() {
		common.Filter(&objects, common.CertObjectPredicate)
	}

	common.Filter(&objects, common.Not(common.AdditionalCASecretPredicate))

	return objects, nil
}
