package k8scfgwatch

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/contextutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/logging/structured"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	k8sTest "k8s.io/client-go/testing"
)

var (
	log          = logging.LoggerForModule()
	timeout      = 10 * time.Second
	retryBackoff = 30 * time.Second
)

// ConfigMapWatcher watches a config map in a given namespaces and evokes a callback function
// when changes are detected.
type ConfigMapWatcher struct {
	k8sClient    kubernetes.Interface
	modifiedFunc func(*v1.ConfigMap)
}

// NewConfigMapWatcher creates a new config map watcher.
func NewConfigMapWatcher(k8sClient kubernetes.Interface, modifiedFunc func(*v1.ConfigMap)) *ConfigMapWatcher {
	return &ConfigMapWatcher{k8sClient: k8sClient, modifiedFunc: modifiedFunc}
}

// Watch a config map in the given namespace bound by the context.
//
// Performs an initial get of the config map, which is followed by a continuous watch.
func (w *ConfigMapWatcher) Watch(ctx concurrency.Waitable, namespace string, name string) {
	err := w.init(ctx, namespace, name)
	if err != nil {
		log.Errorw(fmt.Sprintf("Failed initial get of config map %s/%s", name, namespace), structured.Err(err))
	}
	go w.run(ctx, namespace, name)
}

func (w *ConfigMapWatcher) init(ctx concurrency.Waitable, namespace string, name string) error {
	initCtx, cancel := contextutil.ContextWithTimeoutIfNotExists(concurrency.AsContext(ctx), timeout)
	defer cancel()
	cfgMap, err := w.k8sClient.CoreV1().ConfigMaps(namespace).Get(initCtx, name, metav1.GetOptions{})
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
	watcher, err := w.k8sClient.CoreV1().ConfigMaps(namespace).Watch(
		concurrency.AsContext(ctx),
		metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: namespace}),
	)
	if err != nil {
		log.Errorw(fmt.Sprintf("Unable to start watching config map %s/%s", name, namespace), structured.Err(err))
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

// Used in unit tests to react on watch events.
type testWatchReactor struct {
	watcher watch.Interface
	err     error
}

func (w *testWatchReactor) Handles(_ k8sTest.Action) bool {
	return true
}

func (w *testWatchReactor) React(_ k8sTest.Action) (bool, watch.Interface, error) {
	return true, w.watcher, w.err
}

// NewTestWatchReactor creates a new test watch reactor for testing.
func NewTestWatchReactor(_ *testing.T, watcher watch.Interface) k8sTest.WatchReactor {
	return &testWatchReactor{watcher: watcher}
}
