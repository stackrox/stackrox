package ginkgo

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/chaos"
	"github.com/stackrox/rox/pkg/testutils/clients"
	"github.com/stackrox/rox/pkg/testutils/credentials"
	"github.com/stackrox/rox/pkg/testutils/external"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// StackRoxTestSuite provides a complete testing environment for StackRox E2E tests
type StackRoxTestSuite struct {
	// Core clients
	StackRoxClients *clients.StackRoxClients
	K8sClient       kubernetes.Interface
	Credentials     *credentials.Credentials

	// External service clients
	RegistryClients     []external.RegistryClient
	NotificationClients []external.NotificationClient
	StorageClients      []external.StorageClient

	// Chaos engineering
	ChaosMonkey *chaos.AdmissionControllerChaos

	// Configuration
	config        *SuiteConfig
	cleanupFuncs  []func()
}

// SuiteConfig configures the test suite behavior
type SuiteConfig struct {
	EnableChaos          bool
	ChaosConfig          *chaos.ChaosConfig
	DefaultTimeout       time.Duration
	AlertWaitTimeout     time.Duration
	DeploymentTimeout    time.Duration
	PolicyEnforcementTimeout time.Duration
	ParallelNodes        int
}

// DefaultSuiteConfig returns sensible default configuration
func DefaultSuiteConfig() *SuiteConfig {
	return &SuiteConfig{
		EnableChaos:              false,
		ChaosConfig:              chaos.DefaultAdmissionControllerConfig(),
		DefaultTimeout:           5 * time.Minute,
		AlertWaitTimeout:         2 * time.Minute,
		DeploymentTimeout:        3 * time.Minute,
		PolicyEnforcementTimeout: 2 * time.Minute,
		ParallelNodes:            1,
	}
}

// NewStackRoxTestSuite creates a new test suite with proper resource management
func NewStackRoxTestSuite(config *SuiteConfig) *StackRoxTestSuite {
	if config == nil {
		config = DefaultSuiteConfig()
	}

	return &StackRoxTestSuite{
		config:       config,
		cleanupFuncs: make([]func(), 0),
	}
}

// SetupSuite initializes the test suite (call this in BeforeSuite)
func (s *StackRoxTestSuite) SetupSuite() {
	By("Loading credentials and initializing clients")

	// Load credentials
	var err error
	s.Credentials, err = credentials.Load()
	Expect(err).NotTo(HaveOccurred(), "Failed to load credentials")

	// Create Kubernetes client
	s.K8sClient, err = s.createK8sClient()
	Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client")

	// Create StackRox clients
	s.StackRoxClients = clients.NewStackRoxClients(GinkgoT(), s.Credentials)
	DeferCleanup(func() {
		if s.StackRoxClients != nil {
			s.StackRoxClients.Close()
		}
	})

	// Initialize external service clients
	s.setupExternalServices()

	// Setup chaos monkey if enabled
	if s.config.EnableChaos {
		s.setupChaosMonkey()
	}

	By("Test suite initialization complete")
}

// createK8sClient creates a Kubernetes client using the current kubeconfig
func (s *StackRoxTestSuite) createK8sClient() (kubernetes.Interface, error) {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, fmt.Errorf("could not load default Kubernetes client config: %w", err)
	}

	restCfg, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get REST client config from kubernetes config: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes client from REST config: %w", err)
	}

	return k8sClient, nil
}

// setupExternalServices initializes all available external service clients
func (s *StackRoxTestSuite) setupExternalServices() {
	By("Setting up external service clients")

	// Setup registry clients
	s.RegistryClients = external.GetAvailableRegistries(s.Credentials)
	GinkgoWriter.Printf("Initialized %d registry clients\n", len(s.RegistryClients))

	// Setup notification clients
	s.NotificationClients = external.GetAvailableNotificationClients(s.Credentials)
	GinkgoWriter.Printf("Initialized %d notification clients\n", len(s.NotificationClients))

	// Setup storage clients
	s.StorageClients = external.GetAvailableStorageClients(s.Credentials)
	GinkgoWriter.Printf("Initialized %d storage clients\n", len(s.StorageClients))

	// Log which services are using mocks vs real implementations
	if s.Credentials.ShouldUseMockServices() {
		GinkgoWriter.Printf("Using mock external services (test env: %s)\n", s.Credentials.TestEnv)
	} else {
		GinkgoWriter.Printf("Using real external services (test env: %s)\n", s.Credentials.TestEnv)
	}
}

// setupChaosMonkey initializes chaos engineering
func (s *StackRoxTestSuite) setupChaosMonkey() {
	By("Setting up chaos monkey")

	s.ChaosMonkey = chaos.NewAdmissionControllerChaos(s.K8sClient, s.config.ChaosConfig)
	GinkgoWriter.Printf("Chaos monkey initialized (enabled: %v)\n", s.ChaosMonkey.IsEnabled())
}

