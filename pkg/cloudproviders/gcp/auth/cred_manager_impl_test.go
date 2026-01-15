package auth

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
	cases := map[string]struct {
		setupFn  func(k8sClient *fake.Clientset) error
		expected string
		changes  int
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
			changes:  1,
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
			changes:  2,
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

				return k8sClient.CoreV1().Secrets(namespace).Delete(
					context.Background(),
					secretName,
					metav1.DeleteOptions{},
				)
			},
			expected: "",
			changes:  2,
		},
		"no secret": {
			setupFn:  func(k8sClient *fake.Clientset) error { return nil },
			expected: "",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var changeCount atomic.Int32

			// Use Eventually to retry the entire operation with a configurable timeout.
			// This handles the k8s fake client potentially losing events.
			require.EventuallyWithT(t, func(ct *assert.CollectT) {
				k8sClient := fake.NewClientset()
				manager := newCredentialsManagerImpl(k8sClient, namespace, secretName, func() {
					changeCount.Add(1)
				})
				manager.Start()
				defer manager.Stop()
				require.Eventually(ct, manager.informer.HasSynced, 5*time.Second, 100*time.Millisecond)

				changeCount.Store(0)
				require.NoError(ct, c.setupFn(k8sClient))

				assert.Eventually(ct, func() bool {
					manager.mutex.RLock()
					defer manager.mutex.RUnlock()
					return changeCount.Load() == int32(c.changes) &&
						string(manager.stsConfig) == c.expected
				}, 200*time.Millisecond, 10*time.Millisecond)
			}, 10*time.Second, 200*time.Millisecond, "callbacks not invoked as expected or state incorrect (changes: %d/%d)",
				changeCount.Load(), c.changes)
		})
	}
}
