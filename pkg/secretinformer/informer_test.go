package secretinformer

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	namespace  = "stackrox"
	secretName = "cloudCredentials"
	secretKey  = "key"
	secretData = "fake data"
)

func TestSecretInformer(t *testing.T) {
	t.Parallel()
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
				// Allow the state to propagate.
				time.Sleep(100 * time.Millisecond)
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
				// Allow the state to propagate.
				time.Sleep(100 * time.Millisecond)
				err = k8sClient.CoreV1().Secrets(namespace).Delete(
					context.Background(), secretName, metav1.DeleteOptions{},
				)
				return err
			},
			expectedOnAddCnt:    1,
			expectedOnDeleteCnt: 1,
			expectedData:        secretData,
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			k8sClient := fake.NewSimpleClientset()
			var onAddCnt, onUpdateCnt, onDeleteCnt int
			var mutex sync.RWMutex
			informer := NewSecretInformer(
				namespace,
				secretName,
				k8sClient,
				func(s *v1.Secret) {
					mutex.Lock()
					defer mutex.Unlock()
					onAddCnt++
					assert.Equal(t, c.expectedData, string(s.Data[secretKey]))
				},
				func(s *v1.Secret) {
					mutex.Lock()
					defer mutex.Unlock()
					onUpdateCnt++
					assert.Equal(t, c.expectedData, string(s.Data[secretKey]))
				},
				func() {
					mutex.Lock()
					defer mutex.Unlock()
					onDeleteCnt++
				},
			)

			err := informer.Start()
			require.NoError(t, err)
			defer informer.Stop()
			err = c.setupFn(k8sClient)
			require.NoError(t, err)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				mutex.RLock()
				defer mutex.RUnlock()
				assert.Equal(t, c.expectedOnAddCnt, onAddCnt)
				assert.Equal(t, c.expectedOnUpdateCnt, onUpdateCnt)
				assert.Equal(t, c.expectedOnDeleteCnt, onDeleteCnt)
			}, 5*time.Second, 100*time.Millisecond)
		})
	}
}
