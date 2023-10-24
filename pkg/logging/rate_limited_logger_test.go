package logging

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stackrox/rox/pkg/cache/lru"
	lruMocks "github.com/stackrox/rox/pkg/cache/lru/mocks"
	logMocks "github.com/stackrox/rox/pkg/logging/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
)

const (
	testBurstSize     = 3
	testCacheSize     = 500
	testLimiterPeriod = 300 * time.Millisecond
)

func TestRateLimitedLogger(t *testing.T) {
	suite.Run(t, new(rateLimitedLoggerTestSuite))
}

type rateLimitedLoggerTestSuite struct {
	suite.Suite

	mockLogger *logMocks.MockLogger
	rlLogger   *RateLimitedLogger

	testLRU *expirable.LRU[string, *rateLimitedLog]
}

func newTestRateLimitedLogger(_ *testing.T, logger Logger, c lru.LRU[string, *rateLimitedLog]) *RateLimitedLogger {
	testLogger := &RateLimitedLogger{
		logger,
		c,
	}
	runtime.SetFinalizer(testLogger, stopLogger)
	return testLogger
}

func (s *rateLimitedLoggerTestSuite) SetupTest() {
	mockController := gomock.NewController(s.T())
	s.mockLogger = logMocks.NewMockLogger(mockController)
	s.testLRU = expirable.NewLRU[string, *rateLimitedLog](testCacheSize, onEvict, testLimiterPeriod)
	s.rlLogger = newTestRateLimitedLogger(s.T(), s.mockLogger, s.testLRU)
}

func (s *rateLimitedLoggerTestSuite) TearDownTest() {
	s.mockLogger.EXPECT().Logf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	s.rlLogger.rateLimitedLogs.Purge()
	s.rlLogger.stop()
}

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

func getLogCallerLineNum(lineOffset int) int {
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		return 0
	}
	line += lineOffset
	return line
}

const (
	testCallerFile = "pkg/logging/rate_limited_logger_test.go"
)

