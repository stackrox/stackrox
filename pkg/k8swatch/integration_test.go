//go:build integration

// Integration tests for k8swatch. Run against a real k8s cluster:
//
//	# Using an existing cluster:
//	KUBECONFIG=~/.kube/config go test -tags integration -v ./pkg/k8swatch/
//
//	# Using KinD:
//	kind create cluster --name k8squatch-test
//	go test -tags integration -v ./pkg/k8swatch/
//	kind delete cluster --name k8squatch-test
//
// These tests create and delete resources in a "k8squatch-test" namespace.
package k8swatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const testNamespace = "k8squatch-test"

// setupCluster returns a k8s clientset and an HTTP client configured for the API server.
func setupCluster(t *testing.T) (kubernetes.Interface, *http.Client, string) {
	t.Helper()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err, "Failed to load kubeconfig from %s", kubeconfig)

	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create kubernetes clientset")

	// Verify cluster is reachable
	_, err = clientset.Discovery().ServerVersion()
	require.NoError(t, err, "Cluster not reachable — is KinD running?")

	// Create test namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})

	// Build an HTTP client that uses the same TLS config as the k8s client
	transport, err := config.TransportConfig()
	require.NoError(t, err)

	tlsConfig, err := transport.TLSConfigFor(transport)
	if err != nil || tlsConfig == nil {
		// Fallback for insecure configs (like KinD with --insecure)
		tlsConfig = nil
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// API server base URL
	baseURL := config.Host

	t.Cleanup(func() {
		clientset.CoreV1().Namespaces().Delete(context.Background(), testNamespace, metav1.DeleteOptions{})
	})

	return clientset, httpClient, baseURL
}

func TestIntegration_WatchConfigMaps(t *testing.T) {
	clientset, httpClient, baseURL := setupCluster(t)

	var events []string
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start watching configmaps in our test namespace
	adapter := NewInformerAdapterForTest(baseURL,
		fmt.Sprintf("/api/v1/namespaces/%s/configmaps", testNamespace),
		httpClient,
		func() runtime.Object { return &corev1.ConfigMap{} },
	)

	adapter.AddEventHandler(&testK8sHandler{
		onEvent: func(eventType, name string) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, fmt.Sprintf("%s/%s", eventType, name))
			t.Logf("Event: %s %s", eventType, name)
		},
	})

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	// Wait for initial sync
	require.Eventually(t, func() bool {
		return adapter.HasSynced()
	}, 10*time.Second, 100*time.Millisecond, "adapter should sync")

	// Create a configmap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm-1", Namespace: testNamespace},
		Data:       map[string]string{"key": "value1"},
	}
	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)

	// Wait for ADDED event
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e == "ADDED/test-cm-1" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "should receive ADDED event")

	// Update the configmap
	cm.Data["key"] = "value2"
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err)

	// Wait for MODIFIED event
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e == "MODIFIED/test-cm-1" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "should receive MODIFIED event")

	// Delete the configmap
	err = clientset.CoreV1().ConfigMaps(testNamespace).Delete(ctx, "test-cm-1", metav1.DeleteOptions{})
	require.NoError(t, err)

	// Wait for DELETED event
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e == "DELETED/test-cm-1" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "should receive DELETED event")

	close(stopCh)

	mu.Lock()
	defer mu.Unlock()
	t.Logf("Total events received: %d", len(events))
	assert.Contains(t, events, "ADDED/test-cm-1")
	assert.Contains(t, events, "MODIFIED/test-cm-1")
	assert.Contains(t, events, "DELETED/test-cm-1")
}

func TestIntegration_InitialListPicksUpExisting(t *testing.T) {
	clientset, httpClient, baseURL := setupCluster(t)

	ctx := context.Background()

	// Create resources BEFORE starting the watcher
	for i := 0; i < 3; i++ {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pre-existing-%d", i),
				Namespace: testNamespace,
			},
			Data: map[string]string{"index": fmt.Sprintf("%d", i)},
		}
		_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(ctx, cm, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	// Now start watching — should receive the pre-existing resources via LIST
	var addedNames []string
	var mu sync.Mutex

	adapter := NewInformerAdapterForTest(baseURL,
		fmt.Sprintf("/api/v1/namespaces/%s/configmaps", testNamespace),
		httpClient,
		func() runtime.Object { return &corev1.ConfigMap{} },
	)

	adapter.AddEventHandler(&testK8sHandler{
		onEvent: func(eventType, name string) {
			if eventType == "ADDED" {
				mu.Lock()
				defer mu.Unlock()
				addedNames = append(addedNames, name)
			}
		},
	})

	stopCh := make(chan struct{})
	go adapter.Run(stopCh)

	require.Eventually(t, func() bool {
		return adapter.HasSynced()
	}, 10*time.Second, 100*time.Millisecond)

	// Give events a moment to deliver
	time.Sleep(500 * time.Millisecond)
	close(stopCh)

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, len(addedNames), 3, "should have received at least 3 pre-existing configmaps")
	for i := 0; i < 3; i++ {
		assert.Contains(t, addedNames, fmt.Sprintf("pre-existing-%d", i))
	}

	// Cleanup
	for i := 0; i < 3; i++ {
		clientset.CoreV1().ConfigMaps(testNamespace).Delete(ctx, fmt.Sprintf("pre-existing-%d", i), metav1.DeleteOptions{})
	}
}

