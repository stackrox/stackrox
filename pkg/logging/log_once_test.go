package logging

import (
	"strconv"
	"testing"

	logMocks "github.com/stackrox/rox/pkg/logging/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
)

// TestLogOncefReal and TestLogOncePerKeyfReal demonstrate the usage of LogOncef and LogOncePerKeyf. The use real
// logger so you can run them and see what actually gets written to the output.
func TestLogOncefReal(t *testing.T) {
	logger := LoggerForModule()
	LogOncef(logger, zapcore.InfoLevel, "this message is only %s", "logged once")
	LogOncef(logger, zapcore.InfoLevel, "this message is only %s", "logged once")
}

func TestLogOncePerKeyfReal(t *testing.T) {
	logger := LoggerForModule()
	LogOncePerKeyf("key1", logger, zapcore.InfoLevel, "this message is only logged once per %s", "key1")
	LogOncePerKeyf("key2", logger, zapcore.InfoLevel, "this message is only logged once per %s", "key2")
	LogOncePerKeyf("key1", logger, zapcore.InfoLevel, "this message is only logged once per %s", "key1")
}

func TestLogOnce(t *testing.T) {
	suite.Run(t, new(logOnceTestSuite))
}

type logOnceTestSuite struct {
	suite.Suite
	mockLogger *logMocks.MockLogger
}

func (s *logOnceTestSuite) SetupTest() {
	mockController := gomock.NewController(s.T())
	s.mockLogger = logMocks.NewMockLogger(mockController)
	clearMemory()
}

func (s *logOnceTestSuite) TearDownTest() {
	clearMemory()
}

var _ suite.SetupTestSuite = (*logOnceTestSuite)(nil)
var _ suite.TearDownTestSuite = (*logOnceTestSuite)(nil)

func clearMemory() {
	logOnceSeen.Clear()
	logOnceMemoryUsed.Store(0)
	logOnceLimitNotified.Store(false)
	logOnceMaxMemory = logOnceDefaultMaxMemory
}

func (s *logOnceTestSuite) TestLogOncef() {
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "hello world %d, %s", 6, "yes!").Times(1)

	LogOncef(s.mockLogger, zapcore.WarnLevel, "hello world %d, %s", 6, "yes!")
	LogOncef(s.mockLogger, zapcore.PanicLevel, "hello world %d, %s", 500, "really?")
}

func (s *logOnceTestSuite) TestLogOncePerKeyf() {
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 1", 4).Times(1)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 2", 1).Times(1)

	LogOncePerKeyf("sensor 1", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 1", 4)
	LogOncePerKeyf("sensor 1", s.mockLogger, zapcore.WarnLevel, "This sensor %s is unhealthy %d seconds", "sensor 1", 34)

	LogOncePerKeyf("sensor 2", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 2", 1)
}

func (s *logOnceTestSuite) TestLogOncefSizeLimit() {
	s.mockLogger.EXPECT().Logf(gomock.Any(), gomock.Any()).AnyTimes()

	var capturedWarning string
	s.mockLogger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Do(func(fmt any, _ ...any) {
		capturedWarning = fmt.(string)
	}).Times(1)

	testCount := logOnceDefaultMaxMemory * 2
	for i := range testCount {
		LogOncef(s.mockLogger, zapcore.WarnLevel, "test message "+strconv.Itoa(int(i))) //nolint:govet
	}

	s.Regexp("logOnceMaxMemory.* limit reached", capturedWarning)

	s.Equal(logOnceDefaultMaxMemory, logOnceMemoryUsed.Load())

	var countInMap int32 = 0
	logOnceSeen.Range(func(_ any, _ any) bool {
		countInMap++
		return true
	})

	s.Equal(logOnceDefaultMaxMemory, countInMap)
}

func (s *logOnceTestSuite) TestGetLogOnceMaxMemory() {
	s.Run("default", func() {
		s.Assert().Equal((int32)(10000), getLogOnceMaxMemory())
	})

	cases := map[string]int32{
		"15":             15,
		"2000000000":     2000000000,
		"3000000000":     10000,
		"-6":             10000,
		"plenty, maybe?": 10000,
		"":               10000,
	}

	for input, expected := range cases {
		s.Run(input, func() {
			s.T().Setenv("ROX_MAX_LOG_ONCE_MEMORY", input)
			s.Assert().Equal(expected, getLogOnceMaxMemory())
		})
	}
}

func BenchmarkLogOnce(b *testing.B) {
	logger := LoggerForModule()

	b.Run("including cold start", func(b *testing.B) {
		clearMemory()
		b.ResetTimer()
		for i := range b.N {
			LogOncef(logger, zapcore.InfoLevel, "first benchmark message %d", i)
		}
	})

	b.Run("when already seen", func(b *testing.B) {
		clearMemory()
		LogOncef(logger, zapcore.InfoLevel, "second benchmark message %d", 0)
		b.ResetTimer()
		for i := range b.N {
			LogOncef(logger, zapcore.InfoLevel, "second benchmark message %d", i)
		}
	})

	b.Run("when already seen - parallel", func(b *testing.B) {
		clearMemory()
		LogOncef(logger, zapcore.InfoLevel, "third benchmark message %d", 0)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				LogOncef(logger, zapcore.InfoLevel, "third benchmark message %d", 0)
			}
		})
	})

	b.Run("with key", func(b *testing.B) {
		clearMemory()
		LogOncePerKeyf("mykey", logger, zapcore.InfoLevel, "fourth benchmark message %d", 0)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				LogOncePerKeyf("mykey", logger, zapcore.InfoLevel, "fourth benchmark message %d", 0)
			}
		})
	})
}
