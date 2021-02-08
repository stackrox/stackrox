package logging

import (
	"testing"

	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMappingZapLevelsValidLevels(t *testing.T) {
	for k := range levelToZapLevel {
		_, ok := validLevels[k]
		assert.True(t, ok)
	}
}

func TestMappingValidLevelsZapLevels(t *testing.T) {
	for k := range validLevels {
		_, ok := levelToZapLevel[k]
		assert.True(t, ok)
	}
}

func TestLevelForLabel(t *testing.T) {
	for _, label := range []string{"warn", "WARN", "WaRn"} {
		lvl, ok := LevelForLabel(label)
		assert.Equal(t, WarnLevel, lvl)
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
	_, ok := LabelForLevel(-1)
	assert.False(t, ok)
	label := LabelForLevelOrInvalid(-1)
	assert.Equal(t, "Invalid", label)
}

func TestSortedLevels(t *testing.T) {
	assert.Equal(t, sortedLevels, SortedLevels())
}

func TestSetGlobalLogLevel(t *testing.T) {
	mInfo := ModuleForName(uuid.NewV4().String())
	assert.Equal(t, GetGlobalLogLevel(), mInfo.GetLogLevel())

	SetGlobalLogLevel(DebugLevel)
	mDebug := ModuleForName(uuid.NewV4().String())
	assert.Equal(t, GetGlobalLogLevel(), mDebug.GetLogLevel())
	assert.Equal(t, GetGlobalLogLevel(), mInfo.GetLogLevel())
}
