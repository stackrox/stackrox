package logging

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetCallingModule(t *testing.T) {
	assert.Equal(t, "pkg/logging", getCallingModule(0))
}

func TestCurrentModuleReturnsValidModule(t *testing.T) {
	assert.NotNil(t, CurrentModule())
}

func TestForEachModuleVisitsIndividualModules(t *testing.T) {
	// Make sure that pkg/logging is registered
	assert.NotNil(t, CurrentModule())
	assert.NotNil(t, ModuleForName("some/module"))

	visited := map[string]struct{}{}

	ForEachModule(func(name string, m *Module) {
		if m != nil {
			visited[name] = struct{}{}
		}
	}, SelectAll)

	assert.Contains(t, visited, "pkg/logging")
	assert.Contains(t, visited, "some/module")
}

func TestParseDefaultModuleLevels_Success(t *testing.T) {
	levels, errs := parseDefaultModuleLevels("foo=Info,, bar =debug,")
	assert.Equal(t, levels, map[string]zapcore.Level{
		"foo": zapcore.InfoLevel,
		"bar": zapcore.DebugLevel,
	})
	assert.Empty(t, errs)
}

func TestParseDefaultModuleLevels_Errs(t *testing.T) {
	levels, errs := parseDefaultModuleLevels("foo=Info, baz , bar =random, qux=debug,")
	assert.Equal(t, levels, map[string]zapcore.Level{
		"foo": zapcore.InfoLevel,
		"qux": zapcore.DebugLevel,
	})
	assert.Len(t, errs, 2)
}

func TestNilModuleAlwaysReturnsFalseOnUnref(t *testing.T) {
	var m *Module
	assert.False(t, m.unref())
}

func TestModuleSetLogLevel(t *testing.T) {
	CurrentModule().SetLogLevel(zapcore.DebugLevel)
	assert.Equal(t, zapcore.DebugLevel, CurrentModule().GetLogLevel())
}

func TestModuleRefCountingWorks(t *testing.T) {
	module := newModule(uuid.Must(uuid.NewV4()).String(), zap.NewAtomicLevelAt(zapcore.InfoLevel))
	module.ref()
	assert.False(t, module.unref())
	assert.True(t, module.unref())
}

func TestLoggerCreatedFromModuleReferencesModule(t *testing.T) {
	module := newModule(uuid.Must(uuid.NewV4()).String(), zap.NewAtomicLevelAt(zapcore.InfoLevel))
	logger := module.Logger()
	assert.Equal(t, module, logger.module)
}

func TestRegistryPurgesModulesWithRefCountOfZero(t *testing.T) {
	name := uuid.Must(uuid.NewV4()).String()
	modules.getOrAddModule(name)
	modules.unrefModule(name)

	assert.NotContains(t, modules.modules, name)
}

func TestLoggerCreatedFromModuleUpdatesLogLevel(t *testing.T) {
	for zapLevel := range validLevels {
		module := newModule(uuid.Must(uuid.NewV4()).String(), zap.NewAtomicLevelAt(zapcore.InfoLevel))
		logger := module.Logger()
		module.SetLogLevel(zapLevel)
		assert.True(t, logger.SugaredLogger().Desugar().Core().Enabled(zapLevel))
		// uses internal knowledge of how Enabled(level) method works
		assert.False(t, logger.SugaredLogger().Desugar().Core().Enabled(zapLevel-1))
	}
}

func TestLoggerLevelUpdatesWithGlobalLevel(t *testing.T) {
	module := ModuleForName(uuid.Must(uuid.NewV4()).String())
	logger := module.Logger()
	level := GetGlobalLogLevel()

	module.SetLogLevel(zapcore.DebugLevel)

	// verify the global level remains unchanged
	assert.Equal(t, level, GetGlobalLogLevel())
	assert.True(t, logger.SugaredLogger().Desugar().Core().Enabled(zapcore.DebugLevel))

	SetGlobalLogLevel(zapcore.WarnLevel)
	// verify logger level changes with global level
	assert.Equal(t, zapcore.WarnLevel, GetGlobalLogLevel())
	assert.True(t, logger.SugaredLogger().Desugar().Core().Enabled(zapcore.WarnLevel))
	assert.False(t, logger.SugaredLogger().Desugar().Core().Enabled(zapcore.InfoLevel))
}
