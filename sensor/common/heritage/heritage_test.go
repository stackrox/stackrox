package heritage

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 1,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID: "a",
					PodIP:       "1.1.1.1",
					SensorStart: time.Unix(0, 0),
				},
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
			},
		},
		"Cleanup should not remove anything if less entries than minSize": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 5,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID: "a",
					PodIP:       "1.1.1.1",
					SensorStart: time.Unix(0, 0),
				},
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
			},
		},
		"Max-Size should remove the oldest entry on reverse-sorted slice": {
			args: args{
				// in is reverse-sorted by last-update
				in: []SensorMetadata{
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 0,
				maxSize: 1,
			},
			want: []SensorMetadata{
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
			},
		},
		"Max-Size should remove the oldest entry on sorted slice": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
				},
				now:     time.Unix(999_999, 0),
				maxAge:  0,
				minSize: 0,
				maxSize: 1,
			},
			want: []SensorMetadata{
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
			},
		},
		"Max-Age should remove the oldest entries on sorted slice": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  100 * time.Second,
				minSize: 0,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
			},
		},
		"Max-Age should remove all entries if they are older than max age": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
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
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  5 * time.Second,
				minSize: 1,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
			},
		},
		"Max-Age should sort but remove nothing if all entries are younger than max age": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  500 * time.Second,
				minSize: 0,
				maxSize: 0,
			},
			want: []SensorMetadata{
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
				{
					ContainerID: "a",
					PodIP:       "1.1.1.1",
					SensorStart: time.Unix(0, 0),
				},
			},
		},
		"Sorting order should be by the most recent start timestamp": {
			args: args{
				in: []SensorMetadata{
					{
						ContainerID: "a",
						PodIP:       "1.1.1.1",
						SensorStart: time.Unix(0, 0),
					},
					{
						ContainerID: "b",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
					{
						ContainerID: "c",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(100, 0),
					},
					{
						ContainerID: "d",
						PodIP:       "1.1.1.2",
						SensorStart: time.Unix(200, 0),
					},
				},
				now:     time.Unix(200, 0),
				maxAge:  9999 * time.Second, // we don't want to purge anything in this test case
				minSize: 0,
				maxSize: 5,
			},
			want: []SensorMetadata{
				{
					ContainerID: "d",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(200, 0), // start of `d` is closer to 200 than start of `a`
				},
				{
					ContainerID: "b",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
				{
					ContainerID: "c",
					PodIP:       "1.1.1.2",
					SensorStart: time.Unix(100, 0),
				},
				{
					ContainerID: "a",
					PodIP:       "1.1.1.1",
					SensorStart: time.Unix(0, 0),
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

type dummyWriter struct {
	cmState []*SensorMetadata
}

func (d *dummyWriter) Write(_ context.Context, input ...*SensorMetadata) error {
	d.cmState = input
	return nil
}

func (d *dummyWriter) Read(_ context.Context) ([]*SensorMetadata, error) {
	return d.cmState, nil
}

func newSensorMetadata(cID, podIP string, sensorStart ...time.Time) *SensorMetadata {
	sm := &SensorMetadata{
		ContainerID: cID,
		PodIP:       podIP,
	}
	if len(sensorStart) > 0 {
		sm.SensorStart = sensorStart[0]
	}
	return sm
}

func Test_upsertConfigMap(t *testing.T) {
	now := time.Now()
	a1 := newSensorMetadata("a", "1.1.1.1")
	a2 := newSensorMetadata("a", "1.1.1.2")
	b1 := newSensorMetadata("b", "1.1.1.1")
	b99 := newSensorMetadata("b", "1.1.1.99")

	tests := map[string]struct {
		initialConfigMap       []*SensorMetadata
		currentSensor          *SensorMetadata
		wantWrite              bool
		expectedConfigMapState []*SensorMetadata
	}{
		"Missing podID should result in write to cm": {
			initialConfigMap:       []*SensorMetadata{},
			currentSensor:          a1,
			wantWrite:              true,
			expectedConfigMapState: []*SensorMetadata{a1},
		},
		"Updates only to podID should be treated as new data entry": {
			initialConfigMap:       []*SensorMetadata{a1},
			currentSensor:          a2,
			wantWrite:              true,
			expectedConfigMapState: []*SensorMetadata{a1, a2},
		},
		"Updates only to containerID should be treated as new data entry": {
			initialConfigMap:       []*SensorMetadata{a1},
			currentSensor:          b1,
			wantWrite:              true,
			expectedConfigMapState: []*SensorMetadata{a1, b1},
		},
		"updates on timestamps should result in no writes to cm": {
			initialConfigMap:       []*SensorMetadata{a1},
			currentSensor:          newSensorMetadata("a", "1.1.1.1", time.Unix(9999, 00)),
			wantWrite:              false,
			expectedConfigMapState: []*SensorMetadata{a1},
		},
		"Updates to timestamps on further positions in cache should yield no writes to cm": {
			initialConfigMap:       []*SensorMetadata{b99, a1},
			currentSensor:          newSensorMetadata("a", "1.1.1.1", time.Unix(9999, 00)),
			wantWrite:              false,
			expectedConfigMapState: []*SensorMetadata{b99, a1},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := &dummyWriter{cmState: tt.initialConfigMap}
			hm := newManager(w, tt.currentSensor)
			gotWrite, gotErr := hm.upsertConfigMap(context.Background(), now)
			assert.Equal(t, tt.wantWrite, gotWrite)
			assert.NoError(t, gotErr)

			// Assert on the configMap contents
			gotState, _ := w.Read(t.Context())
			require.Len(t, gotState, len(tt.expectedConfigMapState))
			for i, entry := range gotState {
				assert.Equal(t, tt.expectedConfigMapState[i].PodIP, entry.PodIP)
				assert.Equal(t, tt.expectedConfigMapState[i].ContainerID, entry.ContainerID)
			}
		})
	}
}

func newManager(writer configMapWriter, currentSensor *SensorMetadata) *Manager {
	m := &Manager{
		cacheIsPopulated: atomic.Bool{},
		cache:            make([]*SensorMetadata, 0),
		namespace:        "test",
		currentSensor:    *currentSensor,
		cmWriter:         writer,
		maxSize:          10,
		minSize:          2,
		maxAge:           heritageMaxAge,
	}
	m.cacheIsPopulated.Store(false)
	return m
}
