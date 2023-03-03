package intervals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type IntervalsTestSuite struct {
	suite.Suite
	randFloat64 func() float64
}

func TestScanSuite(t *testing.T) {
	suite.Run(t, new(IntervalsTestSuite))
}

func (s *IntervalsTestSuite) SetupSuite() {
	s.randFloat64 = randFloat64
}

func (s *IntervalsTestSuite) TearDownTest() {
	randFloat64 = s.randFloat64
}

func (s *IntervalsTestSuite) Test_Next() {
	defaultIntervals := NodeScanIntervals{
		base:      time.Hour * 10,
		deviation: 0.1,
	}
	tests := []struct {
		name      string
		intervals NodeScanIntervals
		rand      float64
		want      time.Duration
	}{
		{
			name:      "default interval with rand 0",
			intervals: defaultIntervals,
			rand:      0,
			want:      time.Hour * 9,
		},
		{
			name:      "default interval with rand 1",
			intervals: defaultIntervals,
			rand:      1,
			want:      time.Hour * 11,
		},
		{
			name:      "default interval with rand 0.5",
			intervals: defaultIntervals,
			rand:      0.25,
			want:      time.Hour*9 + time.Minute*30,
		},
		{
			name: "deviation set to 0",
			intervals: NodeScanIntervals{
				deviation: 0,
				base:      time.Hour,
			},
			rand: 0,
			want: time.Hour,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			randFloat64 = func() float64 { return tt.rand }
			got := tt.intervals.Next()
			s.Assert().Equal(tt.want, got)
		})
	}
}
