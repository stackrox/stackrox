package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/spf13/afero"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryToMigrationDir(t *testing.T) {
	for name, tc := range map[string]struct {
		entry     string
		want      migrationDir
		wantError bool
	}{
		"valid migration": {
			entry: "m_199_to_m_200_my_migration",
			want:  migrationDir{dir: "m_199_to_m_200_my_migration", seqNum: 199},
		},
		"trailing separator": {
			entry: "m_209_to_m_210_",
			want:  migrationDir{dir: "m_209_to_m_210_", seqNum: 209},
		},
		"non migration file": {
			entry: "registry.go",
		},
		"helper directory": {
			entry: "cvehelper",
		},
		"not anchored at the start": {
			entry: "prefix_m_209_to_m_210_my_migration",
		},
		"missing trailing separator": {
			entry: "m_209_to_m_210",
		},
		"non numeric sequence": {
			entry: "m_abc_to_m_210_my_migration",
		},
		"overflowing sequence": {
			entry:     "m_999999999999999999999999_to_m_210_my_migration",
			wantError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := directoryToMigrationDir(tc.entry)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestImportToMigrationDir(t *testing.T) {
	for name, tc := range map[string]struct {
		line      string
		want      migrationDir
		wantError bool
	}{
		"valid import": {
			line: `_ "github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_my_migration"`,
			want: migrationDir{dir: `m_209_to_m_210_my_migration"`, seqNum: 209},
		},
		"non migration line": {
			line: "import (",
		},
		"invalid sequence": {
			line: "m_abc_to_m_210_my_migration",
		},
		"overflowing sequence": {
			line:      "m_999999999999999999999999_to_m_210_my_migration",
			wantError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := importToMigrationDir(tc.line)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestToMigrationDir(t *testing.T) {
	for name, tc := range map[string]struct {
		match     []string
		want      migrationDir
		wantError bool
	}{
		"valid match": {
			match: []string{"m_199_to_m_200_my_migration", "199"},
			want:  migrationDir{dir: "m_199_to_m_200_my_migration", seqNum: 199},
		},
		"nil match": {
			wantError: true,
		},
		"missing capture": {
			match:     []string{"m_199_to_m_200_my_migration"},
			wantError: true,
		},
		"extra captures": {
			match:     []string{"m_199_to_m_200_my_migration", "199", "extra"},
			wantError: true,
		},
		"non numeric capture": {
			match:     []string{"m_199_to_m_200_my_migration", "abc"},
			wantError: true,
		},
		"overflowing capture": {
			match:     []string{"m_999999999999999999999999_to_m_210_my_migration", "999999999999999999999999"},
			wantError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := toMigrationDir(tc.match)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFilterUnsupportedMigrations(t *testing.T) {
	migs := []migrationDir{
		{dir: "m_209_to_m_210_a", seqNum: 209},
		{dir: "m_213_to_m_214_b", seqNum: 213},
		{dir: "m_219_to_m_220_c", seqNum: 219},
		{dir: "m_220_to_m_221_d", seqNum: 220},
		{dir: "m_226_to_m_227_e", seqNum: 226},
	}

	for name, tc := range map[string]struct {
		oldestSupportedSeqNum uint64
		want                  []migrationDir
	}{
		"filters below cutoff": {
			oldestSupportedSeqNum: 220,
			want:                  []migrationDir{migs[0], migs[1], migs[2]},
		},
		"cutoff itself is supported": {
			oldestSupportedSeqNum: 209,
			want:                  []migrationDir{},
		},
		"all unsupported": {
			oldestSupportedSeqNum: 227,
			want:                  migs,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, filterUnsupportedMigrations(migs, tc.oldestSupportedSeqNum))
		})
	}
}

func TestIsUnsupportedMigration(t *testing.T) {
	unsupported := []migrationDir{
		{dir: "m_209_to_m_210_a", seqNum: 209},
		{dir: "m_213_to_m_214_b", seqNum: 213},
	}

	for name, tc := range map[string]struct {
		line string
		want bool
	}{
		"unsupported import": {
			line: `_ "github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_a"`,
			want: true,
		},
		"supported import": {
			line: `_ "github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_d"`,
		},
		"migration pattern not in unsupported list": {
			line: `_ "github.com/stackrox/rox/migrator/migrations/m_210_to_m_211_c"`,
		},
		"non migration line": {
			line: "import (",
		},
		"unparseable migration line": {
			line: `_ "github.com/stackrox/rox/migrator/migrations/m_999999999999999999999999_to_m_210_a"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isUnsupportedImport(unsupported, tc.line))
		})
	}
}

func TestOldestSupportedSeqNum(t *testing.T) {
	productstreams.OverrideBumpsForTesting(t, `bumps:
  - from: "3.74"
    to: "4.0"
  - from: "4.11"
    to: "5.0"
`)

	for name, tc := range map[string]struct {
		current productstreams.XYVersion
		want    uint64
		wantErr bool
	}{
		"current stream":      {current: productstreams.XYVersion{X: 5, Y: 1}, want: 220},
		"major boundary":      {current: productstreams.XYVersion{X: 5, Y: 0}, want: 213},
		"phantom version":     {current: productstreams.XYVersion{X: 4, Y: 14}, want: 213},
		"N-1":                 {current: productstreams.XYVersion{X: 4, Y: 11}, want: 211},
		"N-2":                 {current: productstreams.XYVersion{X: 4, Y: 10}, want: 209},
		"no release metadata": {current: productstreams.XYVersion{X: 3, Y: 74}, wantErr: true},
		"unknown major":       {current: productstreams.XYVersion{X: 99, Y: 0}, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := oldestSupportedSeqNum(tc.current)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFilterUnsupportedImports(t *testing.T) {
	unsupported := []migrationDir{
		{dir: "m_209_to_m_210_a", seqNum: 209},
		{dir: "m_213_to_m_214_b", seqNum: 213},
	}

	for name, tc := range map[string]struct {
		content string
		want    []string
	}{
		"filters unsupported imports": {
			content: strings.Join([]string{
				"// Code generated by make bootstrap_migration generator. DO NOT EDIT.",
				"package runner",
				"",
				"import (",
				`	_ "github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_a"`,
				`	_ "github.com/stackrox/rox/migrator/migrations/m_210_to_m_211_c"`,
				`	_ "github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_b"`,
				")",
			}, "\n"),
			want: []string{
				"// Code generated by make bootstrap_migration generator. DO NOT EDIT.",
				"package runner",
				"",
				"import (",
				`	_ "github.com/stackrox/rox/migrator/migrations/m_210_to_m_211_c"`,
				")",
			},
		},
		"keeps non migration lines": {
			content: "package runner\n\nimport ()\n",
			want:    []string{"package runner", "", "import ()"},
		},
		"empty input": {
			content: "",
			want:    nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			lines, err := filterUnsupportedImports(unsupported, strings.NewReader(tc.content))
			require.NoError(t, err)
			assert.Equal(t, tc.want, lines)
		})
	}
}

func TestFilterUnsupportedImportsReadError(t *testing.T) {
	lines, err := filterUnsupportedImports(nil, iotest.ErrReader(errors.New("read error")))
	require.Error(t, err)
	assert.Nil(t, lines)
}

func TestRenderCleanRunnerAll(t *testing.T) {
	for name, lines := range map[string][]string{
		"empty":       nil,
		"single line": {"package runner"},
		"multiple lines": {
			"// Code generated by make bootstrap_migration generator. DO NOT EDIT.",
			"package runner",
			"",
			"import (",
			`	_ "github.com/stackrox/rox/migrator/migrations/m_210_to_m_211_c"`,
			")",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, renderCleanRunnerAll(lines, &buf))

			var want strings.Builder
			for _, line := range lines {
				want.WriteString(line)
				want.WriteString("\n")
			}
			assert.Equal(t, want.String(), buf.String())
		})
	}
}

func TestRenderCleanRunnerAllWriteError(t *testing.T) {
	err := renderCleanRunnerAll([]string{"package runner"}, errWriter{})
	require.Error(t, err)
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestListMigrations(t *testing.T) {
	memFS := withMemFS(t)
	for _, dir := range []string{
		"m_209_to_m_210_a",
		"m_213_to_m_214_b",
		"cvehelper",
		"m_0_to_m_1_zero",
	} {
		require.NoError(t, memFS.MkdirAll(filepath.Join(migrationsDirectory, dir), 0o755))
	}
	require.NoError(t, afero.WriteFile(memFS, filepath.Join(migrationsDirectory, "registry.go"), []byte("package migrations"), 0o644))

	got, err := listMigrations()
	require.NoError(t, err)
	assert.Equal(t, []migrationDir{
		{dir: "m_209_to_m_210_a", seqNum: 209},
		{dir: "m_213_to_m_214_b", seqNum: 213},
	}, got)
}

func TestListMigrationsMissingDirectory(t *testing.T) {
	withMemFS(t)

	_, err := listMigrations()
	require.Error(t, err)
}

func TestListMigrationsInvalidSequence(t *testing.T) {
	memFS := withMemFS(t)
	require.NoError(t, memFS.MkdirAll(filepath.Join(migrationsDirectory, "m_999999999999999999999999_to_m_210_a"), 0o755))

	_, err := listMigrations()
	require.Error(t, err)
}

func TestCleanupMigrationDirectories(t *testing.T) {
	memFS := withMemFS(t)
	migs := []migrationDir{
		{dir: "m_209_to_m_210_a", seqNum: 209},
		{dir: "m_213_to_m_214_b", seqNum: 213},
	}
	for _, mig := range migs {
		require.NoError(t, memFS.MkdirAll(filepath.Join(migrationsDirectory, mig.dir), 0o755))
	}
	kept := filepath.Join(migrationsDirectory, "m_220_to_m_221_c")
	require.NoError(t, memFS.MkdirAll(kept, 0o755))

	require.NoError(t, cleanupMigrationDirectories(migs))

	for _, mig := range migs {
		exists, err := afero.DirExists(memFS, filepath.Join(migrationsDirectory, mig.dir))
		require.NoError(t, err)
		assert.False(t, exists, mig.dir)
	}
	exists, err := afero.DirExists(memFS, kept)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCleanupMigratorRunnerAll(t *testing.T) {
	memFS := withMemFS(t)
	unsupported := []migrationDir{
		{dir: "m_209_to_m_210_a", seqNum: 209},
		{dir: "m_213_to_m_214_b", seqNum: 213},
	}
	runnerPath := writeRunnerAll(t, memFS, runnerAllContent(
		[]string{"m_209_to_m_210_a", "m_213_to_m_214_b"},
		[]string{"m_210_to_m_211_c"},
	))

	require.NoError(t, cleanupMigratorRunnerAll(unsupported))

	got, err := afero.ReadFile(memFS, runnerPath)
	require.NoError(t, err)
	assert.Equal(t, runnerAllContent(nil, []string{"m_210_to_m_211_c"}), string(got))
}

func TestCleanupMigratorRunnerAllMissingFile(t *testing.T) {
	withMemFS(t)

	err := cleanupMigratorRunnerAll(nil)
	require.Error(t, err)
}

func TestRunWithOldestSeqNum(t *testing.T) {
	memFS := withMemFS(t)
	unsupported := []string{
		"m_209_to_m_210_a",
		"m_213_to_m_214_b",
		"m_219_to_m_220_c",
	}
	supported := []string{
		"m_220_to_m_221_d",
		"m_226_to_m_227_e",
	}
	for _, dir := range unsupported {
		require.NoError(t, memFS.MkdirAll(filepath.Join(migrationsDirectory, dir), 0o755))
	}
	for _, dir := range supported {
		require.NoError(t, memFS.MkdirAll(filepath.Join(migrationsDirectory, dir), 0o755))
	}
	runnerPath := writeRunnerAll(t, memFS, runnerAllContent(unsupported, supported))

	require.NoError(t, runWithOldestSeqNum(220))

	for _, dir := range unsupported {
		exists, err := afero.DirExists(memFS, filepath.Join(migrationsDirectory, dir))
		require.NoError(t, err)
		assert.False(t, exists, dir)
	}
	for _, dir := range supported {
		exists, err := afero.DirExists(memFS, filepath.Join(migrationsDirectory, dir))
		require.NoError(t, err)
		assert.True(t, exists, dir)
	}
	got, err := afero.ReadFile(memFS, runnerPath)
	require.NoError(t, err)
	assert.Equal(t, runnerAllContent(nil, supported), string(got))
}

func TestRunWithOldestSeqNumMissingRunnerFile(t *testing.T) {
	memFS := withMemFS(t)
	unsupported := filepath.Join(migrationsDirectory, "m_209_to_m_210_a")
	require.NoError(t, memFS.MkdirAll(unsupported, 0o755))

	err := runWithOldestSeqNum(220)
	require.Error(t, err)

	exists, err := afero.DirExists(memFS, unsupported)
	require.NoError(t, err)
	assert.True(t, exists, "directories must not be removed when runner cleanup fails")
}

func TestRunWithOldestSeqNumMissingMigrationsDirectory(t *testing.T) {
	withMemFS(t)

	require.Error(t, runWithOldestSeqNum(220))
}

func TestCleanupMigrationDirectoriesRemoveError(t *testing.T) {
	memFS := withMemFS(t)
	migs := []migrationDir{{dir: "m_209_to_m_210_a", seqNum: 209}}
	require.NoError(t, memFS.MkdirAll(filepath.Join(migrationsDirectory, migs[0].dir), 0o755))
	fs = removeAllErrorFS{memFS}

	require.Error(t, cleanupMigrationDirectories(migs))
}

func TestCleanupMigratorRunnerAllWriteOpenError(t *testing.T) {
	memFS := withMemFS(t)
	writeRunnerAll(t, memFS, runnerAllContent(nil, []string{"m_210_to_m_211_c"}))
	fs = openFileErrorFS{memFS}

	require.Error(t, cleanupMigratorRunnerAll(nil))
}

func TestCleanupMigratorRunnerAllReadError(t *testing.T) {
	memFS := withMemFS(t)
	writeRunnerAll(t, memFS, runnerAllContent(nil, []string{"m_210_to_m_211_c"}))
	fs = readErrorFS{memFS}

	require.Error(t, cleanupMigratorRunnerAll(nil))
}

type readErrorFS struct {
	afero.Fs
}

func (f readErrorFS) Open(name string) (afero.File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	return readErrorFile{File: file}, nil
}

type readErrorFile struct {
	afero.File
}

func (readErrorFile) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

type removeAllErrorFS struct {
	afero.Fs
}

func (removeAllErrorFS) RemoveAll(string) error {
	return errors.New("remove error")
}

type openFileErrorFS struct {
	afero.Fs
}

func (openFileErrorFS) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, errors.New("open file error")
}

func withMemFS(t *testing.T) afero.Fs {
	t.Helper()
	memFS := afero.NewMemMapFs()
	oldFS, oldRepoRoot, oldMigrationsDirectory := fs, repoRoot, migrationsDirectory
	fs = memFS
	repoRoot = "/repo"
	migrationsDirectory = filepath.Join(repoRoot, "migrator", "migrations")
	t.Cleanup(func() {
		fs, repoRoot, migrationsDirectory = oldFS, oldRepoRoot, oldMigrationsDirectory
	})
	return memFS
}

func writeRunnerAll(t *testing.T, memFS afero.Fs, content string) string {
	t.Helper()
	path := filepath.Join(repoRoot, "migrator", "runner", "all.go")
	require.NoError(t, memFS.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, afero.WriteFile(memFS, path, []byte(content), 0o644))
	return path
}

func runnerAllContent(unsupported, supported []string) string {
	lines := []string{
		"// Code generated by make bootstrap_migration generator. DO NOT EDIT.",
		"package runner",
		"",
		"import (",
	}
	for _, group := range [][]string{unsupported, supported} {
		for _, dir := range group {
			lines = append(lines, fmt.Sprintf("\t_ %q", "github.com/stackrox/rox/migrator/migrations/"+dir))
		}
	}
	lines = append(lines, ")")
	return strings.Join(lines, "\n") + "\n"
}
