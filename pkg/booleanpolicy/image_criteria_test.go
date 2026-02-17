package booleanpolicy

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestImageCriteria(t *testing.T) {
	t.Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	suite.Run(t, new(ImageCriteriaTestSuite))
}

type ImageCriteriaTestSuite struct {
	basePoliciesTestSuite
}

func (suite *ImageCriteriaTestSuite) TestNVDCVSSCriteria() {
	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				Name:            "nginx",
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "HEARTBLEEDDEPSHA"},
			},
		},
	}

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, &storage.Image{
		Id:   "HEARTBLEEDDEPSHA",
		Name: &storage.ImageName{FullName: "heartbleed"},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, NvdCvss: 8, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.2"},
						FirstImageOccurrence: protoTs},
				}},
			},
		},
	})

	nvdCvssPolicyGroup := &storage.PolicyGroup{
		FieldName: fieldnames.NvdCvss,
		Values: []*storage.PolicyValue{
			{
				Value: "> 6",
			},
		},
	}

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, nvdCvssPolicyGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)
	require.Contains(suite.T(), violations.AlertViolations[0].GetMessage(), "NVD CVSS")

}

func (suite *ImageCriteriaTestSuite) TestFixableAndImageFirstOccurenceCriteria() {
	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				Name:            "nginx",
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "HEARTBLEEDDEPSHA"},
			},
		},
	}

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, &storage.Image{
		Id:   "HEARTBLEEDDEPSHA",
		Name: &storage.ImageName{FullName: "heartbleed"},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.2"},
						FirstImageOccurrence: protoTs},
				}},
			},
		},
	})

	fixablePolicyGroup := &storage.PolicyGroup{
		FieldName: fieldnames.Fixable,
		Values:    []*storage.PolicyValue{{Value: "true"}},
	}
	firstImageOccurrenceGroup := &storage.PolicyGroup{
		FieldName: fieldnames.DaysSinceImageFirstDiscovered,
		Values:    []*storage.PolicyValue{{Value: "2"}},
	}

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, fixablePolicyGroup, firstImageOccurrenceGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)

}

func (suite *ImageCriteriaTestSuite) TestFixableAndFixTimestampAvailableCriteria() {
	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				Name:            "nginx",
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "HEARTBLEEDDEPSHA"},
			},
		},
	}

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, &storage.Image{
		Id:   "HEARTBLEEDDEPSHA",
		Name: &storage.ImageName{FullName: "heartbleed"},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.2"},
						FixAvailableTimestamp: protoTs},
				}},
			},
		},
	})

	fixablePolicyGroup := &storage.PolicyGroup{
		FieldName: fieldnames.Fixable,
		Values:    []*storage.PolicyValue{{Value: "true"}},
	}
	fixTimestampAvailableGroup := &storage.PolicyGroup{
		FieldName: fieldnames.DaysSinceFixAvailable,
		Values:    []*storage.PolicyValue{{Value: "2"}},
	}

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, fixablePolicyGroup, fixTimestampAvailableGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)

}

