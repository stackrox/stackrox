package main

import (
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexReportFixtureIsValid(t *testing.T) {
	var indexReport v4.IndexReport
	require.NoError(t, jsonutil.JSONBytesToProto(indexReportFixture, &indexReport))

	contents := indexReport.GetContents()
	require.NotNil(t, contents)
	require.NotEmpty(t, contents.GetPackages())
	require.NotEmpty(t, contents.GetRepositories())
	require.NotEmpty(t, contents.GetEnvironments())

	for _, envList := range contents.GetEnvironments() {
		for _, env := range envList.GetEnvironments() {
			for _, repoID := range env.GetRepositoryIds() {
				_, exists := contents.GetRepositories()[repoID]
				assert.Truef(t, exists, "repository %q referenced by environment is missing", repoID)
			}
		}
	}
}