// TeardownSuite cleans up the test suite (call this in AfterSuite)
func (s *StackRoxTestSuite) TeardownSuite() {
	By("Cleaning up test suite")

	// Run all registered cleanup functions
	for i := len(s.cleanupFuncs) - 1; i >= 0; i-- {
		s.cleanupFuncs[i]()
	}

	By("Test suite cleanup complete")
}

// RegisterCleanup adds a cleanup function to be called during teardown
func (s *StackRoxTestSuite) RegisterCleanup(cleanup func()) {
	s.cleanupFuncs = append(s.cleanupFuncs, cleanup)
}

// Helper methods for common test operations

// CreateTestPolicy creates a policy with automatic cleanup
func (s *StackRoxTestSuite) CreateTestPolicy(ctx context.Context, config *clients.PolicyConfig) *clients.PolicyConfig {
	policy, err := s.StackRoxClients.Policies.CreatePolicy(ctx, config)
	Expect(err).NotTo(HaveOccurred(), "Failed to create test policy")

	// Schedule cleanup
	DeferCleanup(func() {
		err := s.StackRoxClients.Policies.DeletePolicy(context.Background(), policy.GetId())
		if err != nil {
			GinkgoWriter.Printf("Warning: Failed to cleanup policy %s: %v\n", policy.GetId(), err)
		}
	})

	// Update config with created policy ID
	config.Name = policy.GetName()
	return config
}

// WaitForPolicyAlerts waits for alerts to be generated for a policy
func (s *StackRoxTestSuite) WaitForPolicyAlerts(ctx context.Context, policyID string, minCount int) {
	opts := &clients.AlertWaitOptions{
		Timeout:       s.config.AlertWaitTimeout,
		CheckInterval: 5 * time.Second,
		MinAlertCount: minCount,
	}

	Eventually(func() int {
		alerts, err := s.StackRoxClients.Alerts.GetAlertsForPolicy(ctx, policyID)
		if err != nil {
			return 0
		}
		return len(alerts)
	}, opts.Timeout, opts.CheckInterval).Should(BeNumerically(">=", minCount),
		fmt.Sprintf("Expected at least %d alerts for policy %s", minCount, policyID))
}

// CreateViolatingDeployment creates a deployment that violates specific policy categories
func (s *StackRoxTestSuite) CreateViolatingDeployment(deploymentName string, policyCategory string) {
	By(fmt.Sprintf("Creating violating deployment %s for category %s", deploymentName, policyCategory))

	// This would integrate with existing deployment creation logic
	// TODO: Implement based on existing setupDeployment function

	// Schedule cleanup
	DeferCleanup(func() {
		By(fmt.Sprintf("Cleaning up deployment %s", deploymentName))
		// TODO: Implement based on existing teardownDeployment function
	})
}

// TestWithChaos runs a test function with chaos monkey active
func (s *StackRoxTestSuite) TestWithChaos(ctx context.Context, testFunc func(context.Context)) {
	if !s.config.EnableChaos || s.ChaosMonkey == nil {
		// Run test without chaos
		testFunc(ctx)
		return
	}

	By("Starting chaos monkey")
	err := s.ChaosMonkey.Start(ctx)
	Expect(err).NotTo(HaveOccurred(), "Failed to start chaos monkey")

	// Run the test
	testFunc(ctx)

	By("Waiting for system recovery after chaos")
	err = s.ChaosMonkey.WaitForPodRecovery(ctx, 2*time.Minute)
	Expect(err).NotTo(HaveOccurred(), "System failed to recover from chaos")
}

// BDD Helper Functions

// GivenPolicy creates a policy in the Given step
func (s *StackRoxTestSuite) GivenPolicy(name string, enforcement bool, categories ...string) *clients.PolicyConfig {
	config := &clients.PolicyConfig{
		Name:        name,
		Categories:  categories,
		Enforcement: enforcement,
		Scope:       clients.RuntimeScope,
	}

	return s.CreateTestPolicy(context.Background(), config)
}

// WhenDeploymentCreated creates a deployment in the When step
func (s *StackRoxTestSuite) WhenDeploymentCreated(name, image string) {
	s.CreateViolatingDeployment(name, "test")
}

// ThenAlertGenerated verifies alert generation in the Then step
func (s *StackRoxTestSuite) ThenAlertGenerated(policyID string, expectedCount int) {
	s.WaitForPolicyAlerts(context.Background(), policyID, expectedCount)
}

// SetupParallelExecution configures Ginkgo for parallel test execution
func SetupParallelExecution(parallelNodes int) {
	// This would be called in the main test file to configure Ginkgo
	if parallelNodes > 1 {
		GinkgoWriter.Printf("Configuring parallel execution with %d nodes\n", parallelNodes)
		// Ginkgo parallel configuration would be set here
	}
}

// Helper for creating contexts with appropriate timeouts
func (s *StackRoxTestSuite) NewTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.config.DefaultTimeout)
}

// Helper for creating contexts for policy enforcement tests
func (s *StackRoxTestSuite) NewPolicyTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.config.PolicyEnforcementTimeout)
}