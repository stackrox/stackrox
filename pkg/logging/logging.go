// Package logging provides the logger used in StackRox Go programs.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const (
	//PanicLevel log level
	PanicLevel = 70
	//FatalLevel log level
	FatalLevel = 60
	//ErrorLevel log level
	ErrorLevel = 50
	//WarnLevel log level
	WarnLevel = 40
	//InternalLevel log level
	InternalLevel = 35
	//InfoLevel log level
	InfoLevel = 30
	//DebugLevel log level
	DebugLevel = 20
	//InitRetryLevel shows failures within loops
	InitRetryLevel = 15
	//TraceLevel log level
	TraceLevel = 10
)

var (
	// DefaultDestination is the default logging destination, which is currently os.Stdout
	DefaultDestination = os.Stdout
	//RootLogger is the convenience logger used when module specific loggers are not specified
	RootLogger = New("root logger")
	//all registered loggers thus far
	knownLoggers = make(map[string]*Logger)
	//DefaultLevel is the default log level
	DefaultLevel = InfoLevel

	//ValidLevels is a map of all valid level names to their int values
	ValidLevels = map[string]int{
		"Panic":     PanicLevel,
		"Fatal":     FatalLevel,
		"Error":     ErrorLevel,
		"Warn":      WarnLevel,
		"Internal":  InternalLevel,
		"Info":      InfoLevel,
		"Debug":     DebugLevel,
		"InitRetry": InitRetryLevel,
		"Trace":     TraceLevel,
	}

	// ReverseValidLabels is a reverse mapping for ints to strings denoting log levels.
	ReverseValidLabels = reverseLogMap()
)

//Fields provides the correctly typed input for the WithFields() method
type Fields map[string]interface{}

func init() {
	initLevel := os.Getenv("LOGLEVEL")
	for name, value := range ValidLevels {
		if initLevel == name {
			SetGlobalLogLevel(value)
		}
	}
	//print whatever called rootlogger, not rootlogger calling Logger methods.
	//Ie, void printing: logging.go:259...
	RootLogger.StackTraceLevel = 3
}

//SetGlobalLogLevel sets the log level on all loggers, regardless of module
func SetGlobalLogLevel(level int) {
	for _, l := range knownLoggers {
		l.levelLock.Lock()
		l.logLevel = level
		l.levelLock.Unlock()
	}
	DefaultLevel = level
	RootLogger.Debugf("Set log level to: %d", level)
}

//Logger wraps default go log implementation to allow log levels
type Logger struct {
	internal        *log.Logger
	logLevel        int
	levelLock       sync.RWMutex
	fieldCache      map[string]interface{}
	cacheLock       sync.RWMutex
	StackTraceLevel int
	alternateWriter io.Writer
}

//New returns a new Logger
func New(module string) *Logger {
	baseLogger := log.New(DefaultDestination, module+": ", log.Lshortfile|log.Ldate|log.Lmicroseconds|log.LUTC)
	newLogger := Logger{
		baseLogger,
		DefaultLevel,
		sync.RWMutex{},
		make(map[string]interface{}),
		sync.RWMutex{},
		2,
		nil,
	}
	knownLoggers[module] = &newLogger
	return &newLogger
}

//SetOutput redirects log messages to a writer other than std out
func (l *Logger) SetOutput(w io.Writer) {
	l.internal.SetOutput(w)
	l.alternateWriter = w
}

// LogLevel returns the int value for the current log level.
func (l *Logger) LogLevel() int {
	l.levelLock.RLock()
	defer l.levelLock.RUnlock()

	return l.logLevel
}

// SetLogLevel sets the log level with provided level.
func (l *Logger) SetLogLevel(level int) {
	l.levelLock.Lock()
	defer l.levelLock.Unlock()

	l.logLevel = level
}

func (l *Logger) consumeCache() string {
	append := ""

	l.cacheLock.Lock()
	defer l.cacheLock.Unlock()

	for k, v := range l.fieldCache {
		append += fmt.Sprintf("\t%s: %s", k, v)
	}
	l.fieldCache = make(map[string]interface{}) //empty cache

	return append
}

//GetLogLevel returns the log level in human readable string format
func (l *Logger) GetLogLevel() string {
	l.levelLock.RLock()
	defer l.levelLock.RUnlock()

	for name, value := range ValidLevels {
		if l.logLevel == value {
			return name
		}
	}
	return "Invalid"
}

