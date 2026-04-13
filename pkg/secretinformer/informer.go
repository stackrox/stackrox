package secretinformer

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const resyncTime = 10 * time.Minute

var (
	secretGVR = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
)

// SecretInformer is a convenience wrapper around a Kubernetes informer for a specific secret.
type SecretInformer struct {
	namespace  string
	secretName string

	dynamicClient dynamic.Interface
	handler       cache.ResourceEventHandlerRegistration
	onAddFn       func(*v1.Secret)
	onUpdateFn    func(*v1.Secret)
	onDeleteFn    func()
	stopCh        concurrency.Signal
}

var _ cache.ResourceEventHandler = &SecretInformer{}

// NewSecretInformer creates a new secret informer.
func NewSecretInformer(
	namespace string,
	secretName string,
	dynamicClient dynamic.Interface,
	onAddFn func(*v1.Secret),
	onUpdateFn func(*v1.Secret),
	onDeleteFn func(),
) *SecretInformer {
	return &SecretInformer{
		namespace:     namespace,
		secretName:    secretName,
		dynamicClient: dynamicClient,
		onAddFn:       onAddFn,
		onUpdateFn:    onUpdateFn,
		onDeleteFn:    onDeleteFn,
		stopCh:        concurrency.NewSignal(),
	}
}

// Start initiates the secret informer loop.
func (c *SecretInformer) Start() error {
	tweakListOptions := func(opts *metav1.ListOptions) {
		opts.FieldSelector = "metadata.name=" + c.secretName
	}
	sif := dynamicinformer.NewFilteredDynamicSharedInformerFactory(c.dynamicClient, resyncTime, c.namespace, tweakListOptions)

	handler, err := sif.ForResource(secretGVR).Informer().AddEventHandler(c)
	if err != nil {
		return errors.Wrapf(err,
			"could not add event handler to informer for secret %q/%q",
			c.namespace,
			c.secretName,
		)
	}
	c.handler = handler
	sif.Start(c.stopCh.WaitC())
	return nil
}

// Stop ends the secret informer loop.
func (c *SecretInformer) Stop() {
	c.stopCh.Signal()
}

// HasSynced reports if the informer handler has synced, meaning it has had
// all items in the initial list delivered.
func (c *SecretInformer) HasSynced() bool {
	if c == nil || c.handler == nil {
		return false
	}
	return c.handler.HasSynced()
}

// OnAdd is called when the secret is added.
func (c *SecretInformer) OnAdd(obj interface{}, _ bool) {
	uns, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	var secret v1.Secret
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.Object, &secret); err != nil {
		return
	}
	c.onAddFn(&secret)
}

// OnUpdate is called when the secret is updated.
func (c *SecretInformer) OnUpdate(_, newObj interface{}) {
	uns, ok := newObj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	var secret v1.Secret
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.Object, &secret); err != nil {
		return
	}
	c.onUpdateFn(&secret)
}

// OnDelete is called when the secret is deleted.
func (c *SecretInformer) OnDelete(_ interface{}) {
	c.onDeleteFn()
}
