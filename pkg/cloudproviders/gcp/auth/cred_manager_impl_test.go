package auth

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

			k8sClient := fake.NewClientset()

			watchRegistered := make(chan struct{})
			var watchOnce sync.Once
			k8sClient.PrependWatchReactor("secrets", func(action k8stesting.Action) (bool, watch.Interface, error) {
				w, err := k8sClient.Tracker().Watch(action.GetResource(), action.GetNamespace())
				watchOnce.Do(func() { close(watchRegistered) })
				return true, w, err
			})

			manager := newCredentialsManagerImpl(k8sClient, namespace, secretName, func() {
				changeCount.Add(1)
			})
			manager.Start()
			defer manager.Stop()

			// Wait HasSynced first.
			require.Eventually(t, manager.informer.HasSynced, 30*time.Second, 100*time.Millisecond)

			// Wait that watch is executed and events will be properly received.
			select {
			case <-watchRegistered:
			case <-time.After(10 * time.Second):
				require.FailNow(t, "timed out waiting for watch to be registered")
			}

			require.NoError(t, c.setupFn(k8sClient))

			assert.Eventually(t, func() bool {
				manager.mutex.RLock()
				defer manager.mutex.RUnlock()
				return changeCount.Load() == int32(c.changes) &&
					string(manager.stsConfig) == c.expected
			}, 10*time.Second, 50*time.Millisecond,
				"callbacks not invoked as expected or state incorrect (changes: %d/%d)",
				changeCount.Load(), c.changes)
		})
	}
}
