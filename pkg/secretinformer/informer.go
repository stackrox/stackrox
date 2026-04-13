package secretinformer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8swatch"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
)

var (
	log = logging.LoggerForModule()
)

// SecretInformer is a convenience wrapper around a Kubernetes watcher for a specific secret.
// It uses a custom HTTP-based watch implementation to avoid pulling in the entire
// client-go/informers dependency tree.
type SecretInformer struct {
	namespace  string
	secretName string

	_          dynamic.Interface // deprecated parameter, kept for backwards compatibility only
	k8sClient  *http.Client
	onAddFn    func(*v1.Secret)
	onUpdateFn func(*v1.Secret)
	onDeleteFn func()
	stopCh     concurrency.Signal
	hasSynced  concurrency.Signal
	cancelFunc context.CancelFunc
}

// NewSecretInformer creates a new secret informer.
// The dynamicClient parameter is deprecated and unused but kept for backwards compatibility.
func NewSecretInformer(
	namespace string,
	secretName string,
	_ dynamic.Interface, // deprecated, unused
	onAddFn func(*v1.Secret),
	onUpdateFn func(*v1.Secret),
	onDeleteFn func(),
) *SecretInformer {
	return &SecretInformer{
		namespace:  namespace,
		secretName: secretName,
		k8sClient:  k8swatch.InClusterClient(),
		onAddFn:    onAddFn,
		onUpdateFn: onUpdateFn,
		onDeleteFn: onDeleteFn,
		stopCh:     concurrency.NewSignal(),
		hasSynced:  concurrency.NewSignal(),
	}
}

// Start initiates the secret informer loop using a custom HTTP watch.
func (c *SecretInformer) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	go func() {
		<-c.stopCh.WaitC()
		cancel()
	}()

	go c.watchSecret(ctx)
	return nil
}

// watchSecret implements a custom watch for a single secret using HTTP streaming.
func (c *SecretInformer) watchSecret(ctx context.Context) {
	// Do initial LIST to get current state
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", c.namespace, c.secretName)
	baseURL := "https://kubernetes.default.svc"

	// Initial GET to check if secret exists
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+apiPath, nil)
	if err != nil {
		log.Warnf("secretinformer: failed to create initial request for %s/%s: %v", c.namespace, c.secretName, err)
		c.hasSynced.Signal()
		return
	}

	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Warnf("secretinformer: failed to read token: %v", err)
		c.hasSynced.Signal()
		return
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	resp, err := c.k8sClient.Do(req)
	if err != nil {
		log.Debugf("secretinformer: initial GET failed for %s/%s: %v (secret may not exist yet)", c.namespace, c.secretName, err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var secret v1.Secret
			if err := json.NewDecoder(resp.Body).Decode(&secret); err == nil {
				c.onAddFn(&secret)
			}
		}
	}

	c.hasSynced.Signal()

	// Now watch for changes
	watchPath := fmt.Sprintf("/api/v1/namespaces/%s/secrets?watch=true&fieldSelector=metadata.name=%s", c.namespace, c.secretName)

	watcher := k8swatch.New(watchPath, c.k8sClient, func(eventType string, raw json.RawMessage) {
		var secret v1.Secret
		if err := json.Unmarshal(raw, &secret); err != nil {
			log.Warnf("secretinformer: failed to unmarshal %s event: %v", eventType, err)
			return
		}

		switch eventType {
		case "ADDED":
			c.onAddFn(&secret)
		case "MODIFIED":
			c.onUpdateFn(&secret)
		case "DELETED":
			c.onDeleteFn()
		}
	})

	watcher.Run(ctx)
}

// Stop ends the secret informer loop.
func (c *SecretInformer) Stop() {
	c.stopCh.Signal()
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

// HasSynced reports if the informer handler has synced, meaning it has had
// all items in the initial list delivered.
func (c *SecretInformer) HasSynced() bool {
	return c.hasSynced.IsDone()
}
