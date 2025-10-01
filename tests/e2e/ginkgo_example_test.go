//go:build test_e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/testutils/external"
	ginkgoHelper "github.com/stackrox/rox/pkg/testutils/ginkgo"
	"google.golang.org/grpc"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

// Example test demonstrating the complete Ginkgo BDD framework
var _ = Describe("Policy Field Validation Example", func() {
	var (
		conn      *grpc.ClientConn
		policySvc v1.PolicyServiceClient
		alertSvc  v1.AlertServiceClient
		ctx       context.Context
	)

	BeforeEach(func() {
		// Create grpc connection
		conn = centralgrpc.GRPCConnectionToCentral(GinkgoT())
		policySvc = v1.NewPolicyServiceClient(conn)
		alertSvc = v1.NewAlertServiceClient(conn)

		// Create test context with timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)

		DeferCleanup(func() {
			cancel()
			conn.Close()
		})
	})

	Context("Runtime Policy Categories", func() {
		DescribeTable("Runtime enforcement scenarios",
			func(category string, enforcement bool, expectedBehavior PolicyBehavior) {
				By(fmt.Sprintf("Given a %s policy with enforcement=%v", category, enforcement))

				policy, err := policySvc.PostPolicy(ctx, &v1.PostPolicyRequest{
					Policy: &storage.Policy{
						Name:       fmt.Sprintf("Test %s Policy", category),
						Categories: []string{category},
						Severity:   storage.Severity_HIGH_SEVERITY,
						// Additional policy configuration would go here
					},
				})
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() {
					_, _ = policySvc.DeletePolicy(context.Background(), &v1.ResourceByID{Id: policy.GetId()})
				})

				By("When deploying a violating application")
				deploymentName := fmt.Sprintf("test-%s-deployment", strings.ToLower(strings.ReplaceAll(category, " ", "-")))
				// TODO: Create violating deployment here

				By(fmt.Sprintf("Then should %s", expectedBehavior.Description))

				if expectedBehavior.ShouldAlert {
					Eventually(func() []*storage.ListAlert {
						resp, err := alertSvc.ListAlerts(ctx, &v1.ListAlertsRequest{
							Query: fmt.Sprintf("Policy Id:%s", policy.GetId()),
						})
						if err != nil {
							return nil
						}
						return resp.Alerts
					}, expectedBehavior.AlertTimeout, 10*time.Second).Should(
						HaveLen(1),
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
				ShouldBlock:      true,
				ShouldAlert:      true,
				ExpectedSeverity: storage.Severity_HIGH_SEVERITY,
				Description:      "block deployment and generate alert",
				AlertTimeout:     2 * time.Minute,
				BlockingTimeout:  2 * time.Minute,
			}),

			Entry("Privilege Escalation + Monitor", "Privilege Escalation", false, PolicyBehavior{
				ShouldBlock:      false,
				ShouldAlert:      true,
				ExpectedSeverity: storage.Severity_HIGH_SEVERITY,
				Description:      "allow deployment but generate alert",
				AlertTimeout:     2 * time.Minute,
				BlockingTimeout:  0, // Not applicable
			}),

			Entry("Container Security + Enforce", "Container Security", true, PolicyBehavior{
				ShouldBlock:      true,
				ShouldAlert:      true,
				ExpectedSeverity: storage.Severity_MEDIUM_SEVERITY,
				Description:      "block deployment and generate alert",
				AlertTimeout:     2 * time.Minute,
				BlockingTimeout:  2 * time.Minute,
			}),

			Entry("Network Policy + Monitor", "Network Policy", false, PolicyBehavior{
				ShouldBlock:      false,
				ShouldAlert:      true,
				ExpectedSeverity: storage.Severity_HIGH_SEVERITY,
				Description:      "allow deployment but generate alert",
				AlertTimeout:     2 * time.Minute,
				BlockingTimeout:  0,
			}),

			// Additional entries would be generated programmatically
			// to cover all 173 scenarios from the original PolicyFieldsTest
		)
	})

	Context("Build-time Policy Categories", func() {
		DescribeTable("Build-time enforcement scenarios",
			func(category string, enforcement bool, imageConfig ImageConfig, expectedBehavior PolicyBehavior) {
				By(fmt.Sprintf("Given a %s policy with enforcement=%v", category, enforcement))

				policy, err := policySvc.PostPolicy(ctx, &v1.PostPolicyRequest{
					Policy: &storage.Policy{
						Name:       fmt.Sprintf("Test %s Policy", category),
						Categories: []string{category},
						Severity:   storage.Severity_HIGH_SEVERITY,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() {
					_, _ = policySvc.DeletePolicy(context.Background(), &v1.ResourceByID{Id: policy.GetId()})
				})

				By("When scanning an image that violates the policy")

				// Use registry client for scanning if available
				registryClient, err := external.NewGCRClient()
				if err != nil {
					Skip("No registry client available for image scanning")
				}

				scanResult, err := registryClient.ScanImage(imageConfig.ImageName)

				if expectedBehavior.ShouldAlert {
					Expect(err).NotTo(HaveOccurred())
					Expect(scanResult.Vulnerabilities).NotTo(BeEmpty())
				}

				By(fmt.Sprintf("Then should %s", expectedBehavior.Description))

				if expectedBehavior.ShouldAlert {
					Eventually(func() []*storage.ListAlert {
						resp, err := alertSvc.ListAlerts(ctx, &v1.ListAlertsRequest{
							Query: fmt.Sprintf("Image:%s", imageConfig.ImageName),
						})
						if err != nil {
							return nil
						}
						return resp.Alerts
					}, expectedBehavior.AlertTimeout, 10*time.Second).Should(
						HaveLen(1),
						"Expected build-time policy alert",
					)
				}
			},

			Entry("Image Vulnerabilities + Enforce + Critical CVE",
				"Image Vulnerabilities", true,
				ImageConfig{ImageName: "vulnerable:latest"},
				PolicyBehavior{
					ShouldBlock:      true,
					ShouldAlert:      true,
					ExpectedSeverity: storage.Severity_CRITICAL_SEVERITY,
					Description:      "block image and generate alert",
					AlertTimeout:     1 * time.Minute,
					BlockingTimeout:  1 * time.Minute,
				},
			),

			Entry("Dockerfile Security + Monitor",
				"Dockerfile Security", false,
				ImageConfig{ImageName: "insecure-dockerfile:latest"},
				PolicyBehavior{
					ShouldBlock:      false,
					ShouldAlert:      true,
					ExpectedSeverity: storage.Severity_MEDIUM_SEVERITY,
					Description:      "allow image but generate alert",
					AlertTimeout:     1 * time.Minute,
					BlockingTimeout:  0,
				},
			),
		)
	})

	Context("External Service Integration", func() {
		It("should test notification delivery", func() {
			notificationClient, err := external.NewSlackClient()
			if err != nil {
				Skip("No Slack notification client available")
			}

			By("Sending test notification")
			testMessage := &external.NotificationMessage{
				Title:    "Test Notification",
				Text:     "This is a test notification from Ginkgo BDD tests",
				Channel:  "#test-channel",
				Severity: "INFO",
			}

			err = notificationClient.SendMessage(testMessage)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should test backup operations", func() {
			storageClient, err := external.NewS3Client()
			if err != nil {
				Skip("No S3 storage client available")
			}

			By("Creating and uploading a test backup")
			backupData := strings.NewReader("test backup data")
			result, err := storageClient.UploadBackup("test-backup", backupData)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BackupName).To(Equal("test-backup"))

			DeferCleanup(func() {
				_ = storageClient.DeleteBackup("test-backup")
			})

			By("Verifying backup exists")
			backups, err := storageClient.ListBackups()
			Expect(err).NotTo(HaveOccurred())
			Expect(backups).To(ContainElement(HaveField("Name", "test-backup")))
		})
	})
})

// Supporting types for the test

type PolicyBehavior struct {
	ShouldBlock      bool
	ShouldAlert      bool
	ExpectedSeverity storage.Severity
	Description      string
	AlertTimeout     time.Duration
	BlockingTimeout  time.Duration
}

type ImageConfig struct {
	ImageName string
	Registry  string
}
