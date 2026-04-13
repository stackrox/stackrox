package k8scfgwatch

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

var (
	log          = logging.LoggerForModule()
	timeout      = 10 * time.Second
	retryBackoff = 30 * time.Second

	configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
)

// ConfigMapWatcher watches a config map in a given namespaces and evokes a callback function
// when changes are detected.
type ConfigMapWatcher struct {
	dynamicClient dynamic.Interface
	modifiedFunc  func(*v1.ConfigMap)
}

// NewConfigMapWatcher creates a new config map watcher.
func NewConfigMapWatcher(dynamicClient dynamic.Interface, modifiedFunc func(*v1.ConfigMap)) *ConfigMapWatcher {
	return &ConfigMapWatcher{dynamicClient: dynamicClient, modifiedFunc: modifiedFunc}
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
	unstructured, err := w.dynamicClient.Resource(configMapGVR).Namespace(namespace).Get(initCtx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var cfgMap v1.ConfigMap
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &cfgMap); err != nil {
		return err
	}
	w.modifiedFunc(&cfgMap)
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
	watcher, err := w.dynamicClient.Resource(configMapGVR).Namespace(namespace).Watch(
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
			uns, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}
			var cm v1.ConfigMap
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.Object, &cm); err != nil {
				log.Errorw("Failed to convert ConfigMap from unstructured", logging.Err(err))
				continue
			}
			w.modifiedFunc(&cm)
		}
	}
}
