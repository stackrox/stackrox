package k8scfgwatch

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"go.uber.org/zap"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

var log = logging.LoggerForModule()

type ConfigMapWatcher struct {
	k8sClient    *kubernetes.Clientset
	modifiedFunc func(*v1.ConfigMap)
}

func NewConfigMapWatcher(k8sClient *kubernetes.Clientset, modifiedFunc func(*v1.ConfigMap)) *ConfigMapWatcher {
	return &ConfigMapWatcher{k8sClient: k8sClient, modifiedFunc: modifiedFunc}
}

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
				default:
				}
			} else {
				// If eventChannel is closed the server has closed the connection.
				// We want to return and create another watcher.
				return
			}
		}
	}
}
