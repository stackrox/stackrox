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
			k8sClient := fake.NewSimpleClientset()

			var wgAdd, wgUp, wgDel sync.WaitGroup
			wgAdd.Add(c.expectedOnAddCnt)
			wgUp.Add(c.expectedOnUpdateCnt)
			wgDel.Add(c.expectedOnDeleteCnt)

			informer := NewSecretInformer(
				namespace,
				secretName,
				k8sClient,
				func(s *v1.Secret) {
					assert.Equal(t, c.expectedData, string(s.Data[secretKey]))
					wgAdd.Done()
				},
				func(s *v1.Secret) {
					assert.Equal(t, c.expectedData, string(s.Data[secretKey]))
					wgUp.Done()
				},
				func() {
					wgDel.Done()
				},
			)

			err := informer.Start()
			require.NoError(t, err)
			defer informer.Stop()
			require.Eventually(t, informer.HasSynced, 5*time.Second, 100*time.Millisecond)
			err = c.setupFn(k8sClient)
			require.NoError(t, err)

			wgAdd.Wait()
			wgUp.Wait()
			wgDel.Wait()
			// Test is OK if not killed after timeout.
		})
	}
}
