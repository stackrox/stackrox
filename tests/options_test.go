package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/search"
	"github.com/stackrox/rox/central/search/options"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsMapExist(t *testing.T) {
	t.Parallel()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewSearchServiceClient(conn)

	for _, categories := range [][]v1.SearchCategory{
		{},
		{v1.SearchCategory_ALERTS},
		{v1.SearchCategory_DEPLOYMENTS},
		{v1.SearchCategory_IMAGES},
		{v1.SearchCategory_POLICIES},
		search.GetGlobalSearchCategories().AsSlice(),
	} {
		cat := categories
		t.Run(fmt.Sprintf("%v", categories), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			resp, err := service.Options(ctx, &v1.SearchOptionsRequest{Categories: cat})
			cancel()
			require.NoError(t, err)
			if len(cat) == 0 {
				cat = search.GetGlobalSearchCategories().AsSlice()
			}
			assert.ElementsMatch(t, options.GetOptions(cat), resp.GetOptions())
		})
	}
}

func TestOptionsMap(t *testing.T) {
	expectedOptions := []string{
		"Add Capabilities", "CPU Cores Limit", "CPU Cores Request", "CVE", "CVE Published On", "CVE Snoozed", "CVSS",
		"Cluster", "Component", "Component Version", "Deployment", "Deployment Annotation", "Deployment Label",
		"Deployment Type", "Dockerfile Instruction Keyword", "Dockerfile Instruction Value", "Drop Capabilities",
		"Environment Key", "Environment Value", "Environment Variable Source", "Exposed Node Port", "Exposing Service",
		"Exposing Service Port", "Exposure Level", "External Hostname", "External IP", "Image", "Image Command",
		"Image Created Time", "Image Entrypoint", "Image Label", "Image OS", "Image Pull Secret", "Image Registry",
		"Image Remote", "Image Scan Time", "Image Tag", "Image Top CVSS", "Image User", "Image Volumes",
		"Max Exposure Level", "Memory Limit (MB)", "Memory Request (MB)", "Namespace", "Namespace ID",
		"Orchestrator Component", "Pod Label", "Port", "Port Protocol", "Privileged", "Process Arguments",
		"Process Name", "Process Path", "Process UID", "Read Only Root Filesystem", "Secret", "Secret Path",
		"Service Account", "Service Account Permission Level", "Volume Destination", "Volume Name", "Volume ReadOnly",
		"Volume Source", "Volume Type", "Vulnerability State",
	}
	categories := []v1.SearchCategory{
		v1.SearchCategory_DEPLOYMENTS,
		v1.SearchCategory_COMPONENT_VULN_EDGE,
		v1.SearchCategory_IMAGE_COMPONENT_EDGE,
		v1.SearchCategory_IMAGE_COMPONENTS,
		v1.SearchCategory_IMAGE_VULN_EDGE,
		v1.SearchCategory_IMAGE_VULNERABILITIES,
		v1.SearchCategory_IMAGES,
		v1.SearchCategory_NODE_COMPONENT_EDGE,
		v1.SearchCategory_NODE_COMPONENTS,
		v1.SearchCategory_NODE_VULNERABILITIES,
		v1.SearchCategory_NODES,
	}
	for _, category := range categories {
		t.Run(category.String(), func(t *testing.T) {
			t.Parallel()
			options := options.GetOptions([]v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
			assert.Equal(t, expectedOptions, options)
		})
	}
}
