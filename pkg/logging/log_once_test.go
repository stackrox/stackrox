package logging

import (
	"strconv"
	"testing"

	logMocks "github.com/stackrox/rox/pkg/logging/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
)

func TestLogOnce(t *testing.T) {
	suite.Run(t, new(logOnceTestSuite))
}

type logOnceTestSuite struct {
	suite.Suite
	mockLogger *logMocks.MockLogger
	realLogger Logger
}

func (s *logOnceTestSuite) SetupSuite() {
	s.realLogger = LoggerForModule()
}

func (s *logOnceTestSuite) SetupTest() {
	mockController := gomock.NewController(s.T())
	s.mockLogger = logMocks.NewMockLogger(mockController)
	logOnceSeen.Clear()
	logOnceMemoryUsed.Store(0)
}

func (s *logOnceTestSuite) TestLogOncef() {
	s.mockLogger.EXPECT().Logf(zapcore.WarnLevel, "hello world %d, %s", 6, "yes!").MinTimes(1).MaxTimes(1)

	LogOncef(s.mockLogger, zapcore.WarnLevel, "hello world %d, %s", 6, "yes!")
	LogOncef(s.mockLogger, zapcore.WarnLevel, "hello world %d, %s", 500, "really?")
}

func (s *logOnceTestSuite) TestLogOncePerKeyf() {
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "This sensor %s is unealthy %d seconds", "sensor 1", 4).MinTimes(1).MaxTimes(1)
	s.mockLogger.EXPECT().Logf(zapcore.InfoLevel, "This sensor %s is unealthy %d seconds", "sensor 2", 1).MinTimes(1).MaxTimes(1)

	LogOncePerKeyf("sensor 1", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unealthy %d seconds", "sensor 1", 4)
	LogOncePerKeyf("sensor 1", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unealthy %d seconds", "sensor 1", 34)

	LogOncePerKeyf("sensor 2", s.mockLogger, zapcore.InfoLevel, "This sensor %s is unealthy %d seconds", "sensor 2", 1)
}

func (s *logOnceTestSuite) TestLogOncefSizeLimit() {
	testCount := maxLogOnceMemory * 2
	s.mockLogger.EXPECT().Logf(gomock.Any(), gomock.Any()).AnyTimes()
	for i := range testCount {
		LogOncef(s.mockLogger, zapcore.WarnLevel, "test message "+strconv.Itoa(i))
	}

	s.Equal(int64(maxLogOnceMemory), logOnceMemoryUsed.Load())

	inMap := 0
	logOnceSeen.Range(func(_ any, _ any) bool {
		inMap++
		return true
	})

	s.Equal(maxLogOnceMemory, inMap)
}

func (s *logOnceTestSuite) TestLogOncefReal() {
	LogOncef(s.realLogger, zapcore.InfoLevel, "this message is only logged once")
	LogOncef(s.realLogger, zapcore.InfoLevel, "this message is only logged once")
}
