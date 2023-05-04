package logging

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/logging/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
)

const (
	burstSize     = 3
	cacheSize     = 5
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
	s.rlLogger.stop()
}

func (s *rateLimitedLoggerTestSuite) TestBaseFunctions() {
	errorLog := "This is an error log"
	warnLog := "This is a warn log"
	infoLog := "This is an info log"
	debugLog := "This is a debug log"

	s.mockLogger.EXPECT().Error(errorLog, 1)
	s.rlLogger.Error(errorLog, 1)

	s.mockLogger.EXPECT().Warn(3, warnLog)
	s.rlLogger.Warn(3, warnLog)

	s.mockLogger.EXPECT().Info(infoLog, 5, 7)
	s.rlLogger.Info(infoLog, 5, 7)

	s.mockLogger.EXPECT().Debug(9, 2, 4, 6, debugLog)
	s.rlLogger.Debug(9, 2, 4, 6, debugLog)
}

func (s *rateLimitedLoggerTestSuite) TestFormatFunctions() {
	errorTemplateWithoutField := "This is an error template without arg conversion."
	warnTemplateWithoutField := "This is a warn template without arg conversion."
	infoTemplateWithoutField := "This is an info template without arg conversion."
	debugTemplateWithoutField := "This is a debug template without arg conversion."

	s.mockLogger.EXPECT().Errorf(errorTemplateWithoutField)
	s.rlLogger.Errorf(errorTemplateWithoutField)

	s.mockLogger.EXPECT().Warnf(warnTemplateWithoutField)
	s.rlLogger.Warnf(warnTemplateWithoutField)

	s.mockLogger.EXPECT().Infof(infoTemplateWithoutField)
	s.rlLogger.Infof(infoTemplateWithoutField)

	s.mockLogger.EXPECT().Debugf(debugTemplateWithoutField)
	s.rlLogger.Debugf(debugTemplateWithoutField)

	templateWithFields := "This is a template for %s logs with %d arguments to convert"

	s.mockLogger.EXPECT().Errorf(templateWithFields, "error", 2)
	s.rlLogger.Errorf(templateWithFields, "error", 2)

	s.mockLogger.EXPECT().Warnf(templateWithFields, "warn", 2)
	s.rlLogger.Warnf(templateWithFields, "warn", 2)

	s.mockLogger.EXPECT().Infof(templateWithFields, "info", 2)
	s.rlLogger.Infof(templateWithFields, "info", 2)

	s.mockLogger.EXPECT().Debugf(templateWithFields, "debug", 2)
	s.rlLogger.Debugf(templateWithFields, "debug", 2)
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsBurst() {
	templateWithFields := "This is a template for %s logs with %d arguments to convert"
	limiter := "test limiter"

	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)

	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, "").Times(burstSize)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, "").Times(burstSize)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, "").Times(burstSize)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, "").Times(burstSize)

	for i := 0; i < 3*burstSize; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	cacheKeys := s.rlLogger.rateLimitedLogs.Keys()
	for _, k := range cacheKeys {
		v, f := s.rlLogger.rateLimitedLogs.Peek(k)
		s.True(f)
		s.NotNil(v)
		expectedCount := atomic.Int32{}
		expectedCount.Swap(2 * burstSize)
		if v != nil {
			s.Equal(expectedCount.Load(), v.count.Load())
		}
	}
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsCooldown() {
	templateWithFields := "This is a template for %s logs with %d arguments to convert"
	limiter := "test limiter"

	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)

	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, "").Times(2)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, "").Times(2)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, "").Times(2)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, "").Times(2)

	for i := 0; i < 2; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	time.Sleep(LoggingInterval)

	// Burst limit should allow one more trace
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, "").Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, "").Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, "").Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, "").Times(1)

	for i := 0; i < 2; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	time.Sleep(LoggingInterval)

	// Burst limit should not allow any trace
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, "").Times(0)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, "").Times(0)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, "").Times(0)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, "").Times(0)

	for i := 0; i < 2; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	time.Sleep(LoggingInterval)

	// Rate limiter should allow one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 4, 0.2, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, limiterSuffix).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, limiterSuffix).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, limiterSuffix).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, limiterSuffix).Times(1)

	for i := 0; i < 2; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	cacheKeys := s.rlLogger.rateLimitedLogs.Keys()
	for _, k := range cacheKeys {
		v, f := s.rlLogger.rateLimitedLogs.Peek(k)
		s.True(f)
		s.NotNil(v)
		expectedCount := atomic.Int32{}
		expectedCount.Swap(1)
		if v != nil {
			s.Equal(expectedCount.Load(), v.count.Load())
		}
	}
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsSameLimiterDifferentLogs() {
	template1 := "This is a log to be rate limited"
	template2 := "This is another log to be rate limited"
	limiter := "common limiter"

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", template1, "").Times(burstSize)

	for i := 0; i < 2*burstSize; i++ {
		s.rlLogger.InfoL(limiter, template1)
	}

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", template2, "").Times(burstSize)

	for i := 0; i < 2*burstSize; i++ {
		s.rlLogger.InfoL(limiter, template2)
	}

}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsCacheEviction() {
	evictionTemplate := "This is a log that will be evicted"
	limiter := "limiter"

	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", evictionTemplate, "").Times(burstSize)

	for i := 0; i < 2*burstSize; i++ {
		s.rlLogger.InfoL(limiter, evictionTemplate)
	}

	evictionSuffix := fmt.Sprintf(limitedLogSuffixFormat, burstSize, 0.0, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", evictionTemplate, evictionSuffix).Times(1)

	fillerTemplate := "There are now %d fillers in cache"
	for i := 0; i < cacheSize; i++ {
		expected := fmt.Sprintf(fillerTemplate, i+1)
		s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", expected, "").Times(1)
		s.rlLogger.DebugL(limiter, fillerTemplate, i+1)
	}
}

func (s *rateLimitedLoggerTestSuite) TestRateLimitedFunctionsTimedFlush() {

	templateWithFields := "This is a template for %s logs with %d arguments to convert"
	limiter := "test limiter"

	resolvedErrorMsg := fmt.Sprintf(templateWithFields, "error", 2)
	resolvedWarnMsg := fmt.Sprintf(templateWithFields, "warn", 2)
	resolvedInfoMsg := fmt.Sprintf(templateWithFields, "info", 2)
	resolvedDebugMsg := fmt.Sprintf(templateWithFields, "debug", 2)

	time.Sleep(100 * time.Millisecond)

	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, "").Times(burstSize)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, "").Times(burstSize)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, "").Times(burstSize)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, "").Times(burstSize)

	for i := 0; i < 3*burstSize; i++ {
		s.rlLogger.ErrorL(limiter, templateWithFields, "error", 2)
		s.rlLogger.WarnL(limiter, templateWithFields, "warn", 2)
		s.rlLogger.InfoL(limiter, templateWithFields, "info", 2)
		s.rlLogger.DebugL(limiter, templateWithFields, "debug", 2)
	}

	// flush should send one trace
	limiterSuffix := fmt.Sprintf(limitedLogSuffixFormat, 2*burstSize, 0.9, limiter)
	s.mockLogger.EXPECT().Logf(zapcore.ErrorLevel, "%s%s", resolvedErrorMsg, limiterSuffix).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "%s%s", resolvedWarnMsg, limiterSuffix).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "%s%s", resolvedInfoMsg, limiterSuffix).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.DebugLevel, "%s%s", resolvedDebugMsg, limiterSuffix).Times(1)

	time.Sleep(2 * time.Second)

	cacheKeys := s.rlLogger.rateLimitedLogs.Keys()
	for _, k := range cacheKeys {
		v, f := s.rlLogger.rateLimitedLogs.Peek(k)
		s.True(f)
		s.NotNil(v)
		expectedCount := atomic.Int32{}
		expectedCount.Swap(0)
		if v != nil {
			s.Equal(expectedCount.Load(), v.count.Load())
		}
	}
}
