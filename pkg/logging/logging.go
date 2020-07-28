// Package logging provides the logger used in StackRox Go programs.
//
// This package supports runtime configuration via the following
// environment variables:
//   * LOGLEVEL supporting the following values (case insensitive), order is indicative of importance:
//     * fatal
//     * panic
//     * error
//     * warn
//     * internal (deprecated, mapped to info)
//     * info
//     * debug
//     * initretry (deprecated, mapped to debug)
//     * trace (deprecated, mapped to debug)
//   * LOGENCODING supporting the following values:
//     * json
//     * console
//   * MODULE_LOGLEVELS supporting ,-separated module=level pairs, e.g.: grpc=debug,kubernetes=warn
//   * MAX_LOG_LINE_QUOTA in the format max/duration_in_seconds, e.g.: 100/10
//
// LOGLEVEL semantics follow common conventions, i.e., any log message with a level less than the
// currently set log level will be discarded.
package logging

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	//FatalLevel log level
	FatalLevel int32 = 70
	//PanicLevel log level
	PanicLevel int32 = 60
	//ErrorLevel log level
	ErrorLevel int32 = 50
	//WarnLevel log level
	WarnLevel int32 = 40
	//InternalLevel log level
	InternalLevel int32 = 35
	//InfoLevel log level
	InfoLevel int32 = 30
	//DebugLevel log level
	DebugLevel int32 = 20
	//InitRetryLevel shows failures within loops
	InitRetryLevel int32 = 15
	//TraceLevel log level
	TraceLevel int32 = 10

	// defaultDestination is the default logging destination, which is currently os.Stderr
	defaultDestination = "stderr"

	// Our project prefix. For all subpackages of this, we strip this prefix.
	projectPrefix = "github.com/stackrox/rox"

	// LoggingPath is the common log file so we can export it
	LoggingPath = "/var/log/stackrox/log.txt"
)

var (
	console = struct {
		encoding   string
		encodeTime zapcore.TimeEncoder
		separator  string
		fieldOrder string
	}{
		encoding:   "console",
		encodeTime: zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05.000000"),
		separator:  " ",
		fieldOrder: "N:TC:L:",
	}

	json = struct {
		encoding   string
		encodeTime zapcore.TimeEncoder
	}{
		encoding:   "json",
		encodeTime: zapcore.RFC3339NanoTimeEncoder,
	}

	// config is the default logging config used for the root logger
	// and all subsequent logger instances. The log encoding defaults to console.
	config = zap.Config{
		OutputPaths:      []string{defaultDestination},
		ErrorOutputPaths: []string{defaultDestination},
		Encoding:         console.encoding,
		EncoderConfig: zapcore.EncoderConfig{
			// Keys can be anything except the empty string.
			TimeKey:    "time",
			LevelKey:   "level",
			NameKey:    "name",
			CallerKey:  "caller",
			MessageKey: "msg",
			LineEnding: zapcore.DefaultLineEnding,
			EncodeLevel: func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(zapLevelPrefix[l])
			},
			EncodeTime:     console.encodeTime,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller: func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
				fn := caller.File
				if idx := strings.LastIndex(caller.File, "/"); idx != -1 {
					fn = fn[idx+1:]
				}
				enc.AppendString(fn + ":" + strconv.Itoa(caller.Line))
			},
			ConsoleSeparator:  console.separator,
			ConsoleFieldOrder: console.fieldOrder,
		},
	}

	//defaultLevel is the default log level
	defaultLevel = InfoLevel

	//validLevels is a map of all valid level severities to their name
	validLevels = map[int32]string{
		PanicLevel:     "Panic",
		FatalLevel:     "Fatal",
		ErrorLevel:     "Error",
		WarnLevel:      "Warn",
		InternalLevel:  "Internal",
		InfoLevel:      "Info",
		DebugLevel:     "Debug",
		InitRetryLevel: "InitRetry",
		TraceLevel:     "Trace",
	}

	levelToZapLevel = map[int32]zapcore.Level{
		PanicLevel:    zapcore.PanicLevel,
		FatalLevel:    zapcore.FatalLevel,
		ErrorLevel:    zapcore.ErrorLevel,
		WarnLevel:     zapcore.WarnLevel,
		InternalLevel: zapcore.InfoLevel,
		InfoLevel:     zapcore.InfoLevel,
		DebugLevel:    zapcore.DebugLevel,
	}

	// We manually specify this LUT and do *not* populate
	// it by iterating over levelToZapLevel as both InternalLevel
	// and InfoLevel map to zapcore.InfoLevel. Due to golang's
	// random iteration order, the reverse lookup would become non-deterministic.
	zapLevelToLevel = map[zapcore.Level]int32{
		zapcore.PanicLevel: PanicLevel,
		zapcore.FatalLevel: FatalLevel,
		zapcore.ErrorLevel: ErrorLevel,
		zapcore.WarnLevel:  WarnLevel,
		zapcore.InfoLevel:  InfoLevel,
		zapcore.DebugLevel: DebugLevel,
	}

	zapLevelPrefix = map[zapcore.Level]string{
		zapcore.PanicLevel: "Panic",
		zapcore.FatalLevel: "Fatal",
		zapcore.ErrorLevel: "Error",
		zapcore.WarnLevel:  "Warn",
		zapcore.InfoLevel:  "Info",
		zapcore.DebugLevel: "Debug",
	}

	// validLabels maps (lowercase) strings to their respective log level/severity. It should only be used for lookups,
	// as the keys do not refer to the label names as they should be printed.
	validLabels = func() map[string]int32 {
		m := make(map[string]int32, len(validLevels))
		for k, v := range validLevels {
			m[strings.ToLower(v)] = k
		}
		return m
	}()

	// SortedLevels is a slice of log levels/severities, sorted in ascending order of severity.
	sortedLevels = func() []int32 {
		severities := make([]int32, 0, len(validLevels))
		for severity := range validLevels {
			severities = append(severities, severity)
		}
		sort.Slice(severities, func(i, j int) bool {
			return severities[i] < severities[j]
		})
		return severities
	}()

	//rootLogger is the convenience logger used when module specific loggers are not specified
	rootLogger *Logger

	// thisModuleLogger is the logger for logging in this module.
	thisModuleLogger *Logger
)

