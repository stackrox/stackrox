package booleanpolicy

// SPIKE: Container Age criterion tests.
//
// These tests will not compile until make proto-generated-srcs is run to add
// OldestContainerStarted to the generated storage.Container struct.
//
// To run once proto is regenerated:
//   go test ./pkg/booleanpolicy/... -run TestContainerAgeCriteria
//
// The test intentionally uses the same structure as image_criteria_test.go so the
// pattern is familiar to reviewers.

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestContainerAgeCriteria(t *testing.T) {
	suite.Run(t, new(ContainerAgeCriteriaTestSuite))
}

type ContainerAgeCriteriaTestSuite struct {
	basePoliciesTestSuite
}

// helper: deployment with a single container whose oldest_container_started is set
// to `daysAgo` days before now. This simulates what the enrichment step would set.
func depWithContainerAge(daysAgo int) *storage.Deployment {
	started := time.Now().AddDate(0, 0, -daysAgo)
	protoTs, err := protocompat.ConvertTimeToTimestampOrError(started)
	if err != nil {
		panic(err)
	}
	return &storage.Deployment{
		Id:        "test-dep",
		Namespace: "default",
		Containers: []*storage.Container{
			{
				Name: "app",
				// SPIKE: OldestContainerStarted is set transiently by the enrichment step
				// (sensor-side or central-side, see sensor/common/detector/container_age_enrichment.go).
				// In production it is never persisted; in tests we set it directly.
				OldestContainerStarted: protoTs,
			},
		},
	}
}

func (suite *ContainerAgeCriteriaTestSuite) TestContainerAge_ViolatesWhenOlderThanThreshold() {
	dep := depWithContainerAge(35)
	suite.addDepAndImages(dep)

	policy := policyWithSingleKeyValue(fieldnames.ContainerAge, "30", false)
	matcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)

	violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), violations.AlertViolations, "expected violation for 35-day-old container against 30-day threshold")
}

func (suite *ContainerAgeCriteriaTestSuite) TestContainerAge_NoViolationWhenYoungerThanThreshold() {
	dep := depWithContainerAge(10)
	suite.addDepAndImages(dep)

	policy := policyWithSingleKeyValue(fieldnames.ContainerAge, "30", false)
	matcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)

	violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), violations.AlertViolations, "expected no violation for 10-day-old container against 30-day threshold")
}

func (suite *ContainerAgeCriteriaTestSuite) TestContainerAge_NoViolationWhenFieldNotPopulated() {
	// Simulates a deployment where the enrichment step has not run or the container
	// has no live pod instances (e.g., replica count = 0). oldest_container_started
	// is nil, so the criterion must not fire.
	dep := &storage.Deployment{
		Id:        "empty-dep",
		Namespace: "default",
		Containers: []*storage.Container{
			{Name: "app"},
		},
	}
	suite.addDepAndImages(dep)

	policy := policyWithSingleKeyValue(fieldnames.ContainerAge, "1", false)
	matcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)

	violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), violations.AlertViolations, "criterion must not fire when oldest_container_started is nil")
}

func (suite *ContainerAgeCriteriaTestSuite) TestContainerAge_ViolationMessageIncludesContainerName() {
	dep := depWithContainerAge(35)
	suite.addDepAndImages(dep)

	policy := policyWithSingleKeyValue(fieldnames.ContainerAge, "30", false)
	matcher, err := BuildDeploymentMatcher(policy)
	require.NoError(suite.T(), err)

	violations, err := matcher.MatchDeployment(nil, enhancedDeployment(dep, suite.getImagesForDeployment(dep)))
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), violations.AlertViolations)
	assert.Contains(suite.T(), violations.AlertViolations[0].GetMessage(), "app",
		"violation message should reference the container name")
}
