package logging

import (
	"testing"

	uuid "github.com/satori/go.uuid"
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
	levels, errs := parseDefaultModuleLevels("foo=Info,, bar =debug, qux = Trace,")
	assert.Equal(t, levels, map[string]int32{
		"foo": InfoLevel,
		"bar": DebugLevel,
		"qux": TraceLevel,
	})
	assert.Empty(t, errs)
}

func TestParseDefaultModuleLevels_Errs(t *testing.T) {
	levels, errs := parseDefaultModuleLevels("foo=Info, baz , bar =random, qux=Trace,")
	assert.Equal(t, levels, map[string]int32{
		"foo": InfoLevel,
		"qux": TraceLevel,
	})
	assert.Len(t, errs, 2)
}

func TestNilModuleAlwaysReturnsFalseOnUnref(t *testing.T) {
	var m *Module
	assert.False(t, m.unref())
}

func TestModuleSetLogLevel(t *testing.T) {
	CurrentModule().SetLogLevel(DebugLevel)
	assert.Equal(t, DebugLevel, CurrentModule().GetLogLevel())
}

func TestModuleRefCountingWorks(t *testing.T) {
	module := newModule(uuid.NewV4().String(), zap.NewAtomicLevelAt(zapcore.InfoLevel))
	module.ref()
	assert.False(t, module.unref())
	assert.True(t, module.unref())
}

func TestLoggerCreatedFromModuleReferencesModule(t *testing.T) {
	module := newModule(uuid.NewV4().String(), zap.NewAtomicLevelAt(zapcore.InfoLevel))
	logger := module.Logger()
	assert.Equal(t, module, logger.module)
}

func TestRegistryPurgesModulesWithRefCountOfZero(t *testing.T) {
	name := uuid.NewV4().String()
	modules.getOrAddModule(name)
	modules.unrefModule(name)

	assert.NotContains(t, modules.modules, name)
}