func getLogCallerPrefix(line int) string {
	return fmt.Sprintf("%s:%d - ", testCallerFile, line)
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

	logLineNum := getLogCallerLineNum(5)
	prefix := getLogCallerPrefix(logLineNum)
	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s%s", prefix, resolvedErrorMsg, "").Times(1)
	for i := 0; i < 3*testBurstSize; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
	}

	s.validateRateLimitedLogCount(3*testBurstSize - 1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsWarnLBurst() {
	limiter := "test limiter"

	logLineNum := getLogCallerLineNum(5)
	prefix := getLogCallerPrefix(logLineNum)
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s%s", prefix, resolvedWarnMsg, "").Times(1)
	for i := 0; i < 3*testBurstSize; i++ {
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
	}

	s.validateRateLimitedLogCount(3*testBurstSize - 1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsInfoLBurst() {
	limiter := "test limiter"

	logLineNum := getLogCallerLineNum(5)
	prefix := getLogCallerPrefix(logLineNum)
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, resolvedInfoMsg, "").Times(1)
	for i := 0; i < 3*testBurstSize; i++ {
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
	}

	s.validateRateLimitedLogCount(3*testBurstSize - 1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsDebugLBurst() {
	limiter := "test limiter"

	lineNum := getLogCallerLineNum(5)
	prefix := getLogCallerPrefix(lineNum)
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix, resolvedDebugMsg, "").Times(1)
	for i := 0; i < 3*testBurstSize; i++ {
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	s.validateRateLimitedLogCount(3*testBurstSize - 1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsSameLimiterDifferentLogs() {
	template1 := "This is a log to be rate limited"
	template2 := "This is another log to be rate limited"
	limiter := "common limiter"

	lineNum := getLogCallerLineNum(2)
	prefix := getLogCallerPrefix(lineNum)
	logInfo := func(info string) { s.rlLogger.InfoL(limiter, info) }

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, template1, "").Times(1)

	for i := 0; i < 2*testBurstSize; i++ {
		logInfo(template1)
	}

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, template2, "").Times(1)

	for i := 0; i < 2*testBurstSize; i++ {
		logInfo(template2)
	}

	s.validateRateLimitedLogCount(2*testBurstSize - 1)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsCacheEviction() {
	evictionTemplate := "This is a log that will be evicted"
	limiter := "limiter"
	lineNum1 := getLogCallerLineNum(2)
	prefix1 := getLogCallerPrefix(lineNum1)
	logInfo := func(info string) { s.rlLogger.InfoL(limiter, info) }
	lineNum2 := getLogCallerLineNum(2)
	prefix2 := getLogCallerPrefix(lineNum2)
	logDebug := func(template string, arg int) { s.rlLogger.DebugL(limiter, template, arg) }

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix1, evictionTemplate, "").Times(1)

	for i := 0; i < 2*testBurstSize; i++ {
		logInfo(evictionTemplate)
	}

	evictionSuffix := fmt.Sprintf(limitedLogSuffixFormat, 2*testBurstSize-1, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix1, evictionTemplate, evictionSuffix).Times(1)

	fillerTemplate := "There are now %d fillers in cache"
	for i := 0; i < cacheSize; i++ {
		expected := fmt.Sprintf(fillerTemplate, i+1)
		s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s%s", prefix2, expected, "").Times(1)
		logDebug(fillerTemplate, i+1)
	}
}

func TestMockedRateLimitedLogger(t *testing.T) {
	suite.Run(t, new(mockedRateLimitedLoggerTestSuite))
}

type mockedRateLimitedLoggerTestSuite struct {
	suite.Suite

	mockLogger *logMocks.MockLogger
	rlLogger   *RateLimitedLogger

	testLRU *lruMocks.MockLRU[string, *rateLimitedLog]
}

func (s *mockedRateLimitedLoggerTestSuite) SetupTest() {
	mockController := gomock.NewController(s.T())
	s.mockLogger = logMocks.NewMockLogger(mockController)
	s.testLRU = lruMocks.NewMockLRU[string, *rateLimitedLog](mockController)
	s.rlLogger = newTestRateLimitedLogger(s.T(), s.mockLogger, s.testLRU)
}

func (s *mockedRateLimitedLoggerTestSuite) TearDownTest() {
	s.mockLogger.EXPECT().Logf(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	s.testLRU.EXPECT().Purge().AnyTimes()
	s.rlLogger.rateLimitedLogs.Purge()
	s.rlLogger.stop()
}

func (s *mockedRateLimitedLoggerTestSuite) TestCacheEvictionRace() {
	limiter := "eviction"
	log := "This is a rate limited log"
	cachedLog := &rateLimitedLog{
		logger:  s.mockLogger,
		id:      "00000000-1111-2222-3333-111111111111",
		level:   zapcore.InfoLevel,
		limiter: limiter,
		payload: log,
		file:    testCallerFile,
		line:    123,
		count:   atomic.Int32{},
	}
	s.testLRU.EXPECT().Get(gomock.Any()).Times(1).Return(cachedLog, true)
	s.testLRU.EXPECT().Get(gomock.Any()).Times(1).Return(nil, true)
	prefix := getLogCallerPrefix(123)
	suffix := ""
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", prefix, log, suffix)
	s.rlLogger.InfoL(limiter, log)
}

func (s *mockedRateLimitedLoggerTestSuite) TestCacheLookupReturnsValidNil() {
	limiter := "eviction"
	log := "This is a rate limited log"
	s.testLRU.EXPECT().Get(gomock.Any()).Times(1).Return(nil, true)
	s.testLRU.EXPECT().Remove(gomock.Any())
	s.testLRU.EXPECT().Add(gomock.Any(), gomock.Any())
	suffix := ""
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s%s", gomock.Any(), log, suffix)
	s.rlLogger.InfoL(limiter, log)
}
