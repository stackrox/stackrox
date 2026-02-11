package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformVulnerabilityToResponse(t *testing.T) {
	t.Run("with valid CVE", func(t *testing.T) {
		vuln := &storage.EmbeddedVulnerability{
			Cve:                   "CVE-2021-1234",
			FirstSystemOccurrence: protocompat.TimestampNow(),
			FirstImageOccurrence:  protocompat.TimestampNow(),
		}

		result := transformVulnerabilityToResponse(vuln)

		assert.NotNil(t, result)
		assert.Equal(t, "CVE-2021-1234", result.GetId())
		assert.NotNil(t, result.GetFirstSystemOccurrence())
		assert.NotNil(t, result.GetFirstImageOccurrence())
		assert.Nil(t, result.GetSuppression())
	})

	t.Run("with suppression", func(t *testing.T) {
		vuln := &storage.EmbeddedVulnerability{
			Cve:                "CVE-2021-1234",
			Suppressed:         true,
			SuppressActivation: protocompat.TimestampNow(),
			SuppressExpiry:     protocompat.TimestampNow(),
		}

		result := transformVulnerabilityToResponse(vuln)

		assert.NotNil(t, result)
		assert.Equal(t, "CVE-2021-1234", result.GetId())
		assert.NotNil(t, result.GetSuppression())
		assert.NotNil(t, result.GetSuppression().GetSuppressActivation())
		assert.NotNil(t, result.GetSuppression().GetSuppressExpiry())
	})

	t.Run("without CVE returns nil", func(t *testing.T) {
		vuln := &storage.EmbeddedVulnerability{
			Cve:                   "",
			FirstSystemOccurrence: protocompat.TimestampNow(),
		}

		result := transformVulnerabilityToResponse(vuln)

		assert.Nil(t, result)
	})
}

func TestTransformComponentToResponse(t *testing.T) {
	t.Run("with vulnerabilities", func(t *testing.T) {
		comp := &storage.EmbeddedImageScanComponent{
			Name:     "openssl",
			Version:  "1.0.0",
			Location: "/usr/lib",
			HasLayerIndex: &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: 0,
			},
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                   "CVE-2021-1234",
					FirstSystemOccurrence: protocompat.TimestampNow(),
					FirstImageOccurrence:  protocompat.TimestampNow(),
				},
				{
					Cve:                   "CVE-2021-5678",
					FirstSystemOccurrence: protocompat.TimestampNow(),
				},
			},
		}

		result := transformComponentToResponse(comp)

		assert.NotNil(t, result)
		assert.Equal(t, "openssl", result.GetName())
		assert.Equal(t, "1.0.0", result.GetVersion())
		assert.Equal(t, "/usr/lib", result.GetLocation())
		assert.Zero(t, result.GetLayerIndex())
		require.Len(t, result.GetVulnerabilities(), 2)
		assert.Equal(t, "CVE-2021-1234", result.GetVulnerabilities()[0].GetId())
		assert.Equal(t, "CVE-2021-5678", result.GetVulnerabilities()[1].GetId())
	})

	t.Run("no vulnerabilities returns nil", func(t *testing.T) {
		comp := &storage.EmbeddedImageScanComponent{
			Name:    "openssl",
			Version: "1.0.0",
			Vulns:   []*storage.EmbeddedVulnerability{},
		}

		result := transformComponentToResponse(comp)

		assert.Nil(t, result)
	})

	t.Run("only vulnerabilities without CVE returns nil", func(t *testing.T) {
		comp := &storage.EmbeddedImageScanComponent{
			Name:    "openssl",
			Version: "1.0.0",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:                   "",
					FirstSystemOccurrence: protocompat.TimestampNow(),
				},
			},
		}

		result := transformComponentToResponse(comp)

		assert.Nil(t, result)
	})

	t.Run("mixed valid and invalid CVEs", func(t *testing.T) {
		comp := &storage.EmbeddedImageScanComponent{
			Name:    "openssl",
			Version: "1.0.0",
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve: "",
				},
				{
					Cve: "CVE-2021-1234",
				},
				{
					Cve: "",
				},
			},
		}

		result := transformComponentToResponse(comp)

		assert.NotNil(t, result)
		require.Len(t, result.GetVulnerabilities(), 1)
		assert.Equal(t, "CVE-2021-1234", result.GetVulnerabilities()[0].GetId())
	})
}

func TestTransformImageToResponse_Integration(t *testing.T) {
	t.Run("filters components without vulnerabilities", func(t *testing.T) {
		components := []*storage.EmbeddedImageScanComponent{
			{
				Name:    "no-vulns",
				Version: "1.0.0",
				Vulns:   []*storage.EmbeddedVulnerability{},
			},
			{
				Name:    "with-vulns",
				Version: "2.0.0",
				Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2021-1234"},
				},
			},
		}

		result := []*v1.ImageVulnerabilitiesResponse_Image_Component{}

		for _, comp := range components {
			if responseComp := transformComponentToResponse(comp); responseComp != nil {
				result = append(result, responseComp)
			}
		}

		require.Len(t, result, 1)
		assert.Equal(t, "with-vulns", result[0].GetName())
	})
}
