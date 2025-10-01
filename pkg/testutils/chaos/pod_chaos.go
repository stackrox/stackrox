package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

var log = logging.LoggerForModule()

// PodChaos provides chaos engineering for Kubernetes pods via random deletion
type PodChaos struct {
	k8sClient    kubernetes.Interface
	namespace    string
	podSelector  map[string]string
	enabled      bool
	killInterval time.Duration
	maxKills     int
	killCount    int
}

// Config configures the chaos monkey behavior
type Config struct {
	Namespace    string
	PodSelector  map[string]string
	KillInterval time.Duration
	MaxKills     int
	Enabled      bool
}

// NewPodChaos creates a new chaos monkey for pods matching the given selector
func NewPodChaos(k8sClient kubernetes.Interface, config *Config) *PodChaos {
	return &PodChaos{
		k8sClient:    k8sClient,
		namespace:    config.Namespace,
		podSelector:  config.PodSelector,
		enabled:      config.Enabled,
		killInterval: config.KillInterval,
		maxKills:     config.MaxKills,
		killCount:    0,
	}
}

// Start begins the chaos monkey operations
func (c *PodChaos) Start(ctx context.Context) error {
	if !c.enabled {
		log.Info("Pod chaos disabled, skipping")
		return nil
	}

	labelSelector := labels.SelectorFromSet(c.podSelector).String()
	log.Infof("Starting pod chaos monkey in namespace %s with selector %s", c.namespace, labelSelector)

	// Start chaos operations in a goroutine
	go func() {
		ticker := time.NewTicker(c.killInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Pod chaos stopped due to context cancellation")
				return
			case <-ticker.C:
				if c.killCount >= c.maxKills {
					log.Infof("Pod chaos reached max kills (%d), stopping", c.maxKills)
					return
				}

				if err := c.killRandomPod(ctx); err != nil {
					log.Errorf("Failed to kill pod: %v", err)
				}
			}
		}
	}()

	return nil
}

// killRandomPod kills a random pod matching the selector
func (c *PodChaos) killRandomPod(ctx context.Context) error {
	// List pods matching the selector
	labelSelector := labels.SelectorFromSet(c.podSelector).String()
	podList, err := c.k8sClient.CoreV1().Pods(c.namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(podList.Items) == 0 {
		log.Warnf("No pods found with selector %s in namespace %s", labelSelector, c.namespace)
		return nil
	}

	// Select a random pod
	randomIndex := rand.Intn(len(podList.Items))
	targetPod := podList.Items[randomIndex]

	log.Infof("Pod chaos killing pod: %s", targetPod.Name)

	// Delete the pod with zero grace period for immediate termination
	gracePeriodSeconds := int64(0)
	err = c.k8sClient.CoreV1().Pods(c.namespace).Delete(ctx, targetPod.Name, metaV1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s: %w", targetPod.Name, err)
	}

	c.killCount++
	log.Infof("Successfully killed pod %s (kill count: %d/%d)", targetPod.Name, c.killCount, c.maxKills)

	return nil
}

// Stop stops the chaos monkey (implemented via context cancellation)
func (c *PodChaos) Stop() {
	log.Info("Pod chaos stop requested")
}

// GetKillCount returns the number of pods killed so far
func (c *PodChaos) GetKillCount() int {
	return c.killCount
}

// IsEnabled returns whether chaos monkey is enabled
func (c *PodChaos) IsEnabled() bool {
	return c.enabled
}

// WaitForPodRecovery waits for pods to be ready again after chaos
func (c *PodChaos) WaitForPodRecovery(ctx context.Context, timeout time.Duration) error {
	labelSelector := labels.SelectorFromSet(c.podSelector).String()
	log.Infof("Waiting for pod recovery with selector %s (timeout: %v)", labelSelector, timeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pod recovery: %w", ctx.Err())
		case <-ticker.C:
			podList, err := c.k8sClient.CoreV1().Pods(c.namespace).List(ctx, metaV1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				log.Errorf("Failed to list pods during recovery check: %v", err)
				continue
			}

			if len(podList.Items) == 0 {
				log.Debug("No pods found, waiting...")
				continue
			}

			// Check if all pods are ready
			allReady := true
			for _, pod := range podList.Items {
				if pod.Status.Phase != "Running" {
					allReady = false
					break
				}

				// Check container readiness
				for _, condition := range pod.Status.Conditions {
					if condition.Type == "Ready" && condition.Status != "True" {
						allReady = false
						break
					}
				}

				if !allReady {
					break
				}
			}

			if allReady {
				log.Infof("All pods recovered (%d pods ready)", len(podList.Items))
				return nil
			}

			log.Debugf("Pods not fully ready yet (%d pods)", len(podList.Items))
		}
	}
}

// TestWrapper provides helper functions for running tests with chaos
type TestWrapper struct {
	chaos   *PodChaos
	timeout time.Duration
}

// NewTestWrapper creates a test wrapper for chaos operations
func NewTestWrapper(k8sClient kubernetes.Interface, config *Config, timeout time.Duration) *TestWrapper {
	return &TestWrapper{
		chaos:   NewPodChaos(k8sClient, config),
		timeout: timeout,
	}
}

// RunWithChaos executes a test function while chaos monkey is active
func (w *TestWrapper) RunWithChaos(ctx context.Context, testFunc func(context.Context) error) error {
	// Start chaos monkey
	if err := w.chaos.Start(ctx); err != nil {
		return fmt.Errorf("failed to start chaos monkey: %w", err)
	}

	// Wait a bit for chaos to potentially trigger
	time.Sleep(2 * time.Second)

	// Run the test function
	testErr := testFunc(ctx)

	// Wait for system recovery
	if err := w.chaos.WaitForPodRecovery(ctx, w.timeout); err != nil {
		log.Errorf("Failed to wait for pod recovery: %v", err)
		// Don't fail the test due to recovery timeout, log it instead
	}

	return testErr
}

// Convenience config builders for common StackRox components

// AdmissionControllerConfig returns chaos configuration for admission controller pods
func AdmissionControllerConfig(namespace string) *Config {
	return &Config{
		Namespace: namespace,
		PodSelector: map[string]string{
			"app": "admission-control",
		},
		KillInterval: 5 * time.Minute,
		MaxKills:     3,
		Enabled:      true,
	}
}

// SensorConfig returns chaos configuration for sensor pods
func SensorConfig(namespace string) *Config {
	return &Config{
		Namespace: namespace,
		PodSelector: map[string]string{
			"app": "sensor",
		},
		KillInterval: 3 * time.Minute,
		MaxKills:     5,
		Enabled:      true,
	}
}

// CentralConfig returns chaos configuration for central pods
func CentralConfig(namespace string) *Config {
	return &Config{
		Namespace: namespace,
		PodSelector: map[string]string{
			"app": "central",
		},
		KillInterval: 10 * time.Minute,
		MaxKills:     2,
		Enabled:      true,
	}
}
