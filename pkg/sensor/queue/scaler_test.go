package queue

import (
	"os"
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
		expectedQueueSize int
	}{
		"50% memlimit": {
			inputQueueSize:    100,
			sensorMemLimit:    2097152000,
			expectedQueueSize: 50,
		},
		"50% memlimit - rounding up": {
			inputQueueSize:    5,
			sensorMemLimit:    2097152000,
			expectedQueueSize: 3,
		},
		"200% memlimit": {
			inputQueueSize:    100,
			sensorMemLimit:    8388608000,
			expectedQueueSize: 200,
		},
		"At least size 1": {
			inputQueueSize:    100,
			sensorMemLimit:    1,
			expectedQueueSize: 1,
		},
		"At least size 1 on memlimit 0": {
			inputQueueSize:    100,
			sensorMemLimit:    0,
			expectedQueueSize: 1,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			err := os.Setenv("ROX_MEMLIMIT", strconv.Itoa(c.sensorMemLimit))
			s.NoError(err)

			actual, err := ScaleSize(c.inputQueueSize)
			s.NoError(err)
			s.Equal(c.expectedQueueSize, actual)
		})
	}
}

func (s *scalerTestSuite) TestScaleSizeEnvConversion() {
	err := os.Setenv("ROX_MEMLIMIT", "definitelyNotAnInteger")
	s.NoError(err)

	_, err = ScaleSize(100)
	s.ErrorContains(err, "strconv.ParseInt: parsing")
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
			sensorMemLimit: 2097152000,
			expected:       50,
		},
		"Don't scale non default": {
			setting:        env.RegisterIntegerSetting("TEST_1", 100),
			setValue:       42,
			sensorMemLimit: 2097152000,
			expected:       42,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			err := os.Setenv("ROX_MEMLIMIT", strconv.Itoa(c.sensorMemLimit))
			s.NoError(err)
			err = os.Setenv("TEST_1", strconv.Itoa(c.setValue))
			s.NoError(err)

			s.Equal(c.expected, ScaleSizeOnNonDefault(c.setting))
		})
	}
}
