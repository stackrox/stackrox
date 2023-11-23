package gcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	fakeSTSConfig = `{
   "type": "external_account",
   "audience": "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/test-pool/providers/test-provider",
   "subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
   "token_url": "https://sts.googleapis.com/v1/token",
   "service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/fake@stackrox.com:generateAccessToken",
   "credential_source": {
      "file": "/var/run/secrets/openshift/serviceaccount/token",
      "format": {
         "type": "text"
      }
   }
}`
	namespace  = "stackrox"
	secretName = "gcp-cloud-credentials" // #nosec G101
)

func TestCredentialManager(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		setupFn  func(k8sClient *fake.Clientset) error
		expected string
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
			expected: fakeSTSConfig,
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
			expected: fakeSTSConfig,
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
			expected: "",
		},
		"no secret": {
			setupFn:  func(k8sClient *fake.Clientset) error { return nil },
			expected: "",
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			k8sClient := fake.NewSimpleClientset()
			manager := newCredentialsManagerImpl(k8sClient, namespace, secretName, func() {})
			manager.Start()
			defer manager.Stop()

			err := c.setupFn(k8sClient)
			require.NoError(t, err)

			// Assert that the secret data has been updated.
			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				manager.mutex.RLock()
				defer manager.mutex.RUnlock()
				assert.Equal(t, []byte(c.expected), manager.stsConfig)
			}, 5*time.Second, 100*time.Millisecond)
		})
	}
}
