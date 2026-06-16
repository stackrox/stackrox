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
		preCreateSecret     bool
		setupFn             func(t *testing.T, k8sClient *fake.Clientset)
		expectedOnAddCnt    int
		expectedOnUpdateCnt int
		expectedOnDeleteCnt int
		expectedData        string
	}{
		"secret added": {
			setupFn: func(t *testing.T, k8sClient *fake.Clientset) {
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
				require.NoError(t, err)
			},
			expectedOnAddCnt: 1,
			expectedData:     secretData,
		},
		"secret updated": {
			setupFn: func(t *testing.T, k8sClient *fake.Clientset) {
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
				require.NoError(t, err)
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
				require.NoError(t, err)
			},
			expectedOnAddCnt:    1,
			expectedOnUpdateCnt: 1,
			expectedData:        secretData,
		},
		"secret deleted": {
			preCreateSecret: true,
			setupFn: func(t *testing.T, k8sClient *fake.Clientset) {
				err := k8sClient.CoreV1().Secrets(namespace).Delete(
					context.Background(), secretName, metav1.DeleteOptions{},
				)
				require.NoError(t, err)
			},
			expectedOnAddCnt:    1,
			expectedOnDeleteCnt: 1,
			expectedData:        secretData,
		},
		"no secret": {
			setupFn: func(_ *testing.T, _ *fake.Clientset) {},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var onAddCnt, onUpdateCnt, onDeleteCnt atomic.Int32

			k8sClient := fake.NewClientset()
			if c.preCreateSecret {
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
				require.NoError(t, err)
			}

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
			require.Eventually(t, informer.HasSynced, 30*time.Second, 100*time.Millisecond)

			if c.preCreateSecret {
				require.Eventually(t, func() bool {
					return onAddCnt.Load() > 0
				}, 30*time.Second, 50*time.Millisecond,
					"add callback not invoked for pre-created secret")
			}

			c.setupFn(t, k8sClient)

			assert.Eventually(t, func() bool {
				return onAddCnt.Load() == int32(c.expectedOnAddCnt) &&
					onUpdateCnt.Load() == int32(c.expectedOnUpdateCnt) &&
					onDeleteCnt.Load() == int32(c.expectedOnDeleteCnt)
			}, 30*time.Second, 50*time.Millisecond,
				"callbacks not invoked as expected (add: %d/%d, update: %d/%d, delete: %d/%d)",
				onAddCnt.Load(), c.expectedOnAddCnt, onUpdateCnt.Load(), c.expectedOnUpdateCnt,
				onDeleteCnt.Load(), c.expectedOnDeleteCnt)
		})
	}
}
