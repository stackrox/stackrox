package deploytime

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	excludedImage   = "registry.example.com/team/legacy-app:1.0"
	vulnerableImage = "registry.example.com/team/payments:2.3"
)

// oldCriticalCVEPolicy mirrors a "Block Critical CVE older than 180 days" policy that runs at
// both the build and deploy stages and excludes a single image.
func oldCriticalCVEPolicy(exclusions ...*storage.Exclusion) *storage.Policy {
	return &storage.Policy{
		Id:              "old-critical-cve",
		Name:            "Block Critical CVE older than 180 days",
		PolicyVersion:   policyversion.CurrentVersion().String(),
		Severity:        storage.Severity_CRITICAL_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY},
		Exclusions:      exclusions,
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{
				{FieldName: fieldnames.Severity, Values: []*storage.PolicyValue{{Value: "CRITICAL"}}},
				{FieldName: fieldnames.DaysSinceImageFirstDiscovered, Values: []*storage.PolicyValue{{Value: "180"}}},
			},
		}},
	}
}

func imageExclusion(fullName string) *storage.Exclusion {
	return &storage.Exclusion{Image: &storage.Exclusion_Image{Name: fullName}}
}

func imageWithOldCriticalCVE(t *testing.T, fullName string) *storage.Image {
	firstSeen, err := protocompat.ConvertTimeToTimestampOrError(time.Now().AddDate(0, 0, -365))
	require.NoError(t, err)
	return &storage.Image{
		Id:   fullName + "-sha",
		Name: &storage.ImageName{FullName: fullName},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{{
				Name:    "openssl",
				Version: "1.0.1",
				Vulns: []*storage.EmbeddedVulnerability{{
					Cve:                  "CVE-2014-0160",
					Severity:             storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
					FirstImageOccurrence: firstSeen,
				}},
			}},
		},
	}
}

func deploymentRunning(name string, image *storage.Image) booleanpolicy.EnhancedDeployment {
	return booleanpolicy.EnhancedDeployment{
		Deployment: &storage.Deployment{
			Id:        name + "-id",
			Name:      name,
			Namespace: "shop",
			ClusterId: "cluster-1",
			Containers: []*storage.Container{{
				Name:  name,
				Image: &storage.ContainerImage{Id: image.GetId(), Name: image.GetName()},
			}},
		},
		Images: []*storage.Image{image},
	}
}

func detectorFor(t *testing.T, policy *storage.Policy) Detector {
	policySet := detection.NewPolicySet()
	require.NoError(t, policySet.UpsertPolicy(policy))
	return NewDetector(policySet)
}

// TestImageExclusionDoesNotDisableDeployTimeDetection reproduces the customer report: adding an
// image exclusion to a build and deploy policy stopped the policy from alerting on every deployment,
// including deployments that do not run the excluded image.
func TestImageExclusionDoesNotDisableDeployTimeDetection(t *testing.T) {
	ctx := DetectionContext{}
	payments := deploymentRunning("payments", imageWithOldCriticalCVE(t, vulnerableImage))

	t.Run("without an exclusion, the vulnerable deployment alerts", func(t *testing.T) {
		alerts, err := detectorFor(t, oldCriticalCVEPolicy()).Detect(ctx, payments)
		require.NoError(t, err)
		assert.Len(t, alerts, 1)
	})

	t.Run("with an image exclusion, a deployment that does not run the excluded image still alerts", func(t *testing.T) {
		policy := oldCriticalCVEPolicy(imageExclusion(excludedImage))
		alerts, err := detectorFor(t, policy).Detect(ctx, payments)
		require.NoError(t, err)
		require.Len(t, alerts, 1)
		assert.Equal(t, "payments", alerts[0].GetDeployment().GetName())
	})

	t.Run("with an image exclusion, a deployment running the excluded image still alerts", func(t *testing.T) {
		// Image exclusions only apply at build time, so a deployment running the excluded image is
		// still evaluated at deploy time.
		legacy := deploymentRunning("legacy-app", imageWithOldCriticalCVE(t, excludedImage))
		policy := oldCriticalCVEPolicy(imageExclusion(excludedImage))
		alerts, err := detectorFor(t, policy).Detect(ctx, legacy)
		require.NoError(t, err)
		require.Len(t, alerts, 1)
		assert.Equal(t, "legacy-app", alerts[0].GetDeployment().GetName())
	})

	t.Run("a deployment exclusion still skips the named deployment", func(t *testing.T) {
		policy := oldCriticalCVEPolicy(&storage.Exclusion{Deployment: &storage.Exclusion_Deployment{Name: "payments"}})
		alerts, err := detectorFor(t, policy).Detect(ctx, payments)
		require.NoError(t, err)
		assert.Empty(t, alerts)
	})
}

// TestImageExclusionStillAppliesAtBuildTime checks that the fix does not change build-time
// behavior: the excluded image is skipped, other images are still checked.
func TestImageExclusionStillAppliesAtBuildTime(t *testing.T) {
	policy := oldCriticalCVEPolicy(imageExclusion(excludedImage))
	compiled, err := detection.CompilePolicy(policy)
	require.NoError(t, err)

	assert.False(t, compiled.AppliesTo(imageWithOldCriticalCVE(t, excludedImage)))
	assert.True(t, compiled.AppliesTo(imageWithOldCriticalCVE(t, vulnerableImage)))
}
