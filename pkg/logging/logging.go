// Package logging provides the logger used in StackRox Go programs.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/stackrox/rox/pkg/sync"
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

	// Our project prefix. For all subpackages of this, we strip this prefix.
	projectPrefix = "github.com/stackrox/rox"

	// The common log file so we can export it
	loggingPath = "/var/log/stackrox/log.txt"
)

var (
	// defaultDestination is the default logging destination, which is currently os.Stdout
	defaultDestination io.Writer = os.Stdout
	//all registered loggers thus far
	allLoggers = []*weakLoggerRef(nil)
	// numGCdLoggers is the total numbers of loggers whose (former) weak references are present in allLoggers but have
	// since been GC'd.
	numGCdLoggers int32
	// numActiveLoggers is the number of loggers that are alive.
	numActiveLoggers int32
	// loggers by name
	loggersByName = make(map[string]*weakLoggerRef)
	//synchronized access to allLoggers and loggersByName
	knownLoggersLock = sync.Mutex{}

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
		severities := make(severitySlice, 0, len(validLevels))
		for severity := range validLevels {
			severities = append(severities, severity)
		}
		sort.Sort(severities)
		return []int32(severities)
	}()

	// levelPrefixes is a map from level severity to log prefix (of form "<label>  : ", such that all prefixes are of
	// the same length).
	levelPrefixes = func() map[int32]string {
		// TODO(mi): Re-enable this, if desired. It makes sure all logging prefixes are of the same length. However,
		// it currently doesn't make sense because our log message format :/
		//maxLen := 0
		//for _, name := range validLevels {
		//	l := len(name)
		//	if l > maxLen {
		//		maxLen = l
		//	}
		//}
		//format := fmt.Sprintf("%%-%ds: ", maxLen)
		result := make(map[int32]string, len(validLabels))
		for level, name := range validLevels {
			//result[level] = fmt.Sprintf(format, name)
			result[level] = fmt.Sprintf("%s: ", name)
		}
		return result
	}()

	// Given an absolute Go path, extract the package name from it. We assume that the package name is everything
	// starting from the last domain name-like path component (i.e., containing a dot), until (and excluding) the last
	// '/'.
	// Special case: go test, when run with coverage enabled, will change the location of the library fails to something
	// like "pkg/logging/_test/_obj_test", and since Go was too good to have a preprocessor with #file/#line statements,
	// there's no way around that. Rob Pike offers this solution in all seriousness: "If a test fails, run it again
	// without coverage enabled. Problem solved.". Since *something* about this doesn't feel like a proper solution, we
	// instead heuristically strip components starting with an underscore and ending in '_test' from the end of the path
	// sequence.
	fileToPackageRE = regexp.MustCompile(`^(?:.*/)?([^/]+\.[^/.]+(?:/[^/.]+)*?)/(?:(?:_[^/]*)?_test/)*[^/]+\.go$`)

	//rootLogger is the convenience logger used when module specific loggers are not specified
	rootLogger *Logger

	// thisModuleLogger is the logger for logging in this module.
	thisModuleLogger *Logger
)

type severitySlice []int32

func (s severitySlice) Len() int           { return len(s) }
func (s severitySlice) Less(i, j int) bool { return s[i] < s[j] }
func (s severitySlice) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

