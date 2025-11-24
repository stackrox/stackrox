package testdata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGenerateIndexReportStructure(t *testing.T) {
	opts := Options{
		VsockCID:        999,
		NumPackages:     10,
		NumRepositories: 3,
	}
	report, err := GenerateIndexReport(opts)
	require.NoError(t, err)

	require.Equal(t, "999", report.GetVsockCid())
	pkgs := report.GetIndexV4().GetContents().GetPackages()
	repos := report.GetIndexV4().GetContents().GetRepositories()

	require.Len(t, pkgs, opts.NumPackages)
	require.Len(t, repos, opts.NumRepositories)

	require.Equal(t, "pkg-0", pkgs["pkg-0"].GetId())
	require.Equal(t, "repo-0", repos["repo-0"].GetId())
	require.Equal(t, "binary", pkgs["pkg-0"].GetKind())
	require.Equal(t, "amd64", pkgs["pkg-0"].GetArch())
}

func TestRandomizationChangesHashAndVersions(t *testing.T) {
	opts := Options{
		VsockCID:    42,
		NumPackages: 20,
		Randomize:   true,
		Seed:        12345,
	}

	reportOne, err := GenerateIndexReport(opts)
	require.NoError(t, err)

	opts.Seed = 54321
	reportTwo, err := GenerateIndexReport(opts)
	require.NoError(t, err)

	require.NotEqual(t, reportOne.GetIndexV4().GetHashId(), reportTwo.GetIndexV4().GetHashId())
	require.NotEqual(t, reportOne.GetIndexV4().GetContents().GetPackages()["pkg-0"].GetVersion(),
		reportTwo.GetIndexV4().GetContents().GetPackages()["pkg-0"].GetVersion())
}

func TestSerializeAndLoadFixture(t *testing.T) {
	opts := Options{
		VsockCID:        77,
		NumPackages:     15,
		NumRepositories: 5,
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "fixture.pb")
	require.NoError(t, WriteFixture(path, opts))

	report, err := LoadFixture(path)
	require.NoError(t, err)
	require.Equal(t, "77", report.GetVsockCid())
}

func TestPayloadSizesWithinBounds(t *testing.T) {
	cases := []struct {
		name     string
		opts     Options
		minBytes int
		maxBytes int
	}{
		{
			name:     "small",
			opts:     Options{NumPackages: 500, NumRepositories: 50},
			minBytes: 2_200_000,
			maxBytes: 3_000_000,
		},
		{
			name:     "average",
			opts:     Options{NumPackages: 700, NumRepositories: 70},
			minBytes: 3_200_000,
			maxBytes: 4_200_000,
		},
		{
			name:     "large",
			opts:     Options{NumPackages: 1500, NumRepositories: 150},
			minBytes: 6_500_000,
			maxBytes: 8_400_000,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			report, err := GenerateIndexReport(c.opts)
			require.NoError(t, err)
			data, err := proto.Marshal(report)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(data), c.minBytes)
			require.LessOrEqual(t, len(data), c.maxBytes)
		})
	}
}

func TestGoGenerateProducesFixtures(t *testing.T) {
	tmpDir := t.TempDir()
	cmd := testGenerateCommand(t, tmpDir)
	require.NoError(t, cmd)

	for _, name := range []string{"indexreport_small.pb", "indexreport_avg.pb", "indexreport_large.pb"} {
		path := filepath.Join(tmpDir, name)
		_, err := os.Stat(path)
		require.NoError(t, err, "expected fixture %s to exist", name)
	}
}

func testGenerateCommand(t *testing.T, outDir string) error {
	t.Helper()
	return runGenerate(outDir)
}

func runGenerate(outDir string) error {
	// mimic cmd/generate via library call to avoid invoking subprocess in tests.
	specs := map[string]Options{
		"indexreport_small.pb": {VsockCID: 101, NumPackages: 500, NumRepositories: 50},
		"indexreport_avg.pb":   {VsockCID: 202, NumPackages: 700, NumRepositories: 70},
		"indexreport_large.pb": {VsockCID: 303, NumPackages: 1500, NumRepositories: 150},
	}
	for name, opts := range specs {
		if err := WriteFixture(filepath.Join(outDir, name), opts); err != nil {
			return err
		}
	}
	return nil
}

func TestEmbeddedFixtures(t *testing.T) {
	data, err := EmbeddedFixture("small")
	require.NoError(t, err)

	report, err := LoadReportFromBytes(data)
	require.NoError(t, err)
	require.NotNil(t, report.GetIndexV4())

	_, err = EmbeddedFixture("unknown")
	require.Error(t, err)
}
