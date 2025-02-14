package conversion

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

////////////////////////////////////////////////////////////////////////////////
// From pkg/postgres/pgutils/utils.go                                         //
//                                                                            //

// convertEnumSliceToIntArray converts an enum slice into a Postgres intarray
func convertEnumSliceToIntArray[T ~int32](enumSlice []T) []int32 {
	resultSlice := make([]int32, 0, len(enumSlice))
	for _, v := range enumSlice {
		resultSlice = append(resultSlice, int32(v))
	}
	return resultSlice
}

////////////////////////////////////////////////////////////////////////////////
// From pkg/protocompat/time.go                                               //
//                                                                            //

// nilOrTime allows for a proto timestamp to be stored a timestamp type in Postgres
func nilOrTime(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	ts, err := convertTimestampToTimeOrError(t)
	if err != nil {
		return nil
	}
	ts = ts.Round(time.Microsecond)
	return &ts
}

// convertTimestampToTimeOrError converts a proto timestamp
// to a golang Time, or returns an error if there is one.
func convertTimestampToTimeOrError(pbTime *timestamppb.Timestamp) (time.Time, error) {
	return pbTime.AsTime(), pbTime.CheckValid()
}
