package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface exposed for logging purposes.
//
//go:generate mockgen-wrapper
type Logger interface {
	Log(level zapcore.Level, args ...interface{})
	Logf(level zapcore.Level, format string, args ...interface{})

	Panic(args ...interface{})
	Panicf(template string, args ...interface{})
	Panicw(msg string, keysAndValues ...interface{})

	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Errorw(msg string, keysAndValues ...interface{})

	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Warnw(msg string, keysAndValues ...interface{})

	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Infow(msg string, keysAndValues ...interface{})

	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Debugw(msg string, keysAndValues ...interface{})

	SugaredLogger() *zap.SugaredLogger
}

// LoggerImpl wraps a zap.SugaredLogger.
type LoggerImpl struct {
	InnerLogger *zap.SugaredLogger
	module      *Module
	opts        *options
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

// SugaredLogger exposes the underlying sugared logger.
func (l *LoggerImpl) SugaredLogger() *zap.SugaredLogger {
	return l.InnerLogger
}

// Panic uses fmt.Sprintf to construct and log a message, then panics.
func (l *LoggerImpl) Panic(args ...interface{}) {
	l.InnerLogger.Panic(args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func (l *LoggerImpl) Panicf(template string, args ...interface{}) {
	l.InnerLogger.Panicf(template, args...)
}

// Panicw logs a message with some additional context, then panics.
// The variadic key-value pairs are treated as in zap SugaredLogger With.
func (l *LoggerImpl) Panicw(msg string, keysAndValues ...interface{}) {
	l.InnerLogger.Panicw(msg, keysAndValues...)
}

// Fatal uses fmt.Sprintf to construct and log a message, then calls os.Exit.
func (l *LoggerImpl) Fatal(args ...interface{}) {
	l.InnerLogger.Fatal(args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func (l *LoggerImpl) Fatalf(template string, args ...interface{}) {
	l.InnerLogger.Fatalf(template, args...)
}

// Fatalw logs a message with some additional context, then calls os.Exit.
// The variadic key-value pairs are treated as in zap SugaredLogger With.
func (l *LoggerImpl) Fatalw(msg string, keysAndValues ...interface{}) {
	l.InnerLogger.Fatalw(msg, keysAndValues...)
}

// Error uses fmt.Sprintf to construct and log a message.
func (l *LoggerImpl) Error(args ...interface{}) {
	l.InnerLogger.Error(args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func (l *LoggerImpl) Errorf(template string, args ...interface{}) {
	l.InnerLogger.Errorf(template, args...)
}

// Errorw logs a message with some additional context.
// The variadic key-value pairs are treated as in zap SugaredLogger With.
func (l *LoggerImpl) Errorw(msg string, keysAndValues ...interface{}) {
	l.InnerLogger.Errorw(msg, keysAndValues...)

	l.createAdministrationEventFromLog(msg, "error", keysAndValues...)
}

// Warn uses fmt.Sprintf to construct and log a message.
func (l *LoggerImpl) Warn(args ...interface{}) {
	l.InnerLogger.Warn(args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func (l *LoggerImpl) Warnf(template string, args ...interface{}) {
	l.InnerLogger.Warnf(template, args...)
}

// Warnw logs a message with some additional context.
// The variadic key-value pairs are treated as in zap SugaredLogger With.
func (l *LoggerImpl) Warnw(msg string, keysAndValues ...interface{}) {
	l.InnerLogger.Warnw(msg, keysAndValues...)

	l.createAdministrationEventFromLog(msg, "warn", keysAndValues...)
}

// Info uses fmt.Sprintf to construct and log a message.
func (l *LoggerImpl) Info(args ...interface{}) {
	l.InnerLogger.Info(args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func (l *LoggerImpl) Infof(template string, args ...interface{}) {
	l.InnerLogger.Infof(template, args...)
}

// Infow logs a message with some additional context.
// The variadic key-value pairs are treated as in zap SugaredLogger With.
func (l *LoggerImpl) Infow(msg string, keysAndValues ...interface{}) {
	l.InnerLogger.Infow(msg, keysAndValues...)

	l.createAdministrationEventFromLog(msg, "info", keysAndValues...)
}

// Debug uses fmt.Sprintf to construct and log a message.
func (l *LoggerImpl) Debug(args ...interface{}) {
	l.InnerLogger.Debug(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func (l *LoggerImpl) Debugf(template string, args ...interface{}) {
	l.InnerLogger.Debugf(template, args...)
}

// Debugw logs a message with some additional context.
// The variadic key-value pairs are treated as in zap SugaredLogger With.
func (l *LoggerImpl) Debugw(msg string, keysAndValues ...interface{}) {
	l.InnerLogger.Debugw(msg, keysAndValues...)
}

func (l *LoggerImpl) createAdministrationEventFromLog(msg string, level string, keysAndValues ...interface{}) {
	// Short-circuit if no log event stream or converter is found.
	if l.opts.AdministrationEventsStream == nil || l.opts.AdministrationEventsConverter == nil {
		return
	}

	// We will use the log converter to convert logs to an events.AdministrationEvent.
	if event := l.opts.AdministrationEventsConverter.Convert(msg, level, l.Module().Name(), keysAndValues...); event != nil {
		l.opts.AdministrationEventsStream.Produce(event)
	}
}
