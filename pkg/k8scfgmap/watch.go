package k8scfgmap

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	k8sTest "k8s.io/client-go/testing"
)

var log = logging.LoggerForModule()

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
func (w *ConfigMapWatcher) Watch(ctx concurrency.Waitable, namespace string, name string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			watcher, err := w.k8sClient.CoreV1().ConfigMaps(namespace).Watch(
				concurrency.AsContext(ctx),
				metav1.SingleObject(metav1.ObjectMeta{Name: name, Namespace: namespace}))
			if err != nil {
				log.Errorw("Unable to create config map watcher", zap.Error(err))
				continue
			}
			w.onChange(ctx, watcher.ResultChan())
			watcher.Stop()
		}
	}
}

func (w *ConfigMapWatcher) onChange(ctx concurrency.Waitable, eventChannel <-chan watch.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, open := <-eventChannel:
			if open {
				switch event.Type {
				case watch.Added:
					fallthrough
				case watch.Modified:
					if cm, ok := event.Object.(*v1.ConfigMap); ok {
						w.modifiedFunc(cm)
					}
				}
			} else {
				// If eventChannel is closed the server has closed the connection.
				// We want to return and create another watcher.
				return
			}
		}
	}
}

// WatchReactor is used to test config map watchers.
type WatchReactor struct {
	Action  k8sTest.Action
	Watcher watch.Interface
	Err     error
}

// Handles dummy actions.
func (w *WatchReactor) Handles(_ k8sTest.Action) bool {
	return true
}

// React to watch events.
func (w *WatchReactor) React(_ k8sTest.Action) (bool, watch.Interface, error) {
	return true, w.Watcher, w.Err
}
