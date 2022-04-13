package logging

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
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
