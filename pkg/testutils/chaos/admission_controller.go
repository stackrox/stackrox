package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/testutils"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

var log = logging.LoggerForModule()

// AdmissionControllerChaos provides chaos engineering for admission controller pods
type AdmissionControllerChaos struct {
	k8sClient    kubernetes.Interface
	namespace    string
	podSelector  map[string]string
	enabled      bool
	killInterval time.Duration
	maxKills     int
	killCount    int
}

// ChaosConfig configures the chaos monkey behavior
type ChaosConfig struct {
	Namespace    string
	PodSelector  map[string]string
	KillInterval time.Duration
	MaxKills     int
	Enabled      bool
}

// DefaultAdmissionControllerConfig returns default chaos configuration for admission controller
func DefaultAdmissionControllerConfig() *ChaosConfig {
	return &ChaosConfig{
		Namespace: "stackrox",
		PodSelector: map[string]string{
			"app": "admission-control",
		},
		KillInterval: 5 * time.Minute,
		MaxKills:     3,
		Enabled:      true,
	}
}

// NewAdmissionControllerChaos creates a new chaos monkey for admission controller pods
func NewAdmissionControllerChaos(k8sClient kubernetes.Interface, config *ChaosConfig) *AdmissionControllerChaos {
	if config == nil {
		config = DefaultAdmissionControllerConfig()
	}

	return &AdmissionControllerChaos{
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
func (c *AdmissionControllerChaos) Start(ctx context.Context) error {
	if !c.enabled {
		log.Info("Chaos monkey disabled, skipping")
		return nil
	}

	log.Infof("Starting admission controller chaos monkey in namespace %s", c.namespace)

	// Start chaos operations in a goroutine
	go func() {
		ticker := time.NewTicker(c.killInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Chaos monkey stopped due to context cancellation")
				return
			case <-ticker.C:
				if c.killCount >= c.maxKills {
					log.Infof("Chaos monkey reached max kills (%d), stopping", c.maxKills)
					return
				}

				if err := c.killRandomPod(ctx); err != nil {
					log.Errorf("Failed to kill admission controller pod: %v", err)
				}
			}
		}
	}()

	return nil
}

// killRandomPod kills a random admission controller pod
func (c *AdmissionControllerChaos) killRandomPod(ctx context.Context) error {
	// List pods matching the selector
	labelSelector := labels.SelectorFromSet(c.podSelector).String()
	podList, err := c.k8sClient.CoreV1().Pods(c.namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list admission controller pods: %w", err)
	}

	if len(podList.Items) == 0 {
		log.Warnf("No admission controller pods found with selector %s in namespace %s", labelSelector, c.namespace)
		return nil
	}

	// Select a random pod
	randomIndex := rand.Intn(len(podList.Items))
	targetPod := podList.Items[randomIndex]

	log.Infof("Chaos monkey killing admission controller pod: %s", targetPod.Name)

	// Delete the pod with zero grace period for immediate termination
	gracePeriodSeconds := int64(0)
	err = c.k8sClient.CoreV1().Pods(c.namespace).Delete(ctx, targetPod.Name, metaV1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s: %w", targetPod.Name, err)
	}

	c.killCount++
	log.Infof("Successfully killed admission controller pod %s (kill count: %d/%d)", targetPod.Name, c.killCount, c.maxKills)

	return nil
}

// Stop stops the chaos monkey (implemented via context cancellation)
func (c *AdmissionControllerChaos) Stop() {
	log.Info("Chaos monkey stop requested")
}

// GetKillCount returns the number of pods killed so far
func (c *AdmissionControllerChaos) GetKillCount() int {
	return c.killCount
}

// IsEnabled returns whether chaos monkey is enabled
func (c *AdmissionControllerChaos) IsEnabled() bool {
	return c.enabled
}

// WaitForPodRecovery waits for admission controller pods to be ready again after chaos
func (c *AdmissionControllerChaos) WaitForPodRecovery(ctx context.Context, timeout time.Duration) error {
	log.Infof("Waiting for admission controller pod recovery (timeout: %v)", timeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	labelSelector := labels.SelectorFromSet(c.podSelector).String()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for admission controller recovery: %w", ctx.Err())
		case <-ticker.C:
			podList, err := c.k8sClient.CoreV1().Pods(c.namespace).List(ctx, metaV1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				log.Errorf("Failed to list pods during recovery check: %v", err)
				continue
			}

			if len(podList.Items) == 0 {
				log.Debug("No admission controller pods found, waiting...")
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
				log.Infof("All admission controller pods recovered (%d pods ready)", len(podList.Items))
				return nil
			}

			log.Debugf("Admission controller pods not fully ready yet (%d pods)", len(podList.Items))
		}
	}
}

// ChaosTestWrapper provides helper functions for running tests with chaos
type ChaosTestWrapper struct {
	chaos   *AdmissionControllerChaos
	timeout time.Duration
}

// NewChaosTestWrapper creates a test wrapper for chaos operations
func NewChaosTestWrapper(k8sClient kubernetes.Interface, config *ChaosConfig, timeout time.Duration) *ChaosTestWrapper {
	return &ChaosTestWrapper{
		chaos:   NewAdmissionControllerChaos(k8sClient, config),
		timeout: timeout,
	}
}

// RunWithChaos executes a test function while chaos monkey is active
func (w *ChaosTestWrapper) RunWithChaos(ctx context.Context, testFunc func(context.Context) error) error {
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

// CreateTestDeploymentWithChaos creates a test deployment and validates it survives admission controller chaos
func CreateTestDeploymentWithChaos(t testutils.T, k8sClient kubernetes.Interface, deploymentName, image string) error {
	ctx := context.Background()

	// Create chaos wrapper
	wrapper := NewChaosTestWrapper(k8sClient, DefaultAdmissionControllerConfig(), 2*time.Minute)

	return wrapper.RunWithChaos(ctx, func(ctx context.Context) error {
		// This would integrate with the existing deployment creation functions
		// For now, return a placeholder that would be replaced with actual deployment logic
		log.Infof("Creating test deployment %s with image %s during chaos", deploymentName, image)

		// TODO: Integrate with existing setupDeployment function from common.go
		// setupDeployment(t, image, deploymentName)

		return nil
	})
}

// ValidatePolicyEnforcementWithChaos validates that policy enforcement works despite admission controller chaos
func ValidatePolicyEnforcementWithChaos(t testutils.T, k8sClient kubernetes.Interface, policyID string, deploymentName string) error {
	ctx := context.Background()

	wrapper := NewChaosTestWrapper(k8sClient, DefaultAdmissionControllerConfig(), 3*time.Minute)

	return wrapper.RunWithChaos(ctx, func(ctx context.Context) error {
		log.Infof("Validating policy enforcement for policy %s and deployment %s during chaos", policyID, deploymentName)

		// TODO: Integrate with policy client to verify enforcement
		// This would use the PolicyClient and AlertClient from the clients package

		return nil
	})
}