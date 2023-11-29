package imagescan

import (
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	Pod = helper.K8sResourceInfo{Kind: "Pod", YamlFile: "pod.yaml"}

	Policies = []*storage.Policy{
		{
			Id:         uuid.NewV4().String(),
			Name:       "Red Hat Package Manager in Image",
			Disabled:   false,
			Categories: []string{"Security Best Practices"},
			LifecycleStages: []storage.LifecycleStage{
				storage.LifecycleStage_DEPLOY, storage.LifecycleStage_BUILD,
			},
			EventSource:   0,
			PolicyVersion: "1.1",
			PolicySections: []*storage.PolicySection{
				{
					SectionName: "",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       "Image Component",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "rpm|microdnf|dnf|yum=",
								},
							},
						},
					},
				},
			},
			CriteriaLocked:     true,
			MitreVectorsLocked: true,
			IsDefault:          true,
		},
	}
)

type ImageScanSuite struct {
	testContext *helper.TestContext
	suite.Suite
}

func Test_ImageScan(t *testing.T) {
	suite.Run(t, new(ImageScanSuite))
}

func (s *ImageScanSuite) SetupSuite() {
	customConfig := helper.DefaultCentralConfig()
	customConfig.InitialSystemPolicies = Policies
	testContext, err := helper.NewContextWithConfig(s.T(), customConfig)
	s.Require().NoError(err)
	s.testContext = testContext
}

func (s *ImageScanSuite) TearDownTest() {
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

func (s *ImageScanSuite) Test_AlertsUpdatedOnImageUpdate() {
	s.testContext.RunTest(s.T(),
		helper.WithResources([]helper.K8sResourceInfo{Pod}),
		helper.WithTestCase(func(t *testing.T, tc *helper.TestContext, resource map[string]k8s.Object) {
			var image *storage.ContainerImage
			// Image should be received by central
			fmt.Println("lvm: waiting for pod")
			tc.LastDeploymentStateWithTimeout(t, "myapp", func(dp *storage.Deployment, _ central.ResourceAction) error {
				if len(dp.GetContainers()) != 1 {
					return errors.Errorf("expected 1 container found %d", len(dp.GetContainers()))
				}

				if dp.GetContainers()[0].GetImage().GetId() == "" {
					return errors.New("image ID should not be empty")
				}

				image = dp.GetContainers()[0].GetImage()
				return nil
			}, "myapp should have started the container and have an imageID", 2*time.Minute)

			// There should be no violation yet, because there are no components provided for this image
			tc.NoViolations(t, "myapp", "violation found for deployment")
			tc.GetFakeCentral().StubMessage(&central.MsgToSensor{
				Msg: &central.MsgToSensor_UpdatedImage{
					UpdatedImage: &storage.Image{
						Id:    image.GetId(),
						Name:  image.GetName(),
						Names: []*storage.ImageName{image.GetName()},
						Scan: &storage.ImageScan{
							ScannerVersion: "2.0",
							Components: []*storage.EmbeddedImageScanComponent{
								{
									Name:    "rpm",
									Version: "3.2.1",
								},
							},
						},
					},
				},
			})

			tc.GetFakeCentral().StubMessage(&central.MsgToSensor{
				Msg: &central.MsgToSensor_ReprocessDeployments{
					ReprocessDeployments: &central.ReprocessDeployments{},
				},
			})

			// Violation should eventually happen for myapp, since the image scanned has rpm installed
			tc.LastViolationStateWithTimeout(t, "myapp", func(result *central.AlertResults) error {
				if !checkIfAlertsHaveViolation(result, Policies[0].GetName()) {
					return errors.New("violation not found for deployment")
				}
				return nil
			}, "Should have violation", 2*time.Minute)

		}))
}

func checkIfAlertsHaveViolation(result *central.AlertResults, name string) bool {
	if result == nil {
		return false
	}

	alerts := result.GetAlerts()
	if len(alerts) == 0 {
		return false
	}
	for _, alert := range alerts {
		if alert.GetPolicy().GetName() == name {
			return true
		}
	}
	return false
}
