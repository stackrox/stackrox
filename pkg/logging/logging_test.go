package logging

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLevelForLabel(t *testing.T) {
	for _, label := range []string{"warn", "WARN", "WaRn"} {
		lvl, ok := LevelForLabel(label)
		assert.Equal(t, zapcore.WarnLevel, lvl)
		assert.True(t, ok)
	}
	for _, label := range []string{"foo", "bar", "Trace", "something", "else", "WTF", "@$%@$&Y)(RW(*U(@Y$"} {
		_, ok := LevelForLabel(label)
		assert.False(t, ok)
	}
}

func TestLabelForLevel(t *testing.T) {
	for level, expectedLabel := range validLevels {
		actualLabel, ok := LabelForLevel(level)
		assert.True(t, ok)
		assert.Equal(t, expectedLabel, actualLabel)
		assert.Equal(t, expectedLabel, LabelForLevelOrInvalid(level))
	}
	_, ok := LabelForLevel(-2)
	assert.False(t, ok)
	label := LabelForLevelOrInvalid(-2)
	assert.Equal(t, "Invalid", label)
}

func TestZapSortedLevels(t *testing.T) {
	assert.Equal(t, sortedLevels, SortedLevels())
}

func TestSetGlobalLogLevel(t *testing.T) {
	mInfo := ModuleForName(uuid.NewV4().String())
	assert.Equal(t, GetGlobalLogLevel(), mInfo.GetLogLevel())

	SetGlobalLogLevel(zapcore.DebugLevel)
	mDebug := ModuleForName(uuid.NewV4().String())
	assert.Equal(t, GetGlobalLogLevel(), mDebug.GetLogLevel())
	assert.Equal(t, GetGlobalLogLevel(), mInfo.GetLogLevel())
}

func Test_withRotatingCore(t *testing.T) {
	t.Setenv(env.LoggingMaxBackups.EnvVar(), "2")
	t.Setenv(env.LoggingMaxSizeMB.EnvVar(), "1")
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	logname1 := filepath.Join(dir1, "test.log")
	logname2 := filepath.Join(dir2, "test.log")
	cfg := config
	cfg.OutputPaths = []string{logname1, logname2}
	core := withRotatingCore(&cfg)

	logger, err := cfg.Build(zap.WrapCore(core))
	require.NoError(t, err)
	logger.Info("begin")
	for i := 0; i < 10000; i++ {
		logger.Info("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	}

	require.NoError(t, logger.Sync())
	fi, err := os.Stat(logname1)
	require.NoError(t, err)
	assert.Equal(t, int64(451500), fi.Size()) // may change if code changes.
	fi, err = os.Stat(logname2)
	require.NoError(t, err)
	assert.Equal(t, int64(451500), fi.Size())

	t.Run("test the backup cut line", func(t *testing.T) {
		// Ensure the last log starts with a timestamp and filename:
		// 2024/11/21 15:41:07.549096: logging_test.go:70 Info: ...
		f, err := os.Open(logname1)
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		buf := make([]byte, 43)
		n, err := f.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, len(buf), n)
		assert.True(t, regexp.MustCompile(
			`^\d{4}/\d\d/\d\d \d\d:\d\d:\d\d\.\d{6}: logging_test\.go`).Match(buf),
			string(buf))
	})
	t.Run("test backup file", func(t *testing.T) {
		found := 0
		err := filepath.WalkDir(dir1, func(path string, d fs.DirEntry, err error) error {
			if ok, _ := filepath.Match("test-*.log", d.Name()); ok {
				found++
				fi, err := d.Info()
				require.NoError(t, err)
				assert.Equal(t, int64(1048559), fi.Size())
				f, err := os.Open(path)
				require.NoError(t, err)
				defer func() { _ = f.Close() }()
				// Ensure the first log line of the backup ends with begin:
				// 2024/11/21 15:41:07.549096: logging_test.go:70 Info: begin
				buf := make([]byte, 58)
				n, err := f.Read(buf)
				require.NoError(t, err)
				assert.Equal(t, len(buf), n)
				assert.True(t, strings.HasSuffix(string(buf), "begin"))
			}
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, found)
	})
}
