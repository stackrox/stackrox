package logging

import (
	"runtime"
	"strings"

	"github.com/stackrox/rox/pkg/sync"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// SelectAll is a selector that selects all modules.
	SelectAll []string

	modules = &registry{
		modules: map[string]*Module{},
	}
)

// CurrentModule returns the module corresponding to the caller.
func CurrentModule() *Module {
	return currentModule(3)
}

func currentModule(skip int) *Module {
	return modules.getOrAddModule(getCallingModule(skip))
}

// ModuleForName returns the Module corresponding to name.
func ModuleForName(name string) *Module {
	return modules.getOrAddModule(name)
}

// ForEachModule invokes f for known modules. If selector is not empty,
// f is invoked for every entry in selector:
//   - a pointer to a known Module instance if the entry is known
//   - a nil pointer if the entry is not known
func ForEachModule(f func(name string, m *Module), selector []string) {
	modules.forEachModule(f, selector)
}

// Module bundles up logging-specific information about a module.
type Module struct {
	logLevel zap.AtomicLevel
	name     string
	// Technically, the reference count would not have to an atomic as
	// per-module access is guarded by the global registry lock. However,
	// we remind ourselves and our future self that we have to think about
	// concurrency and enjoy compact statements like m.refCounter.Dec() == 0.
	refCounter *atomic.Int32
}

// newModule creates a new module with name, logLevel and initial referenceCount of 1.
func newModule(name string, logLevel zap.AtomicLevel) *Module {
	return &Module{
		logLevel:   logLevel,
		refCounter: atomic.NewInt32(1),
		name:       name,
	}
}

// Logger returns a new logger for m.
func (m *Module) Logger(opts ...OptionsFunc) *LoggerImpl {
	return CreateLogger(m, 0, opts...)
}

// Name returns the name of m
func (m *Module) Name() string {
	return m.name
}

// GetLogLevel returns the log level of the module.
func (m *Module) GetLogLevel() zapcore.Level {
	return m.logLevel.Level()
}

// SetLogLevel adjusts the log level of m to level.
//
// Adjusting a per-module log level propagates to all
// loggers created for the respective module.
func (m *Module) SetLogLevel(level zapcore.Level) {
	m.logLevel.SetLevel(level)
}

// ref increments the reference count of m by 1.
func (m *Module) ref() {
	if m != nil {
		m.refCounter.Inc()
	}
}

// unref decrements the reference count of m by 1 and returns true
// if the counter has reached 0.
func (m *Module) unref() bool {
	if m != nil {
		return m.refCounter.Dec() == 0
	}
	return false
}

// registry keeps track of modules indexed by name.
type registry struct {
	sync.Mutex
	modules map[string]*Module
}

func (r *registry) forEachModule(f func(name string, m *Module), selector []string) {
	r.Lock()
	defer r.Unlock()

	if len(selector) > 0 {
		for _, name := range selector {
			f(name, r.modules[name])
		}
	} else {
		for k, v := range r.modules {
			f(k, v)
		}
	}
}

func (r *registry) getOrAddModule(name string) *Module {
	r.Lock()
	defer r.Unlock()

	m, known := r.modules[name]
	if !known {
		m = newModule(name, zap.NewAtomicLevelAt(GetGlobalLogLevel()))
		r.modules[name] = m
	} else {
		m.ref()
	}

	return m
}

func (r *registry) unrefModule(name string) {
	r.Lock()
	defer r.Unlock()

	if r.modules[name].unref() {
		delete(r.modules, name)
	}
}

// callerFileToPackage takes the path of the source file of the caller (<skip> frames up the call stack), and returns
// the corresponding Go package. If no package could be determined, the empty string is returned.
func callerFileToPackage(skip int) string {
	callers := [1]uintptr{}
	if runtime.Callers(2+skip, callers[:]) != 1 {
		return ""
	}

	name := runtime.FuncForPC(callers[0]).Name()
	dotIdx := -1

	// Find the leftmost '.' after the rightmost '/'.
	for i := len(name) - 1; i >= 0 && name[i] != '/'; i-- {
		if name[i] == '.' {
			dotIdx = i
		}
	}
	if dotIdx != -1 {
		name = name[:dotIdx]
	}

	return name
}

// getCallingModule returns the short name of the module calling this function, skipping <skip> stack frames in the
// call stack.
// The short name is determined as follows:
//   - If the package is a subpackage of "<projectPrefix>/pkg", the short name is the result of stripping
//     "<projectPrefix>/".
//   - Otherwise, if the package is a subpackage of "<projectPrefix>/", the short name is the result of stripping
//     "<projectPrefix>/" and the first component of the remaining path name, including any trailing slashes. If the
//     resulting string is the empty string, the short name is "main".
//   - Otherwise, the short name is the full package name.
func getCallingModule(skip int) string {
	callingPackage := callerFileToPackage(skip)
	if callingPackage == "" {
		return ""
	}

	shortenedCallingPackage := strings.TrimPrefix(callingPackage, projectPrefix+"/")
	if len(shortenedCallingPackage) == len(callingPackage) {
		return callingPackage
	}
	components := strings.SplitN(shortenedCallingPackage, "/", 2)
	if components[0] == "pkg" {
		return shortenedCallingPackage
	} else if len(components) == 1 {
		return "main"
	}
	return components[1]
}