func TestIntegration_WatchMultipleResourceTypes(t *testing.T) {
	_, httpClient, baseURL := setupCluster(t)

	var wg sync.WaitGroup
	results := make(map[string]bool)
	var mu sync.Mutex

	resourcePaths := []struct {
		path    string
		factory func() runtime.Object
		name    string
	}{
		{"/api/v1/namespaces", func() runtime.Object { return &corev1.Namespace{} }, "namespaces"},
		{"/api/v1/services", func() runtime.Object { return &corev1.Service{} }, "services"},
		{"/api/v1/nodes", func() runtime.Object { return &corev1.Node{} }, "nodes"},
	}

	stopCh := make(chan struct{})

	for _, r := range resourcePaths {
		r := r
		wg.Add(1)
		go func() {
			defer wg.Done()
			adapter := NewInformerAdapterForTest(baseURL, r.path, httpClient,
				r.factory,
			)
			adapter.AddEventHandler(&testK8sHandler{
				onEvent: func(eventType, name string) {
					mu.Lock()
					defer mu.Unlock()
					results[r.name] = true
				},
			})
			adapter.Run(stopCh)
		}()
	}

	// Wait for events from multiple resource types
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		// Namespaces should always have events (at least kube-system exists)
		return results["namespaces"]
	}, 10*time.Second, 100*time.Millisecond, "should receive namespace events")

	close(stopCh)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, results["namespaces"], "should have received namespace events")
	t.Logf("Resource types with events: %v", results)
}

func TestIntegration_MemoryFootprint(t *testing.T) {
	_, httpClient, baseURL := setupCluster(t)

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Start 5 watchers (simulating a subset of sensor's informers)
	paths := []string{
		"/api/v1/namespaces",
		"/api/v1/services",
		"/api/v1/configmaps",
		"/api/v1/secrets",
		"/api/v1/serviceaccounts",
	}

	stopCh := make(chan struct{})
	for _, path := range paths {
		path := path
		adapter := NewInformerAdapterForTest(baseURL, path, httpClient,
			func() runtime.Object { return &corev1.ConfigMap{} }, // generic — we just want to measure overhead
		)
		adapter.AddEventHandler(&testK8sHandler{
			onEvent: func(_, _ string) {},
		})
		go adapter.Run(stopCh)
	}

	// Let them sync and stabilize
	time.Sleep(3 * time.Second)
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	close(stopCh)

	heapDelta := int64(memAfter.HeapAlloc) - int64(memBefore.HeapAlloc)
	goroutineDelta := int(memAfter.NumGC) // approximate — goroutine count isn't in MemStats

	t.Logf("Memory for 5 k8squatch watchers:")
	t.Logf("  HeapAlloc before: %d KB", memBefore.HeapAlloc/1024)
	t.Logf("  HeapAlloc after:  %d KB", memAfter.HeapAlloc/1024)
	t.Logf("  Delta:            %d KB", heapDelta/1024)
	t.Logf("  Mallocs delta:    %d", memAfter.Mallocs-memBefore.Mallocs)

	// 5 watchers should use less than 2 MB of heap
	assert.Less(t, heapDelta, int64(2*1024*1024),
		"5 k8squatch watchers should use less than 2 MB heap")
}

// testK8sHandler adapts k8s event handler to a simple callback.
type testK8sHandler struct {
	onEvent func(eventType, name string)
}

func (h *testK8sHandler) OnAdd(obj interface{}, _ bool) {
	if cm, ok := obj.(metav1.ObjectMetaAccessor); ok {
		h.onEvent("ADDED", cm.GetObjectMeta().GetName())
	}
}

func (h *testK8sHandler) OnUpdate(_, newObj interface{}) {
	if cm, ok := newObj.(metav1.ObjectMetaAccessor); ok {
		h.onEvent("MODIFIED", cm.GetObjectMeta().GetName())
	}
}

func (h *testK8sHandler) OnDelete(obj interface{}) {
	if cm, ok := obj.(metav1.ObjectMetaAccessor); ok {
		h.onEvent("DELETED", cm.GetObjectMeta().GetName())
	}
}
