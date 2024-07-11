package intervals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestNewNodeScanIntervalFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		envInterval   string
		envDeviation  string
		envMaxInitial string
		want          NodeScanIntervals
	}{
		{
			name:          "when interval, deviation and initial are set then interval is set",
			envInterval:   "1h",
			envDeviation:  "30m",
			envMaxInitial: "1m",
			want: NodeScanIntervals{
				base:       time.Hour,
				deviation:  0.5,
				initialMax: time.Minute,
			},
		},
		{
			name:         "when deviation greater or equal to interval",
			envInterval:  "4h",
			envDeviation: "4h",
			want: NodeScanIntervals{
				base:       time.Hour * 4,
				deviation:  1,
				initialMax: time.Minute * 5,
			},
		},
	}
	for _, tt := range tests {
		t.Setenv("ROX_NODE_SCANNING_INTERVAL", tt.envInterval)
		t.Setenv("ROX_NODE_SCANNING_INTERVAL_DEVIATION", tt.envDeviation)
		t.Setenv("ROX_NODE_SCANNING_MAX_INITIAL_WAIT", tt.envMaxInitial)
		t.Run(tt.name, func(t *testing.T) {
			got := NewNodeScanIntervalFromEnv()
			assert.Exactly(t, got, tt.want)
		})
	}
}

func (s *IntervalsTestSuite) TestNodeScanIntervals_Initial() {
	i := &NodeScanIntervals{
		initialMax: time.Minute,
	}
	randFloat64 = func() float64 { return 0.5 }
	s.Equal(time.Second*30, i.Initial())
}