func (suite *ImageCriteriaTestSuite) TestDaysSinceCVEPublishedCriteria() {
	heartbleedDep := &storage.Deployment{
		Id: "HEARTBLEEDDEPID",
		Containers: []*storage.Container{
			{
				Name:            "nginx",
				SecurityContext: &storage.SecurityContext{Privileged: true},
				Image:           &storage.ContainerImage{Id: "HEARTBLEEDDEPSHA"},
			},
		},
	}

	ts := time.Now().AddDate(0, 0, -5)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(ts)
	require.NoError(suite.T(), err)

	suite.addDepAndImages(heartbleedDep, &storage.Image{
		Id:   "HEARTBLEEDDEPSHA",
		Name: &storage.ImageName{FullName: "heartbleed"},
		Scan: &storage.ImageScan{
			Components: []*storage.EmbeddedImageScanComponent{
				{Name: "heartbleed", Version: "1.2", Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2014-0160", Link: "https://heartbleed", Cvss: 6, SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{FixedBy: "v1.2"},
						PublishedOn: protoTs},
				}},
			},
		},
	})

	fixablePolicyGroup := &storage.PolicyGroup{
		FieldName: fieldnames.Fixable,
		Values:    []*storage.PolicyValue{{Value: "true"}},
	}
	cvePublishedGroup := &storage.PolicyGroup{
		FieldName: fieldnames.DaysSincePublished,
		Values:    []*storage.PolicyValue{{Value: "2"}},
	}

	policy := policyWithGroups(storage.EventSource_NOT_APPLICABLE, fixablePolicyGroup, cvePublishedGroup)

	deployment := suite.deployments["HEARTBLEEDDEPID"]
	depMatcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)
	violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(deployment, suite.getImagesForDeployment(deployment)))
	require.Len(suite.T(), violations.AlertViolations, 1)
	require.NoError(suite.T(), err)

}

func (suite *ImageCriteriaTestSuite) TestImageOS() {
	depToImg := make(map[*storage.Deployment]*storage.Image)
	for _, imgName := range []string{
		"unknown",
		"alpine:v3.4",
		"alpine:v3.11",
		"ubuntu:20.04",
		"debian:8",
		"debian:10",
	} {
		img := imageWithOS(imgName)
		dep := fixtures.GetDeployment().CloneVT()
		dep.Containers = []*storage.Container{
			{
				Name:  imgName,
				Image: types.ToContainerImage(img),
			},
		}
		depToImg[dep] = img
	}

	for _, testCase := range []struct {
		value           string
		expectedMatches []string
	}{
		{
			value:           "unknown",
			expectedMatches: []string{"unknown"},
		},
		{
			value:           "alpine",
			expectedMatches: []string{},
		},
		{
			value:           "alpine.*",
			expectedMatches: []string{"alpine:v3.4", "alpine:v3.11"},
		},
		{
			value:           "debian:8",
			expectedMatches: []string{"debian:8"},
		},
		{
			value:           "centos",
			expectedMatches: nil,
		},
	} {
		c := testCase

		suite.T().Run(fmt.Sprintf("DeploymentMatcher %+v", c), func(t *testing.T) {
			depMatcher, err := BuildDeploymentMatcher(policyWithSingleKeyValue(fieldnames.ImageOS, c.value, false))
			require.NoError(t, err)
			depMatched := set.NewStringSet()
			for dep, img := range depToImg {
				violations, err := depMatcher.MatchDeployment(nil, enhancedDeployment(dep, []*storage.Image{img}))
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					depMatched.Add(img.GetScan().GetOperatingSystem())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Container '%s' has image with base OS '%s'", dep.GetContainers()[0].GetName(), img.GetScan().GetOperatingSystem()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, depMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", depMatched.AsSlice(), c.value, c.expectedMatches)
		})

		suite.T().Run(fmt.Sprintf("ImageMatcher %+v", c), func(t *testing.T) {
			imgMatcher, err := BuildImageMatcher(policyWithSingleKeyValue(fieldnames.ImageOS, c.value, false))
			require.NoError(t, err)
			imgMatched := set.NewStringSet()
			for _, img := range depToImg {
				violations, err := imgMatcher.MatchImage(nil, img)
				require.NoError(t, err)
				if len(violations.AlertViolations) > 0 {
					imgMatched.Add(img.GetScan().GetOperatingSystem())
					require.Len(t, violations.AlertViolations, 1)
					assert.Equal(t, fmt.Sprintf("Image has base OS '%s'", img.GetScan().GetOperatingSystem()), violations.AlertViolations[0].GetMessage())
				}
			}
			assert.ElementsMatch(t, imgMatched.AsSlice(), c.expectedMatches, "Got %v for policy %v; expected: %v", imgMatched.AsSlice(), c.value, c.expectedMatches)
		})
	}
}

