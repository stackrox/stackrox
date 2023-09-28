package openvex

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/openvex/go-vex/pkg/vex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	cimg, err := imgUtils.GenerateImageFromString("docker.io/daha97/openvex:1.23.4")
	require.NoError(t, err)
	img := types.ToImage(cimg)

	now := time.Now()
	report := &vex.VEX{
		Metadata: vex.Metadata{
			Timestamp: &now,
		},
		Statements: []vex.Statement{
			{
				ID: "asdasd",
				Vulnerability: vex.Vulnerability{
					Name: "CVE-2021-36087",
				},
				Status:        "not_affected",
				Justification: "vulnerable_code_not_in_execute_path",
			},
		},
	}
	rawReport, err := json.Marshal(report)
	require.NoError(t, err)

	img.OpenVexReport = []*storage.OpenVex{
		{
			OpenVexReport: rawReport,
		},
	}
	img.Scan = &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Name:    "libsepol",
				Version: "3.1-1",
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve: "CVE-2021-36087",
					},
					{
						Cve: "CVE-2021-123",
					},
					{
						Cve: "CVE-2021-5343",
					},
					{
						Cve: "CVE-2021-65634",
					},
				},
			},
		},
	}

	filtered, err := Filter(img)
	assert.NoError(t, err)
	assert.True(t, filtered)
	assert.Len(t, img.GetScan().GetComponents()[0].GetVulns(), 3)
}