func init() {
	initLevel := os.Getenv("LOGLEVEL")
	value, ok := LevelForLabel(initLevel)
	if ok {
		SetGlobalLogLevel(value)
	}
	// If !ok, defer printing a warning message until we've created a logger for this module. This has to wait, since
	// we want to be able to create it with the log level set above.

	thisModule := getCallingModule(0)
	if thisModule == "" {
		thisModule = "pkg/logging"
	}

	// Use direct calls to createLogger in this function, as New/NewOrGet/LoggerForModule refer to thisModuleLogger.
	thisModuleLogger = createLogger(thisModule, 0)
	if !ok && initLevel != "" {
		thisModuleLogger.Warnf("Invalid LOGLEVEL value '%s', defaulting to %s", initLevel, LabelForLevelOrInvalid(defaultLevel))
	}

	logFile, err := os.OpenFile(loggingPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err == nil {
		defaultDestination = io.MultiWriter(os.Stdout, logFile)
	}

	rootLogger = createLogger("root logger", 0)
	// rootLogger is only accessed via module-level functions, hence we have one more caller than usual.
	rootLogger.stackTraceLevel++

	// Nothing can access these fields before init() is complete, hence we do not have to worry about locking or
	// clashes.
	loggersByName[rootLogger.module] = rootLogger.selfRef()
	loggersByName[thisModuleLogger.module] = thisModuleLogger.selfRef()
	allLoggers = append(allLoggers, rootLogger.selfRef(), thisModuleLogger.selfRef())
	atomic.StoreInt32(&numActiveLoggers, 2)
}

// forEachLogger invokes the given function on each active logger instance, and performs a compaction simultaneously.
func forEachLogger(fn func(*Logger)) {
	currI := 0
	for i, ref := range allLoggers {
		l := ref.get()
		if l != nil {
			fn(l)
			if i != currI {
				allLoggers[currI] = ref
			}
			currI++
		}
	}

	// Check if any weak references have been cleaned up, and if so, compact the loggersByName map as well.
	if currI < len(allLoggers) {
		allLoggers = allLoggers[:currI]
		compactLoggersByName()
	}
}

// compactLoggersIfRequired performs a compaction on the allLoggers slice (and the loggersByName map) if the number of
// GC'd weak references in allLoggers exceeds 25% of its size.
// Note: the lock must be held for this operation.
func compactLoggersIfRequired() {
	oldNumLoggers := len(allLoggers)
	if atomic.LoadInt32(&numGCdLoggers) < int32(oldNumLoggers/4) {
		return
	}
	forEachLogger(func(*Logger) {})
	atomic.AddInt32(&numGCdLoggers, -int32(len(allLoggers)-oldNumLoggers))
	compactLoggersByName()
}

// compactLoggersByName iterates through the loggersByName map, and deletes all entries corresponding to weak references
// that have been GC'd.
// Note: the lock must be held for this operation.
func compactLoggersByName() {
	for k, v := range loggersByName {
		if v.get() == nil {
			delete(loggersByName, k)
		}
	}
}

// ForEachLogger invokes the given function for each active logger.
func ForEachLogger(fn func(*Logger)) {
	knownLoggersLock.Lock()
	defer knownLoggersLock.Unlock()

	forEachLogger(fn)
}

// GetAllLoggers retrieves a slice containing all active loggers.
func GetAllLoggers() []*Logger {
	knownLoggersLock.Lock()
	defer knownLoggersLock.Unlock()
	result := make([]*Logger, 0, len(allLoggers))

	forEachLogger(func(l *Logger) {
		result = append(result, l)
	})
	return result
}

// SetGlobalLogLevel sets the log level on all loggers for all modules.
func SetGlobalLogLevel(level int32) {
	atomic.StoreInt32(&defaultLevel, level)

	if thisModuleLogger != nil {
		thisModuleLogger.Debugf("Set log level to: %d", level)
	}

	ForEachLogger(func(l *Logger) {
		l.SetLogLevel(level)
	})
}

// GetGlobalLogLevel returns the global log level (it is still possible that module loggers log at a different level).
func GetGlobalLogLevel() int32 {
	return atomic.LoadInt32(&defaultLevel)
}

// Logger wraps default go log implementation to allow log levels
type Logger struct {
	internal        *log.Logger
	module          string
	logLevel        int32
	fields          Fields
	stackTraceLevel int
	creationSite    string
	// A weak reference to the logger.
	weakSelfRef *weakLoggerRef
}

// A special value to indicate that access to the pointer is currently locked by another thread. The current thread
// must spin until the pointer holds a different value.
var tempUnavailable = uintptr(1)

// A weak reference to a logger. Allows us to maintain a global list of loggers by name, without having them linger on
// forever.
// For more information on weak references, see
// http://www.programmr.com/blogs/what-every-java-developer-should-know-strong-and-weak-references
type weakLoggerRef struct {
	// Note: The first two fields are used with atomic 64-bit operations. On i386, atomic 64-bit operations crash
	// if the data is not 64-bit aligned. Go guarantees that the first field in an allocated struct is
	// 64-bit aligned, so these have to appear first.

	// These fields are protected by the spinlock on ptr
	numGets         uint64
	finalizeNumGets uint64

	// This fields either holds the pointer to the logger, nil (if the logger has been garbage collected), or
	// tempUnavailable to indicate the value is spin-locked.
	ptr uintptr
}

// get converts a weak reference into a strong one.
func (r *weakLoggerRef) get() *Logger {
	if r == nil {
		return nil
	}
	// Spinlock the pointer.
	ptr := atomic.SwapUintptr(&r.ptr, tempUnavailable)
	for ptr == tempUnavailable {
		runtime.Gosched() // Yield control to a different goroutine.
		ptr = atomic.SwapUintptr(&r.ptr, tempUnavailable)
	}
	// Sigh.. the standard go vet complains about a possible misuse of unsafe.Pointer. uintptr is not supposed to be
	// stored (and then converted back to an unsafe.Pointer) due to garbage collection issues. However, fooling garbage
	// collection is *exactly* our aim, so the warning is meaningless.
	// To see why this is safe, consider that all paths through the finalizer (see below) either re-enqueue the
	// finalizer, or store a nil pointer in r.ptr upon return (and while holding the lock). Hence, no garbage collection
	// can have succeeded if this yields a non-nil pointer.
	l := (*Logger)(unsafe.Pointer(ptr)) // NOVET
	// A finalizer call might have started just before the above line. While holding the spinlock, increment the numGets
	// counter to make sure the weak reference doesn't get cleared.
	atomic.AddUint64(&r.numGets, 1)
	// Release the spinlock after we've obtained a strong reference and incremented the numGets counter.
	atomic.StoreUintptr(&r.ptr, ptr)
	return l
}

// Finalization of loggers
// Doing proper finalization is complicated, as we have to watch out for the following race condition:
// - finalizer for logger l gets called, context switch happens immediately after entering the finalizer, before
//   anything else.
// - Other goroutine calls get() on the weak reference. Since the finalizer has done nothing at this point, there is
//   no indication that this ref was about to be gc'd. A strong reference hence will be returned.
// - Finalizer resumes. There is no way of telling that a strong reference was just acquired, or rather, the situation
//   the finalizer sees at this point is indistinguishable from a situation where the get() occurred earlier and the
//   strong reference is no longer alive.
//
// Since it's impossible to find out if a get() call occurred *just before* a finalizer, we instead require two
// finalizer runs, which allows us to check that no get() call occurred *between* those two runs.
// To check this, we keep track of the number of get() invocations, and the value we observed in the last finalizer run.
// Only if this number is unchanged between two invocations (checking this of course has to be done while holding the
// spinlock), we clear the weak reference and allow garbage collection. Otherwise, we re-enqueue the finalizer, letting
// the runtime re-invoke it when all strong references have again disappeared.

// finalizeLogger implements the finalizer that is responsible for atomically clearing the weak reference.
func finalizeLogger(l *Logger) {
	weakRef := l.weakSelfRef // non-nil -- we only register the finalizer after setting this.
	ptr := atomic.SwapUintptr(&weakRef.ptr, tempUnavailable)
	if ptr == tempUnavailable {
		// Pointer is locked - somebody is calling get() right now. Don't even bother doing anything else, just
		// re-enqueue.
		// Note: we still update the finalizeNumGets. We "risk" reading a value that will soon be overwritten, which
		// doesn't really matter. On the other hand, if we don't update the value here, the finalizer may have to run
		// three times, as at this point we may already past the last get() invocation.
		atomic.StoreUint64(&weakRef.finalizeNumGets, atomic.LoadUint64(&weakRef.numGets))
		runtime.SetFinalizer(l, finalizeLogger)
		return
	}

	numGets := atomic.LoadUint64(&weakRef.numGets)
	if atomic.SwapUint64(&weakRef.finalizeNumGets, numGets) == numGets {
		// Finalization successful - no new get() invocations since last finalizer invocation
		atomic.StoreUintptr(&weakRef.ptr, uintptr(unsafe.Pointer(nil))) // Unlock
		atomic.AddInt32(&numGCdLoggers, 1)
		atomic.AddInt32(&numActiveLoggers, -1)
		return
	}

	// Finalization unsuccessful - there have been get() invocations since last finalizer invocation, so re-enqueue.
	// Note that we also updated the finalizeNumGets in the swap() call above.
	atomic.StoreUintptr(&weakRef.ptr, ptr) // unlock the pointer
	// Re-enqueue the finalizer.
	runtime.SetFinalizer(l, finalizeLogger)
}

// callerFileToPackage takes the path of the source file of the caller (<skip> frames up the call stack), and returns
// the corresponding Go package. If no package could be determined, the empty string is returned.
func callerFileToPackage(skip int) string {
	_, callerFile, _, ok := runtime.Caller(1 + skip)
	if !ok {
		return ""
	}
	if !strings.HasPrefix(callerFile, "/") {
		// In bazel builds, callerFile will be relative to the project root.
		callerFile = fmt.Sprintf("%s/%s", projectPrefix, callerFile)
	}
	submatches := fileToPackageRE.FindStringSubmatch(callerFile)
	if len(submatches) != 2 {
		return ""
	}
	return submatches[1]
}

// getCallingModule returns the short name of the module calling this function, skipping <skip> stack frames in the
// call stack.
// The short name is determined as follows:
// - If the package is a subpackage of "<projectPrefix>/pkg", the short name is the result of stripping
//   "<projectPrefix>/".
// - Otherwise, if the package is a subpackage of "<projectPrefix>/", the short name is the result of stripping
//   "<projectPrefix>/" and the first component of the remaining path name, including any trailing slashes. If the
//   resulting string is the empty string, the short name is "main".
// - Otherwise, the short name is the full package name.
func getCallingModule(skip int) string {
	callingPackage := callerFileToPackage(1 + skip)
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

// createLogger creates (but does not register) a new logger instance.
func createLogger(module string, skip int) *Logger {
	var creationSite string
	_, creationFile, creationLine, ok := runtime.Caller(1 + skip)
	if !ok {
		creationSite = "<unknown>"
	} else {
		creationSite = fmt.Sprintf("%s:%d", creationFile, creationLine)
	}

	baseLogger := log.New(defaultDestination, module+": ", log.Lshortfile|log.Ldate|log.Lmicroseconds|log.LUTC)
	newLogger := &Logger{
		internal:        baseLogger,
		module:          module,
		logLevel:        GetGlobalLogLevel(),
		stackTraceLevel: 3, // Info/Debug/...[f] -> log[f] -> log.Output
		creationSite:    creationSite,
	}

	return newLogger
}

// selfRef obtains a weak reference to this logger.
func (l *Logger) selfRef() *weakLoggerRef {
	if l.weakSelfRef == nil {
		l.weakSelfRef = &weakLoggerRef{
			ptr:     uintptr(unsafe.Pointer(l)),
			numGets: 1,
		}
		runtime.SetFinalizer(l, finalizeLogger)
	}
	return l.weakSelfRef
}

// LoggerForModule returns a logger instance for the module from which it is instantiated.
func LoggerForModule() *Logger {
	module := getCallingModule(1)
	if module == "" {
		rootLogger.Errorf("Could not determine calling module! Returning <unknown>")
		module = "<unknown>"
	}
	return newOrGet(module)
}

// NewOrGet returns a logger instance for the given module. If there isn't such a logger yet, it will be created.
func NewOrGet(module string) *Logger {
	return newOrGet(module)
}

func newOrGet(module string) *Logger {
	knownLoggersLock.Lock()
	defer knownLoggersLock.Unlock()

	logger := loggersByName[module].get()
	if logger != nil {
		return logger
	}
	logger = createLogger(module, 2)
	// Compact loggers before adding new ones if the number of GC'd loggers is too high.
	compactLoggersIfRequired()
	loggersByName[module] = logger.selfRef()
	allLoggers = append(allLoggers, logger.selfRef())
	atomic.AddInt32(&numActiveLoggers, 1)
	return logger
}

// New returns a new logger instance for the given module. If there already exists a logger for the same module, a
// warning message will be printed.
func New(module string) *Logger {
	newLogger := createLogger(module, 1)

	knownLoggersLock.Lock()
	defer knownLoggersLock.Unlock()
	// Compact loggers before adding new ones if the number of GC'd loggers is too high.
	compactLoggersIfRequired()
	allLoggers = append(allLoggers, newLogger.selfRef())
	atomic.AddInt32(&numActiveLoggers, 1)
	if existingLogger := loggersByName[module].get(); existingLogger != nil {
		thisModuleLogger.Warnf("Duplicate logger for module '%s' created at %s; existing logger created at %s",
			module, newLogger.creationSite, existingLogger.creationSite)
	} else {
		loggersByName[module] = newLogger.selfRef()
	}
	return newLogger
}

// GetLoggerByModule returns the logger for the given module, or nil if no logger for that module was registered.
func GetLoggerByModule(module string) *Logger {
	knownLoggersLock.Lock()
	defer knownLoggersLock.Unlock()

	return loggersByName[module].get()
}

// GetLoggersByModule returns a list of loggers for the given modules. Any modules for which no loggers are registered
// will be returned in the unknownModules return value.
func GetLoggersByModule(modules []string) (loggers []*Logger, unknownModules []string) {
	knownLoggersLock.Lock()
	defer knownLoggersLock.Unlock()

	loggers = make([]*Logger, 0, len(modules))
	for _, module := range modules {
		if l := loggersByName[module].get(); l != nil {
			loggers = append(loggers, l)
		} else {
			unknownModules = append(unknownModules, module)
		}
	}
	return
}

// GetModule retrieves the module of a logger.
func (l *Logger) GetModule() string {
	return l.module
}

//SetOutput redirects log messages to a writer other than std out
func (l *Logger) SetOutput(w io.Writer) {
	l.internal.SetOutput(w)
}

// LogLevel returns the int value for the current log level.
func (l *Logger) LogLevel() int32 {
	return atomic.LoadInt32(&l.logLevel)
}

// SetLogLevel sets the log level with provided level.
func (l *Logger) SetLogLevel(level int32) {
	atomic.StoreInt32(&l.logLevel, level)
}

//GetLogLevel returns the log level in human readable string format
func (l *Logger) GetLogLevel() string {
	return LabelForLevelOrInvalid(l.LogLevel())
}

func (l *Logger) log(level int32, args ...interface{}) {
	if l.LogLevel() <= level {
		_ = l.internal.Output(l.stackTraceLevel, levelPrefixes[level]+fmt.Sprint(args...)+l.fields.String())
	}
}

func (l *Logger) logf(level int32, format string, args ...interface{}) {
	if l.LogLevel() <= level {
		_ = l.internal.Output(l.stackTraceLevel, levelPrefixes[level]+fmt.Sprintf(format, args...)+l.fields.String())
	}
}

//Trace provide super low level detail
func (l *Logger) Trace(args ...interface{}) {
	l.log(TraceLevel, args...)
}

//Tracef provide super low level detail
func (l *Logger) Tracef(format string, args ...interface{}) {
	l.logf(TraceLevel, format, args...)
}

//Retry describes the inner contents of loops that retry an action several times
func (l *Logger) Retry(args ...interface{}) {
	l.log(InitRetryLevel, args...)
}

//Retryf describes the inner contents of loops that retry an action several times
func (l *Logger) Retryf(format string, args ...interface{}) {
	l.logf(InitRetryLevel, format, args...)
}

//Debug provides standard debug messages
func (l *Logger) Debug(args ...interface{}) {
	l.log(DebugLevel, args...)
}

//Debugf provides standard debug messages
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logf(DebugLevel, format, args...)
}

//Info displays information
func (l *Logger) Info(args ...interface{}) {
	l.log(InfoLevel, args...)
}

//Infof displays information
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logf(InfoLevel, format, args...)
}

