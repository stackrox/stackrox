package benchmark

import (
	"embed"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	benchmarkDir = "fixtures"
)

var (
	//go:embed fixtures/*.json
	benchmarksFS embed.FS
)

func loadTestFixtures() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
	files, err := benchmarksFS.ReadDir(benchmarkDir)
	if err != nil {
		return nil, err
	}

	var benchmarks []*storage.ComplianceOperatorBenchmarkV2
	errList := errorhelpers.NewErrorList("Load Compliance Operator Benchmarks")
	for _, f := range files {
		b, err := readBenchmarksFile(filepath.Join(benchmarkDir, f.Name()))
		if err != nil {
			errList.AddError(err)
			continue
		}
		benchmarks = append(benchmarks, b)
	}
	return benchmarks, errList.ToError()
}

func readBenchmarksFile(path string) (*storage.ComplianceOperatorBenchmarkV2, error) {
	contents, err := benchmarksFS.ReadFile(path)
	utils.CrashOnError(err)

	var benchmark storage.ComplianceOperatorBenchmarkV2
	err = jsonutil.JSONBytesToProto(contents, &benchmark)
	if err != nil {
		log.Errorf("Unable to unmarshal benchmark (%s) json: %v", path, err)
		return nil, err
	}
	return &benchmark, nil
}

func TestGetBenchmarkShortNameFromProfileName_BackwardCompatibility(t *testing.T) {
	benchmarks, err := loadTestFixtures()
	require.NoError(t, err, "Failed to load compliance operator benchmarks")
	require.NotEmpty(t, benchmarks, "Expected to load at least one benchmark from fixtures")

	for _, benchmark := range benchmarks {
		t.Run(benchmark.GetName(), func(t *testing.T) {
			require.NotEmpty(t, benchmark.GetProfiles(), "Benchmark %s should have at least one profile", benchmark.GetName())

			expectedShortName := benchmark.GetShortName()
			for _, profile := range benchmark.GetProfiles() {
				profileName := profile.GetProfileName()

				actualShortName := GetBenchmarkShortNameFromProfileName(profileName)

				assert.Equal(t, expectedShortName, actualShortName,
					"Profile %q should return shortName %q, but got %q",
					profileName, expectedShortName, actualShortName)
			}
		})
	}
}

func TestGetBenchmarkFromProfile_BackwardCompatibility(t *testing.T) {
	benchmarks, err := loadTestFixtures()
	require.NoError(t, err, "Failed to load compliance operator benchmarks")
	require.NotEmpty(t, benchmarks, "Expected to load at least one benchmark from fixtures")

	for _, benchmark := range benchmarks {
		t.Run(benchmark.GetName(), func(t *testing.T) {
			require.NotEmpty(t, benchmark.GetProfiles(), "Benchmark %s should have at least one profile", benchmark.GetName())

			expectedShortName := benchmark.GetShortName()
			expectedProvider := benchmark.GetProvider()
			for _, profile := range benchmark.GetProfiles() {
				actualBenchmark, err := GetBenchmarkFromProfile(&storage.ComplianceOperatorProfileV2{
					Name: profile.GetProfileName(),
				})

				require.NoError(t, err)
				assert.Equal(t, expectedShortName, actualBenchmark.GetShortName())
				assert.Equal(t, expectedProvider, actualBenchmark.GetProvider())
			}
		})
	}
}

func TestGetBenchmarkShortNameFromProfileName_UnknownProfile(t *testing.T) {
	assert.Equal(t, "", GetBenchmarkShortNameFromProfileName("unknown-profile-v1"))
	assert.Equal(t, "", GetBenchmarkShortNameFromProfileName(""))
}
