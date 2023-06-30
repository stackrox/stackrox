package logging

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/logging/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
)

const (
	burstSize     = 3
	cacheSize     = 500
	limiterLines  = 1
	limiterPeriod = 300 * time.Millisecond

	LoggingInterval = 120 * time.Millisecond
)

func TestRateLimitedLogger(t *testing.T) {
	suite.Run(t, new(rateLimitedLoggerTestSuite))
}

type rateLimitedLoggerTestSuite struct {
	suite.Suite

	mockLogger *mocks.MockLogger
	rlLogger   *RateLimitedLogger
}

func (s *rateLimitedLoggerTestSuite) SetupTest() {
	mockController := gomock.NewController(s.T())
	s.mockLogger = mocks.NewMockLogger(mockController)
	s.rlLogger = NewRateLimitLogger(s.mockLogger, cacheSize, limiterLines, limiterPeriod, burstSize)
}

func (s *rateLimitedLoggerTestSuite) TearDownTest() {
	s.mockLogger.EXPECT().Logf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	s.rlLogger.rateLimitedLogs.Purge()
	s.rlLogger.stop()
}

// TODO: ROX-17312: For all tests, test each function (log level) independently.

func (s *rateLimitedLoggerTestSuite) TestBaseFunctionError() {
	errorLog := "This is an error log"

	s.mockLogger.EXPECT().Error(errorLog, 1)
	s.rlLogger.Error(errorLog, 1)
}

func (s *rateLimitedLoggerTestSuite) TestBaseFunctionWarn() {
	warnLog := "This is a warn log"

	s.mockLogger.EXPECT().Warn(3, warnLog)
	s.rlLogger.Warn(3, warnLog)
}

func (s *rateLimitedLoggerTestSuite) TestBaseFunctionInfo() {
	infoLog := "This is an info log"

	s.mockLogger.EXPECT().Info(infoLog, 5, 7)
	s.rlLogger.Info(infoLog, 5, 7)
}

func (s *rateLimitedLoggerTestSuite) TestBaseFunctionDebug() {
	debugLog := "This is a debug log"

	s.mockLogger.EXPECT().Debug(9, 2, 4, 6, debugLog)
	s.rlLogger.Debug(9, 2, 4, 6, debugLog)
}

const (
	templateWithFields = "This is a template for %s logs with %d arguments to convert"
)

func (s *rateLimitedLoggerTestSuite) TestFormatFunctionErrorf() {
	errorTemplateWithoutField := "This is an error template without arg conversion."

	s.mockLogger.EXPECT().Errorf(errorTemplateWithoutField)
	s.rlLogger.Errorf(errorTemplateWithoutField)

	s.mockLogger.EXPECT().Errorf(templateWithFields, "error", 2)
	s.rlLogger.Errorf(templateWithFields, "error", 2)
}

func (s *rateLimitedLoggerTestSuite) TestFormatFunctionWarnf() {
	warnTemplateWithoutField := "This is a warn template without arg conversion."

	s.mockLogger.EXPECT().Warnf(warnTemplateWithoutField)
	s.rlLogger.Warnf(warnTemplateWithoutField)

	s.mockLogger.EXPECT().Warnf(templateWithFields, "warn", 2)
	s.rlLogger.Warnf(templateWithFields, "warn", 2)
}

func (s *rateLimitedLoggerTestSuite) TestFormatFunctionInfof() {
	infoTemplateWithoutField := "This is an info template without arg conversion."

	s.mockLogger.EXPECT().Infof(infoTemplateWithoutField)
	s.rlLogger.Infof(infoTemplateWithoutField)

	s.mockLogger.EXPECT().Infof(templateWithFields, "info", 2)
	s.rlLogger.Infof(templateWithFields, "info", 2)
}

func (s *rateLimitedLoggerTestSuite) TestFormatFunctionDebugf() {
	debugTemplateWithoutField := "This is a debug template without arg conversion."

	s.mockLogger.EXPECT().Debugf(debugTemplateWithoutField)
	s.rlLogger.Debugf(debugTemplateWithoutField)

	s.mockLogger.EXPECT().Debugf(templateWithFields, "debug", 2)
	s.rlLogger.Debugf(templateWithFields, "debug", 2)
}

func getLogCallerPrefix(lineOffset int) string {
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	// file = getTrimmedFilePath(file)
	file := "pkg/logging/rate_limited_logger_test.go"
	line += lineOffset
	return fmt.Sprintf("%s:%d - ", file, line)
}

