package v2tostorage

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestScanComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    *v2.ScanComponent
		expected *storage.EmbeddedImageScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "component with top cvss",
			input: &v2.ScanComponent{
				Name:         "vulnerable-lib",
				Version:      "1.2.3",
				SetTopCvss:   &v2.ScanComponent_TopCvss{TopCvss: 9.8},
				RiskScore:    8.5,
				Architecture: "amd64",
			},
			expected: &storage.EmbeddedImageScanComponent{
				Name:         "vulnerable-lib",
				Version:      "1.2.3",
				RiskScore:    8.5,
				Architecture: "amd64",
				SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: 9.8,
				},
			},
		},
		{
			name: "component without top cvss",
			input: &v2.ScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
			},
			expected: &storage.EmbeddedImageScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanComponent(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    []*v2.ScanComponent
		expected []*storage.EmbeddedImageScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []*v2.ScanComponent{},
			expected: nil,
		},
		{
			name: "multiple components",
			input: []*v2.ScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
				{
					Name:    "component2",
					Version: "2.0.0",
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
					Source:  storage.SourceType_OS,
				},
				{
					Name:    "component2",
					Version: "2.0.0",
					Source:  storage.SourceType_PYTHON,
				},
			},
		},
		{
			name: "with nil component",
			input: []*v2.ScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
				nil,
				{
					Name:    "component2",
					Version: "2.0.0",
				},
			},
			expected: []*storage.EmbeddedImageScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
				},
				{
					Name:    "component2",
					Version: "2.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScanComponents(tt.input)
			protoassert.SlicesEqual(t, tt.expected, result)
		})
	}
}
