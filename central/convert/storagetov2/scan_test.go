package storagetov2

import (
	"testing"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestDataSource(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.DataSource
		expected *v2.DataSource
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete datasource",
			input: &storage.DataSource{
				Id:     "ds-123",
				Name:   "production-scanner",
				Mirror: "scanner.prod.example.com",
			},
			expected: &v2.DataSource{
				Id:     "ds-123",
				Name:   "production-scanner",
				Mirror: "scanner.prod.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DataSource(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.EmbeddedImageScanComponent
		expected *v2.ScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "component with top cvss",
			input: &storage.EmbeddedImageScanComponent{
				Name:    "vulnerable-lib",
				Version: "1.2.3",
				License: &storage.License{
					Name: "MIT",
					Type: "permissive",
					Url:  "https://opensource.org/licenses/MIT",
				},
				Source:       storage.SourceType_PYTHON,
				Location:     "/usr/lib/python/vulnerable-lib",
				RiskScore:    8.5,
				FixedBy:      "1.2.4",
				Architecture: "amd64",
				SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{
					TopCvss: 9.8,
				},
				Executables: []*storage.EmbeddedImageScanComponent_Executable{
					{
						Path:         "/usr/bin/vuln-exec",
						Dependencies: []string{"lib1", "lib2"},
					},
				},
			},
			expected: &v2.ScanComponent{
				Name:    "vulnerable-lib",
				Version: "1.2.3",
				License: &v2.License{
					Name: "MIT",
					Type: "permissive",
					Url:  "https://opensource.org/licenses/MIT",
				},
				Source:       v2.SourceType_PYTHON,
				Location:     "/usr/lib/python/vulnerable-lib",
				RiskScore:    8.5,
				FixedBy:      "1.2.4",
				Architecture: "amd64",
				SetTopCvss: &v2.ScanComponent_TopCvss{
					TopCvss: 9.8,
				},
				Executables: []*v2.ScanComponent_Executable{
					{
						Path:         "/usr/bin/vuln-exec",
						Dependencies: []string{"lib1", "lib2"},
					},
				},
			},
		},
		{
			name: "component without top cvss",
			input: &storage.EmbeddedImageScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
				Source:  storage.SourceType_GO,
			},
			expected: &v2.ScanComponent{
				Name:    "safe-lib",
				Version: "2.0.0",
				Source:  v2.SourceType_GO,
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
func TestConvertSourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    storage.SourceType
		expected v2.SourceType
	}{
		{
			name:     "OS source type",
			input:    storage.SourceType_OS,
			expected: v2.SourceType_OS,
		},
		{
			name:     "Python source type",
			input:    storage.SourceType_PYTHON,
			expected: v2.SourceType_PYTHON,
		},
		{
			name:     "Java source type",
			input:    storage.SourceType_JAVA,
			expected: v2.SourceType_JAVA,
		},
		{
			name:     "Go source type",
			input:    storage.SourceType_GO,
			expected: v2.SourceType_GO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSourceType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanComponents(t *testing.T) {
	tests := []struct {
		name     string
		input    []*storage.EmbeddedImageScanComponent
		expected []*v2.ScanComponent
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []*storage.EmbeddedImageScanComponent{},
			expected: nil,
		},
		{
			name: "multiple components",
			input: []*storage.EmbeddedImageScanComponent{
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
			expected: []*v2.ScanComponent{
				{
					Name:    "component1",
					Version: "1.0.0",
					Source:  v2.SourceType_OS,
				},
				{
					Name:    "component2",
					Version: "2.0.0",
					Source:  v2.SourceType_PYTHON,
				},
			},
		},
		{
			name: "with nil component",
			input: []*storage.EmbeddedImageScanComponent{
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
			expected: []*v2.ScanComponent{
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

func TestExecutable(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.EmbeddedImageScanComponent_Executable
		expected *v2.ScanComponent_Executable
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete executable",
			input: &storage.EmbeddedImageScanComponent_Executable{
				Path:         "/usr/bin/test-exec",
				Dependencies: []string{"dependency1", "dependency2"},
			},
			expected: &v2.ScanComponent_Executable{
				Path:         "/usr/bin/test-exec",
				Dependencies: []string{"dependency1", "dependency2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Executable(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}

func TestLicense(t *testing.T) {
	tests := []struct {
		name     string
		input    *storage.License
		expected *v2.License
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "complete license",
			input: &storage.License{
				Name: "Apache-2.0",
				Type: "permissive",
				Url:  "https://opensource.org/licenses/Apache-2.0",
			},
			expected: &v2.License{
				Name: "Apache-2.0",
				Type: "permissive",
				Url:  "https://opensource.org/licenses/Apache-2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := License(tt.input)
			protoassert.Equal(t, tt.expected, result)
		})
	}
}
