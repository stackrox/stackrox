package logging

import (
	"go.uber.org/zap"
)

// Logger wraps a zap.SugaredLogger.
type Logger struct {
	*zap.SugaredLogger
	module *Module
}

// Module module returns the module that l belongs to.
func (l *Logger) Module() *Module {
	return l.module
}

// finalize finalizes l and decrements the per-module reference count.
func (l *Logger) finalize() {
	if l.module != nil {
		modules.unrefModule(l.module.name)
	}
}
