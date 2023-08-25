package k8scfgwatch

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
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
	t.Parallel()
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
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			k8sClient := fake.NewSimpleClientset()
			watcher := watch.NewFake()
			watchReactor := NewTestWatchReactor(t, watcher)
			k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReactor}
			currentCfgData := ""
			mutex := sync.RWMutex{}
			cfgWatcher := NewConfigMapWatcher(k8sClient, func(cm *v1.ConfigMap) {
				mutex.Lock()
				defer mutex.Unlock()
				currentCfgData = cm.Data[cfgKey]
			})
			cfgWatcher.Watch(context.TODO(), cfgNamespace, cfgName)

			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgName, Namespace: cfgNamespace},
				Data:       map[string]string{cfgKey: cfgValue},
			}
			c.triggerFunc(watcher, cm)

			// Assert that the config map data has been updated.
			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				mutex.RLock()
				defer mutex.RUnlock()
				assert.Equal(t, cfgValue, currentCfgData)
			}, 5*time.Second, 100*time.Millisecond)
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
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			k8sClient := fake.NewSimpleClientset()
			watcher := watch.NewFake()
			watchReactor := NewTestWatchReactor(t, watcher)
			k8sClient.WatchReactionChain = []k8sTest.WatchReactor{watchReactor}
			currentCfgData := ""
			mutex := sync.RWMutex{}
			cfgWatcher := NewConfigMapWatcher(k8sClient, func(cm *v1.ConfigMap) {
				mutex.Lock()
				defer mutex.Unlock()
				currentCfgData = cm.Data[cfgKey]
			})
			ctx, cancel := context.WithTimeout(context.TODO(), 50*time.Millisecond)
			defer cancel()
			cfgWatcher.Watch(ctx, cfgNamespace, cfgName)
			time.Sleep(100 * time.Millisecond)

			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: cfgName, Namespace: cfgNamespace},
				Data:       map[string]string{cfgKey: cfgValue},
			}
			c.triggerFunc(watcher, cm)

			// Assert that the config map data has NOT been updated after context cancellation.
			assert.Never(t, func() bool {
				mutex.RLock()
				defer mutex.RUnlock()
				return currentCfgData != ""
			}, 500*time.Millisecond, 50*time.Millisecond)
		})
	}
}