func init() {
	initLevel := os.Getenv("LOGLEVEL")
	value, ok := LevelForLabel(initLevel)
	if ok {
		SetGlobalLogLevel(value)
	}

	zapLevel := levelToZapLevelOrDefault(value, zapcore.InfoLevel)

	switch le := os.Getenv("LOGENCODING"); le {
	case "", console.encoding:
		config.Encoding = console.encoding
		config.EncoderConfig.EncodeTime = console.encodeTime
	case json.encoding:
		config.Encoding = json.encoding
		config.EncoderConfig.EncodeTime = json.encodeTime
	default:
		panic(fmt.Sprintf("unknown log encoding %s", le))
	}

	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// To the alert reader: While we could theoretically create a zapcore.Core instance and use
	// the logFile to create a MultiSyncWriter, we stick with using the config-based approach
	// such that we can easily propagate changes to log levels.
	if logFile, err := os.OpenFile(LoggingPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666); err == nil {
		defer func() {
			_ = logFile.Close()
		}()
		config.OutputPaths = append(
			config.OutputPaths, LoggingPath,
		)
	}

	if buildinfo.ReleaseBuild {
		config.DisableStacktrace = true
		config.Sampling = &zap.SamplingConfig{
			// The default sampling config assumes an interval of 1s.
			Initial: int(math.Max(1, float64(maxLogLineQuotaPerInterval/logLineQuotaIntervalSecs))),
			// Do not try to distill a representative sample and instead drop log messages.
			Thereafter: 1,
		}
	} else {
		// Configures logging at the DPanic log-level to panic.
		config.Development = true
	}

	// If !ok, defer printing a warning message until we've created a logger for this module. This has to wait, since
	// we want to be able to create it with the log level set above.
	thisModule := getCallingModule(0)
	if thisModule == "" {
		thisModule = "pkg/logging"
	}

	var defaultLevelsByModuleParsingErrs []error
	defaultLevelsByModule, defaultLevelsByModuleParsingErrs := parseDefaultModuleLevels(os.Getenv("MODULE_LOGLEVELS"))

	// Use direct calls to createLogger in this function, as New/NewOrGet/CurrentModule().Logger() refer to thisModuleLogger.
	thisModuleLogger = createLogger(ModuleForName(thisModule))
	if !ok && initLevel != "" {
		thisModuleLogger.Warnf("Invalid LOGLEVEL value '%s', defaulting to %s", initLevel, LabelForLevelOrInvalid(defaultLevel))
	}

	if len(defaultLevelsByModuleParsingErrs) > 0 {
		thisModuleLogger.Warn("Malformed entries in MODULE_LOGLEVELS string:")
		for _, err := range defaultLevelsByModuleParsingErrs {
			thisModuleLogger.Warnf("  %v", err)
		}
	} else {
		for k, v := range defaultLevelsByModule {
			modules.getOrAddModule(k).SetLogLevel(v)
		}
	}

	rootLogger = createLogger(ModuleForName("root logger"))
}

// SetGlobalLogLevel sets the log level on all loggers for all modules.
func SetGlobalLogLevel(level int32) {
	l, known := levelToZapLevel[level]
	if !known {
		if thisModuleLogger != nil {
			thisModuleLogger.Debugf("Ignoring unknown log level: %d", level)
		}
		return
	}

	atomic.StoreInt32(&defaultLevel, level)

	if thisModuleLogger != nil {
		thisModuleLogger.Debugf("Set log level to: %s", l)
	}

	config.Level.SetLevel(l)
	ForEachModule(func(name string, m *Module) {
		m.SetLogLevel(level)
	}, SelectAll)
}

// GetGlobalLogLevel returns the global log level (it is still possible that module loggers log at a different level).
func GetGlobalLogLevel() int32 {
	return atomic.LoadInt32(&defaultLevel)
}

