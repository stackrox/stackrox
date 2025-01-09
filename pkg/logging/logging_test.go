package logging

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

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
	// Keep current log + 1 rotation file.
	t.Setenv(env.LoggingMaxRotationFiles.EnvVar(), "1")
	t.Setenv(env.LoggingMaxSizeMB.EnvVar(), "1")
	maxSizeInBytes := int64(env.LoggingMaxSizeMB.IntegerSetting() * 1024 * 1024)
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	logname1 := filepath.Join(dir1, "test.log")
	logname2 := filepath.Join(dir2, "test.log")
	// Clone the global config:
	cfg := config
	// ... but do not log to the standard streams:
	cfg.OutputPaths = []string{}
	core := withRotatingCores(&cfg, []string{logname1, logname2})

	logger, err := cfg.Build(zap.WrapCore(core))
	require.NoError(t, err)
	logger.Info("begin")
	for i := 0; i < 10000; i++ {
		logger.Info("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	}

	require.NoError(t, logger.Sync())
	fi, err := os.Stat(logname1)
	require.NoError(t, err)
	assert.Greater(t, fi.Size(), maxSizeInBytes>>4)
	assert.Less(t, fi.Size(), maxSizeInBytes)
	fi, err = os.Stat(logname2)
	require.NoError(t, err)
	assert.Greater(t, fi.Size(), maxSizeInBytes>>4)

	logLineRegex := regexp.MustCompile(`^\d{4}/\d\d/\d\d \d\d:\d\d:\d\d\.\d{6}: logging_test\.go`)

	t.Run("test the rotation cut line", func(t *testing.T) {
		// Ensure the last log starts with a timestamp and filename:
		// 2024/11/21 15:41:07.549096: logging_test.go:70 Info: ...
		f, err := os.Open(logname1)
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		buf := make([]byte, 43)
		n, err := f.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, len(buf), n)
		assert.True(t, logLineRegex.Match(buf), string(buf))
	})

	oldestRoll := ""
	t.Run("test rotation file", func(t *testing.T) {
		found := 0
		err := ForEachRotation(logname1, func(rotationFileName string) error {
			// Skip the current file name.
			if rotationFileName == logname1 {
				return nil
			}
			found++
			oldestRoll = rotationFileName
			fi, err := os.Stat(rotationFileName)
			require.NoError(t, err)
			assert.Greater(t, fi.Size(), maxSizeInBytes-(maxSizeInBytes>>10), rotationFileName)
			assert.Less(t, fi.Size(), maxSizeInBytes)
			f, err := os.Open(rotationFileName)
			require.NoError(t, err)
			defer func() { _ = f.Close() }()
			// Ensure the first line of the rotation file ends with "begin":
			// 2024/11/21 15:41:07.549096: logging_test.go:70 Info: begin
			buf := make([]byte, 58)
			n, err := f.Read(buf)
			require.NoError(t, err)
			assert.Equal(t, len(buf), n)
			assert.True(t, logLineRegex.Match(buf), string(buf))
			assert.True(t, strings.HasSuffix(string(buf), "begin"))
			return err
		})
		require.NoError(t, err)
		assert.Equal(t, 1, found)
	})
	t.Run("ensure the oldest rotation is deleted", func(t *testing.T) {
		// Write more to trigger a rotation.
		for i := 0; i < 4000; i++ {
			logger.Info("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
		}
		assert.NoError(t, logger.Sync())
		// lumberjack removes the old files asynchronously, therefore the test
		// has to wait for it.
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			_, err := os.Stat(oldestRoll)
			require.ErrorIs(c, err, fs.ErrNotExist)
		}, 5*time.Second, 500*time.Millisecond)

		t.Run("ensure there are still only 2 files", func(t *testing.T) {
			found := 0
			err := ForEachRotation(logname1, func(_ string) error {
				found += 1
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, 2, found)
		})
	})

}

func TestForEachRotation(t *testing.T) {
	dir := t.TempDir()
	prefix := "test"
	ext := ".log"
	// Timestamps to create test rotation files:
	for _, timestamp := range []string{
		"", // current log file.
		"-2017-11-04T18-30-00.000",
		"-2016-11-04T18-30-00.001",
		"-2016-11-04T18-30-00.003",
		"-2016-11-04T18-30-00.002",
		"-2016-11-04T18-30-00.000",
		"-2017-12-04T18-30-00.000",
		"-not-matched-file",
	} {
		f, _ := os.Create(filepath.Join(dir, prefix+timestamp+ext))
		_, _ = f.WriteString(timestamp)
		_ = f.Close()
	}
	var lines = make([]string, 0, 3)
	err := ForEachRotation(filepath.Join(dir, prefix+ext), func(filepath string) error {
		b, err := os.ReadFile(filepath)
		assert.NoError(t, err)
		lines = append(lines, string(b))
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, lines, []string{
		"-2016-11-04T18-30-00.000",
		"-2016-11-04T18-30-00.001",
		"-2016-11-04T18-30-00.002",
		"-2016-11-04T18-30-00.003",
		"-2017-11-04T18-30-00.000",
		"-2017-12-04T18-30-00.000",
		"",
	}, "files have to be read in the older-first order")
}