//Trace provide super low level detail
func (l *Logger) Trace(args ...interface{}) {
	level := l.LogLevel()

	if level <= TraceLevel {
		l.internal.Output(l.StackTraceLevel, "Trace: "+fmt.Sprint(args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Tracef provide super low level detail
func (l *Logger) Tracef(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= TraceLevel {
		l.internal.Output(l.StackTraceLevel, "Trace: "+fmt.Sprintf(format, args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Retry describes the inner contents of loops that retry an action several times
func (l *Logger) Retry(args ...interface{}) {
	level := l.LogLevel()

	if level <= InitRetryLevel {
		l.internal.Output(l.StackTraceLevel, "Retry: "+fmt.Sprint(args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Retryf describes the inner contents of loops that retry an action several times
func (l *Logger) Retryf(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= InitRetryLevel {
		l.internal.Output(l.StackTraceLevel, "Retry: "+fmt.Sprintf(format, args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Debug provides standard debug messages
func (l *Logger) Debug(args ...interface{}) {
	level := l.LogLevel()

	if level <= DebugLevel {
		l.internal.Output(l.StackTraceLevel, "Debug: "+fmt.Sprint(args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Debugf provides standard debug messages
func (l *Logger) Debugf(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= DebugLevel {
		l.internal.Output(l.StackTraceLevel, "Debug: "+fmt.Sprintf(format, args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Info displays information
func (l *Logger) Info(args ...interface{}) {
	level := l.LogLevel()

	if level <= InfoLevel {
		l.internal.Output(l.StackTraceLevel, "Info: "+fmt.Sprint(args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Infof displays information
func (l *Logger) Infof(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= InfoLevel {
		l.internal.Output(l.StackTraceLevel, "Info: "+fmt.Sprintf(format, args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Warn displays a warning
func (l *Logger) Warn(args ...interface{}) {
	level := l.LogLevel()

	if level <= WarnLevel {
		l.internal.Output(l.StackTraceLevel, "Warn: "+fmt.Sprint(args...)+l.consumeCache())
	}
	l.consumeCache()
}

//Warnf displays a warning
func (l *Logger) Warnf(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= WarnLevel {
		l.internal.Output(l.StackTraceLevel, "Warn: "+fmt.Sprintf(format, args...)+
			l.consumeCache())
	}
	l.consumeCache()
}

//Error logs an error
func (l *Logger) Error(args ...interface{}) {
	level := l.LogLevel()

	if level <= ErrorLevel {
		l.internal.Output(l.StackTraceLevel, "Error: "+fmt.Sprint(args...)+
			l.consumeCache())
	}
	l.consumeCache()
}

//Errorf logs an error
func (l *Logger) Errorf(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= ErrorLevel {
		l.internal.Output(l.StackTraceLevel, "Error: "+fmt.Sprintf(format, args...)+
			l.consumeCache())
	}
	l.consumeCache()
}

//Fatal logs and exits
func (l *Logger) Fatal(args ...interface{}) {
	level := l.LogLevel()

	if level <= FatalLevel {
		l.internal.Output(l.StackTraceLevel, "Fatal: "+fmt.Sprint(args...)+l.consumeCache())
		os.Exit(1)
	}
	l.consumeCache()
}

//Fatalf logs and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= FatalLevel {
		l.internal.Output(l.StackTraceLevel, "Fatal: "+fmt.Sprintf(format, args...)+l.consumeCache())
		os.Exit(1)
	}
	l.consumeCache()
}

//Panic logs and throws a panic
func (l *Logger) Panic(args ...interface{}) {
	level := l.LogLevel()

	if level <= PanicLevel {
		l.internal.Output(l.StackTraceLevel, "Panic: "+fmt.Sprint(args...)+l.consumeCache())
		panic(fmt.Sprint(args...))
	}
	l.consumeCache()
}

//Panicf logs and throws a panic
func (l *Logger) Panicf(format string, args ...interface{}) {
	level := l.LogLevel()

	if level <= PanicLevel {
		l.internal.Output(l.StackTraceLevel, "Panic: "+fmt.Sprintf(format, args...)+l.consumeCache())
		panic(fmt.Sprintf(format, args...))
	}
	l.consumeCache()
}

//Log logs the message regardless of loglevel
func (l *Logger) Log(args ...interface{}) {
	l.internal.Output(l.StackTraceLevel, fmt.Sprint(args)+l.consumeCache())
}

//Logf logs the message regardless of loglevel
func (l *Logger) Logf(format string, args ...interface{}) {
	l.internal.Output(l.StackTraceLevel, fmt.Sprintf(format, args)+l.consumeCache())
}

//Write prints the arguments without the log prefixes - if and only if - SetOutput has been previously called
func (l *Logger) Write(args ...interface{}) {
	if l.alternateWriter != nil {
		l.alternateWriter.Write([]byte(fmt.Sprint(args...) + l.consumeCache()))
	} else {
		l.Log(args...)
	}
}

//Writef prints the formatted arguments without the log prefixes - if and only if - SetOutput has been previously called
func (l *Logger) Writef(format string, args ...interface{}) {
	if l.alternateWriter != nil {
		l.alternateWriter.Write([]byte(fmt.Sprintf(format, args) + l.consumeCache()))
	} else {
		l.Logf(format, args)
	}
}

//WithFields provides custom formatted output
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	temp := l.Copy()

	temp.cacheLock.Lock()
	defer temp.cacheLock.Unlock()

	for k, v := range fields {
		temp.fieldCache[k] = v
	}

	return temp
}

//WithError is a convenience wrapper for WithFields and a single error type
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(Fields{"error": err.Error()})
}

//Copy returns a deep copy of a given logger
func (l *Logger) Copy() *Logger {
	l.cacheLock.RLock()
	defer l.cacheLock.RUnlock()

	l.levelLock.RLock()
	defer l.levelLock.RUnlock()

	copy := Logger{
		internal:        l.internal,
		logLevel:        l.logLevel,
		fieldCache:      make(map[string]interface{}, len(l.fieldCache)),
		StackTraceLevel: l.StackTraceLevel,
		alternateWriter: l.alternateWriter,
	}

	for k, v := range l.fieldCache {
		copy.fieldCache[k] = v
	}

	return &copy
}

//convenience methods log apply to root logger

// Debug implements logging.Logger interface.
func Debug(args ...interface{}) { RootLogger.Debug(args...) }

// Debugf implements logging.Logger interface.
func Debugf(format string, args ...interface{}) { RootLogger.Debugf(format, args...) }

// Error implements logging.Logger interface.
func Error(args ...interface{}) { RootLogger.Error(args...) }

// Errorf implements logging.Logger interface.
func Errorf(format string, args ...interface{}) { RootLogger.Errorf(format, args...) }

// Fatal implements logging.Logger interface.
func Fatal(args ...interface{}) { RootLogger.Fatal(args...) }

// Fatalf implements logging.Logger interface.
func Fatalf(format string, args ...interface{}) { RootLogger.Fatalf(format, args...) }

// Fatalln implements logging.Logger interface.
func Fatalln(args ...interface{}) { RootLogger.internal.Fatalln(args...) }

// Info implements logging.Logger interface.
func Info(args ...interface{}) { RootLogger.Info(args...) }

// Infof implements logging.Logger interface.
func Infof(format string, args ...interface{}) { RootLogger.Infof(format, args...) }

// Panic implements logging.Logger interface.
func Panic(args ...interface{}) { RootLogger.Panic(args...) }

// Panicf implements logging.Logger interface.
func Panicf(format string, args ...interface{}) { RootLogger.Panicf(format, args...) }

// Panicln implements logging.Logger interface.
func Panicln(args ...interface{}) { RootLogger.internal.Panicln(args...) }

// Print implements logging.Logger interface.
func Print(args ...interface{}) { RootLogger.internal.Print(args...) }

// Printf implements logging.Logger interface.
func Printf(format string, args ...interface{}) { RootLogger.internal.Printf(format, args...) }

// Println implements logging.Logger interface.
func Println(args ...interface{}) { RootLogger.internal.Println(args...) }

// Warn implements logging.Logger interface.
func Warn(args ...interface{}) { RootLogger.Warn(args...) }

// Warnf implements logging.Logger interface.
func Warnf(format string, args ...interface{}) { RootLogger.Warnf(format, args...) }

//Log logs the message regardless of log level
func Log(args ...interface{}) {
	RootLogger.Log(args)
}

//Logf logs the message regardless of log level
func Logf(format string, args ...interface{}) {
	RootLogger.Logf(format, args)
}

//Write prints the arguments without the log prefixes - if and only if - SetOutput has been previously called
func Write(args ...interface{}) {
	RootLogger.Write(args...)
}

//Writef prints the formatted arguments without the log prefixes - if and only if - SetOutput has been previously called
func Writef(format string, args ...interface{}) {
	RootLogger.Writef(format, args)
}

//WithFields provides custom formatted output
func WithFields(fields map[string]interface{}) *Logger {
	temp := RootLogger.Copy()

	temp.cacheLock.Lock()
	defer temp.cacheLock.Unlock()

	for k, v := range fields {
		temp.fieldCache[k] = v
	}

	return temp
}

//WithError is a convenience wrapper for WithFields and a single error type
func WithError(err error) *Logger {
	return WithFields(Fields{"error": err.Error()})
}

//GetLogLevel returns the log level in human readable string format
func GetLogLevel() string {
	return RootLogger.GetLogLevel()
}

func reverseLogMap() map[int]string {
	m := make(map[int]string)
	for k, v := range ValidLevels {
		m[v] = k
	}
	return m
}
