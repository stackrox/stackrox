package logging

import (
	"strconv"
	"testing"

	logMocks "github.com/stackrox/rox/pkg/logging/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
)

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

func (s *logOnceTestSuite) TestLogOncef() {
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "hello world %d, %s", 6, "yes!").MinTimes(1).MaxTimes(1)

	LogOncef(s.mockLogger, zapcore.WarnLevel, "hello world %d, %s", 6, "yes!")
	LogOncef(s.mockLogger, zapcore.WarnLevel, "hello world %d, %s", 500, "really?")
}

func (s *logOnceTestSuite) TestLogOncePerKeyf() {
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 1", 4).MinTimes(1).MaxTimes(1)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 2", 1).MinTimes(1).MaxTimes(1)

	LogOncePerKeyf("sensor 1", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 1", 4)
	LogOncePerKeyf("sensor 1", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 1", 34)

	LogOncePerKeyf("sensor 2", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unhealthy %d seconds", "sensor 2", 1)
}

func (s *logOnceTestSuite) TestLogOncefSizeLimit() {
	testCount := maxLogOnceMemory * 2
	s.mockLogger.EXPECT().Logf(gomock.Any(), gomock.Any()).AnyTimes()
	s.mockLogger.EXPECT().Warnf("maxLogOnceMemory=%d limit reached", maxLogOnceMemory).MinTimes(1).MaxTimes(1)
	for i := range testCount {
		LogOncef(s.mockLogger, zapcore.WarnLevel, "test message "+strconv.Itoa(i)) //nolint:govet
	}

	s.Equal(int64(maxLogOnceMemory), logOnceMemoryUsed.Load())

	countInMap := 0
	logOnceSeen.Range(func(_ any, _ any) bool {
		countInMap++
		return true
	})

	s.Equal(maxLogOnceMemory, countInMap)
}

func clearMemory() {
	logOnceSeen.Clear()
	logOnceMemoryUsed.Store(0)
	logOnceLimitNotified.Store(false)
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
