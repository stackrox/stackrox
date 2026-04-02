package secretinformer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

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

			// Use Eventually to retry the entire operation with a configurable timeout.
			// This handles the k8s fake client potentially losing events.
			require.EventuallyWithT(t, func(ct *assert.CollectT) {
				k8sClient := fake.NewClientset()
				informer := NewSecretInformer(
					namespace,
					secretName,
					k8sClient,
					func(s *v1.Secret) {
						assert.Equal(ct, c.expectedData, string(s.Data[secretKey]))
						onAddCnt.Add(1)
					},
					func(s *v1.Secret) {
						assert.Equal(ct, c.expectedData, string(s.Data[secretKey]))
						onUpdateCnt.Add(1)
					},
					func() {
						onDeleteCnt.Add(1)
					},
				)
				err := informer.Start()
				require.NoError(ct, err)
				defer informer.Stop()
				require.Eventually(ct, informer.HasSynced, 5*time.Second, 100*time.Millisecond)

				onAddCnt.Store(0)
				onUpdateCnt.Store(0)
				onDeleteCnt.Store(0)
				require.NoError(ct, c.setupFn(k8sClient))

				assert.Eventually(ct, func() bool {
					return onAddCnt.Load() == int32(c.expectedOnAddCnt) &&
						onUpdateCnt.Load() == int32(c.expectedOnUpdateCnt) &&
						onDeleteCnt.Load() == int32(c.expectedOnDeleteCnt)
				}, 200*time.Millisecond, 10*time.Millisecond)
			}, 10*time.Second, 200*time.Millisecond, "callbacks not invoked as expected (add: %d/%d, update: %d/%d, delete: %d/%d)",
				onAddCnt.Load(), c.expectedOnAddCnt, onUpdateCnt.Load(), c.expectedOnUpdateCnt,
				onDeleteCnt.Load(), c.expectedOnDeleteCnt)
		})
	}
}
