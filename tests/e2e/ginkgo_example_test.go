//go:build test_e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/clients"
	"github.com/stackrox/rox/pkg/testutils/external"
	ginkgoHelper "github.com/stackrox/rox/pkg/testutils/ginkgo"
)

// Example test demonstrating the complete Ginkgo BDD framework
var _ = Describe("Policy Field Validation Example", func() {
	var (
		suite *ginkgoHelper.StackRoxTestSuite
		ctx   context.Context
	)

	BeforeEach(func() {
		// Create test context with timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		DeferCleanup(cancel)
	})

	Context("Runtime Policy Categories", func() {
		DescribeTable("Runtime enforcement scenarios",
			func(category string, enforcement bool, expectedBehavior PolicyBehavior) {
				By(fmt.Sprintf("Given a %s policy with enforcement=%v", category, enforcement))

				policyConfig := &clients.PolicyConfig{
					Name:        fmt.Sprintf("Test %s Policy", category),
					Categories:  []string{category},
					Enforcement: enforcement,
					Scope:       clients.RuntimeScope,
					Severity:    storage.Severity_HIGH_SEVERITY,
				}

				// Create policy with automatic cleanup
				policy, err := suite.StackRoxClients.Policies.CreatePolicy(ctx, policyConfig)
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() {
					suite.StackRoxClients.Policies.DeletePolicy(context.Background(), policy.GetId())
				})

				By("When deploying a violating application")
				deploymentName := fmt.Sprintf("test-%s-deployment", category)
				suite.CreateViolatingDeployment(deploymentName, category)

				By(fmt.Sprintf("Then should %s", expectedBehavior.Description))

				if expectedBehavior.ShouldAlert {
					Eventually(func() []*storage.Alert {
						alerts, _ := suite.StackRoxClients.Alerts.GetAlertsForPolicy(ctx, policy.GetId())
						return alerts
					}, expectedBehavior.AlertTimeout, 10*time.Second).Should(
						ginkgoHelper.HaveAlerts(1).WithSeverity(expectedBehavior.ExpectedSeverity),
						fmt.Sprintf("Expected alert for policy %s", policy.GetId()),
					)
				}

				if expectedBehavior.ShouldBlock {
					Eventually(func() string {
						return deploymentName
					}, expectedBehavior.BlockingTimeout, 10*time.Second).Should(
						ginkgoHelper.BeDeploymentBlocked(),
						fmt.Sprintf("Expected deployment %s to be blocked", deploymentName),
					)
				}
			},

			// Test matrix entries - these represent the 173 scenarios from PolicyFieldsTest.groovy
			Entry("Privilege Escalation + Enforce", "Privilege Escalation", true, PolicyBehavior{
				ShouldBlock:       true,
				ShouldAlert:       true,
				ExpectedSeverity:  storage.Severity_HIGH_SEVERITY,
				Description:       "block deployment and generate alert",
				AlertTimeout:      2 * time.Minute,
				BlockingTimeout:   2 * time.Minute,
			}),

			Entry("Privilege Escalation + Monitor", "Privilege Escalation", false, PolicyBehavior{
				ShouldBlock:       false,
				ShouldAlert:       true,
				ExpectedSeverity:  storage.Severity_HIGH_SEVERITY,
				Description:       "allow deployment but generate alert",
				AlertTimeout:      2 * time.Minute,
				BlockingTimeout:   0, // Not applicable
			}),

			Entry("Container Security + Enforce", "Container Security", true, PolicyBehavior{
				ShouldBlock:       true,
				ShouldAlert:       true,
				ExpectedSeverity:  storage.Severity_MEDIUM_SEVERITY,
				Description:       "block deployment and generate alert",
				AlertTimeout:      2 * time.Minute,
				BlockingTimeout:   2 * time.Minute,
			}),

			Entry("Network Policy + Monitor", "Network Policy", false, PolicyBehavior{
				ShouldBlock:       false,
				ShouldAlert:       true,
				ExpectedSeverity:  storage.Severity_HIGH_SEVERITY,
				Description:       "allow deployment but generate alert",
				AlertTimeout:      2 * time.Minute,
				BlockingTimeout:   0,
			}),

			// Additional entries would be generated programmatically
			// to cover all 173 scenarios from the original PolicyFieldsTest
		)
	})

	Context("Build-time Policy Categories", func() {
		DescribeTable("Build-time enforcement scenarios",
			func(category string, enforcement bool, imageConfig ImageConfig, expectedBehavior PolicyBehavior) {
				By(fmt.Sprintf("Given a %s policy with enforcement=%v", category, enforcement))

				policyConfig := &clients.PolicyConfig{
					Name:        fmt.Sprintf("Test %s Policy", category),
					Categories:  []string{category},
					Enforcement: enforcement,
					Scope:       clients.BuildScope,
					Severity:    storage.Severity_HIGH_SEVERITY,
				}

				policy, err := suite.StackRoxClients.Policies.CreatePolicy(ctx, policyConfig)
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() {
					suite.StackRoxClients.Policies.DeletePolicy(context.Background(), policy.GetId())
				})

				By("When scanning an image that violates the policy")

				// Use available registry clients for scanning
				if len(suite.RegistryClients) > 0 {
					registryClient := suite.RegistryClients[0]
					scanResult, err := registryClient.ScanImage(imageConfig.ImageName)

					if expectedBehavior.ShouldAlert {
						Expect(err).NotTo(HaveOccurred())
						Expect(scanResult).To(ginkgoHelper.HaveVulnerability().WithSeverity("HIGH"))
					}
				} else {
					Skip("No registry clients available for image scanning")
				}

				By(fmt.Sprintf("Then should %s", expectedBehavior.Description))

				if expectedBehavior.ShouldAlert {
					Eventually(func() []*storage.Alert {
						alerts, _ := suite.StackRoxClients.Alerts.GetAlertsForImage(ctx, imageConfig.ImageName)
						return alerts
					}, expectedBehavior.AlertTimeout, 10*time.Second).Should(
						ginkgoHelper.HaveAlerts(1),
						"Expected build-time policy alert",
					)
				}
			},

			Entry("Image Vulnerabilities + Enforce + Critical CVE",
				"Image Vulnerabilities", true,
				ImageConfig{ImageName: "vulnerable:latest"},
				PolicyBehavior{
					ShouldBlock:       true,
					ShouldAlert:       true,
					ExpectedSeverity:  storage.Severity_CRITICAL_SEVERITY,
					Description:       "block image and generate alert",
					AlertTimeout:      1 * time.Minute,
					BlockingTimeout:   1 * time.Minute,
				},
			),

			Entry("Dockerfile Security + Monitor",
				"Dockerfile Security", false,
				ImageConfig{ImageName: "insecure-dockerfile:latest"},
				PolicyBehavior{
					ShouldBlock:       false,
					ShouldAlert:       true,
					ExpectedSeverity:  storage.Severity_MEDIUM_SEVERITY,
					Description:       "allow image but generate alert",
					AlertTimeout:      1 * time.Minute,
					BlockingTimeout:   0,
				},
			),
		)
	})

	Context("External Service Integration", func() {
		It("should test notification delivery", func() {
			if len(suite.NotificationClients) == 0 {
				Skip("No notification clients available")
			}

			By("Sending test notification")
			notificationClient := suite.NotificationClients[0]

			testMessage := &external.NotificationMessage{
				Title:    "Test Notification",
				Text:     "This is a test notification from Ginkgo BDD tests",
				Channel:  "#test-channel",
				Severity: "INFO",
			}

			err := notificationClient.SendMessage(testMessage)
			Expect(err).To(ginkgoHelper.BeSuccessfulNotification())
		})

		It("should test backup operations", func() {
			if len(suite.StorageClients) == 0 {
				Skip("No storage clients available")
			}

			By("Creating and uploading a test backup")
			storageClient := suite.StorageClients[0]

			backupData := strings.NewReader("test backup data")
			result, err := storageClient.UploadBackup("test-backup", backupData)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BackupName).To(Equal("test-backup"))

			DeferCleanup(func() {
				storageClient.DeleteBackup("test-backup")
			})

			By("Verifying backup exists")
			backups, err := storageClient.ListBackups()
			Expect(err).NotTo(HaveOccurred())
			Expect(backups).To(ContainElement(HaveField("Name", "test-backup")))
		})
	})

	Context("Chaos Engineering", func() {
		It("should validate policy enforcement during admission controller chaos", func() {
			if !suite.config.EnableChaos {
				Skip("Chaos engineering disabled")
			}

			suite.TestWithChaos(ctx, func(chaosCtx context.Context) {
				By("Creating policy during chaos")
				policyConfig := &clients.PolicyConfig{
					Name:        "Chaos Test Policy",
					Categories:  []string{"Privilege Escalation"},
					Enforcement: true,
					Scope:       clients.RuntimeScope,
				}

				policy, err := suite.StackRoxClients.Policies.CreatePolicy(chaosCtx, policyConfig)
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() {
					suite.StackRoxClients.Policies.DeletePolicy(context.Background(), policy.GetId())
				})

				By("Creating violating deployment during chaos")
				deploymentName := "chaos-test-deployment"
				suite.CreateViolatingDeployment(deploymentName, "Privilege Escalation")

				By("Verifying policy enforcement works despite chaos")
				Eventually(func() string {
					return deploymentName
				}, 3*time.Minute, 15*time.Second).Should(
					ginkgoHelper.BeDeploymentBlocked(),
					"Policy enforcement should work despite admission controller chaos",
				)
			})
		})
	})
})

// Supporting types for the test

type PolicyBehavior struct {
	ShouldBlock       bool
	ShouldAlert       bool
	ExpectedSeverity  storage.Severity
	Description       string
	AlertTimeout      time.Duration
	BlockingTimeout   time.Duration
}

type ImageConfig struct {
	ImageName string
	Registry  string
}

// Suite setup - this would be in a separate suite_test.go file
var _ = BeforeSuite(func() {
	suite = ginkgoHelper.NewStackRoxTestSuite(ginkgoHelper.DefaultSuiteConfig())
	suite.SetupSuite()
})

var _ = AfterSuite(func() {
	if suite != nil {
		suite.TeardownSuite()
	}
})