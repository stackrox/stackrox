package secretinformer

import (
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const resyncTime = 10 * time.Minute

// SecretInformer is a convenience wrapper around a Kubernetes informer for a specific secret.
type SecretInformer struct {
	namespace  string
	secretName string

	k8sClient  kubernetes.Interface
	onAddFn    func(*v1.Secret)
	onUpdateFn func(*v1.Secret)
	onDeleteFn func()
	stopCh     chan struct{}
}

var _ cache.ResourceEventHandler = &SecretInformer{}

// NewSecretInformer creates a new secret informer.
func NewSecretInformer(
	namespace string,
	secretName string,
	k8sClient kubernetes.Interface,
	onAddFn func(*v1.Secret),
	onUpdateFn func(*v1.Secret),
	onDeleteFn func(),
) *SecretInformer {
	return &SecretInformer{
		namespace:  namespace,
		secretName: secretName,
		k8sClient:  k8sClient,
		onAddFn:    onAddFn,
		onUpdateFn: onUpdateFn,
		onDeleteFn: onDeleteFn,
		stopCh:     make(chan struct{}),
	}
}

// Start initiates the secret informer loop.
func (c *SecretInformer) Start() error {
	nsOption := informers.WithNamespace(c.namespace)
	labelOption := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.FieldSelector = "metadata.name=" + c.secretName
	})
	sif := informers.NewSharedInformerFactoryWithOptions(c.k8sClient, resyncTime, nsOption, labelOption)

	if _, err := sif.Core().V1().Secrets().Informer().AddEventHandler(c); err != nil {
		return errors.Wrap(err, "could not add event handler")
	}
	sif.Start(c.stopCh)
	sif.WaitForCacheSync(c.stopCh)

	return nil
}

// Stop ends the secret informer loop.
func (c *SecretInformer) Stop() {
	close(c.stopCh)
}

// OnAdd is called when the secret is added.
func (c *SecretInformer) OnAdd(obj interface{}, _ bool) {
	if secret, ok := obj.(*v1.Secret); ok {
		c.onAddFn(secret)
	}
}

// OnUpdate is called when the secret is updated.
func (c *SecretInformer) OnUpdate(_, newObj interface{}) {
	if secret, ok := newObj.(*v1.Secret); ok {
		c.onUpdateFn(secret)
	}
}

// OnDelete is called when the secret is deleted.
func (c *SecretInformer) OnDelete(_ interface{}) {
	c.onDeleteFn()
}
