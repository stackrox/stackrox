package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/retry"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nginxImage struct {
	version          string
	SHA              string
	activeComponents int
}

func (i *nginxImage) getImage() string {
	return fmt.Sprintf("docker.io/library/nginx:%s@%s", i.version, i.SHA)
}

var (
	avmDeploymentName = "nginx-avm"
	nginxImages       = []nginxImage{
		{
			version:          "1.14.0",
			SHA:              "sha256:8b600a4d029481cc5b459f1380b30ff6cb98e27544fc02370de836e397e34030",
			activeComponents: 5,
		},
		{
			version:          "1.18.0",
			SHA:              "sha256:e90ac5331fe095cea01b121a3627174b2e33e06e83720e9a934c7b8ccc9c55a0",
			activeComponents: 11,
		},
		{
			version:          "1.20.0",
			SHA:              "sha256:ea4560b87ff03479670d15df426f7d02e30cb6340dcd3004cdfc048d6a1d54b4",
			activeComponents: 11,
		},
	}
	once sync.Once
)

type ActiveContext struct {
	ContainerName string `json:"containerName"`
}

type ActiveState struct {
	State          string          `json:"state"`
	ActiveContexts []ActiveContext `json:"activeContexts"`
}

type ActiveComponent struct {
	IDStruct
	ActiveState ActiveState `json:"activeState"`
}

type ActiveVulnerability struct {
	IDStruct
	ActiveState ActiveState `json:"activeState"`
}

type ComponentsAndVulnsWithActiveState struct {
	IDStruct
	Components []ActiveComponent     `json:"components"`
	Vulns      []ActiveVulnerability `json:"vulns"`
}

func TestActiveVulnerability(t *testing.T) {
	waitForImageScanned(t)
	for idx, tc := range nginxImages {
		t.Run(tc.version, func(t *testing.T) {
			runTestActiveVulnerability(t, idx, tc)
		})
	}
}

func runTestActiveVulnerability(t *testing.T, idx int, testCase nginxImage) {
	log.Infof("test case %v", testCase)
	deploymentName := fmt.Sprintf("%s-%d", avmDeploymentName, idx)
	setupDeployment(t, testCase.getImage(), deploymentName)
	defer teardownDeployment(t, deploymentName)
	fmt.Println(idx, testCase, deploymentName)
	deploymentID := getDeploymentID(t, deploymentName)
	checkActiveVulnerability(t, testCase, deploymentID)
}

func TestActiveVulnerability_SetImage(t *testing.T) {
	waitForImageScanned(t)
	setupDeploymentWithReplicas(t, nginxImages[0].getImage(), avmDeploymentName, 3)
	defer teardownDeployment(t, avmDeploymentName)
	deploymentID := getDeploymentID(t, avmDeploymentName)

	checkActiveVulnerability(t, nginxImages[0], deploymentID)

	// Upgrade image and check result
	setImage(t, avmDeploymentName, deploymentID, "nginx", nginxImages[1].getImage())
	checkActiveVulnerability(t, nginxImages[1], deploymentID)

	// Downgrade image and check result
	setImage(t, avmDeploymentName, deploymentID, "nginx", nginxImages[0].getImage())
	checkActiveVulnerability(t, nginxImages[0], deploymentID)
}

func checkActiveVulnerability(t *testing.T, image nginxImage, deploymentID string) {
	waitForCondition(t, func() bool {
		deployment := getDeploymentActiveStates(t, deploymentID)
		return image.activeComponents <= getActiveComponentCount(deployment)
	}, "active components populated", 5*time.Minute, 30*time.Second)
	fromDeployment := getDeploymentActiveStates(t, deploymentID)
	assert.LessOrEqual(t, image.activeComponents, getActiveComponentCount(fromDeployment))
	// The active vulns are not stable over time. But at least one vuln should exist.
	assert.NotZero(t, getActiveVulnCount(t, fromDeployment))

	fromImage := getImageActiveStates(t, image.SHA, deploymentID)
	assert.LessOrEqual(t, image.activeComponents, getActiveComponentCount(fromImage))
	assert.Equal(t, getActiveVulnCount(t, fromDeployment), getActiveVulnCount(t, fromImage))
}

func getActiveComponentCount(entity ComponentsAndVulnsWithActiveState) int {
	var count int
	var activeComponents []string
	for _, component := range entity.Components {
		if component.ActiveState.State == "Active" {
			activeComponents = append(activeComponents, string(component.ID))
			count++
		}
	}
	log.Infof("Found %d active components(s) for %s: %v", count, entity.ID, activeComponents)
	return count
}

func getActiveVulnCount(t *testing.T, entity ComponentsAndVulnsWithActiveState) int {
	var count int
	var activeVulns []string
	for _, vuln := range entity.Vulns {
		if vuln.ActiveState.State == "Active" {
			activeVulns = append(activeVulns, string(vuln.ID))
			count++
		}
	}
	log.Infof("Found %d active vuln(s) for %s: %v", count, entity.ID, activeVulns)
	return count
}

func getDeploymentActiveStates(t *testing.T, deploymentID string) ComponentsAndVulnsWithActiveState {
	var resp struct {
		Deployment ComponentsAndVulnsWithActiveState `json:"deployment"`
	}
	makeGraphQLRequest(t, `
		query getDeploymentCVE($deploymentID: ID!) {
			deployment(id: $deploymentID) {
				id
				components {
					id
					activeState {
						state
						activeContexts {
							containerName
						}
					}
				}
				vulns {
					id
					activeState {
						state
						activeContexts {
							containerName
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"deploymentID": deploymentID,
	}, &resp, timeout)
	return resp.Deployment
}

func getImageActiveStates(t *testing.T, imageID, deploymentID string) ComponentsAndVulnsWithActiveState {
	var resp struct {
		Image ComponentsAndVulnsWithActiveState `json:"image"`
	}
	makeGraphQLRequest(t, `
		query getImageCVE($imageID: ID!, $scopeQuery: String) {
			image(id: $imageID) {
				id
				components {
					id
					activeState(query: $scopeQuery) {
						state
						activeContexts {
							containerName
						}
					}
				}
				vulns {
					id
					activeState(query: $scopeQuery) {
						state
						activeContexts {
							containerName
						}
					}
				}
			}
		}
	`, map[string]interface{}{
		"imageID":    imageID,
		"scopeQuery": fmt.Sprintf("DEPLOYMENT ID:%q", deploymentID),
	}, &resp, timeout)
	return resp.Image
}

func waitForImageScanned(t *testing.T) {
	once.Do(func() {
		conn := centralgrpc.GRPCConnectionToCentral(t)
		imageService := v1.NewImageServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		for _, image := range nginxImages {
			log.Infof("wait for image %s scanned ...", image.getImage())
			err := retry.WithRetry(func() error {
				_, err := imageService.ScanImage(ctx, &v1.ScanImageRequest{
					ImageName: image.getImage(),
				})
				return err
			}, retry.Tries(3), retry.OnFailedAttempts(func(_ error) {
				time.Sleep(5 * time.Second)
			}))
			require.NoError(t, err, "fail to prepare images for testing. This may be caused by network issue.")
		}
	})
}
