package aws

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	fakeSTSConfig = `
    [default]
    sts_regional_endpoints = regional
    role_name: fake-role
    web_identity_token_file: /var/run/secrets/openshift/serviceaccount/token`

	namespace  = "stackrox"
	secretName = "aws-cloud-credentials"
)

func TestCredentialManager(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		setupFn         func(k8sClient *fake.Clientset) error
		fileExists      bool
		expectedContent string
	}{
		"secret added": {
			setupFn: func(k8sClient *fake.Clientset) error {
				_, err := k8sClient.CoreV1().Secrets(namespace).Create(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							cloudCredentialsKey: []byte(fakeSTSConfig),
						},
					},
					metav1.CreateOptions{},
				)
				return err
			},
			fileExists:      true,
			expectedContent: fakeSTSConfig,
		},
		"secret updated": {
			setupFn: func(k8sClient *fake.Clientset) error {
				_, err := k8sClient.CoreV1().Secrets(namespace).Create(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							cloudCredentialsKey: []byte("xxx"),
						},
					},
					metav1.CreateOptions{},
				)
				if err != nil {
					return err
				}
				// Allow the state to propagate.
				time.Sleep(10 * time.Millisecond)

				_, err = k8sClient.CoreV1().Secrets(namespace).Update(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							cloudCredentialsKey: []byte(fakeSTSConfig),
						},
					},
					metav1.UpdateOptions{},
				)
				return err
			},
			fileExists:      true,
			expectedContent: fakeSTSConfig,
		},
		"secret deleted": {
			setupFn: func(k8sClient *fake.Clientset) error {
				_, err := k8sClient.CoreV1().Secrets(namespace).Create(
					context.Background(),
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{Name: secretName},
						Data: map[string][]byte{
							cloudCredentialsKey: []byte(fakeSTSConfig),
						},
					},
					metav1.CreateOptions{},
				)
				if err != nil {
					return err
				}
				// Allow the state to propagate.
				time.Sleep(10 * time.Millisecond)

				return k8sClient.CoreV1().Secrets(namespace).Delete(
					context.Background(),
					secretName,
					metav1.DeleteOptions{},
				)
			},
			fileExists: false,
		},
		"no secret": {
			setupFn: func(k8sClient *fake.Clientset) error { return nil },
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			k8sClient := fake.NewSimpleClientset()
			// Randomize file name to make sure test runs don't interfere with each other.
			fileName := fmt.Sprintf("%s-%d", secretName, rand.Int())
			manager := newAWSCredentialsManagerImpl(k8sClient, namespace, secretName, fileName)
			manager.Start()
			defer manager.Stop()

			err := c.setupFn(k8sClient)
			require.NoError(t, err)

			// Assert that the secret data has been updated.
			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				manager.mutex.RLock()
				defer manager.mutex.RUnlock()
				assert.Equal(t, c.fileExists, len(manager.stsConfig) > 0)
				if c.fileExists {
					stsConfig, err := os.ReadFile(manager.mirroredFileName)
					assert.NoError(t, err)
					assert.Equal(t, []byte(c.expectedContent), stsConfig)
				}
			}, 5*time.Second, 100*time.Millisecond)
		})
	}
}