func (suite *ImageCriteriaTestSuite) TestImageVerified() {
	const (
		verifier0  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000001"
		verifier1  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000002"
		verifier2  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000003"
		verifier3  = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000004"
		unverifier = "io.stackrox.signatureintegration.00000000-0000-0000-0000-00000000000F"
	)

	var images = []*storage.Image{
		suite.imageWithSignatureVerificationResults("image_no_results", []*storage.ImageSignatureVerificationResult{{}}),
		suite.imageWithSignatureVerificationResults("image_empty_results", []*storage.ImageSignatureVerificationResult{{
			VerifierId: "",
			Status:     storage.ImageSignatureVerificationResult_UNSET,
		}}),
		suite.imageWithSignatureVerificationResults("image_nil_results", nil),
		suite.imageWithSignatureVerificationResults("verified_by_0", []*storage.ImageSignatureVerificationResult{{
			VerifierId:              verifier0,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_0"},
		}}),
		suite.imageWithSignatureVerificationResults("unverified_image", []*storage.ImageSignatureVerificationResult{{
			VerifierId: unverifier,
			Status:     storage.ImageSignatureVerificationResult_UNSET,
		}}),
		suite.imageWithSignatureVerificationResults("verified_by_3", []*storage.ImageSignatureVerificationResult{{
			VerifierId: verifier2,
			Status:     storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
		}, {
			VerifierId:              verifier3,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_3"},
		}}),
		suite.imageWithSignatureVerificationResults("verified_by_2_and_3", []*storage.ImageSignatureVerificationResult{{
			VerifierId:              verifier2,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_2_and_3"},
		}, {
			VerifierId:              verifier3,
			Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
			VerifiedImageReferences: []string{"verified_by_2_and_3"},
		}}),
	}

	var allImages set.FrozenStringSet
	{
		ai := set.NewStringSet()
		for _, img := range images {
			ai.Add(img.GetName().GetFullName())
		}
		allImages = ai.Freeze()
	}

	getViolationMessage := func(img *storage.Image) string {
		message := strings.Builder{}
		message.WriteString("Image signature is not verified by the specified signature integration(s)")
		successfulVerifierIDs := []string{}
		for _, r := range img.GetSignatureVerificationData().GetResults() {
			if r.GetVerifierId() != "" && r.GetStatus() == storage.ImageSignatureVerificationResult_VERIFIED {
				successfulVerifierIDs = append(successfulVerifierIDs, r.GetVerifierId())
			}
		}
		if len(successfulVerifierIDs) > 0 {
			message.WriteString(fmt.Sprintf(" (it is verified by other integration(s): %s)", printer.StringSliceToSortedSentence(successfulVerifierIDs)))
		}
		message.WriteString(".")
		return message.String()
	}

	suite.Run("Test disallowed AND operator", func() {
		_, err := BuildImageMatcher(policyWithSingleFieldAndValues(fieldnames.ImageSignatureVerifiedBy,
			[]string{verifier0}, false, storage.BooleanOperator_AND))
		suite.EqualError(err,
			"policy validation error: operator AND is not allowed for field \"Image Signature Verified By\"")
	})

	for i, testCase := range []struct {
		values          []string
		expectedMatches set.FrozenStringSet
	}{
		{
			values:          []string{unverifier},
			expectedMatches: allImages,
		},
		{
			values:          []string{verifier0},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_0")),
		},
		{
			values:          []string{verifier1},
			expectedMatches: allImages,
		},
		{
			values:          []string{verifier2},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_2_and_3")),
		},
		{
			values:          []string{verifier3},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_3", "verified_by_2_and_3")),
		},
		{
			values:          []string{verifier0, verifier2},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_0", "verified_by_2_and_3")),
		},
		{
			values:          []string{verifier2, verifier3},
			expectedMatches: allImages.Difference(set.NewFrozenStringSet("verified_by_3", "verified_by_2_and_3")),
		},
	} {
		c := testCase

		suite.Run(fmt.Sprintf("ImageMatcher %d: %+v", i, c), func() {
			imgMatcher, err := BuildImageMatcher(policyWithSingleFieldAndValues(fieldnames.ImageSignatureVerifiedBy,
				c.values, false, storage.BooleanOperator_OR))
			suite.NoError(err)
			matchedImages := set.NewStringSet()
			for _, img := range images {
				violations, err := imgMatcher.MatchImage(nil, img)
				suite.NoError(err)
				if len(violations.AlertViolations) == 0 {
					continue
				}
				matchedImages.Add(img.GetName().GetFullName())
				suite.Truef(c.expectedMatches.Contains(img.GetName().GetFullName()), "Image %q should not match",
					img.GetName().GetFullName())

				for _, violation := range violations.AlertViolations {
					suite.Equal(getViolationMessage(img), violation.GetMessage())
				}
			}
			suite.True(c.expectedMatches.Difference(matchedImages.Freeze()).IsEmpty(), matchedImages)
		})
	}
}

