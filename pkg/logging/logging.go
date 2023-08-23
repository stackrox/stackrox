// Package logging provides the logger used in StackRox Go programs.
//
// This package supports runtime configuration via the following
// environment variables:
//
// 1. LOGLEVEL supporting the following values (case insensitive), order is indicative of importance:
//
//   - fatal
//   - panic
//   - error
//   - warn
//   - info
//   - debug
//
// 2. LOGENCODING supporting the following values:
//
//   - json
//   - console
//
// 3. MODULE_LOGLEVELS supporting ,-separated module=level pairs, e.g.: grpc=debug,kubernetes=warn
//
// 4. MAX_LOG_LINE_QUOTA in the format max/duration_in_seconds, e.g.: 100/10
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

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/notifications"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// defaultDestination is the default logging destination,
	// which is currently os.Stderr.
	defaultDestination = "stderr"

	// Our project prefix. For all subpackages of this, we strip this prefix.
	projectPrefix = "github.com/stackrox/rox"

	// LoggingPath is the common log file so we can export it.
	LoggingPath = "/var/log/stackrox/log.txt"

	// defaultLevel is the default log level.
	defaultLevel = zapcore.InfoLevel

	// Aliases for zapcore.* log levels to abstract away zapcore-based
	// implementation and not to require clients to import zapcore lib
	// explicitly.

	// WarnLevel log level
	WarnLevel = zapcore.WarnLevel
)

// Options for the logger.
type Options struct {
	notificationStream    notifications.Stream
	notificationConverter notifications.LogConverter
}

// OptionsFunc allows setting log options for a logger.
type OptionsFunc = func(option *Options)

var (
	console = struct {
		encoding   string
		encodeTime zapcore.TimeEncoder
		separator  string
	}{
		encoding:   "console",
		encodeTime: zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05.000000"),
		separator:  " ",
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
				enc.AppendString(validLevels[l])
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
			ConsoleSeparator: console.separator,
		},
	}

	// validLevels is a map of all valid level severities to their name
	validLevels = map[zapcore.Level]string{
		zapcore.PanicLevel: "Panic",
		zapcore.FatalLevel: "Fatal",
		zapcore.ErrorLevel: "Error",
		zapcore.WarnLevel:  "Warn",
		zapcore.InfoLevel:  "Info",
		zapcore.DebugLevel: "Debug",
	}

	// validLabels maps (lowercase) strings to their respective log level/severity. It should only be used for lookups,
	// as the keys do not refer to the label names as they should be printed.
	validLabels = func() map[string]zapcore.Level {
		m := make(map[string]zapcore.Level, len(validLevels))
		for k, v := range validLevels {
			m[strings.ToLower(v)] = k
		}
		return m
	}()

	// sortedLevels is a slice of log levels/severities, sorted in ascending order of severity.
	sortedLevels = func() []zapcore.Level {
		severities := make([]zapcore.Level, 0, len(validLevels))
		for severity := range validLevels {
			severities = append(severities, severity)
		}
		sort.Slice(severities, func(i, j int) bool {
			return severities[i] < severities[j]
		})
		return severities
	}()

	// rootLogger is the convenience logger used when module specific loggers are not specified
	rootLogger Logger

	// thisModuleLogger is the logger for logging in this module.
	thisModuleLogger Logger
)

func init() {
	initLevelStr, initLevelValid := os.Getenv("LOGLEVEL"), false
	logLevel := defaultLevel
	if value, ok := LevelForLabel(initLevelStr); ok {
		logLevel = value
		initLevelValid = true
	}

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

	config.Level = zap.NewAtomicLevelAt(logLevel)

	// To the alert reader: While we could theoretically create a zapcore.Core instance and use
	// the logFile to create a MultiSyncWriter, we stick with using the config-based approach
	// such that we can easily propagate changes to log levels.
	addOutput(&config, LoggingPath)

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

	// Use direct calls to CreateLogger in this function, as New/NewOrGet/CurrentModule().Logger() refer to thisModuleLogger.
	thisModuleLogger = CreateLogger(ModuleForName(thisModule), 0)
	if !initLevelValid && initLevelStr != "" {
		thisModuleLogger.Warnf("Invalid LOGLEVEL value '%s', defaulting to %s", initLevelStr, LabelForLevelOrInvalid(logLevel))
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

	rootLogger = CreateLogger(ModuleForName("root logger"), 0)
}

func addOutput(config *zap.Config, path string) {
	for _, p := range config.OutputPaths {
		if p == path {
			return
		}
	}
	if logFile, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666); err == nil {
		defer func() {
			_ = logFile.Close()
		}()
		config.OutputPaths = append(
			config.OutputPaths, path,
		)
	}
}

