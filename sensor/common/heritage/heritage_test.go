package heritage

import (
	"reflect"
	"testing"
	"time"
)

func Test_cleanupHeritageData(t *testing.T) {
	type args struct {
		in      []PastSensor
		now     time.Time
		maxAge  time.Duration
		minSize int
		maxSize int
	}
	tests := map[string]struct {
		args args
		want []PastSensor
	}{
		"Cleanup disabled should not remove anything": {
			args: args{
				in: []PastSensor{
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
			want: []PastSensor{
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
				in: []PastSensor{
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
			want: []PastSensor{
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
				in: []PastSensor{
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
			want: []PastSensor{
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
				in: []PastSensor{
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
			want: []PastSensor{
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
				in: []PastSensor{
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
			want: []PastSensor{
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
				in: []PastSensor{
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
			want: []PastSensor{},
		},
		"Max-Age should remove all entries, but  minSize of latest entries should be kept": {
			args: args{
				in: []PastSensor{
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
			want: []PastSensor{
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
				in: []PastSensor{
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
			want: []PastSensor{
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
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			input := make([]*PastSensor, len(tt.args.in))
			// copy to convert []PastSensor into []*PastSensor
			for i, entry := range tt.args.in {
				input[i] = &entry
			}
			got := pruneOldHeritageData(input, tt.args.now, tt.args.maxAge, tt.args.minSize, tt.args.maxSize)
			gotValues := make([]PastSensor, len(got))
			// Copy to convert []*PastSensor into []PastSensor, for DeepEqual assertion
			for i, entry := range got {
				gotValues[i] = *entry
			}
			if !reflect.DeepEqual(gotValues, tt.want) {
				t.Errorf("\ngot  = %v\nwant = %v", got, tt.want)
			}
		})
	}
}
