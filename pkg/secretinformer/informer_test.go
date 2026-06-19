package secretinformer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	namespace  = "stackrox"
	secretName = "cloudCredentials"
	secretKey  = "key"
	secretData = "fake data"
)

func TestSecretInformer(t *testing.T) {
	cases := map[string]struct {
		setupFn             func(k8sClient *fake.Clientset) error
		expectedOnAddCnt    int
		expectedOnUpdateCnt int
		expectedOnDeleteCnt int
		expectedData        string
	}{
		"secret added": {
			setupFn: func(k8sClient *fake.Clientset) error {
				_, err := k8sClient.CoreV1().Secrets(namespace).Create(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							secretKey: []byte(secretData),
						},
					},
					metav1.CreateOptions{},
				)
				return err
			},
			expectedOnAddCnt: 1,
			expectedData:     secretData,
		},
		"secret updated": {
			setupFn: func(k8sClient *fake.Clientset) error {
				_, err := k8sClient.CoreV1().Secrets(namespace).Create(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							secretKey: []byte(secretData),
						},
					},
					metav1.CreateOptions{},
				)
				if err != nil {
					return err
				}
				_, err = k8sClient.CoreV1().Secrets(namespace).Update(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							secretKey: []byte(secretData),
						},
					},
					metav1.UpdateOptions{},
				)
				return err
			},
			expectedOnAddCnt:    1,
			expectedOnUpdateCnt: 1,
			expectedData:        secretData,
		},
		"secret deleted": {
			setupFn: func(k8sClient *fake.Clientset) error {
				_, err := k8sClient.CoreV1().Secrets(namespace).Create(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							secretKey: []byte(secretData),
						},
					},
					metav1.CreateOptions{},
				)
				if err != nil {
					return err
				}
				err = k8sClient.CoreV1().Secrets(namespace).Delete(
					context.Background(), secretName, metav1.DeleteOptions{},
				)
				return err
			},
			expectedOnAddCnt:    1,
			expectedOnDeleteCnt: 1,
			expectedData:        secretData,
		},
		"no secret": {
			setupFn: func(k8sClient *fake.Clientset) error {
				return nil
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var onAddCnt, onUpdateCnt, onDeleteCnt atomic.Int32

			k8sClient := fake.NewClientset()

			watchRegistered := make(chan struct{})
			var watchOnce sync.Once
			k8sClient.PrependWatchReactor("secrets", func(action k8stesting.Action) (bool, watch.Interface, error) {
				w, err := k8sClient.Tracker().Watch(action.GetResource(), action.GetNamespace())
				watchOnce.Do(func() { close(watchRegistered) })
				return true, w, err
			})

			informer := NewSecretInformer(
				namespace,
				secretName,
				k8sClient,
				func(s *v1.Secret) {
					assert.Equal(t, c.expectedData, string(s.Data[secretKey]))
					onAddCnt.Add(1)
				},
				func(s *v1.Secret) {
					assert.Equal(t, c.expectedData, string(s.Data[secretKey]))
					onUpdateCnt.Add(1)
				},
				func() {
					onDeleteCnt.Add(1)
				},
			)
			err := informer.Start()
			require.NoError(t, err)
			defer informer.Stop()

			// There is a problem with fake informer. It happens that Go routine that starts informers,
			// marks HasSynced as a true before watchers are registered. Because of that,
			// events are never received. There are no watchers that are listening to these events
			// after HasSynced is true. That's why we need to wait for HasSynced and Watch event.
			require.Eventually(t, informer.HasSynced, 30*time.Second, 100*time.Millisecond)

			// Wait that watch is executed and events will be properly received.
			select {
			case <-watchRegistered:
			case <-time.After(10 * time.Second):
				require.FailNow(t, "timed out waiting for watch to be registered")
			}

			require.NoError(t, c.setupFn(k8sClient))

			assert.Eventually(t, func() bool {
				return onAddCnt.Load() == int32(c.expectedOnAddCnt) &&
					onUpdateCnt.Load() == int32(c.expectedOnUpdateCnt) &&
					onDeleteCnt.Load() == int32(c.expectedOnDeleteCnt)
			}, 10*time.Second, 50*time.Millisecond,
				"callbacks not invoked as expected (add: %d/%d, update: %d/%d, delete: %d/%d)",
				onAddCnt.Load(), c.expectedOnAddCnt, onUpdateCnt.Load(), c.expectedOnUpdateCnt,
				onDeleteCnt.Load(), c.expectedOnDeleteCnt)
		})
	}
}
