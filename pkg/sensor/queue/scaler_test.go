package queue

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

type scalerTestSuite struct {
	suite.Suite
}

func TestBroker(t *testing.T) {
	suite.Run(t, &scalerTestSuite{})
}

func (s *scalerTestSuite) TestScaleSize() {
	cases := map[string]struct {
		inputQueueSize    int
		sensorMemLimit    int
		bufferCeiling     int
		expectedQueueSize int
	}{
		"50% memlimit": {
			inputQueueSize:    100,
			sensorMemLimit:    int(defaultMemlimit * 0.5),
			expectedQueueSize: 50,
		},
		"50% memlimit - rounding up": {
			inputQueueSize:    5,
			sensorMemLimit:    int(defaultMemlimit * 0.5),
			expectedQueueSize: 3,
		},
		"200% memlimit": {
			inputQueueSize:    100,
			sensorMemLimit:    int(defaultMemlimit * 2),
			expectedQueueSize: 200,
		},
		"At least size 1": {
			inputQueueSize:    100,
			sensorMemLimit:    1,
			expectedQueueSize: 1,
		},
		"Default on memlimit 0": {
			inputQueueSize:    100,
			sensorMemLimit:    0,
			expectedQueueSize: 100,
		},
		"Upper limit hit": {
			inputQueueSize:    100,
			sensorMemLimit:    int(defaultMemlimit * 10),
			expectedQueueSize: 300,
		},
		"Custom upper limit": {
			inputQueueSize:    100,
			sensorMemLimit:    int(defaultMemlimit * 10),
			bufferCeiling:     5,
			expectedQueueSize: 500,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			s.T().Setenv("ROX_MEMLIMIT", strconv.Itoa(c.sensorMemLimit))
			if c.bufferCeiling != 0 {
				s.T().Setenv("ROX_SENSOR_BUFFER_SCALE_CEILING", strconv.Itoa(c.bufferCeiling))
			}

			actual, err := ScaleSize(c.inputQueueSize)

			if err != nil {
				switch {
				case s.ErrorContains(err, "ROX_MEMLIMIT is set to 0"):
					break
				default:
					s.Failf("Encountered unexpected error: %v", err.Error())
				}
			}
			s.Equal(c.expectedQueueSize, actual)
		})
	}
}

func (s *scalerTestSuite) TestScaleSizeEnvConversion() {
	s.T().Setenv("ROX_MEMLIMIT", "definitelyNotAnInteger")

	actual, err := ScaleSize(100)
	s.ErrorContains(err, "strconv.ParseInt: parsing")
	s.Equal(100, actual, "ScaleSize must return the input on error")
}

func (s *scalerTestSuite) TestScaleSizeOnNonDefault() {
	cases := map[string]struct {
		setting        *env.IntegerSetting
		setValue       int
		sensorMemLimit int
		expected       int
	}{
		"Scaling env var 50%": {
			setting:        env.RegisterIntegerSetting("TEST_1", 100),
			setValue:       100,
			sensorMemLimit: int(defaultMemlimit * 0.5),
			expected:       50,
		},
		"Don't scale non default": {
			setting:        env.RegisterIntegerSetting("TEST_1", 100),
			setValue:       42,
			sensorMemLimit: int(defaultMemlimit * 0.5),
			expected:       42,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			s.T().Setenv("ROX_MEMLIMIT", strconv.Itoa(c.sensorMemLimit))
			s.T().Setenv("TEST_1", strconv.Itoa(c.setValue))

			s.Equal(c.expected, ScaleSizeOnNonDefault(c.setting))
		})
	}
}
