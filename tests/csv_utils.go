package tests

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"

	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func verifyRiskEventTimelineCSV(t testutils.T, deploymentID string, eventNamesExpected []string) {
	// Export a CSV of the deployment's event timeline, and verify its content
	const baseURL = "/api/risk/timeline/export/csv"
	params := url.Values{}
	params.Add("query", fmt.Sprintf("Deployment ID:\"%s\"", deploymentID))
	escapedURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Get an HTTP client and query for csv content response
	client := centralgrpc.HTTPClientForCentral(t)
	resp, err := client.Get(escapedURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer utils.IgnoreError(resp.Body.Close)

	// Read the CSV content
	cr := csv.NewReader(resp.Body)
	rows, err := cr.ReadAll()
	require.NoError(t, err)

	// We expect a certain ordering of the columns. Check the first row since it is the header
	// The very first item has the byte order mark as well.
	assert.Equal(
		t,
		[]string{
			"\ufeffEvent Timestamp",
			"Event Type",
			"Event Name",
			"Process Args",
			"Process Parent Name",
			"Process Baselined",
			"Process UID",
			"Process Parent UID",
			"Container Exit Code",
			"Container Exit Reason",
			"Container ID",
			"Container Name",
			"Container Start Time",
			"Deployment ID",
			"Pod ID",
			"Pod Name",
			"Pod Start Time",
			"Pod Container Count",
		},
		rows[0])

	// Remove headers
	rows = rows[1:]

	// Check event names match
	// Index 0 of a row is the timestamp and 2 is the event name
	eventNamesInCSV := sliceutils.Map(rows, func(row []string) string { return row[2] })
	assert.ElementsMatch(t, eventNamesExpected, eventNamesInCSV)

	// All the records should be ordered by their event timestamp in a reverse order (latest first)
	for i := 0; i < len(rows)-1; i++ {
		assert.GreaterOrEqual(t, rows[i][0], rows[i+1][0])
	}
}