func (suite *ImageCriteriaTestSuite) TestImageVerified_WithDeployment() {
	const (
		verifier1 = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000002"
		verifier2 = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000003"
		verifier3 = "io.stackrox.signatureintegration.00000000-0000-0000-0000-000000000004"
	)

	imgVerifiedAndMatchingReference := suite.imageWithSignatureVerificationResults("image_verified_by_1",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId:              verifier1,
				Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
				VerifiedImageReferences: []string{"image_verified_by_1"},
			},
		})

	imgVerifiedAndMatchingMultipleReferences := suite.imageWithSignatureVerificationResults("image_verified_by_2",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId:              verifier3,
				Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
				VerifiedImageReferences: []string{"image_with_alternative_verified_reference", "image_verified_by_2"},
			},
		})

	imgVerifiedButNotMatchingReference := suite.imageWithSignatureVerificationResults("image_with_alternative_verified_reference",
		[]*storage.ImageSignatureVerificationResult{
			{
				VerifierId:              verifier2,
				Status:                  storage.ImageSignatureVerificationResult_VERIFIED,
				VerifiedImageReferences: []string{"image_verified_by_2"},
			},
		})

	cases := map[string]struct {
		deployment       *storage.Deployment
		image            *storage.Image
		matchingVerifier string
		expectViolation  bool
	}{
		"deployment with matching verified image reference shouldn't lead in alert message": {
			deployment:       deploymentWithImage("deployment_with_image_verified_by_1", imgVerifiedAndMatchingReference),
			image:            imgVerifiedAndMatchingReference,
			matchingVerifier: verifier1,
		},
		"deployment with verified result but no matching verified image reference should lead to alert message": {
			deployment:       deploymentWithImage("deployment_with_image_alternative_verified_reference", imgVerifiedButNotMatchingReference),
			image:            imgVerifiedButNotMatchingReference,
			matchingVerifier: verifier2,
			expectViolation:  true,
		},
		"deployment with verified result and multiple matching verified image references shouldn't lead to alert message": {
			deployment:       deploymentWithImage("deployment_with_image_verified_by_2", imgVerifiedAndMatchingMultipleReferences),
			image:            imgVerifiedAndMatchingMultipleReferences,
			matchingVerifier: verifier3,
		},
	}

	for name, c := range cases {
		suite.Run(name, func() {
			deploymentMatcher, err := BuildDeploymentMatcher(policyWithSingleFieldAndValues(fieldnames.ImageSignatureVerifiedBy,
				[]string{c.matchingVerifier}, false, storage.BooleanOperator_OR))
			suite.Require().NoError(err)

			violations, err := deploymentMatcher.MatchDeployment(nil, EnhancedDeployment{
				Deployment: c.deployment,
				Images:     []*storage.Image{c.image},
			})
			suite.Require().NoError(err)

			if c.expectViolation {
				suite.NotEmpty(violations.AlertViolations)
			} else {
				suite.Empty(violations.AlertViolations)
			}
		})
	}
}
