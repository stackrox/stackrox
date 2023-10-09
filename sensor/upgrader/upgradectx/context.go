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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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

	resources              map[schema.GroupVersionKind]*resources.Metadata
	clientSet              kubernetes.Interface
	dynamicClientGenerator dynamic.Interface
	schemaValidator        validation.Schema

	ownerRef *metav1.OwnerReference

	centralHTTPClient *http.Client
	grpcClientConn    *grpc.ClientConn

	podSecurityPoliciesSupported bool
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

	pspSupported := false
	for _, gvk := range common.OrderedBundleResourceTypes {
		if _, ok := resourceMap[gvk]; ok {
			log.Infof("Resource type %s is SUPPORTED", gvk)
			if gvk.Kind == "PodSecurityPolicy" {
				pspSupported = true
			}
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
	schemaValidator, err := common.ValidatorFromOpenAPIDoc(openAPIDoc)
	if err != nil {
		return nil, errors.Wrap(err, "creating validator from OpenAPI schema")
	}

	c := &UpgradeContext{
		ctx:                          ctx,
		config:                       *config,
		resources:                    resourceMap,
		clientSet:                    k8sClientSet,
		dynamicClientGenerator:       dynamicClientGenerator,
		schemaValidator:              schemaValidator,
		podSecurityPoliciesSupported: pspSupported,
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

// InCertRotationMode returns whether this is a cert rotation upgrade.
func (c *UpgradeContext) InCertRotationMode() bool {
	return c.config.InCertRotationMode
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

	req.Header.Set("User-Agent", clientconn.GetUserAgent())

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
func (c *UpgradeContext) ParseAndValidateObject(data []byte) (*unstructured.Unstructured, error) {
	k8sObj, err := k8sutil.UnstructuredFromYAML(string(data))
	if err != nil {
		return nil, err
	}
	if err := c.schemaValidator.ValidateBytes(data); err != nil {
		return nil, errors.Wrap(err, "schema validation failed")
	}
	return k8sObj, nil
}

// Owner returns the owning object of this upgrader instance, if any.
func (c *UpgradeContext) Owner() *k8sobjects.ObjectRef {
	return c.config.Owner
}

// List lists all Kubernetes options of resources of the given purpose, applying the given list options.
func (c *UpgradeContext) List(resourcePurpose resources.Purpose, listOpts *metav1.ListOptions) ([]*unstructured.Unstructured, error) {
	if listOpts == nil {
		listOpts = &metav1.ListOptions{}
	}

	var result []*unstructured.Unstructured

	for _, resourceMD := range c.resources {
		if resourceMD.Purpose&resourcePurpose != resourcePurpose {
			continue
		}
		for _, ns := range common.AllowedNamespaces {
			resourceClient := c.DynamicClientForResource(resourceMD, ns)
			listObj, err := resourceClient.List(c.ctx, *listOpts)
			if err != nil {
				return nil, errors.Wrapf(err, "listing relevant objects of type %v in namespace %s", resourceMD, ns)
			}

			for _, item := range listObj.Items {
				item := item // create a copy to prevent aliasing
				result = append(result, &item)
			}
		}
	}

	return result, nil
}

// ListCurrentObjects returns all Kubernetes objects that are relevant for the upgrade process. The caller is free
// to modify the objects.
func (c *UpgradeContext) ListCurrentObjects() ([]*unstructured.Unstructured, error) {
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

// IsPodSecurityEnabled returns whether or not pod security polices are enabled for this cluster
func (c *UpgradeContext) IsPodSecurityEnabled() bool {
	return c.podSecurityPoliciesSupported
}
