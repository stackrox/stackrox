package k8scfgwatch

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	log          = logging.LoggerForModule()
	timeout      = 10 * time.Second
	retryBackoff = 30 * time.Second
)

// ConfigMapWatcher watches a config map in a given namespaces and evokes a callback function
// when changes are detected.
type ConfigMapWatcher struct {
	configMapsClient configMapsClient
	modifiedFunc     func(*v1.ConfigMap)
}

// configMapsClient is the minimal interface needed for ConfigMap watching.
// Using this instead of kubernetes.Interface avoids importing the full k8s
// client-go scheme which registers all 58+ API groups at init (~1.3 MB).
type configMapsClient interface {
	CoreV1() corev1client.CoreV1Interface
}

// NewConfigMapWatcher creates a new config map watcher.
func NewConfigMapWatcher(k8sClient configMapsClient, modifiedFunc func(*v1.ConfigMap)) *ConfigMapWatcher {
	return &ConfigMapWatcher{configMapsClient: k8sClient, modifiedFunc: modifiedFunc}
}

// Watch a config map in the given namespace bound by the context.
//
// Performs an initial get of the config map, which is followed by a continuous watch.
func (w *ConfigMapWatcher) Watch(ctx concurrency.Waitable, namespace string, name string) {
	err := w.init(ctx, namespace, name)
	if err != nil {
		log.Errorw(fmt.Sprintf("Failed initial get of config map %s/%s", name, namespace), logging.Err(err))
	}
	go w.run(ctx, namespace, name)
}

func (w *ConfigMapWatcher) init(ctx concurrency.Waitable, namespace string, name string) error {
	initCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(concurrency.AsContext(ctx), timeout)
	defer cancel()
	cfgMap, err := w.configMapsClient.CoreV1().ConfigMaps(namespace).Get(initCtx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	w.modifiedFunc(cfgMap)
	return nil
}

func (w *ConfigMapWatcher) run(ctx concurrency.Waitable, namespace string, name string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			w.startWatcher(ctx, namespace, name)
		}
		concurrency.WaitWithTimeout(ctx, retryBackoff)
	}
}

func (w *ConfigMapWatcher) startWatcher(ctx concurrency.Waitable, namespace string, name string) {
	watcher, err := w.configMapsClient.CoreV1().ConfigMaps(namespace).Watch(
		concurrency.AsContext(ctx),
		metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: namespace}),
	)
	if err != nil {
		log.Errorw(fmt.Sprintf("Unable to start watching config map %s/%s", name, namespace), logging.Err(err))
		return
	}
	defer watcher.Stop()
	w.onChange(ctx, watcher.ResultChan())
}

func (w *ConfigMapWatcher) onChange(ctx concurrency.Waitable, eventChannel <-chan watch.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, open := <-eventChannel:
			// If eventChannel is closed the server has closed the connection.
			// We want to return and create another watcher.
			if !open {
				return
			}

			if event.Type != watch.Added && event.Type != watch.Modified {
				continue
			}
			if cm, ok := event.Object.(*v1.ConfigMap); ok {
				w.modifiedFunc(cm)
			}
		}
	}
}
