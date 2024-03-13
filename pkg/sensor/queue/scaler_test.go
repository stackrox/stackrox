package queue

import (
	"os"
	"strconv"
	"testing"

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
		SensorMemLimit    int
		expectedQueueSize int
	}{
		"50% memlimit": {
			inputQueueSize:    100,
			SensorMemLimit:    2097152000,
			expectedQueueSize: 50,
		},
		"50% memlimit - rounding up": {
			inputQueueSize:    5,
			SensorMemLimit:    2097152000,
			expectedQueueSize: 3,
		},
		"200% memlimit": {
			inputQueueSize:    100,
			SensorMemLimit:    8388608000,
			expectedQueueSize: 200,
		},
		"At least size 1": {
			inputQueueSize:    100,
			SensorMemLimit:    1,
			expectedQueueSize: 1,
		},
		"At least size 1 on memlimit 0": {
			inputQueueSize:    100,
			SensorMemLimit:    0,
			expectedQueueSize: 1,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			err := os.Setenv("ROX_MEMLIMIT", strconv.Itoa(c.SensorMemLimit))
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