//Warn displays a warning
func (l *Logger) Warn(args ...interface{}) {
	l.log(WarnLevel, args...)
}

//Warnf displays a warning
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logf(WarnLevel, format, args...)
}

//Error logs an error
func (l *Logger) Error(args ...interface{}) {
	l.log(ErrorLevel, args...)
}

//Errorf logs an error
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logf(ErrorLevel, format, args...)
}

//Fatal logs and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.log(FatalLevel, args...)
	os.Exit(1)
}

//Fatalf logs and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logf(FatalLevel, format, args...)
	os.Exit(1)
}

//Panic logs and throws a panic
func (l *Logger) Panic(args ...interface{}) {
	l.log(PanicLevel, args...)
	panic(fmt.Sprint(args...))
}

//Panicf logs and throws a panic
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.logf(PanicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

//Log logs the message regardless of loglevel
func (l *Logger) Log(args ...interface{}) {
	_ = l.internal.Output(l.stackTraceLevel, fmt.Sprint(args...)+l.fields.String())
}

//Logf logs the message regardless of loglevel
func (l *Logger) Logf(format string, args ...interface{}) {
	_ = l.internal.Output(l.stackTraceLevel, fmt.Sprintf(format, args...)+l.fields.String())
}

//WithFields provides custom formatted output
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		internal:        l.internal,
		logLevel:        l.LogLevel(),
		stackTraceLevel: l.stackTraceLevel,
		fields:          l.fields.update(fields),
	}
}