// SetGlobalLogLevel sets the log level on all loggers for all modules.
func SetGlobalLogLevel(l zapcore.Level) {
	config.Level.SetLevel(l)
	ForEachModule(func(name string, m *Module) {
		m.SetLogLevel(l)
	}, SelectAll)

	// Don't log the log level change when switching to Panic or Fatal.
	if thisModuleLogger != nil && l <= zapcore.ErrorLevel {
		thisModuleLogger.Logf(l, "Log level is set to: %s", l)
	}
}

// GetGlobalLogLevel returns the global log level (it is still possible that module loggers log at a different level).
func GetGlobalLogLevel() zapcore.Level {
	return config.Level.Level()
}

// LoggerForModule returns a logger for the current module.
func LoggerForModule(opts ...OptionsFunc) Logger {
	return currentModule(3).Logger(opts...)
}

// convenience methods log apply to root logger

// Debug implements logging.Logger interface.
func Debug(args ...interface{}) { rootLogger.Debug(args...) }

// Debugf implements logging.Logger interface.
func Debugf(format string, args ...interface{}) { rootLogger.Debugf(format, args...) }

// Error implements logging.Logger interface.
func Error(args ...interface{}) { rootLogger.Error(args...) }

// Errorf implements logging.Logger interface.
func Errorf(format string, args ...interface{}) { rootLogger.Errorf(format, args...) }

// Fatalf implements logging.Logger interface.
func Fatalf(format string, args ...interface{}) { rootLogger.Fatalf(format, args...) }

// Info implements logging.Logger interface.
func Info(args ...interface{}) { rootLogger.Info(args...) }

// Infof implements logging.Logger interface.
func Infof(format string, args ...interface{}) { rootLogger.Infof(format, args...) }

// Panicf implements logging.Logger interface.
func Panicf(format string, args ...interface{}) { rootLogger.Panicf(format, args...) }

// Warn implements logging.Logger interface.
func Warn(args ...interface{}) { rootLogger.Warn(args...) }

// Warnf implements logging.Logger interface.
func Warnf(format string, args ...interface{}) { rootLogger.Warnf(format, args...) }

// LabelForLevel takes a zapcore.Level and returns its name. If the level has no associated name, a zero-valued
// string is returned, and the bool return value will be false.
func LabelForLevel(level zapcore.Level) (string, bool) {
	name, ok := validLevels[level]
	return name, ok
}

// LabelForLevelOrInvalid returns the label for the given log level. If the level is unknown, "Invalid" is returned.
func LabelForLevelOrInvalid(level zapcore.Level) (name string) {
	name, ok := LabelForLevel(level)
	if !ok {
		name = "Invalid"
	}
	return
}

// LevelForLabel returns the severity level for a label, if the label name is known. Otherwise, a zero-valued level is
// returned, and the bool return value will be false.
func LevelForLabel(label string) (zapcore.Level, bool) {
	level, ok := validLabels[strings.ToLower(label)]
	return level, ok
}

// SortedLevels returns a slice containing all levels, in ascending order of severity.
func SortedLevels() []zapcore.Level {
	// Create a copy of the original slice to prevent the caller from modifying logging internals.
	result := make([]zapcore.Level, len(sortedLevels))
	copy(result, sortedLevels)
	return result
}

// CreateLogger creates (but does not register) a new logger instance.
// Skip allows to specify how much layers of nested calls we will skip during logging.
func CreateLogger(module *Module, skip int, opts ...OptionsFunc) *LoggerImpl {
	lc := config
	// Need to increase the skip by 1 by default since we call the logger inline. Otherwise, the location of the caller
	// would also be set to this file.
	return createLoggerWithConfig(&lc, module, skip+1, opts...)
}

func createLoggerWithConfig(lc *zap.Config, module *Module, skip int, opts ...OptionsFunc) *LoggerImpl {
	lc.Level = module.logLevel

	logger, err := lc.Build(zap.AddCallerSkip(skip))
	if err != nil {
		panic(errors.Wrap(err, "failed to instantiate logger"))
	}

	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	result := &LoggerImpl{
		InnerLogger: logger.Named(module.name).Sugar(),
		module:      module,
		opts:        o,
	}

	runtime.SetFinalizer(result, (*LoggerImpl).finalize)

	return result
}

func parseDefaultModuleLevels(str string) (map[string]zapcore.Level, []error) {
	var errs []error
	entries := strings.Split(str, ",")
	result := make(map[string]zapcore.Level, len(entries))
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