func (s *rateLimitedLoggerTestSuite) validateRateLimitedLogCount(expectedLogCount int) {
	cacheKeys := s.rlLogger.rateLimitedLogs.Keys()
	for _, k := range cacheKeys {
		v, f := s.rlLogger.rateLimitedLogs.Peek(k)
		s.True(f)
		s.NotNil(v)
		expectedCount := atomic.Int32{}
		expectedCount.Swap(int32(expectedLogCount))
		if v != nil {
			s.Equal(expectedCount.Load(), v.count.Load())
		}
	}
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsErrorLBurst() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, "").Times(burstSize)

	for i := 0; i < 3*burstSize; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
	}

	s.validateRateLimitedLogCount(2 * burstSize)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsWarnLBurst() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, "").Times(burstSize)

	for i := 0; i < 3*burstSize; i++ {
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
	}

	s.validateRateLimitedLogCount(2 * burstSize)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsInfoLBurst() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, "").Times(burstSize)

	for i := 0; i < 3*burstSize; i++ {
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
	}

	s.validateRateLimitedLogCount(2 * burstSize)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsDebugLBurst() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, "").Times(burstSize)

	for i := 0; i < 3*burstSize; i++ {
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	s.validateRateLimitedLogCount(2 * burstSize)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsErrorLCoolDown() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, "").Times(2)

	logError := func() {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
	}

	for i := 0; i < 2; i++ {
		logError()
	}

	// TODO: ROX-17312: Mock timer, clock and synchronization of logs.
	time.Sleep(LoggingInterval)

	// Burst limit should allow one more trace
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, "").Times(1)

	for i := 0; i < 2; i++ {
		logError()
	}

	time.Sleep(LoggingInterval)

	// Burst limit should not allow any trace
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, "").Times(0)

	for i := 0; i < 2; i++ {
		logError()
	}

	time.Sleep(LoggingInterval)

	// Rate limiter should allow one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 4, 0.2, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, limiterSuffix).Times(1)

	for i := 0; i < 2; i++ {
		logError()
	}

	s.validateRateLimitedLogCount(1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsWarnLCoolDown() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, "").Times(2)

	logWarn := func() {
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
	}

	for i := 0; i < 2; i++ {
		logWarn()
	}

	// TODO: ROX-17312: Mock timer, clock and synchronization of logs.
	time.Sleep(LoggingInterval)

	// Burst limit should allow one more trace
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, "").Times(1)

	for i := 0; i < 2; i++ {
		logWarn()
	}

	time.Sleep(LoggingInterval)

	// Burst limit should not allow any trace
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, "").Times(0)

	for i := 0; i < 2; i++ {
		logWarn()
	}

	time.Sleep(LoggingInterval)

	// Rate limiter should allow one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 4, 0.2, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, limiterSuffix).Times(1)

	for i := 0; i < 2; i++ {
		logWarn()
	}

	s.validateRateLimitedLogCount(1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsInfoLCoolDown() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, "").Times(2)

	logInfo := func() {
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
	}

	for i := 0; i < 2; i++ {
		logInfo()
	}

	// TODO: ROX-17312: Mock timer, clock and synchronization of logs.
	time.Sleep(LoggingInterval)

	// Burst limit should allow one more trace
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, "").Times(1)

	for i := 0; i < 2; i++ {
		logInfo()
	}

	time.Sleep(LoggingInterval)

	// Burst limit should not allow any trace
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, "").Times(0)

	for i := 0; i < 2; i++ {
		logInfo()
	}

	time.Sleep(LoggingInterval)

	// Rate limiter should allow one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 4, 0.2, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, limiterSuffix).Times(1)

	for i := 0; i < 2; i++ {
		logInfo()
	}

	s.validateRateLimitedLogCount(1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsDebugLCoolDown() {
	limiter := "test limiter"

	prefix := getLogCallerPrefix(5)
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, "").Times(2)

	logDebug := func() {
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	for i := 0; i < 2; i++ {
		logDebug()
	}

	// TODO: ROX-17312: Mock timer, clock and synchronization of logs.
	time.Sleep(LoggingInterval)

	// Burst limit should allow one more trace
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, "").Times(1)

	for i := 0; i < 2; i++ {
		logDebug()
	}

	time.Sleep(LoggingInterval)

	// Burst limit should not allow any trace
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, "").Times(0)

	for i := 0; i < 2; i++ {
		logDebug()
	}

	time.Sleep(LoggingInterval)

	// Rate limiter should allow one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 4, 0.2, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, limiterSuffix).Times(1)

	for i := 0; i < 2; i++ {
		logDebug()
	}

	s.validateRateLimitedLogCount(1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsSameLimiterDifferentLogs() {
	template1 := "This is a log to be rate limited"
	template2 := "This is another log to be rate limited"
	limiter := "common limiter"
	prefix := getLogCallerPrefix(1)
	logInfo := func(info string) { s.rlLogger.InfoL(limiter, info) }

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, template1, "").Times(burstSize)

	for i := 0; i < 2*burstSize; i++ {
		logInfo(template1)
	}

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, template2, "").Times(burstSize)

	for i := 0; i < 2*burstSize; i++ {
		logInfo(template2)
	}

}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsCacheEviction() {
	evictionTemplate := "This is a log that will be evicted"
	limiter := "limiter"
	prefix1 := getLogCallerPrefix(1)
	logInfo := func(info string) { s.rlLogger.InfoL(limiter, info) }
	prefix2 := getLogCallerPrefix(1)
	logDebug := func(template string, arg int) { s.rlLogger.DebugL(limiter, template, arg) }

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix1, evictionTemplate, "").Times(burstSize)

	for i := 0; i < 2*burstSize; i++ {
		logInfo(evictionTemplate)
	}

	evictionSuffix := fmt.Sprintf(limitedLogSuffixFormat, burstSize, 0.0, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix1, evictionTemplate, evictionSuffix).Times(1)

	fillerTemplate := "There are now %d fillers in cache"
	for i := 0; i < cacheSize; i++ {
		expected := fmt.Sprintf(fillerTemplate, i+1)
		s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix2, expected, "").Times(1)
		logDebug(fillerTemplate, i+1)
	}
}

