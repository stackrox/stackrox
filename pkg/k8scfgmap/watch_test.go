package k8scfgmap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8sTest "k8s.io/client-go/testing"
)

var (
	cfgName      = "test-cm"
	cfgNamespace = "test-ns"
	cfgKey       = "test-key"
	cfgValue     = "test-value"
)

func TestConfigMapTrigger(t *testing.T) {
	cases := map[string]struct {
		triggerFunc func(*watch.FakeWatcher, *v1.ConfigMap)
	}{
		"config map added": {
			triggerFunc: func(watcher *watch.FakeWatcher, cm *v1.ConfigMap) {
				if !watcher.IsStopped() {
					watcher.Add(cm)
				}
			},
		},
		"config map modified": {
			triggerFunc: func(watcher *watch.FakeWatcher, cm *v1.ConfigMap) {
				if !watcher.IsStopped() {
					watcher.Modify(cm)
				}
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			k8sClient := fake.NewSimpleClientset()
			watcher := watch.NewFake()
			watchReaction := &WatchReactor{
				Watcher: watcher,
			}
			k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReaction}
			actualValue := ""
			cfgWatcher := NewConfigMapWatcher(k8sClient, func(cm *v1.ConfigMap) {
				actualValue = cm.Data[cfgKey]
			})
			go cfgWatcher.Watch(context.TODO(), cfgNamespace, cfgName)

			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgName, Namespace: cfgNamespace},
				Data:       map[string]string{cfgKey: cfgValue},
			}
			c.triggerFunc(watcher, cm)
			// Should be long enough to load the client CA in the background.
			time.Sleep(500 * time.Millisecond)

			assert.Equal(t, cfgValue, actualValue)
		})
	}
}

func TestConfigMapContextCancelled(t *testing.T) {
	cases := map[string]struct {
		triggerFunc func(*watch.FakeWatcher, *v1.ConfigMap)
	}{
		"config map added": {
			triggerFunc: func(watcher *watch.FakeWatcher, cm *v1.ConfigMap) {
				if !watcher.IsStopped() {
					watcher.Add(cm)
				}
			},
		},
		"config map modified": {
			triggerFunc: func(watcher *watch.FakeWatcher, cm *v1.ConfigMap) {
				if !watcher.IsStopped() {
					watcher.Modify(cm)
				}
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			k8sClient := fake.NewSimpleClientset()
			watcher := watch.NewFake()
			watchReaction := &WatchReactor{
				Watcher: watcher,
			}
			k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReaction}
			actualValue := ""
			cfgWatcher := NewConfigMapWatcher(k8sClient, func(cm *v1.ConfigMap) {
				actualValue = cm.Data[cfgKey]
			})
			ctx, cancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
			defer cancel()
			go cfgWatcher.Watch(ctx, cfgNamespace, cfgName)
			time.Sleep(100 * time.Millisecond)

			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgName, Namespace: cfgNamespace},
				Data:       map[string]string{cfgKey: cfgValue},
			}
			c.triggerFunc(watcher, cm)

			assert.Empty(t, actualValue)
		})
	}
}
