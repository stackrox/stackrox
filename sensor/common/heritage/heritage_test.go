package heritage

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_cleanupHeritageData(t *testing.T) {
	type args struct {
		in      []SensorMetadata
		now     time.Time
		maxAge  time.Duration
		minSize int
		maxSize int
	}
	tests := map[string]struct {
		args args
		want []SensorMetadata
	}{
		"Cleanup disabled should not remove anything": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 1,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "a",
					PodIP:        "1.1.1.1",
					SensorStart:  time.Unix(0, 0),
					LatestUpdate: time.Unix(10, 0),
				},
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
			},
		},
		"Cleanup should not remove anything if less entries than minSize": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 5,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "a",
					PodIP:        "1.1.1.1",
					SensorStart:  time.Unix(0, 0),
					LatestUpdate: time.Unix(10, 0),
				},
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
			},
		},
		"Max-Size should remove the oldest entry on reverse-sorted slice": {
			args: args{
				// in is reverse-sorted by last-update
				in: []SensorMetadata{
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 0,
				maxSize: 1,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
			},
		},
		"Max-Size should remove the oldest entry on sorted slice": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 0,
				maxSize: 1,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
			},
		},
		"Max-Age should remove the oldest entries on sorted slice": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  100 * time.Second,
				minSize: 0,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
			},
		},
		"Max-Age should remove all entries if they are older than max age": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  5 * time.Second,
				minSize: 0,
				maxSize: 0,
			},
			want: []SensorMetadata{},
		},
		"Max-Age should remove all entries, but  minSize of latest entries should be kept": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  5 * time.Second,
				minSize: 1,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
			},
		},
		"Max-Age should sort but remove nothing if all entries are younger than max age": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(110, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  500 * time.Second,
				minSize: 0,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(110, 0),
				},
				{
					ContainerID:  "a",
					PodIP:        "1.1.1.1",
					SensorStart:  time.Unix(0, 0),
					LatestUpdate: time.Unix(10, 0),
				},
			},
		},
		"Sorting order should be by the most recently updated, then by most recently start timestamp": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID:  "a",
						PodIP:        "1.1.1.1",
						SensorStart:  time.Unix(0, 0),
						LatestUpdate: time.Unix(10, 0),
					},
					{
						ContainerID:  "b",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(5, 0),
					},
					{
						ContainerID:  "c",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(100, 0),
						LatestUpdate: time.Unix(7, 0),
					},
					{
						ContainerID:  "d",
						PodIP:        "1.1.1.2",
						SensorStart:  time.Unix(200, 0),
						LatestUpdate: time.Unix(10, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  9999 * time.Second, // we don't want to purge anything in this test case
				minSize: 0,
				maxSize: 5,
			},
			want: []SensorMetadata{
				{
					ContainerID:  "d",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(200, 0), // start of `d` is closer to 200 than start of `a`
					LatestUpdate: time.Unix(10, 0),  // tie between `d` and `a`
				},
				{
					ContainerID:  "a",
					PodIP:        "1.1.1.1",
					SensorStart:  time.Unix(0, 0),
					LatestUpdate: time.Unix(10, 0),
				},
				{
					ContainerID:  "c",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(7, 0),
				},
				{
					ContainerID:  "b",
					PodIP:        "1.1.1.2",
					SensorStart:  time.Unix(100, 0),
					LatestUpdate: time.Unix(5, 0),
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			input := make([]*SensorMetadata, len(tt.args.in))
			// copy to convert []SensorMetadata into []*SensorMetadata
			for i, entry := range tt.args.in {
				input[i] = &entry
			}
			got := pruneOldHeritageData(input, tt.args.now, tt.args.maxAge, tt.args.minSize, tt.args.maxSize)
			gotValues := make([]SensorMetadata, len(got))
			// Copy to convert []*SensorMetadata into []SensorMetadata, for DeepEqual assertion
			for i, entry := range got {
				gotValues[i] = *entry
			}
			if !reflect.DeepEqual(gotValues, tt.want) {
				t.Errorf("\ngot  = %v\nwant = %v", sensorMetadataString(got), tt.want)
			}
		})
	}
}

type dummyWriter struct{}

func (d *dummyWriter) Write(_ context.Context, _ ...*SensorMetadata) error {
	return nil
}

func (d *dummyWriter) Read(_ context.Context) ([]*SensorMetadata, error) {
	return []*SensorMetadata{}, nil
}

func Test_writeRateLimiter(t *testing.T) {
	data := []*SensorMetadata{
		{
			ContainerID:  "a",
			PodIP:        "1.1.1.1",
			SensorStart:  time.Unix(0, 0),
			LatestUpdate: time.Unix(10, 0),
		},
		{
			ContainerID:  "b",
			PodIP:        "1.1.1.2",
			SensorStart:  time.Unix(100, 0),
			LatestUpdate: time.Unix(110, 0),
		},
	}
	tests := map[string]struct {
		lastWrite     time.Time
		frequency     time.Duration
		now           time.Time
		cacheHit      bool
		writeExpected bool
	}{
		"Write after 5s should not trigger 1s rate limit": {
			lastWrite:     time.Unix(0, 0),
			frequency:     time.Second,
			now:           time.Unix(5, 0),
			cacheHit:      true,
			writeExpected: true,
		},
		"Write after 5s should trigger 10s rate limit": {
			lastWrite:     time.Unix(0, 0),
			frequency:     10 * time.Second,
			now:           time.Unix(5, 0),
			cacheHit:      true,
			writeExpected: false,
		},
		"Write after 5s should not trigger 10s rate limit on cache-miss": {
			lastWrite:     time.Unix(0, 0),
			frequency:     10 * time.Second,
			now:           time.Unix(5, 0),
			cacheHit:      false,
			writeExpected: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hm := newManager(t, &dummyWriter{}, data, tt.lastWrite, tt.frequency)
			if tt.cacheHit {
				hm.currentSensor.PodIP = "1.1.1.1"
				hm.currentSensor.ContainerID = "a"
			} else {
				hm.currentSensor.PodIP = "1.1.1.199"
				hm.currentSensor.ContainerID = "z"
			}
			gotWrite, gotErr := hm.write(context.Background(), tt.now)
			assert.NoError(t, gotErr)
			assert.Equal(t, tt.writeExpected, gotWrite)
		})
	}
}

func newManager(_ *testing.T, writer configMapWriter, cache []*SensorMetadata, lastWrite time.Time, freq time.Duration) *Manager {
	return &Manager{
		cacheIsPopulated: atomic.Bool{},
		cache:            cache,
		namespace:        "test",
		currentSensor: SensorMetadata{
			SensorVersion: "1.0.0-test",
			SensorStart:   time.Unix(0, 0),
		},
		cmWriter:         writer,
		lastCmWrite:      lastWrite,
		writeCmFrequency: freq,
		maxSize:          10,
		minSize:          2,
		maxAge:           heritageMaxAge,
	}
}