func checkLimiterFlushed(t *testing.T, logger *RateLimitedLogger) {
	cacheKeys := logger.rateLimitedLogs.Keys()
	for _, k := range cacheKeys {
		v, f := logger.rateLimitedLogs.Peek(k)
		assert.True(t, f)
		assert.NotNil(t, v)
		expectedCount := atomic.Int32{}
		expectedCount.Swap(0)
		if v != nil {
			assert.Equal(t, expectedCount.Load(), v.count.Load())
		}
	}
}

func TestRateLimitedFunctionsErrorLTimedFlush(t *testing.T) {
	mockController := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(mockController)

	limiter := "test limiter"

	// Issued traces
	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)

	rlLogger := NewRateLimitLogger(mockLogger, cacheSize, limiterLines, limiterPeriod, burstSize)

	prefix := getLogCallerPrefix(1)
	logError := func() { rlLogger.ErrorL(limiter, templateWithFields, "error", 2) }

	// First burst
	mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, "").Times(burstSize)

	// flush should send one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 2*burstSize, 0.9, limiter)
	mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, limiterSuffix).Times(1)

	// TODO: ROX-17312: Use timer mock and ad-hoc synchronization to avoid sleeping in tests.
	// Avoid concurrency with background logging loop
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 3*burstSize; i++ {
		logError()
	}

	time.Sleep(2 * time.Second)

	checkLimiterFlushed(t, rlLogger)
}

func TestRateLimitedFunctionsWarnLTimedFlush(t *testing.T) {
	mockController := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(mockController)
	limiter := "test limiter"

	// Issued traces
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)

	rlLogger := NewRateLimitLogger(mockLogger, cacheSize, limiterLines, limiterPeriod, burstSize)

	prefix := getLogCallerPrefix(1)
	logWarn := func() { rlLogger.WarnL(limiter, templateWithFields, "warn", 2) }

	// First burst
	mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, "").Times(burstSize)

	// flush should send one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 2*burstSize, 0.9, limiter)
	mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, limiterSuffix).Times(1)

	// TODO: ROX-17312: Use timer mock and ad-hoc synchronization to avoid sleeping in tests.
	// Avoid concurrency with background logging loop
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 3*burstSize; i++ {
		logWarn()
	}

	time.Sleep(2 * time.Second)

	checkLimiterFlushed(t, rlLogger)
}

func TestRateLimitedFunctionsInfoLTimedFlush(t *testing.T) {
	mockController := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(mockController)

	limiter := "test limiter"

	// Issued traces
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)

	rlLogger := NewRateLimitLogger(mockLogger, cacheSize, limiterLines, limiterPeriod, burstSize)

	prefix := getLogCallerPrefix(1)
	logInfo := func() { rlLogger.InfoL(limiter, templateWithFields, "info", 2) }

	// First burst
	mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, "").Times(burstSize)

	// flush should send one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 2*burstSize, 0.9, limiter)
	mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, limiterSuffix).Times(1)

	// TODO: ROX-17312: Use timer mock and ad-hoc synchronization to avoid sleeping in tests.
	// Avoid concurrency with background logging loop
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 3*burstSize; i++ {
		logInfo()
	}

	time.Sleep(2 * time.Second)

	checkLimiterFlushed(t, rlLogger)
}

func TestRateLimitedFunctionsDebugLTimedFlush(t *testing.T) {
	mockController := gomock.NewController(t)
	mockLogger := mocks.NewMockLogger(mockController)

	limiter := "test limiter"

	// Issued traces
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)

	rlLogger := NewRateLimitLogger(mockLogger, cacheSize, limiterLines, limiterPeriod, burstSize)

	prefix := getLogCallerPrefix(1)
	logDebug := func() { rlLogger.DebugL(limiter, templateWithFields, "debug", 2) }

	// First burst
	mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, "").Times(burstSize)

	// flush should send one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 2*burstSize, 0.9, limiter)
	mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, limiterSuffix).Times(1)

	// TODO: ROX-17312: Use timer mock and ad-hoc synchronization to avoid sleeping in tests.
	// Avoid concurrency with background logging loop
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 3*burstSize; i++ {
		logDebug()
	}

	time.Sleep(2 * time.Second)

	checkLimiterFlushed(t, rlLogger)
}
