package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface exposed for logging purposes
type Logger interface {
	Log(level zapcore.Level, args ...interface{})
	Logf(level zapcore.Level, format string, args ...interface{})
	Panicf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Debugf(template string, args ...interface{})
	Panic(args ...interface{})
	Fatal(args ...interface{})
	Error(args ...interface{})
	Warn(args ...interface{})
	Info(args ...interface{})
	Debug(args ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
}

// LoggerImpl wraps a zap.SugaredLogger.
type LoggerImpl struct {
	*zap.SugaredLogger
	module *Module
}

// Log logs at level.
func (l *LoggerImpl) Log(level zapcore.Level, args ...interface{}) {
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
func (l *LoggerImpl) Logf(level zapcore.Level, template string, args ...interface{}) {
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
func (l *LoggerImpl) Module() *Module {
	return l.module
}

// finalize finalizes l and decrements the per-module reference count.
func (l *LoggerImpl) finalize() {
	if l.module != nil {
		modules.unrefModule(l.module.name)
	}
}