//WithError is a convenience wrapper for WithFields and a single error type
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(Fields{"error": err.Error()})
}

//convenience methods log apply to root logger

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
func Fatalln(args ...interface{}) { rootLogger.internal.Fatalln(args...) }

// Info implements logging.Logger interface.
func Info(args ...interface{}) { rootLogger.Info(args...) }

// Infof implements logging.Logger interface.
func Infof(format string, args ...interface{}) { rootLogger.Infof(format, args...) }

// Panic implements logging.Logger interface.
func Panic(args ...interface{}) { rootLogger.Panic(args...) }

// Panicf implements logging.Logger interface.
func Panicf(format string, args ...interface{}) { rootLogger.Panicf(format, args...) }

// Panicln implements logging.Logger interface.
func Panicln(args ...interface{}) { rootLogger.internal.Panicln(args...) }

// Print implements logging.Logger interface.
func Print(args ...interface{}) { rootLogger.internal.Print(args...) }

// Printf implements logging.Logger interface.
func Printf(format string, args ...interface{}) { rootLogger.internal.Printf(format, args...) }

// Println implements logging.Logger interface.
func Println(args ...interface{}) { rootLogger.internal.Println(args...) }

// Warn implements logging.Logger interface.
func Warn(args ...interface{}) { rootLogger.Warn(args...) }

// Warnf implements logging.Logger interface.
func Warnf(format string, args ...interface{}) { rootLogger.Warnf(format, args...) }

//Log logs the message regardless of log level
func Log(args ...interface{}) { rootLogger.Log(args...) }

//Logf logs the message regardless of log level
func Logf(format string, args ...interface{}) { rootLogger.Logf(format, args...) }

//WithFields provides custom formatted output
func WithFields(fields map[string]interface{}) *Logger { return rootLogger.WithFields(fields) }

//WithError is a convenience wrapper for WithFields and a single error type
func WithError(err error) *Logger { return rootLogger.WithError(err) }

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

func getNumActiveLoggers() int {
	return int(atomic.LoadInt32(&numActiveLoggers))
}