// LoggerForModule returns a logger for the current module.
func LoggerForModule() *Logger {
	return currentModule(3).Logger()
}

//convenience methods log apply to root logger

// Log implements logging.Logger interface.
func Log(level int32, args ...interface{}) { rootLogger.Log(level, args...) }

// Logf implements logging.Logger interface.
func Logf(level int32, template string, args ...interface{}) {
	rootLogger.Logf(level, template, args...)
}

// Debug implements logging.Logger interface.
func Debug(args ...interface{}) { rootLogger.Debug(args...) }

// Debugf implements logging.Logger interface.
func Debugf(format string, args ...interface{}) { rootLogger.Debugf(format, args...) }

// Error implements logging.Logger interface.
func Error(args ...interface{}) { rootLogger.Error(args...) }

// Errorf implements logging.Logger interface.
func Errorf(format string, args ...interface{}) { rootLogger.Errorf(format, args...) }

// Fatal implements logging.Logger interface.
func Fatal(args ...interface{}) { rootLogger.Fatal(args...) }

// Fatalf implements logging.Logger interface.
func Fatalf(format string, args ...interface{}) { rootLogger.Fatalf(format, args...) }

// Fatalln implements logging.Logger interface.
func Fatalln(args ...interface{}) { rootLogger.Fatal(args...) }

// Info implements logging.Logger interface.
func Info(args ...interface{}) { rootLogger.Info(args...) }

// Infof implements logging.Logger interface.
func Infof(format string, args ...interface{}) { rootLogger.Infof(format, args...) }

// Panic implements logging.Logger interface.
func Panic(args ...interface{}) { rootLogger.Panic(args...) }

// Panicf implements logging.Logger interface.
func Panicf(format string, args ...interface{}) { rootLogger.Panicf(format, args...) }

// Panicln implements logging.Logger interface.
func Panicln(args ...interface{}) { rootLogger.Panic(args...) }

// Print implements logging.Logger interface.
func Print(args ...interface{}) { rootLogger.Info(args...) }

// Printf implements logging.Logger interface.
func Printf(format string, args ...interface{}) { rootLogger.Infof(format, args...) }

// Println implements logging.Logger interface.
func Println(args ...interface{}) { rootLogger.Info(args...) }

// Warn implements logging.Logger interface.
func Warn(args ...interface{}) { rootLogger.Warn(args...) }

// Warnf implements logging.Logger interface.
func Warnf(format string, args ...interface{}) { rootLogger.Warnf(format, args...) }

// LabelForLevel takes a numeric log level and returns its name. If the level has no associated name, a zero-valued
// string is returned, and the bool return value will be false.
func LabelForLevel(level int32) (string, bool) {
	name, ok := validLevels[level]
	return name, ok
}

// LabelForLevelOrInvalid returns the label for the given log level. If the level is unknown, "Invalid" is returned.
func LabelForLevelOrInvalid(level int32) (name string) {
	name, ok := LabelForLevel(level)
	if !ok {
		name = "Invalid"
	}
	return
}

// LevelForLabel returns the severity level for a label, if the label name is known. Otherwise, a zero-valued level is
// returned, and the bool return value will be false.
func LevelForLabel(label string) (int32, bool) {
	level, ok := validLabels[strings.ToLower(label)]
	return level, ok
}

// SortedLevels returns a slice containing all levels, in ascending order of severity.
func SortedLevels() []int32 {
	// Create a copy of the original slice to prevent the caller from modifying logging internals.
	result := make([]int32, len(sortedLevels))
	copy(result, sortedLevels)
	return result
}

func levelToZapLevelOrDefault(level int32, defaultLevel zapcore.Level) zapcore.Level {
	if l, known := levelToZapLevel[level]; known {
		return l
	}
	return defaultLevel
}

// createLogger creates (but does not register) a new logger instance.
func createLogger(module *Module) *Logger {
	lc := config
	lc.Level = zap.NewAtomicLevelAt(module.logLevel.Level())

	logger, err := lc.Build(zap.AddCallerSkip(0))
	if err != nil {
		panic(errors.Wrap(err, "failed to instantiate logger"))
	}

	result := &Logger{
		SugaredLogger: logger.Named(module.name).Sugar(),
		module:        module,
	}

	runtime.SetFinalizer(result, (*Logger).finalize)

	return result
}

func parseDefaultModuleLevels(str string) (map[string]int32, []error) {
	var errs []error
	entries := strings.Split(str, ",")
	result := make(map[string]int32, len(entries))
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			errs = append(errs, errors.Errorf("malformed entry %q, expecting form <module>=<level>", e))
			continue
		}
		module := strings.TrimSpace(parts[0])
		defaultLevelStr := strings.TrimSpace(parts[1])

		level, ok := LevelForLabel(defaultLevelStr)
		if !ok {
			errs = append(errs, errors.Errorf("malformed default level %q for module %s", defaultLevelStr, module))
			continue
		}
		result[module] = level
	}

	return result, errs
}
