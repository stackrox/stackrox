package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps a zap.SugaredLogger.
type Logger struct {
	*zap.SugaredLogger
	module *Module
}

// Log logs at level.
func (l *Logger) Log(level zapcore.Level, args ...interface{}) {
	switch level {
	case zapcore.PanicLevel:
		l.Panic(args...)
	case zapcore.FatalLevel:
		l.Fatal(args...)
	case zapcore.ErrorLevel:
		l.Error(args...)
	case zapcore.WarnLevel:
		l.Warn(args...)
	case zapcore.InfoLevel:
		l.Info(args...)
	case zapcore.DebugLevel:
		l.Debug(args...)
	}
}

// Logf logs at level.
func (l *Logger) Logf(level zapcore.Level, template string, args ...interface{}) {
	switch level {
	case zapcore.PanicLevel:
		l.Panicf(template, args...)
	case zapcore.FatalLevel:
		l.Fatalf(template, args...)
	case zapcore.ErrorLevel:
		l.Errorf(template, args...)
	case zapcore.WarnLevel:
		l.Warnf(template, args...)
	case zapcore.InfoLevel:
		l.Infof(template, args...)
	case zapcore.DebugLevel:
		l.Debugf(template, args...)
	}
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
